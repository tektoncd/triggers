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
	"io"
	"net/http"
	"os"
	"sync"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/binding"
	cehttp "github.com/cloudevents/sdk-go/v2/protocol/http"
	"github.com/google/uuid"
	"github.com/tektoncd/triggers/pkg/apis/triggers"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	triggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	listersv1alpha1 "github.com/tektoncd/triggers/pkg/client/listers/triggers/v1alpha1"
	listers "github.com/tektoncd/triggers/pkg/client/listers/triggers/v1beta1"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"github.com/tektoncd/triggers/pkg/interceptors/webhook"
	"github.com/tektoncd/triggers/pkg/reconciler/events"
	"github.com/tektoncd/triggers/pkg/resources"
	"github.com/tektoncd/triggers/pkg/sink/cloudevent"
	"github.com/tektoncd/triggers/pkg/template"
	"github.com/tidwall/sjson"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	discoveryclient "k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	"knative.dev/pkg/apis"
	v1 "knative.dev/pkg/apis/duck/v1"
)

// Sink defines the sink resource for processing incoming events for the
// EventListener.
type Sink struct {
	KubeClientSet          kubernetes.Interface
	TriggersClient         triggersclientset.Interface
	DiscoveryClient        discoveryclient.ServerResourcesInterface
	DynamicClient          dynamic.Interface
	HTTPClient             *http.Client
	CEClient               cloudevent.CEClient
	EventListenerName      string
	EventListenerNamespace string
	Logger                 *zap.SugaredLogger
	Recorder               *Recorder
	Auth                   AuthOverride
	PayloadValidation      bool
	CloudEventURI          string
	// WGProcessTriggers keeps track of triggers or triggerGroups currently being processed
	// Currently only used in tests to wait for all triggers to finish processing
	WGProcessTriggers *sync.WaitGroup
	EventRecorder     record.EventRecorder

	// listers index properties about resources
	EventListenerLister         listers.EventListenerLister
	TriggerLister               listers.TriggerLister
	TriggerBindingLister        listers.TriggerBindingLister
	ClusterTriggerBindingLister listers.ClusterTriggerBindingLister
	TriggerTemplateLister       listers.TriggerTemplateLister
	ClusterInterceptorLister    listersv1alpha1.ClusterInterceptorLister
	InterceptorLister           listersv1alpha1.InterceptorLister
}

// Response defines the HTTP body that the Sink responds to events with.
type Response struct {
	// EventListener is the name of the eventListener.
	// Deprecated: use EventListenerUID instead.
	EventListener string `json:"eventListener"`
	// Namespace is the namespace that the eventListener is running in.
	// Deprecated: use EventListenerUID instead.
	Namespace string `json:"namespace,omitempty"`
	// EventListenerUID is the UID of the EventListener
	EventListenerUID string `json:"eventListenerUID"`
	// EventID is a uniqueID that gets assigned to each incoming request
	EventID string `json:"eventID,omitempty"`
	// ErrorMessage gives message about Error which occurs during event processing
	ErrorMessage string `json:"errorMessage,omitempty"`
}

func (r Sink) emitEvents(recorder record.EventRecorder, el *triggersv1.EventListener, eventType string, err error) {
	if os.Getenv("EL_EVENT") == "enable" {
		events.Emit(recorder, eventType, el, err)
	}
}

