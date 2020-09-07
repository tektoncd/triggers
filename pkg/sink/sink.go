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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	triggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"github.com/tektoncd/triggers/pkg/interceptors/bitbucket"
	"github.com/tektoncd/triggers/pkg/interceptors/cel"
	"github.com/tektoncd/triggers/pkg/interceptors/github"
	"github.com/tektoncd/triggers/pkg/interceptors/gitlab"
	"github.com/tektoncd/triggers/pkg/interceptors/webhook"
	"github.com/tektoncd/triggers/pkg/resources"
	"github.com/tektoncd/triggers/pkg/template"
	"go.uber.org/zap"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
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
	HTTPClient             *http.Client
	EventListenerName      string
	EventListenerNamespace string
	Logger                 *zap.SugaredLogger
	Auth                   AuthOverride
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
	el, err := r.TriggersClient.TriggersV1alpha1().EventListeners(r.EventListenerNamespace).Get(r.EventListenerName, metav1.GetOptions{})
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
		go func(t triggersv1.EventListenerTrigger) {
			localRequest := request.Clone(request.Context())
			if err := r.processTrigger(&t, localRequest, event, eventID, eventLog); err != nil {
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
		}(t)
	}

	//The eventlistener waits until all the trigger executions (up-to the creation of the resources) and
	//only when at least one of the execution completed successfully, it returns response code 201(Created) otherwise it returns 202 (Accepted).
	code := http.StatusAccepted
	for i := 0; i < len(el.Spec.Triggers); i++ {
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
		eventLog.Errorf("failed to write back sink response: %w", err)
	}
}

func (r Sink) processTrigger(t *triggersv1.EventListenerTrigger, request *http.Request, event []byte, eventID string, eventLog *zap.SugaredLogger) error {
	if t == nil {
		return errors.New("EventListenerTrigger not defined")
	}

	if t.Template == nil && t.TriggerRef != "" {
		trigger, err := r.TriggersClient.TriggersV1alpha1().Triggers(r.EventListenerNamespace).Get(t.TriggerRef, metav1.GetOptions{})
		if err != nil {
			r.Logger.Fatalf("Error getting Trigger %s in Namespace %s: %s", t.TriggerRef, r.EventListenerNamespace, err)
			return err
		}
		trig, err := triggersv1.ToEventListenerTrigger(trigger.Spec)
		if err != nil {
			r.Logger.Fatalf("Error changing Trigger to EventListenerTrigger: %s", err)
			return err
		}
		t = &trig
	}

	log := eventLog.With(zap.String(triggersv1.TriggerLabelKey, t.Name))

	finalPayload, header, err := r.ExecuteInterceptors(t, request, event, log)
	if err != nil {
		log.Error(err)
		return err
	}

	rt, err := template.ResolveTrigger(*t,
		r.TriggersClient.TriggersV1alpha1().TriggerBindings(r.EventListenerNamespace).Get,
		r.TriggersClient.TriggersV1alpha1().ClusterTriggerBindings().Get,
		r.TriggersClient.TriggersV1alpha1().TriggerTemplates(r.EventListenerNamespace).Get)
	if err != nil {
		log.Error(err)
		return err
	}

	params, err := template.ResolveParams(rt, finalPayload, header)
	if err != nil {
		log.Error(err)
		return err
	}

	log.Infof("ResolvedParams : %+v", params)
	resources := template.ResolveResources(rt.TriggerTemplate, params)
	if err := r.CreateResources(t.ServiceAccountName, resources, t.Name, eventID, log); err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func (r Sink) ExecuteInterceptors(t *triggersv1.EventListenerTrigger, in *http.Request, event []byte, log *zap.SugaredLogger) ([]byte, http.Header, error) {
	if len(t.Interceptors) == 0 {
		return event, in.Header, nil
	}

	// The request body to the first interceptor in the chain should be the received event body.
	request := &http.Request{
		Method: http.MethodPost,
		Header: in.Header,
		URL:    in.URL,
		Body:   ioutil.NopCloser(bytes.NewBuffer(event)),
	}

	// We create a cache against each request, so whenever we make network calls like
	// fetching kubernetes secrets, we can do so only once per request.
	request = interceptors.WithCache(request)

	var resp *http.Response
	for _, i := range t.Interceptors {
		var interceptor interceptors.Interceptor
		switch {
		case i.Webhook != nil:
			interceptor = webhook.NewInterceptor(i.Webhook, r.HTTPClient, r.EventListenerNamespace, log)
		case i.GitHub != nil:
			interceptor = github.NewInterceptor(i.GitHub, r.KubeClientSet, r.EventListenerNamespace, log)
		case i.GitLab != nil:
			interceptor = gitlab.NewInterceptor(i.GitLab, r.KubeClientSet, r.EventListenerNamespace, log)
		case i.CEL != nil:
			interceptor = cel.NewInterceptor(i.CEL, r.KubeClientSet, r.EventListenerNamespace, log)
		case i.Bitbucket != nil:
			interceptor = bitbucket.NewInterceptor(i.Bitbucket, r.KubeClientSet, r.EventListenerNamespace, log)
		default:
			return nil, nil, fmt.Errorf("unknown interceptor type: %v", i)
		}
		var err error
		resp, err = interceptor.ExecuteTrigger(request)
		if err != nil {
			return nil, nil, err
		}

		// Set the next request to be the output of the last response to enable
		// request chaining.
		request = &http.Request{
			Method: http.MethodPost,
			Header: resp.Header,
			URL:    in.URL,
			Body:   ioutil.NopCloser(resp.Body),
		}
	}
	payload, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("error reading final response body: %w", err)
	}
	defer resp.Body.Close()
	return payload, resp.Header, nil
}

func (r Sink) CreateResources(sa string, res []json.RawMessage, triggerName, eventID string, log *zap.SugaredLogger) error {
	discoveryClient := r.DiscoveryClient
	dynamicClient := r.DynamicClient
	var err error
	if len(sa) > 0 {
		// So at start up the discovery and dynamic clients are created using the in cluster config
		// of this pod (i.e. using the credentials of the serviceaccount associated with the EventListener)

		// However, we also have a ServiceAccountName reference with each EventListenerTrigger to allow
		// for more fine grained authorization control around the resources we create below.
		discoveryClient, dynamicClient, err = r.Auth.OverrideAuthentication(sa, r.EventListenerNamespace, log, r.DiscoveryClient, r.DynamicClient)
		if err != nil {
			log.Errorf("problem cloning rest config: %#v", err)
			return err
		}
	}

	for _, rr := range res {
		if err := resources.Create(r.Logger, rr, triggerName, eventID, r.EventListenerName, r.EventListenerNamespace, discoveryClient, dynamicClient); err != nil {
			log.Errorf("problem creating obj: %#v", err)
			return err
		}
	}
	return nil
}
