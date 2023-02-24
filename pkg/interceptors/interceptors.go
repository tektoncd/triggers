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

package interceptors

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	triggersv1alpha1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	triggersv1beta1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"google.golang.org/grpc/codes"
	"knative.dev/pkg/apis"
)

const (
	CoreInterceptorsHost = "tekton-triggers-core-interceptors"
)

// Interceptor is the interface that all interceptors implement.
type Interceptor interface {
	ExecuteTrigger(req *http.Request) (*http.Response, error)
}

// GetInterceptorParams returns InterceptorParams for the current interceptors
func GetInterceptorParams(i *triggersv1beta1.EventInterceptor) map[string]interface{} {
	ip := map[string]interface{}{}
	switch {
	case i.Webhook != nil:
		// WebHook headers are of type map[string][]string.
		// Use old style for now. Upgrade later.
		if i.Webhook != nil {
			ip["objectRef"] = i.Webhook.ObjectRef
			ip["header"] = i.Webhook.Header
		}
	case i.Params != nil:
		for _, p := range i.Params {
			ip[p.Name] = p.Value
		}
	}
	return ip
}

// Fail constructs a InterceptorResponse that should not continue further processing.
func Fail(c codes.Code, msg string) *triggersv1beta1.InterceptorResponse {
	return &triggersv1beta1.InterceptorResponse{
		Continue: false,
		Status: triggersv1beta1.Status{
			Code:    c,
			Message: msg,
		},
	}
}

// Failf constructs a InterceptorResponse that should not continue further processing.
func Failf(c codes.Code, format string, a ...interface{}) *triggersv1beta1.InterceptorResponse {
	return Fail(c, fmt.Sprintf(format, a...))
}

// Canonical updates the map keys to use the Canonical name
func Canonical(h map[string][]string) http.Header {
	c := map[string][]string{}
	for k, v := range h {
		c[http.CanonicalHeaderKey(k)] = v
	}
	return http.Header(c)
}

// UnmarshalParams unmarshalls the passed in InterceptorParams into the provided param struct
func UnmarshalParams(ip map[string]interface{}, p interface{}) error {
	b, err := json.Marshal(ip)
	if err != nil {
		return fmt.Errorf("failed to marshal json: %w", err)
	}

	if err := json.Unmarshal(b, &p); err != nil {
		// Should never happen since Unmarshall only returns err if json is invalid which we already check above
		return fmt.Errorf("invalid json: %w", err)
	}
	return nil
}

type InterceptorGetter func(name string) (*triggersv1alpha1.ClusterInterceptor, error)

// ResolveToURL finds an Interceptor's URL.
func ResolveToURL(getter InterceptorGetter, name string) (*apis.URL, error) {
	ic, err := getter(name)
	if err != nil {
		return nil, fmt.Errorf("url resolution failed for interceptor %s with: %w", name, err)
	}
	if addr := ic.Status.Address; addr != nil {
		if addr.URL != nil {
			return addr.URL, nil
		}
	}
	// If the status does not have a URL, try to generate it from the Spec.
	return ic.ResolveAddress()
}

func Execute(ctx context.Context, client *http.Client, req *triggersv1beta1.InterceptorRequest, url string) (*triggersv1beta1.InterceptorResponse, error) {

	b, err := json.Marshal(req)

	if err != nil {
		return nil, err
	}
	// TODO: Seed context with timeouts
	r, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(b))
	if err != nil {
		return nil, err
	}
	res, err := client.Do(r)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		// TODO: error type for easier checking. wrap in status.Errorf?
		return nil, fmt.Errorf("interceptor response was not 200: %v", string(body))
	}
	iresp := triggersv1beta1.InterceptorResponse{}
	if err := json.Unmarshal(body, &iresp); err != nil {
		return nil, err
	}
	return &iresp, nil
}
