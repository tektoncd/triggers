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
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"

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
	HttpClient             *http.Client
	EventListenerName      string
	EventListenerNamespace string
}

func (r Resource) HandleEvent(response http.ResponseWriter, request *http.Request) {
	el, err := r.TriggersClient.TektonV1alpha1().EventListeners(r.EventListenerNamespace).Get(r.EventListenerName, metav1.GetOptions{})
	if err != nil {
		log.Printf("Error getting EventListener %s in Namespace %s: %s", r.EventListenerName, r.EventListenerNamespace, err)
		response.WriteHeader(http.StatusInternalServerError)
		return
	}

	event, err := ioutil.ReadAll(request.Body)
	if err != nil {
		log.Printf("Error reading event body: %s", err)
		response.WriteHeader(http.StatusInternalServerError)
		return
	}

	eventId := template.Uid()
	log.Printf("EventListener: %s in Namespace: %s handling event (EventID: %s) with payload: %s and header: %v",
		r.EventListenerName, r.EventListenerNamespace, eventId, string(event), request.Header)

	// Execute each Trigger
	for _, trigger := range el.Spec.Triggers {
		go r.executeTrigger(event, request, trigger, eventId)
	}

	// TODO: Do we really need to return the entire body back???
	fmt.Fprintf(response, "EventListener: %s in Namespace: %s handling event (EventID: %s) with payload: %s and header: %v",
		r.EventListenerName, r.EventListenerNamespace, string(eventId), string(event), request.Header)
}

func (r Resource) executeTrigger(payload []byte, request *http.Request, trigger triggersv1.EventListenerTrigger, eventId string) {
	if trigger.Interceptor != nil {
		interceptorUrl, err := GetURI(trigger.Interceptor.ObjectRef, r.EventListenerNamespace) // TODO: Cache this result or do this on initialization
		if err != nil {
			log.Printf("Could not resolve Interceptor Service URI: %q", err)
			return
		}

		modifiedPayload, err := r.processEvent(interceptorUrl, request, payload)
		if err != nil {
			log.Printf("Error Intercepting Event (EventID: %s): %q", eventId, err)
			return
		}
		payload = modifiedPayload
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
	err = createResources(resources, r.RESTClient, r.DiscoveryClient, r.EventListenerNamespace, r.EventListenerName, eventId)
	if err != nil {
		log.Print(err)
	}
}

func (r Resource) processEvent(interceptorUrl *url.URL, request *http.Request, payload []byte) ([]byte, error) {
	outgoing := createOutgoingRequest(context.Background(), request, interceptorUrl, payload)
	respPayload, err := makeRequest(r.HttpClient, outgoing)
	if err != nil {
		return nil, xerrors.Errorf("Not OK response from Event Processor: %w", err)
	}
	return respPayload, nil
}

func createResources(resources []json.RawMessage, restClient restclient.Interface, discoveryClient discoveryclient.DiscoveryInterface, eventListenerNamespace string, eventListenerName string, eventId string) error {
	for _, resource := range resources {
		if err := createResource(resource, restClient, discoveryClient, eventListenerNamespace, eventListenerName, eventId); err != nil {
			return err
		}
	}
	return nil
}

// createResource uses the kubeClient to create the resource defined in the
// TriggerResourceTemplate and returns any errors with this process
func createResource(rt json.RawMessage, restClient restclient.Interface, discoveryClient discoveryclient.DiscoveryInterface, eventListenerNamespace string, eventListenerName string, eventId string) error {
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
		return err
	}
	rt, err = sjson.SetBytes(rt, "metadata.labels."+triggersv1.LabelEscape+triggersv1.EventIDLabelKey, eventId)
	if err != nil {
		log.Print(err)
		return err
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