// HandleEvent processes an incoming HTTP event for the event listener.
func (r Sink) HandleEvent(response http.ResponseWriter, request *http.Request) {

	log := r.Logger.With(
		zap.String("eventlistener", r.EventListenerName),
		zap.String("namespace", r.EventListenerNamespace),
	)
	eventID := template.UUID()
	log = log.With(zap.String(triggers.EventIDLabelKey, eventID))

	elTemp := triggersv1.EventListener{
		TypeMeta: metav1.TypeMeta{
			Kind:       "EventListener",
			APIVersion: "v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.EventListenerName,
			Namespace: r.EventListenerNamespace,
		},
		Spec: triggersv1.EventListenerSpec{
			Resources: triggersv1.Resources{
				KubernetesResource: &triggersv1.KubernetesResource{
					WithPodSpec: v1.WithPodSpec{
						Template: v1.PodSpecable{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{{
									// enabled by default for temporary EL
									Env: []corev1.EnvVar{{
										Name:  "EL_EVENT",
										Value: "true",
									}},
								}},
							},
						},
					},
				},
			},
		},
	}

	r.emitEvents(r.EventRecorder, &elTemp, events.TriggerProcessingStartedV1, nil)
	r.sendCloudEvents(request.Header, elTemp, eventID, events.TriggerProcessingStartedV1)

	event, err := io.ReadAll(request.Body)
	if err != nil {
		log.Errorf("Error reading event body: %s", err)
		r.recordCountMetrics(failTag)
		response.WriteHeader(http.StatusInternalServerError)
		r.emitEvents(r.EventRecorder, &elTemp, events.TriggerProcessingFailedV1, err)
		r.sendCloudEvents(request.Header, elTemp, eventID, events.TriggerProcessingFailedV1)
		return
	}

	el, err := r.EventListenerLister.EventListeners(r.EventListenerNamespace).Get(r.EventListenerName)
	if err != nil {
		log.Errorf("Error getting EventListener %s in Namespace %s: %s", r.EventListenerName, r.EventListenerNamespace, err)
		r.recordCountMetrics(failTag)
		response.WriteHeader(http.StatusInternalServerError)
		r.emitEvents(r.EventRecorder, &elTemp, events.TriggerProcessingFailedV1, err)
		r.sendCloudEvents(request.Header, elTemp, eventID, events.TriggerProcessingFailedV1)
		return
	}

	elUID := string(el.GetUID())
	log = log.With(zap.String("eventlistenerUID", elUID))

	log = log.With(zap.String(triggers.EventIDLabelKey, eventID))
	log.Debugf("handling event with path %s, payload: %s and header: %v", request.URL.Path, string(event), request.Header)
	trItems, err := r.selectTriggers(el.Spec.NamespaceSelector, el.Spec.LabelSelector)
	if err != nil {
		r.Logger.Errorf("unable to select configured mergedTriggers: %s", err)
		response.WriteHeader(http.StatusInternalServerError)
		r.emitEvents(r.EventRecorder, el, events.TriggerProcessingFailedV1, err)
		r.sendCloudEvents(nil, *el, eventID, events.TriggerProcessingFailedV1)
		return
	}

	// Process any ungroupedTriggers
	mergedTriggers, err := r.merge(el.Spec.Triggers, trItems)
	if err != nil {
		log.Errorf("error merging triggers: %s", err)
		response.WriteHeader(http.StatusInternalServerError)
		r.emitEvents(r.EventRecorder, el, events.TriggerProcessingFailedV1, err)
		r.sendCloudEvents(nil, *el, eventID, events.TriggerProcessingFailedV1)
		return
	}
	r.WGProcessTriggers.Add(len(mergedTriggers))
	for _, t := range mergedTriggers {
		go func(t triggersv1.Trigger) {
			defer r.WGProcessTriggers.Done()
			localRequest := request.Clone(request.Context())
			emptyExtensions := make(map[string]interface{})
			r.processTrigger(t, el, localRequest, event, eventID, log, emptyExtensions)
		}(*t)
	}

	// Process grouped triggers
	for _, group := range el.Spec.TriggerGroups {
		r.WGProcessTriggers.Add(1)
		go func(g triggersv1.EventListenerTriggerGroup) {
			defer r.WGProcessTriggers.Done()
			localRequest := request.Clone(request.Context())
			r.processTriggerGroups(g, el, localRequest, event, eventID, log, r.WGProcessTriggers)
		}(group)
	}

	r.recordCountMetrics(successTag)

	body := Response{
		EventListener:    r.EventListenerName,
		EventListenerUID: elUID,
		Namespace:        r.EventListenerNamespace,
		EventID:          eventID,
	}

	msg := cehttp.NewMessageFromHttpRequest(request)
	if encoding := msg.ReadEncoding(); encoding == binding.EncodingUnknown {
		response.WriteHeader(http.StatusAccepted)
		response.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(response).Encode(body); err != nil {
			log.Errorf("failed to write back sink response: %v", err)
			r.emitEvents(r.EventRecorder, el, events.TriggerProcessingFailedV1, err)
			r.sendCloudEvents(nil, *el, eventID, events.TriggerProcessingFailedV1)
		}

	} else {
		responseEvent := cloudevents.NewEvent()
		responseEvent.SetID(uuid.New().String())
		responseEvent.SetSource(r.EventListenerName)

		_ = responseEvent.SetData(cloudevents.ApplicationJSON, body)

		eventResponse := binding.ToMessage(&responseEvent)
		defer func() {
			if err := eventResponse.Finish(nil); err != nil {
				log.Errorf("failed to close cloud event sink response: %v", err)
			}
		}()

		if err := cehttp.WriteResponseWriter(request.Context(), eventResponse, http.StatusAccepted, response); err != nil {
			log.Errorf("failed to write back cloud event sink response: %v", err)
			r.emitEvents(r.EventRecorder, el, events.TriggerProcessingFailedV1, err)
			r.sendCloudEvents(nil, *el, eventID, events.TriggerProcessingFailedV1)
		}
	}
	r.emitEvents(r.EventRecorder, el, events.TriggerProcessingDoneV1, nil)
	r.sendCloudEvents(nil, *el, eventID, events.TriggerProcessingDoneV1)
}

