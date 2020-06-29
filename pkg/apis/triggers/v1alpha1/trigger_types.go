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
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TriggerSpec represents a connection between TriggerBinding, Params,
// and TriggerTemplate; TriggerBinding provides extracted values for
// TriggerTemplate to then create resources from.
type TriggerSpec struct {
	Bindings []*TriggerBinding `json:"bindings"`
	Template TriggerTemplate   `json:"template"`
	// +optional
	Name         string                `json:"name,omitempty"`
	Interceptors []*TriggerInterceptor `json:"interceptors,omitempty"`
	// ServiceAccount optionally associates credentials with each trigger;
	// more granular authorization for
	// who is allowed to utilize the associated pipeline
	// vs. defaulting to whatever permissions are associated
	// with the entire EventListener and associated sink facilitates
	// multi-tenant model based scenarios
	// TODO do we want to restrict this to the event listener namespace and just ask for the service account name here?
	// +optional
	ServiceAccount *corev1.ObjectReference `json:"serviceAccount,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Trigger defines a mapping of an input event to parameters. This is used
// to extract information from events to be passed to TriggerTemplates within a
// Trigger.
// +k8s:openapi-gen=true
type Trigger struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec holds the desired state of the Trigger
	// +optional
	Spec TriggerSpec `json:"spec"`
}

// TriggerInterceptor provides a hook to intercept and pre-process events
type TriggerInterceptor struct {
	Webhook   *WebhookInterceptor   `json:"webhook,omitempty"`
	GitHub    *GitHubInterceptor    `json:"github,omitempty"`
	GitLab    *GitLabInterceptor    `json:"gitlab,omitempty"`
	CEL       *CELInterceptor       `json:"cel,omitempty"`
	Bitbucket *BitbucketInterceptor `json:"bitbucket,omitempty"`
}

// WebhookInterceptor provides a webhook to intercept and pre-process events
type WebhookInterceptor struct {
	// ObjectRef is a reference to an object that will resolve to a cluster DNS
	// name to use as the EventInterceptor. Either objectRef or url can be specified
	// +optional
	ObjectRef *corev1.ObjectReference `json:"objectRef,omitempty"`
	// Header is a group of key-value pairs that can be appended to the
	// interceptor request headers. This allows the interceptor to make
	// decisions specific to an EventListenerTrigger.
	Header []v1beta1.Param `json:"header,omitempty"`
}

// BitbucketInterceptor provides a webhook to intercept and pre-process events
type BitbucketInterceptor struct {
	SecretRef  *SecretRef `json:"secretRef,omitempty"`
	EventTypes []string   `json:"eventTypes,omitempty"`
}

// GitHubInterceptor provides a webhook to intercept and pre-process events
type GitHubInterceptor struct {
	SecretRef  *SecretRef `json:"secretRef,omitempty"`
	EventTypes []string   `json:"eventTypes,omitempty"`
}

// GitLabInterceptor provides a webhook to intercept and pre-process events
type GitLabInterceptor struct {
	SecretRef  *SecretRef `json:"secretRef,omitempty"`
	EventTypes []string   `json:"eventTypes,omitempty"`
}

// CELInterceptor provides a webhook to intercept and pre-process events
type CELInterceptor struct {
	Filter   string       `json:"filter,omitempty"`
	Overlays []CELOverlay `json:"overlays,omitempty"`
}

// CELOverlay provides a way to modify the request body using CEL expressions
type CELOverlay struct {
	Key        string `json:"key,omitempty"`
	Expression string `json:"expression,omitempty"`
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
