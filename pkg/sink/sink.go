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
	"path"

	"github.com/tektoncd/triggers/pkg/interceptors"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"golang.org/x/xerrors"

	"github.com/tektoncd/triggers/pkg/interceptors/webhook"

	pipelineclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	triggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"github.com/tektoncd/triggers/pkg/template"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	discoveryclient "k8s.io/client-go/discovery"
	restclient "k8s.io/client-go/rest"
)

// Sink defines the sink resource for processing incoming events for the
// EventListener.
type Sink struct {
	TriggersClient         triggersclientset.Interface
	DiscoveryClient        discoveryclient.DiscoveryInterface
	RESTClient             restclient.Interface
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
	r.Logger.Debugf("EventListener: %s in Namespace: %s handling event (EventID: %s) with payload: %s and header: %v",
		r.EventListenerName, r.EventListenerNamespace, eventID, string(event), request.Header)

	result := make(chan int, 10)
	// Execute each Trigger

	for _, trigger := range el.Spec.Triggers {
		t := trigger
		var interceptor interceptors.Interceptor
		if t.Interceptor != nil {
			switch {
			case t.Interceptor.Webhook != nil:
				interceptor = webhook.NewInterceptor(t.Interceptor, r.HTTPClient, r.EventListenerNamespace, r.Logger)
			}
		}
		go func() {
			finalPayload := event
			if interceptor != nil {
				payload, err := interceptor.ExecuteTrigger(event, request, t, eventID)
				if err != nil {
					r.Logger.Error(err)
					result <- http.StatusAccepted
					return
				}
				finalPayload = payload
			}
			binding, err := template.ResolveBinding(t,
				r.TriggersClient.TektonV1alpha1().TriggerBindings(r.EventListenerNamespace).Get,
				r.TriggersClient.TektonV1alpha1().TriggerTemplates(r.EventListenerNamespace).Get)
			if err != nil {
				r.Logger.Error(err)
				result <- http.StatusAccepted
				return
			}
			resources, err := template.NewResources(finalPayload, request.Header, binding)
			if err != nil {
				r.Logger.Error(err)
				result <- http.StatusAccepted
				return
			}
			if err = createResources(resources, r.RESTClient, r.DiscoveryClient, r.EventListenerNamespace, r.EventListenerName, eventID, r.Logger); err != nil {
				r.Logger.Error(err)
			}
			result <- http.StatusCreated
		}()
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

func createResources(resources []json.RawMessage, restClient restclient.Interface, discoveryClient discoveryclient.DiscoveryInterface, eventListenerNamespace string, eventListenerName string, eventID string, logger *zap.SugaredLogger) error {
	for _, resource := range resources {
		if err := createResource(resource, restClient, discoveryClient, eventListenerNamespace, eventListenerName, eventID, logger); err != nil {
			return err
		}
	}
	return nil
}

// createResource uses the kubeClient to create the resource defined in the
// TriggerResourceTemplate and returns any errors with this process
func createResource(rt json.RawMessage, restClient restclient.Interface, discoveryClient discoveryclient.DiscoveryInterface, eventListenerNamespace string, eventListenerName string, eventID string, logger *zap.SugaredLogger) error {
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
		return err
	}
	rt, err = sjson.SetBytes(rt, "metadata.labels."+triggersv1.LabelEscape+triggersv1.EventIDLabelKey, eventID)
	if err != nil {
		return err
	}

	resourcename := gjson.GetBytes(rt, "metadata.name")
	resourcekind := gjson.GetBytes(rt, "kind")
	logger.Infof("Generating resource: kind: %s, name: %s ", resourcekind, resourcename)

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

// findAPIResource returns the APIResource definition using the discovery client.
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
