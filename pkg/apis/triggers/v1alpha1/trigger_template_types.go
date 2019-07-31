package v1alpha1

/*
Copyright 2019 The Tekton Authors.
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

import (
	"encoding/json"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Check that TriggerTemplate may be validated and defaulted.
//var _ apis.Validatable = (*TriggerTemplate)(nil)

// +k8s:deepcopy-gen=true
type TriggerTemplateSpec struct {
	Params            []pipelinev1.ParamSpec    `json:"params,omitempty"`
	ResourceTemplates []TriggerResourceTemplate `json:"resourcetemplates,omitempty"`
}

// +k8s:deepcopy-gen=true
type TriggerResourceTemplate struct {
	json.RawMessage `json:",inline"`
}

type TriggerTemplateStatus struct{}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TriggerTemplate takes parameters and uses them to create CRDs
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type TriggerTemplate struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec holds the desired state of the EventListener from the client
	// +optional
	Spec TriggerTemplateSpec `json:"spec"`
	// +optional
	Status TriggerTemplateStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TriggerTemplateList contains a list of TriggerTemplate
// +k8s:deepcopy-gen=true
type TriggerTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TriggerTemplate `json:"items"`
}
