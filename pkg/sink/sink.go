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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	triggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	listers "github.com/tektoncd/triggers/pkg/client/listers/triggers/v1alpha1"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"github.com/tektoncd/triggers/pkg/interceptors/webhook"
	"github.com/tektoncd/triggers/pkg/resources"
	"github.com/tektoncd/triggers/pkg/template"
	"github.com/tidwall/sjson"
	"go.uber.org/zap"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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
	HTTPClient             *http.Client
	EventListenerName      string
	EventListenerNamespace string
	Logger                 *zap.SugaredLogger
	Auth                   AuthOverride

	// listers index properties about resources
	EventListenerLister         listers.EventListenerLister
	TriggerLister               listers.TriggerLister
	TriggerBindingLister        listers.TriggerBindingLister
	ClusterTriggerBindingLister listers.ClusterTriggerBindingLister
	TriggerTemplateLister       listers.TriggerTemplateLister
	InterceptorLister           listers.InterceptorLister
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
	el, err := r.EventListenerLister.EventListeners(r.EventListenerNamespace).Get(r.EventListenerName)
	if err != nil {
		r.Logger.Errorf("Error getting EventListener %s in Namespace %s: %s", r.EventListenerName, r.EventListenerNamespace, err)
		response.WriteHeader(http.StatusInternalServerError)
		return
	}
	event, err := ioutil.ReadAll(request.Body)
	if err != nil {
		r.Logger.Errorf("Error reading event body: %s", err)
		response.WriteHeader(http.StatusInternalServerError)
		return
	}

	eventID := template.UUID()
	eventLog := r.Logger.With(zap.String(triggersv1.EventIDLabelKey, eventID))
	eventLog.Debugf("EventListener: %s in Namespace: %s handling event (EventID: %s) with payload: %s and header: %v",
		r.EventListenerName, r.EventListenerNamespace, eventID, string(event), request.Header)
	var trItems []*triggersv1.Trigger
	if len(el.Spec.NamespaceSelector.MatchNames) == 1 &&
		el.Spec.NamespaceSelector.MatchNames[0] == "*" {
		trList, err := r.TriggerLister.List(labels.Everything())
		if err != nil {
			r.Logger.Errorf("Error getting Triggers: %s", err)
			response.WriteHeader(http.StatusInternalServerError)
			return
		}
		trItems = trList
	} else if len(el.Spec.NamespaceSelector.MatchNames) != 0 {
		for _, v := range el.Spec.NamespaceSelector.MatchNames {
			trList, err := r.TriggerLister.Triggers(v).List(labels.Everything())
			if err != nil {
				r.Logger.Errorf("Error getting Triggers: %s", err)
				response.WriteHeader(http.StatusInternalServerError)
				return
			}
			trItems = append(trItems, trList...)
		}
	}
	triggers, err := r.merge(el.Spec.Triggers, trItems)
	if err != nil {
		r.Logger.Errorf("Error merging Triggers: %s", err)
		response.WriteHeader(http.StatusInternalServerError)
		return
	}
	result := make(chan int, 10)
	// Execute each Trigger
	for _, t := range triggers {
		go func(t triggersv1.Trigger) {
			localRequest := request.Clone(request.Context())
			if err := r.processTrigger(t, localRequest, event, eventID, eventLog); err != nil {
				if kerrors.IsUnauthorized(err) {
					result <- http.StatusUnauthorized
					return
				}
				if kerrors.IsForbidden(err) {
					result <- http.StatusForbidden
					return
				}
				result <- http.StatusAccepted
				return
			}
			result <- http.StatusCreated
		}(*t)
	}

	//The eventlistener waits until all the trigger executions (up-to the creation of the resources) and
	//only when at least one of the execution completed successfully, it returns response code 201(Created) otherwise it returns 202 (Accepted).
	code := http.StatusAccepted
	for i := 0; i < len(triggers); i++ {
		thiscode := <-result
		// current take - if someone is doing unauthorized stuff, we abort immediately;
		// unauthorized should be the final status code vs. the less than comparison
		// below around accepted vs. created
		if thiscode == http.StatusUnauthorized || thiscode == http.StatusForbidden {
			code = thiscode
			break
		}
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
		eventLog.Errorf("failed to write back sink response: %v", err)
	}
}

