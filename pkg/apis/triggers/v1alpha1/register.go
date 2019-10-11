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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	// GroupName is the Kubernetes resource group name for Tekton types.
	GroupName = "tekton.dev"

	// EventListenerLabelKey is used as the label identifier for an EventListener.
	EventListenerLabelKey = "/eventlistener"

	// EventIDLabelKey is used as the label identifier for an EventListener event.
	EventIDLabelKey = "/triggers-eventid"

	// TriggerBindingLabelKey is used as the label identifier for a TriggerBinding.
	TriggerBindingLabelKey = "/triggerbinding"

	// TriggerTemplateKey is used as the label identifier for a TriggerTemplate
	TriggerTemplateKey = "/triggertemplate"

	// LabelEscape is an escaped GroupName safe for use in resource types.
	LabelEscape = "tekton\\.dev"
)

// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: "v1alpha1"}

// Kind takes an unqualified kind and returns back a Group qualified GroupKind
func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

var (
	schemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

	// AddToScheme adds Build types to the scheme.
	AddToScheme = schemeBuilder.AddToScheme
)

// Adds the list of known types to Scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&EventListener{},
		&EventListenerList{},
		&TriggerBinding{},
		&TriggerBindingList{},
		&TriggerTemplate{},
		&TriggerTemplateList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
