package v1alpha1

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
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
	Status Status `json:"status"`
}

type Status struct {
	// The status code, which should be an enum value of [google.rpc.Code][google.rpc.Code].
	Code codes.Code `json:"code,omitempty"`
	// A developer-facing error message, which should be in English.
	Message string `json:"message,omitempty"`
}

func (s Status) Err() StatusError {
	return StatusError{s: s}
}

type StatusError struct {
	s Status
}

func (s StatusError) Error() string {
	return fmt.Sprintf("rpc error: code = %s desc = %s", s.s.Code, s.s.Message)
}

func ParseTriggerID(triggerID string) (namespace, name string) {
	splits := strings.Split(triggerID, "/")
	if len(splits) != 4 {
		return
	}

	return splits[1], splits[3]
}