func (r Sink) merge(et []triggersv1.EventListenerTrigger, trItems []*triggersv1.Trigger) ([]*triggersv1.Trigger, error) {
	triggers := trItems
	for _, t := range et {
		switch {
		case t.Template == nil && t.TriggerRef != "":
			trig, err := r.TriggerLister.Triggers(r.EventListenerNamespace).Get(t.TriggerRef)
			if err != nil {
				r.Logger.Errorf("Error getting Trigger %s in Namespace %s: %s", t.TriggerRef, r.EventListenerNamespace, err)
				continue
			}
			triggers = append(triggers, trig)
		case t.Template != nil:
			triggers = append(triggers, &triggersv1.Trigger{
				ObjectMeta: metav1.ObjectMeta{
					Name:      t.Name,
					Namespace: r.EventListenerNamespace},
				Spec: triggersv1.TriggerSpec{
					ServiceAccountName: t.ServiceAccountName,
					Bindings:           t.Bindings,
					Template:           *t.Template,
					Interceptors:       t.Interceptors,
				},
			})
		default:
			return nil, errors.New("EventListenerTrigger not defined")
		}
	}
	return triggers, nil
}

func (r Sink) processTrigger(t triggersv1.Trigger, request *http.Request, event []byte, eventID string, eventLog *zap.SugaredLogger) error {
	log := eventLog.With(zap.String(triggersv1.TriggerLabelKey, t.Name))

	finalPayload, header, iresp, err := r.ExecuteInterceptors(t, request, event, log, eventID)
	if err != nil {
		log.Error(err)
		return err
	}

	if iresp != nil {
		if !iresp.Continue {
			log.Infof("interceptor stopped trigger processing: %v", iresp.Status.Err())
			return iresp.Status.Err()
		}
	}

	rt, err := template.ResolveTrigger(t,
		r.TriggerBindingLister.TriggerBindings(t.Namespace).Get,
		r.ClusterTriggerBindingLister.Get,
		r.TriggerTemplateLister.TriggerTemplates(t.Namespace).Get)
	if err != nil {
		log.Error(err)
		return err
	}
	extensions := map[string]interface{}{}
	if iresp != nil && iresp.Extensions != nil {
		extensions = iresp.Extensions
	}
	params, err := template.ResolveParams(rt, finalPayload, header, extensions)
	if err != nil {
		log.Error(err)
		return err
	}

	log.Infof("ResolvedParams : %+v", params)
	resources := template.ResolveResources(rt.TriggerTemplate, params)
	if err := r.CreateResources(t.Namespace, t.Spec.ServiceAccountName, resources, t.Name, eventID, log); err != nil {
		log.Error(err)
		return err
	}
	return nil
}

