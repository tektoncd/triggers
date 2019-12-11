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
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/tektoncd/triggers/pkg/interceptors/github"
	"github.com/tektoncd/triggers/pkg/interceptors/gitlab"

	"github.com/tektoncd/triggers/pkg/interceptors"

	"github.com/tidwall/sjson"
	"golang.org/x/xerrors"

	"github.com/tektoncd/triggers/pkg/interceptors/webhook"

	pipelineclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	triggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"github.com/tektoncd/triggers/pkg/template"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	discoveryclient "k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// Sink defines the sink resource for processing incoming events for the
// EventListener.
type Sink struct {
	KubeClientSet          kubernetes.Interface
	TriggersClient         triggersclientset.Interface
	DiscoveryClient        discoveryclient.ServerResourcesInterface
	DynamicClient          dynamic.Interface
	PipelineClient         pipelineclientset.Interface
	HTTPClient             *http.Client
	EventListenerName      string
	EventListenerNamespace string
	Logger                 *zap.SugaredLogger
}

// HandleEvent processes an incoming HTTP event for the event listener.
func (r Sink) HandleEvent(response http.ResponseWriter, request *http.Request) {
	el, err := r.TriggersClient.TektonV1alpha1().EventListeners(r.EventListenerNamespace).Get(r.EventListenerName, metav1.GetOptions{})
	if err != nil {
		r.Logger.Fatalf("Error getting EventListener %s in Namespace %s: %s", r.EventListenerName, r.EventListenerNamespace, err)
		response.WriteHeader(http.StatusInternalServerError)
		return
	}

	event, err := ioutil.ReadAll(request.Body)
	if err != nil {
		r.Logger.Errorf("Error reading event body: %s", err)
		response.WriteHeader(http.StatusInternalServerError)
		return
	}

	eventID := template.UID()
	eventLog := r.Logger.With(zap.String(triggersv1.EventIDLabelKey, eventID))
	eventLog.Debugf("EventListener: %s in Namespace: %s handling event (EventID: %s) with payload: %s and header: %v",
		r.EventListenerName, r.EventListenerNamespace, eventID, string(event), request.Header)

	result := make(chan int, 10)
	// Execute each Trigger

	for _, t := range el.Spec.Triggers {
		log := eventLog.With(zap.String(triggersv1.TriggerLabelKey, t.Name))

		var interceptor interceptors.Interceptor
		if t.Interceptor != nil {
			switch {
			case t.Interceptor.Webhook != nil:
				interceptor = webhook.NewInterceptor(t.Interceptor.Webhook, r.HTTPClient, r.EventListenerNamespace, log)
			case t.Interceptor.Github != nil:
				interceptor = github.NewInterceptor(t.Interceptor.Github, r.KubeClientSet, r.EventListenerNamespace, log)
			case t.Interceptor.Gitlab != nil:
				interceptor = gitlab.NewInterceptor(t.Interceptor.Gitlab, r.KubeClientSet, r.EventListenerNamespace, log)
			}
		}
		go func(t triggersv1.EventListenerTrigger) {
			finalPayload := event
			if interceptor != nil {
				payload, err := interceptor.ExecuteTrigger(event, request, &t, eventID)
				if err != nil {
					log.Error(err)
					result <- http.StatusAccepted
					return
				}
				finalPayload = payload
			}
			rt, err := template.ResolveTrigger(t,
				r.TriggersClient.TektonV1alpha1().TriggerBindings(r.EventListenerNamespace).Get,
				r.TriggersClient.TektonV1alpha1().TriggerTemplates(r.EventListenerNamespace).Get)
			if err != nil {
				log.Error(err)
				result <- http.StatusAccepted
				return
			}

			params, err := template.ResolveParams(rt.TriggerBindings, finalPayload, request.Header, rt.TriggerTemplate.Spec.Params)
			if err != nil {
				log.Error(err)
				result <- http.StatusAccepted
				return
			}
			log.Info("params: %+v", params)
			resources := template.ResolveResources(rt.TriggerTemplate, params)
			if err := r.createResources(resources, t.Name, eventID); err != nil {
				log.Error(err)
			}
			result <- http.StatusCreated
		}(t)
	}

	//The eventlistener waits until all the trigger executions (up-to the creation of the resources) and
	//only when at least one of the execution completed successfully, it returns response code 201(Accepted) otherwise it returns 202 (Created).
	code := http.StatusAccepted
	for i := 0; i < len(el.Spec.Triggers); i++ {
		thiscode := <-result
		if thiscode < code {
			code = thiscode
		}
	}

	// TODO: Do we really need to return the entire body back???
	response.WriteHeader(code)
	fmt.Fprintf(response, "EventListener: %s in Namespace: %s handling event (EventID: %s) with payload: %s and header: %v",
		r.EventListenerName, r.EventListenerNamespace, string(eventID), string(event), request.Header)
}

