/*
Copyright 2019 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package sink

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"time"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	pipelineclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	triggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"

	"github.com/tektoncd/triggers/pkg/template"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"golang.org/x/xerrors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	discoveryclient "k8s.io/client-go/discovery"
	restclient "k8s.io/client-go/rest"
)

type Resource struct {
	TriggersClient         triggersclientset.Interface
	DiscoveryClient        discoveryclient.DiscoveryInterface
	RESTClient             restclient.Interface
	PipelineClient         pipelineclientset.Interface
	EventListenerName      string
	EventListenerNamespace string
}

const (
	// TODO: To be configurable in future
	taskRunPollingInterval = 10 * time.Second
	taskRunPollingRetries  = 5
)

func (r Resource) HandleEvent(response http.ResponseWriter, request *http.Request) {
	el, err := r.TriggersClient.TektonV1alpha1().EventListeners(r.EventListenerNamespace).Get(r.EventListenerName, metav1.GetOptions{})
	if err != nil {
		log.Printf("Error getting EventListener %s in Namespace %s: %s", r.EventListenerName, r.EventListenerNamespace, err)
		return
	}

	event, err := ioutil.ReadAll(request.Body)
	if err != nil {
		log.Printf("Error reading event body: %s", err)
		return
	}

	log.Printf("EventListener: %s in Namespace: %s handling event with payload: %s and header: %v",
		r.EventListenerName, r.EventListenerNamespace, string(event), request.Header)
	// Execute each Trigger
	for _, trigger := range el.Spec.Triggers {
		go r.executeTrigger(event, request.Header, trigger)
	}
	fmt.Fprintf(response, "EventListener: %s in Namespace: %s handling event with payload: %s and header: %v",
		r.EventListenerName, r.EventListenerNamespace, string(event), request.Header)
}

func (r Resource) executeTrigger(payload []byte, header http.Header, trigger triggersv1.EventListenerTrigger) {
	// Secure Endpoint
	if trigger.TriggerValidate != nil {
		if err := r.validateEvent(trigger.TriggerValidate, header, payload); err != nil {
			log.Printf("Error securing Endpoint for TriggerBinding %s in Namespace %s: %s", trigger.Binding.Name, r.EventListenerNamespace, err)
			return
		}
	}

	binding, err := template.ResolveBinding(trigger,
		r.TriggersClient.TektonV1alpha1().TriggerBindings(r.EventListenerNamespace).Get,
		r.TriggersClient.TektonV1alpha1().TriggerTemplates(r.EventListenerNamespace).Get)
	if err != nil {
		log.Print(err)
		return
	}
	resources, err := template.NewResources(payload, trigger.Params, binding)
	if err != nil {
		log.Print(err)
		return
	}
	err = createResources(resources, r.RESTClient, r.DiscoveryClient, r.EventListenerNamespace, r.EventListenerName)
	if err != nil {
		log.Print(err)
	}
}

func createResources(resources []json.RawMessage, restClient restclient.Interface, discoveryClient discoveryclient.DiscoveryInterface, eventListenerNamespace string, eventListenerName string) error {
	for _, resource := range resources {
		if err := createResource(resource, restClient, discoveryClient, eventListenerNamespace, eventListenerName); err != nil {
			return err
		}
	}
	return nil
}

// createResource uses the kubeClient to create the resource defined in the
// TriggerResourceTemplate and returns any errors with this process
func createResource(rt json.RawMessage, restClient restclient.Interface, discoveryClient discoveryclient.DiscoveryInterface, eventListenerNamespace string, eventListenerName string) error {
	// Assume the TriggerResourceTemplate is valid (it has an apiVersion and Kind)
	apiVersion := gjson.GetBytes(rt, "apiVersion").String()
	kind := gjson.GetBytes(rt, "kind").String()
	namespace := gjson.GetBytes(rt, "metadata.namespace").String()
	// Default the resource creation to the EventListenerNamespace if not found in the resource template
	if namespace == "" {
		namespace = eventListenerNamespace
	}
	apiResource, err := findAPIResource(discoveryClient, apiVersion, kind)
	if err != nil {
		return err
	}

	rt, err = sjson.SetBytes(rt, "metadata.labels."+triggersv1.LabelEscape+triggersv1.EventListenerLabelKey, eventListenerName)
	if err != nil {
		log.Print(err)
	}

	uri := createRequestURI(apiVersion, apiResource.Name, namespace, apiResource.Namespaced)
	result := restClient.Post().
		RequestURI(uri).
		Body([]byte(rt)).
		SetHeader("Content-Type", "application/json").
		Do()
	if result.Error() != nil {
		return result.Error()
	}
	return nil
}

// findAPIResource returns the APIResource defintion using the discovery client.
func findAPIResource(discoveryClient discoveryclient.DiscoveryInterface, apiVersion, kind string) (*metav1.APIResource, error) {
	resourceList, err := discoveryClient.ServerResourcesForGroupVersion(apiVersion)
	if err != nil {
		return nil, xerrors.Errorf("Error getting kubernetes server resources for apiVersion %s: %s", apiVersion, err)
	}
	for _, apiResource := range resourceList.APIResources {
		if apiResource.Kind == kind {
			return &apiResource, nil
		}
	}
	return nil, xerrors.Errorf("Error could not find resource with apiVersion %s and kind %s", apiVersion, kind)
}

// createRequestURI returns the URI for a request to the kubernetes API REST endpoint.
// If namespaced is false, then namespace will be excluded from the URI.
func createRequestURI(apiVersion, namePlural, namespace string, namespaced bool) string {
	var uri string
	if apiVersion == "v1" {
		uri = "api/v1"
	} else {
		uri = path.Join(uri, "apis", apiVersion)
	}
	if namespaced {
		uri = path.Join(uri, "namespaces", namespace)
	}
	uri = path.Join(uri, namePlural)
	return uri
}

func (r Resource) validateEvent(triggerValidate *triggersv1.TriggerValidate, headers http.Header, payload []byte) error {
	tr, err := r.createValidateTask(triggerValidate, headers, payload)
	if err != nil {
		return err
	}

	tr, err = r.PipelineClient.TektonV1alpha1().TaskRuns(r.EventListenerNamespace).Create(tr)
	if err != nil {
		return err
	}

	for i := taskRunPollingRetries; i >= 0; i-- {
		tr, err := r.PipelineClient.TektonV1alpha1().TaskRuns(r.EventListenerNamespace).Get(tr.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		if tr.IsSuccessful() {
			break
		}

		if tr.IsDone() && !tr.IsSuccessful() {
			return errors.New("validation taskrun: " + tr.Name + " failed")
		}

		time.Sleep(taskRunPollingInterval)
	}
	return nil
}

func (r Resource) createValidateTask(triggerValidate *triggersv1.TriggerValidate,
	headers http.Header, payload []byte) (*pipelinev1.TaskRun, error) {
	// Checking whether task define in taskref exists or not
	task, err := r.PipelineClient.TektonV1alpha1().Tasks(r.EventListenerNamespace).Get(triggerValidate.TaskRef.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	if task.Spec.Inputs == nil {
		return nil, errors.New("parameters are mandatory for validate task")
	}

	params := []pipelinev1.Param{}
	params = append(params, triggerValidate.Params...)
	params = append(params, pipelinev1.Param{
		Name: "EventBody",
		Value: pipelinev1.ArrayOrString{
			Type:      pipelinev1.ParamTypeString,
			StringVal: string(payload),
		},
	})

	h, err := json.Marshal(headers)
	if err != nil {
		return nil, err
	}

	params = append(params, pipelinev1.Param{
		Name: "EventHeaders",
		Value: pipelinev1.ArrayOrString{
			Type:      pipelinev1.ParamTypeString,
			StringVal: string(h),
		},
	})

	return &pipelinev1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    r.EventListenerNamespace,
			GenerateName: triggerValidate.TaskRef.Name,
			Labels: map[string]string{triggersv1.GroupName +
				triggersv1.EventListenerLabelKey: r.EventListenerName},
		},
		Spec: pipelinev1.TaskRunSpec{
			Inputs: pipelinev1.TaskRunInputs{
				Params: params,
			},
			TaskRef:        &triggerValidate.TaskRef,
			ServiceAccount: triggerValidate.ServiceAccountName,
		},
	}, nil
}
