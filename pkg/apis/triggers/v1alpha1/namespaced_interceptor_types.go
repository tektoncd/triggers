/*
Copyright 2022 The Tekton Authors

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
	"bytes"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

// Check that NamespacedInterceptor may be validated and defaulted.
var _ apis.Validatable = (*NamespacedInterceptor)(nil)
var _ apis.Defaultable = (*NamespacedInterceptor)(nil)

// +genclient
// +genclient:nonNamespaced
// +genreconciler:krshapedlogic=false
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// NamespacedInterceptor describes a pluggable interceptor including configuration
// such as the fields it accepts and its deployment address. The type is based on
// the Validating/MutatingWebhookConfiguration types for configuring AdmissionWebhooks
type NamespacedInterceptor struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec NamespacedInterceptorSpec `json:"spec"`
	// +optional
	Status NamespacedInterceptorStatus `json:"status"`
}

// NamespacedInterceptorSpec describes the Spec for an NamespacedInterceptor
type NamespacedInterceptorSpec struct {
	ClientConfig ClientConfig `json:"clientConfig"`
}

// NamespacedInterceptorStatus holds the status of the NamespacedInterceptor
// +k8s:deepcopy-gen=true
type NamespacedInterceptorStatus struct {
	duckv1.Status `json:",inline"`

	// NamespacedInterceptor is Addressable and exposes the URL where the Interceptor is running
	duckv1.AddressStatus `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// NamespacedInterceptorList contains a list of NamespacedInterceptor
// We don't use this but it's required for certain codegen features.
type NamespacedInterceptorList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NamespacedInterceptor `json:"items"`
}

// ResolveAddress returns the URL where the interceptor is running using its clientConfig
func (it *NamespacedInterceptor) ResolveAddress() (*apis.URL, error) {
	if url := it.Spec.ClientConfig.URL; url != nil {
		return url, nil
	}
	svc := it.Spec.ClientConfig.Service
	if svc == nil {
		return nil, ErrNilURL
	}
	var (
		port *int32
		url  *apis.URL
	)

	if svc.Port != nil {
		port = svc.Port
	}

	if bytes.Equal(it.Spec.ClientConfig.CaBundle, []byte{}) {
		if port == nil {
			port = &defaultHTTPPort
		}
		url = formURL("http", svc, port)
	} else {
		if port == nil {
			port = &defaultHTTPSPort
		}
		url = formURL("https", svc, port)
	}
	return url, nil
}