// ExecuteInterceptor executes all interceptors for the Trigger and returns back the body, header, and InterceptorResponse to use.
// When TEP-0022 is fully implemented, this function will only return the InterceptorResponse and error.
func (r Sink) ExecuteInterceptors(t triggersv1.Trigger, in *http.Request, event []byte, log *zap.SugaredLogger, eventID string) ([]byte, http.Header, *triggersv1.InterceptorResponse, error) {
	if len(t.Spec.Interceptors) == 0 {
		return event, in.Header, nil, nil
	}

	// request is the request sent to the interceptors in the chain. Each interceptor can set the InterceptorParams field
	// or add to the Extensions
	request := triggersv1.InterceptorRequest{
		Body:       string(event),
		Header:     in.Header.Clone(),
		Extensions: map[string]interface{}{}, // Empty extensions for the first interceptor in chain
		Context: &triggersv1.TriggerContext{
			EventURL: in.URL.String(),
			EventID:  eventID,
			// t.Name might not be fully accurate until we get rid of triggers inlined within EventListener
			TriggerID: fmt.Sprintf("namespaces/%s/triggers/%s", t.Namespace, t.Name), // TODO: t.Name might be wrong
		},
	}

	for _, i := range t.Spec.Interceptors {
		if i.Webhook != nil { // Old style interceptor
			body, err := extendBodyWithExtensions([]byte(request.Body), request.Extensions)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("could not merge extensions with body: %w", err)
			}
			req := &http.Request{
				Method: http.MethodPost,
				Header: request.Header,
				URL:    in.URL,
				Body:   ioutil.NopCloser(bytes.NewBuffer(body)),
			}
			interceptor := webhook.NewInterceptor(i.Webhook, r.HTTPClient, t.Namespace, log)
			res, err := interceptor.ExecuteTrigger(req)
			if err != nil {
				return nil, nil, nil, err
			}

			payload, err := ioutil.ReadAll(res.Body)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("error reading webhook interceptor response body: %w", err)
			}
			defer res.Body.Close()
			// Set the next request to be the output of the last response to enable
			// request chaining.
			request.Header = res.Header.Clone()
			request.Body = string(payload)
			continue
		}
		request.InterceptorParams = interceptors.GetInterceptorParams(i)
		url, err := interceptors.ResolveToURL(r.InterceptorLister.Get, interceptors.GetName(i))
		if err != nil {
			return nil, nil, nil, fmt.Errorf("could not resolve interceptor URL: %w", err)
		}

		// TODO: Plumb through context from EL
		interceptorResponse, err := interceptors.Execute(context.Background(), r.HTTPClient, &request, url.String())
		if err != nil {
			return nil, nil, nil, err
		}
		if !interceptorResponse.Continue {
			return nil, nil, interceptorResponse, nil
		}

		if interceptorResponse.Extensions != nil {
			// Merge any extensions and pass it on to the next request in the chain
			for k, v := range interceptorResponse.Extensions {
				request.Extensions[k] = v
			}
		}
		// Clear interceptorParams for the next interceptor in chain
		request.InterceptorParams = map[string]interface{}{}
	}
	return []byte(request.Body), request.Header, &triggersv1.InterceptorResponse{
		Continue:   true,
		Extensions: request.Extensions,
	}, nil
}

func (r Sink) CreateResources(triggerNS, sa string, res []json.RawMessage, triggerName, eventID string, log *zap.SugaredLogger) error {
	discoveryClient := r.DiscoveryClient
	dynamicClient := r.DynamicClient
	var err error
	if len(sa) > 0 {
		// So at start up the discovery and dynamic clients are created using the in cluster config
		// of this pod (i.e. using the credentials of the serviceaccount associated with the EventListener)

		// However, we also have a ServiceAccountName reference with each EventListenerTrigger to allow
		// for more fine grained authorization control around the resources we create below.
		discoveryClient, dynamicClient, err = r.Auth.OverrideAuthentication(sa, triggerNS, log, r.DiscoveryClient, r.DynamicClient)
		if err != nil {
			log.Errorf("problem cloning rest config: %#v", err)
			return err
		}
	}

	for _, rr := range res {
		if err := resources.Create(r.Logger, rr, triggerName, eventID, r.EventListenerName, triggerNS, discoveryClient, dynamicClient); err != nil {
			log.Errorf("problem creating obj: %#v", err)
			return err
		}
	}
	return nil
}

// extendBodyWithExtensions merges the extensions into the given body.
func extendBodyWithExtensions(body []byte, extensions map[string]interface{}) ([]byte, error) {
	for k, v := range extensions {
		vb, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal value to JSON: %w", err)
		}
		body, err = sjson.SetRawBytes(body, fmt.Sprintf("extensions.%s", k), vb)
		if err != nil {
			return nil, fmt.Errorf("failed to sjson extensions to body: %w", err)
		}
	}

	return body, nil
}