func (r Sink) sendCloudEvents(headers http.Header, el triggersv1.EventListener, eventID, eventType string) {
	data, err := json.Marshal(headers)
	if err != nil {
		r.Logger.Errorf("Error marshaling request Headers to json: %s", err)
		return
	}

	// If no cloudEventURI, then don't try to sendCloudEvents
	if r.CloudEventURI == "" {
		return
	}

	resource := cloudevent.Resource{
		EventID:   eventID,
		EventType: eventType,
		TargetURI: r.CloudEventURI,
		Client:    r.CEClient,
		Logger:    r.Logger,
		Data:      data,
		EL:        el,
	}

	go resource.SendCloudEvents()
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

func (r Sink) processTriggerGroups(g triggersv1.EventListenerTriggerGroup, el *triggersv1.EventListener, request *http.Request, event []byte, eventID string, eventLog *zap.SugaredLogger, wg *sync.WaitGroup) {
	log := eventLog.With(zap.String(triggers.TriggerGroupLabelKey, g.Name))

	extensions := map[string]interface{}{}
	payload, header, resp, err := r.ExecuteInterceptors(g.Interceptors, request, event, log, eventID, fmt.Sprintf("namespaces/%s/triggerGroups/%s", r.EventListenerNamespace, g.Name), r.EventListenerNamespace, extensions)
	if err != nil {
		log.Error(err)
		return
	}
	if resp != nil {
		if resp.Extensions != nil {
			for k, v := range resp.Extensions {
				extensions[k] = v
			}
		}
		if !resp.Continue {
			eventLog.Infof("interceptor stopped trigger processing: %v", resp.Status.Err())
			return
		}
	}

	trItems, err := r.selectTriggers(g.TriggerSelector.NamespaceSelector, g.TriggerSelector.LabelSelector)
	if err != nil {
		return
	}

	// Create a new HTTP request that contains the body and header from any interceptors in the TriggerGroup
	// This request will be passed on to the triggers in this group
	triggerReq := request.Clone(request.Context())
	triggerReq.Header = header
	triggerReq.Body = io.NopCloser(bytes.NewBuffer(payload))

	wg.Add(len(trItems))
	for _, t := range trItems {
		go func(t triggersv1.Trigger) {
			defer wg.Done()
			// TODO(dibyom): We might be able to get away with only cloning if necessary
			// i.e. if there are interceptors and iff those interceptors will modify the body/header (i.e. webhook)
			localRequest := triggerReq.Clone(triggerReq.Context())
			r.processTrigger(t, el, localRequest, event, eventID, log, extensions)
		}(*t)
	}
}

func (r Sink) selectTriggers(namespaceSelector triggersv1.NamespaceSelector, labelSelector *metav1.LabelSelector) ([]*triggersv1.Trigger, error) {
	var trItems []*triggersv1.Trigger
	var err error
	targetLabels := labels.Everything()
	if labelSelector != nil {
		targetLabels, err = metav1.LabelSelectorAsSelector(labelSelector)
		if err != nil {
			r.Logger.Errorf("failed to create label selector: %v", err)
			return nil, err
		}
	}
	var triggerFunc func() ([]*triggersv1.Trigger, error)
	switch {
	case len(namespaceSelector.MatchNames) == 1 && namespaceSelector.MatchNames[0] == "*":
		triggerFunc = func() ([]*triggersv1.Trigger, error) {
			return r.TriggerLister.List(targetLabels)
		}
	case len(namespaceSelector.MatchNames) != 0:
		triggerFunc = func() ([]*triggersv1.Trigger, error) {
			var trList []*triggersv1.Trigger
			for _, v := range namespaceSelector.MatchNames {
				trNsList, err := r.TriggerLister.Triggers(v).List(targetLabels)
				if err != nil {
					return nil, err
				}
				trList = append(trList, trNsList...)
			}
			return trList, nil
		}
	case len(namespaceSelector.MatchNames) == 0:
		if labelSelector != nil {
			triggerFunc = func() ([]*triggersv1.Trigger, error) {
				return r.TriggerLister.Triggers(r.EventListenerNamespace).List(targetLabels)
			}
		}
	}
	if triggerFunc == nil {
		return trItems, nil
	}
	trList, err := triggerFunc()
	if err != nil {
		r.Logger.Errorf("Error getting Triggers: %v", err)
		return nil, err
	}
	trItems = append(trItems, trList...)

	return trItems, nil
}

func (r Sink) processTrigger(t triggersv1.Trigger, el *triggersv1.EventListener, request *http.Request, event []byte, eventID string, eventLog *zap.SugaredLogger, extensions map[string]interface{}) {
	log := eventLog.With(zap.String(triggers.TriggerLabelKey, t.Name))

	log.Infof("*** request: %v", request)
	log.Infof("*** event: %v", event)
	log.Infof("*** extensions: %v", extensions)

	finalPayload, header, iresp, err := r.ExecuteTriggerInterceptors(t, request, event, log, eventID, extensions)
	if err != nil {
		log.Error(err)
		return
	}

	if iresp != nil {
		if !iresp.Continue {
			log.Infof("interceptor stopped trigger processing: %v", iresp.Status.Err())
			return
		}
	}

	rt, err := template.ResolveTrigger(t,
		r.TriggerBindingLister.TriggerBindings(t.Namespace).Get,
		r.ClusterTriggerBindingLister.Get,
		r.TriggerTemplateLister.TriggerTemplates(t.Namespace).Get)
	if err != nil {
		log.Error(err)
		return
	}
	if iresp != nil && iresp.Extensions != nil {
		extensions = iresp.Extensions
	}

	params, err := template.ResolveParams(rt, finalPayload, header, extensions, template.NewTriggerContext(eventID))
	if err != nil {
		log.Error(err)
		return
	}

	log.Infof("ResolvedParams : %+v", params)
	resources := template.ResolveResources(rt.TriggerTemplate, params)

	if err := r.CreateResources(t.Namespace, t.Spec.ServiceAccountName, resources, t.Name, eventID, log); err != nil {
		log.Error(err)
		return
	}
	go r.recordResourceCreation(resources)
	r.emitEvents(r.EventRecorder, el, events.TriggerProcessingSuccessfulV1, nil)
	r.sendCloudEvents(request.Header, *el, eventID, events.TriggerProcessingSuccessfulV1)

}

func (r Sink) ExecuteTriggerInterceptors(t triggersv1.Trigger, in *http.Request, event []byte, log *zap.SugaredLogger, eventID string, extensions map[string]interface{}) ([]byte, http.Header, *triggersv1.InterceptorResponse, error) {
	return r.ExecuteInterceptors(t.Spec.Interceptors, in, event, log, eventID, fmt.Sprintf("namespaces/%s/triggers/%s", t.Namespace, t.Name), t.Namespace, extensions)
}

// ExecuteInterceptor executes all interceptors for the Trigger and returns back the body, header, and InterceptorResponse to use.
// When TEP-0022 is fully implemented, this function will only return the InterceptorResponse and error.
func (r Sink) ExecuteInterceptors(trInt []*triggersv1.TriggerInterceptor, in *http.Request, event []byte, log *zap.SugaredLogger, eventID string, triggerID string, namespace string, extensions map[string]interface{}) ([]byte, http.Header, *triggersv1.InterceptorResponse, error) {
	if len(trInt) == 0 {
		return event, in.Header, nil, nil
	}

	request := triggersv1.InterceptorRequest{
		Body:       string(event),
		Header:     in.Header.Clone(),
		Extensions: extensions,
		Context: &triggersv1.TriggerContext{
			EventURL: in.URL.String(),
			EventID:  eventID,
			// t.Name might not be fully accurate until we get rid of triggers inlined within EventListener
			TriggerID: triggerID,
		},
	}

	// request is the request sent to the interceptors in the chain. Each interceptor can set the InterceptorParams field
	// or add to the Extensions

	for _, i := range trInt {
		if i.Webhook != nil { // Old style interceptor
			body, err := extendBodyWithExtensions([]byte(request.Body), request.Extensions)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("could not merge extensions with body: %w", err)
			}
			req := &http.Request{
				Method: http.MethodPost,
				Header: request.Header,
				URL:    in.URL,
				Body:   io.NopCloser(bytes.NewBuffer(body)),
			}
			interceptor := webhook.NewInterceptor(i.Webhook, r.HTTPClient, namespace, log)

			res, err := interceptor.ExecuteTrigger(req)
			if err != nil {
				return nil, nil, nil, err
			}

			payload, err := io.ReadAll(res.Body)
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

		var url *apis.URL
		if i.Ref.Kind == triggersv1.ClusterInterceptorKind {
			ic, err := r.ClusterInterceptorLister.Get(i.GetName())
			if err != nil {
				return nil, nil, nil, fmt.Errorf("url resolution failed for interceptor %s with: %w", i.GetName(), err)
			}
			if ic.Status.Address != nil && ic.Status.Address.URL != nil {
				url = ic.Status.Address.URL
			} else if url, err = ic.ResolveAddress(); err != nil {
				return nil, nil, nil, fmt.Errorf("url resolution failed for interceptor %s with: %w", i.GetName(), err)
			}
			if err != nil {
				return nil, nil, nil, fmt.Errorf("could not resolve clusterinterceptor URL: %w", err)
			}
		} else if i.Ref.Kind == triggersv1.NamespacedInterceptorKind {
			if r.InterceptorLister == nil {
				r.Logger.Debugf("nil lister")
			}
			ic, err := r.InterceptorLister.Interceptors(r.EventListenerNamespace).Get(i.GetName())
			if err != nil {
				return nil, nil, nil, fmt.Errorf("url resolution failed for interceptor %s with: %w", i.GetName(), err)
			}
			if addr := ic.Status.Address; addr != nil && addr.URL != nil {
				url = addr.URL
			} else if url, err = ic.ResolveAddress(); err != nil {
				return nil, nil, nil, fmt.Errorf("url resolution failed for interceptor %s with: %w", i.GetName(), err)
			}
			if err != nil {
				return nil, nil, nil, fmt.Errorf("could not resolve nameSpacedinterceptor URL: %w", err)
			}
		}

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

		// handle form-data payload for slack only
		if v := in.Header.Get("X-Slack-Signature"); v != "" {
			jsonBody, err := json.Marshal(request.Body)
			if err != nil {
				body, _ := io.ReadAll(in.Body)
				request.Body = string(body)
			} else {
				request.Body = string(jsonBody)
			}
		}

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
