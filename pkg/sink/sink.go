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
	"io/ioutil"
	"net/http"

	pipelineclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	triggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"github.com/tektoncd/triggers/pkg/interceptors/cel"
	"github.com/tektoncd/triggers/pkg/interceptors/github"
	"github.com/tektoncd/triggers/pkg/interceptors/gitlab"
	"github.com/tektoncd/triggers/pkg/interceptors/webhook"
	"github.com/tektoncd/triggers/pkg/resources"
	"github.com/tektoncd/triggers/pkg/template"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// Response defines the HTTP body that the Sink responds to events with.
type Response struct {
	// EventListener is the name of the eventListener
	EventListener string `json:"eventListener"`
	// Namespace is the namespace that the eventListener is running in
	Namespace string `json:"namespace,omitempty"`
	// EventID is a uniqueID that gets assigned to each incoming request
	EventID string `json:"eventID,omitempty"`
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
			case t.Interceptor.GitHub != nil:
				interceptor = github.NewInterceptor(t.Interceptor.GitHub, r.KubeClientSet, r.EventListenerNamespace, log)
			case t.Interceptor.GitLab != nil:
				interceptor = gitlab.NewInterceptor(t.Interceptor.GitLab, r.KubeClientSet, r.EventListenerNamespace, log)
			case t.Interceptor.CEL != nil:
				interceptor = cel.NewInterceptor(t.Interceptor.CEL, r.KubeClientSet, r.EventListenerNamespace, log)
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
			log.Infof("params: %+v", params)
			res := template.ResolveResources(rt.TriggerTemplate, params)
			if err := r.createResources(res, t.Name, eventID); err != nil {
				log.Errorf("Could not create resources for %q: %v", t.Name, err)
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

	response.WriteHeader(code)
	response.Header().Set("Content-Type", "application/json")
	body := Response{
		EventListener: r.EventListenerName,
		Namespace:     r.EventListenerNamespace,
		EventID:       eventID,
	}
	if err := json.NewEncoder(response).Encode(body); err != nil {
		eventLog.Errorf("failed to write back sink response: %w", err)
	}
}

func (r Sink) createResources(res []json.RawMessage, triggerName, eventID string) error {
	for _, rr := range res {
		if err := resources.Create(r.Logger, rr, triggerName, eventID, r.EventListenerName, r.EventListenerNamespace, r.DiscoveryClient, r.DynamicClient); err != nil {
			return err
		}
	}
	return nil
}
