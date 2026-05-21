/*
Copyright 2024 The Tekton Authors

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
	"knative.dev/pkg/apis"
)

// Check that ScheduledTemplate may be validated and defaulted.
var _ apis.Validatable = (*ScheduledTemplate)(nil)
var _ apis.Defaultable = (*ScheduledTemplate)(nil)

// ScheduledTemplateSpec holds the desired state of ScheduledTemplate
type ScheduledTemplateSpec struct {
	// +listType=atomic
	// Embed TriggerTemplateSpec to inherit Params and ResourceTemplates
	TriggerTemplateSpec `json:",inline"`
	// URL of the external event trigger is listening to
	CloudEventSink *apis.URL `json:"cloudEventSink,omitempty"`
	// Cron job schedule for the trigger
	Schedule string `json:"schedule,omitempty"`
}

// ScheduledTemplateStatus describes the desired state of ScheduledTemplate
type ScheduledTemplateStatus struct{}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ScheduledTemplate takes parameters and uses them to create CRDs
// +k8s:openapi-gen=true
type ScheduledTemplate struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec holds the desired state of the ScheduledTemplate from the client
	// +optional
	Spec ScheduledTemplateSpec `json:"spec"`
	// +optional
	Status ScheduledTemplateStatus `json:"status,omitempty"`
}

// ScheduledTemplateList contains a list of ScheduledTemplate
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ScheduledTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ScheduledTemplate `json:"items"`
}