func (r Sink) createResources(resources []json.RawMessage, triggerName, eventID string) error {
	for _, resource := range resources {
		if err := r.createResource(resource, triggerName, eventID); err != nil {
			return err
		}
	}
	return nil
}

// createResource uses the kubeClient to create the resource defined in the
// TriggerResourceTemplate and returns any errors with this process
func (r Sink) createResource(rt json.RawMessage, triggerName string, eventID string) error {
	// Add common labels
	rt, err := addLabels(rt, map[string]string{
		triggersv1.EventListenerLabelKey: r.EventListenerName,
		triggersv1.EventIDLabelKey:       eventID,
		triggersv1.TriggerLabelKey:       triggerName,
	})
	if err != nil {
		return err
	}

	// Assume the TriggerResourceTemplate is valid (it has an apiVersion and Kind)
	data := new(unstructured.Unstructured)
	if err := data.UnmarshalJSON(rt); err != nil {
		return err
	}

	namespace := data.GetNamespace()
	// Default the resource creation to the EventListenerNamespace if not found in the resource template
	if namespace == "" {
		namespace = r.EventListenerNamespace
	}

	// Resolve resource kind to the underlying API Resource type.
	apiResource, err := r.findAPIResource(data.GetAPIVersion(), data.GetKind())
	if err != nil {
		return err
	}

	name := data.GetName()
	if name == "" {
		name = data.GetGenerateName()
	}
	r.Logger.Infof("Generating resource: kind: %+v, name: %s", apiResource, name)

	gvr := schema.GroupVersionResource{
		Group:    apiResource.Group,
		Version:  apiResource.Version,
		Resource: apiResource.Name,
	}

	_, err = r.DynamicClient.Resource(gvr).Namespace(namespace).Create(data, metav1.CreateOptions{})
	return err
}

// findAPIResource returns the APIResource definition using the discovery client.
func (r Sink) findAPIResource(apiVersion, kind string) (*metav1.APIResource, error) {
	resourceList, err := r.DiscoveryClient.ServerResourcesForGroupVersion(apiVersion)
	if err != nil {
		return nil, xerrors.Errorf("Error getting kubernetes server resources for apiVersion %s: %s", apiVersion, err)
	}
	for _, apiResource := range resourceList.APIResources {
		if apiResource.Kind != kind {
			continue
		}
		r := &apiResource
		// Resolve GroupVersion from parent list to have consistent resource identifiers.
		if r.Version == "" || r.Group == "" {
			gv, err := schema.ParseGroupVersion(resourceList.GroupVersion)
			if err != nil {
				return nil, xerrors.Errorf("error parsing parsing GroupVersion: %v", err)
			}
			r.Group = gv.Group
			r.Version = gv.Version
		}
		return r, nil
	}
	return nil, xerrors.Errorf("Error could not find resource with apiVersion %s and kind %s", apiVersion, kind)
}

// addLabels adds autogenerated Tekton labels to created resources.
func addLabels(rt json.RawMessage, labels map[string]string) (json.RawMessage, error) {
	var err error
	for k, v := range labels {
		l := fmt.Sprintf("metadata.labels.%s/%s", triggersv1.LabelEscape, strings.TrimLeft(k, "/"))
		rt, err = sjson.SetBytes(rt, l, v)
		if err != nil {
			return rt, err
		}
	}
	return rt, err
}
