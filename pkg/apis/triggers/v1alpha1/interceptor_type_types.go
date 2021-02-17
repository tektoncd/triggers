package v1alpha1

import (
	"errors"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

// Check that EventListener may be validated and defaulted.
var _ apis.Validatable = (*InterceptorType)(nil)
var _ apis.Defaultable = (*InterceptorType)(nil)

// +genclient
// +genclient:nonNamespaced
// +genreconciler:krshapedlogic=false
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// InterceptorType describes a pluggable interceptor including configuration
// such as the fields it accepts and its deployment address
type InterceptorType struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec InterceptorTypeSpec `json:"spec"`
	// +optional
	Status InterceptorTypeStatus `json:"status"`
}

// InterceptorTypeSpec describes the Spec for an InterceptorType
type InterceptorTypeSpec struct {
	ClientConfig ClientConfig `json:"clientConfig"`
}

// InterceptorTypeStatus holds the status of the InterceptorType
// +k8s:deepcopy-gen=true
type InterceptorTypeStatus struct {
	duckv1.Status `json:",inline"`

	// InterceptorType is Addressable and exposes the URL where the Interceptor is running
	duckv1.AddressStatus `json:",inline"`
}

// ClientConfig describes how a client can communicate with the Interceptor
type ClientConfig struct {
	// URL is a fully formed URL pointing to the interceptor
	// Mutually exclusive with Service
	URL *apis.URL `json:"url,omitempty"`

	// Service is a reference to a Service object where the interceptor is running
	// Mutually exclusive with URL
	Service *ServiceReference `json:"service,omitempty"`
}

// ServServiceReference is a reference to a Service object
// with an optional path
type ServiceReference struct {
	// Name is the name of the service
	Name string `json:"name"`

	// Namespace is the namespace of the service
	Namespace string `json:"namespace"`

	// Path is an optional URL path
	// +optional
	Path string

	// Port is a valid port number
	Port int
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// InterceptorTypeList contains a list of InterceptorTypes
// We don't use this but it's required for certain codegen features.
type InterceptorTypeList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InterceptorType `json:"items"`
}

var ErrNilURL = errors.New("interceptor URL was nil")

// ResolveAddress returns the URL where the interceptor is running using its clientConfig
func (it *InterceptorType) ResolveAddress() (*apis.URL, error) {
	if url := it.Spec.ClientConfig.URL; url != nil {
		return url, nil
	}
	svc := it.Spec.ClientConfig.Service
	if svc == nil {
		return nil, ErrNilURL
	}

	url := &apis.URL{
		Scheme: "http", // TODO: Support HTTPs if caBundle is present
		Host:   fmt.Sprintf("%s.%s.svc:%d", svc.Name, svc.Namespace, svc.Port),
		Path:   svc.Path,
	}
	return url, nil
}
