package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duckv1alpha1 "knative.dev/pkg/apis/duck/v1alpha1"
)

// Check that Trigger may be validated and defaulted.
var _ apis.Validatable = (*Trigger)(nil)
var _ apis.Defaultable = (*Trigger)(nil)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Trigger represents a connection between TriggerBinding, Interceptor,
// and TriggerTemplate.
//
// +k8s:openapi-gen=true
type Trigger struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec holds the desired state of the EventListener from the client
	// +optional
	Spec TriggerSpec `json:"spec"`
	// +optional
	Status TriggerStatus `json:"status"`
}

// TriggerStatus defines the observed state of Trigger.
type TriggerStatus struct {
	// Trigger is Addressable.  of the the EventListener sink
	duckv1alpha1.AddressStatus `json:",inline"`
}

// TriggerSpec represents a connection between TriggerBinding, Interceptor,
// and TriggerTemplate; TriggerBinding provides extracted values for
// TriggerTemplate to then create resources from.
type TriggerSpec struct {
	Bindings     []*EventListenerBinding `json:"bindings"`
	Template     EventListenerTemplate   `json:"template"`
	Interceptors []*EventInterceptor     `json:"interceptors,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TriggerList contains a list of Triggers.
// We don't use this but it's required for certain codegen features.
type TriggerList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Trigger `json:"items"`
}
