package v1alpha1

import (
	"context"
	"strings"

	"google.golang.org/grpc/status"
)

type InterceptorInterface interface {
	Process(ctx context.Context, r *InterceptorRequest) *InterceptorResponse
}

// Do not generate DeepCopy(). See #827
// +k8s:deepcopy-gen=false
type InterceptorRequest struct {
	// Body is the incoming HTTP event body
	Body []byte `json:"body,omitempty"`
	// Header are the headers for the incoming HTTP event
	Header map[string][]string `json:"header,omitempty"`
	// Extensions are extra values that are added by previous interceptors in a chain
	Extensions map[string]interface{} `json:"extensions,omitempty"`

	// InterceptorParams are the user specified params for interceptor in the Trigger
	InterceptorParams map[string]interface{} `json:"interceptor_params,omitempty"`

	// Context contains additional metadata about the event being processed
	Context *TriggerContext
}

type TriggerContext struct {
	// EventURL is the URL of the incoming event
	EventURL string `json:"url,omitempty"`
	// EventID is a unique ID assigned by Triggers to each event
	EventID string `json:"event_id,omitempty"`
	// TriggerID is of the form namespace/$ns/triggers/$name
	TriggerID string `json:"trigger_id,omitempty"`
}

// Do not generate Deepcopy(). See #827
// +k8s:deepcopy-gen=false
type InterceptorResponse struct {
	// Extensions are additional fields that is added to the interceptor event.
	// See TEP-0022. Naming TBD.
	Extensions map[string]interface{} `json:"extensions,omitempty"`
	// Continue indicates if the EventListener should continue processing the Trigger or not
	Continue bool `json:"continue,omitempty"`
	// Status is an Error status containing details on any interceptor processing errors
	Status *status.Status `json:"status,omitempty"`
}

func ParseTriggerID(triggerID string) (namespace, name string) {
	splits := strings.Split(triggerID, "/")
	if len(splits) != 4 {
		return
	}

	return splits[1], splits[3]
}
