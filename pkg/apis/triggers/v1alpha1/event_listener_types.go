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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/apis"
)

// Check that EventListener may be validated and defaulted.
var _ apis.Validatable = (*EventListener)(nil)

type EventListenerSpec struct {
	TriggerBindingRefs []TriggerBindingRef `json:"triggerbindingrefs,omitempty"`
}

type TriggerBindingRef struct {
	Name       string `json:"name,omitempty"`
	Namespace  string `json:"namespace,omitempty"`
	APIVersion string `json:"apiversion,omitempty"`
}

type EventListenerStatus struct{}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EventListener exposes a service to accept webhook events.
// +k8s:openapi-gen=true
type EventListener struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec holds the desired state of the EventListener from the client
	// +optional
	Spec EventListenerSpec `json:"spec"`
	// +optional
	Status EventListenerStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EventListenerList contains a list of TriggerBinding
type EventListenerList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EventListener `json:"items"`
}

// GetOwnerReference gets the EventListener as owner reference for any related objects
func (el *EventListener) GetOwnerReference() *metav1.OwnerReference {
	return metav1.NewControllerRef(el, schema.GroupVersionKind{
		Group:   SchemeGroupVersion.Group,
		Version: SchemeGroupVersion.Version,
		Kind:    "EventListener",
	})
}
