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
	"io/ioutil"
	"net/http"
	"path"

	"google.golang.org/grpc/codes"
	"knative.dev/pkg/apis"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	corev1lister "k8s.io/client-go/listers/core/v1"
)

const (
	CoreInterceptorsHost = "tekton-triggers-core-interceptors"
)

// Interceptor is the interface that all interceptors implement.
type Interceptor interface {
	ExecuteTrigger(req *http.Request) (*http.Response, error)
}

type key string

const RequestCacheKey key = "interceptors.RequestCache"

func getCache(req *http.Request) map[string]interface{} {
	if cache, ok := req.Context().Value(RequestCacheKey).(map[string]interface{}); ok {
		return cache
	}

	return make(map[string]interface{})
}

// GetSecretToken queries Kubernetes for the given secret reference. We use this function
// to resolve secret material like GitHub webhook secrets, and call it once for every
// trigger that references it.
//
// As we may have many triggers that all use the same secret, we cache the secret values
// in the request cache.
func GetSecretToken(req *http.Request, sl corev1lister.SecretLister, sr *triggersv1.SecretRef, triggerNS string) ([]byte, error) {
	var cache map[string]interface{}

	cacheKey := path.Join("secret", triggerNS, sr.SecretName, sr.SecretKey)
	if req != nil {
		cache = getCache(req)
		if secretValue, ok := cache[cacheKey]; ok {
			return secretValue.([]byte), nil
		}
	}

	secret, err := sl.Secrets(triggerNS).Get(sr.SecretName)
	if err != nil {
		return nil, err
	}

	secretValue := secret.Data[sr.SecretKey]
	if req != nil {
		cache[cacheKey] = secret.Data[sr.SecretKey]
	}

	return secretValue, nil
}

// GetInterceptorParams returns InterceptorParams for the current interceptors
func GetInterceptorParams(i *triggersv1.EventInterceptor) map[string]interface{} {
	ip := map[string]interface{}{}
	switch {
	case i.Webhook != nil:
		// WebHook headers are of type map[string][]string.
		// Use old style for now. Upgrade later.
		if i.Webhook != nil {
			ip["objectRef"] = i.Webhook.ObjectRef
			ip["header"] = i.Webhook.Header
		}
	case i.DeprecatedGitHub != nil:
		if i.DeprecatedGitHub.EventTypes != nil {
			ip["eventTypes"] = i.DeprecatedGitHub.EventTypes
		}
		if i.DeprecatedGitHub.SecretRef != nil {
			ip["secretRef"] = i.DeprecatedGitHub.SecretRef
		}
	case i.DeprecatedGitLab != nil:
		if i.DeprecatedGitLab.EventTypes != nil {
			ip["eventTypes"] = i.DeprecatedGitLab.EventTypes
		}
		if i.DeprecatedGitLab.SecretRef != nil {
			ip["secretRef"] = i.DeprecatedGitLab.SecretRef
		}
	case i.DeprecatedCEL != nil:
		if i.DeprecatedCEL.Filter != "" {
			ip["filter"] = i.DeprecatedCEL.Filter
		}

		if i.DeprecatedCEL.Overlays != nil {
			ip["overlays"] = i.DeprecatedCEL.Overlays
		}
	case i.DeprecatedBitbucket != nil:
		if i.DeprecatedBitbucket.EventTypes != nil {
			ip["eventTypes"] = i.DeprecatedBitbucket.EventTypes
		}
		if i.DeprecatedBitbucket.SecretRef != nil {
			ip["secretRef"] = i.DeprecatedBitbucket.SecretRef
		}
	case i.Params != nil:
		for _, p := range i.Params {
			ip[p.Name] = p.Value
		}
	}
	return ip
}

// Fail constructs a InterceptorResponse that should not continue further processing.
func Fail(c codes.Code, msg string) *triggersv1.InterceptorResponse {
	return &triggersv1.InterceptorResponse{
		Continue: false,
		Status: triggersv1.Status{
			Code:    c,
			Message: msg,
		},
	}
}

// Failf constructs a InterceptorResponse that should not continue further processing.
func Failf(c codes.Code, format string, a ...interface{}) *triggersv1.InterceptorResponse {
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

type InterceptorGetter func(name string) (*triggersv1.ClusterInterceptor, error)

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

func Execute(ctx context.Context, client *http.Client, req *triggersv1.InterceptorRequest, url string) (*triggersv1.InterceptorResponse, error) {
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
	body, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		// TODO: error type for easier checking. wrap in status.Errorf?
		return nil, fmt.Errorf("interceptor response was not 200: %v", string(body))
	}
	iresp := triggersv1.InterceptorResponse{}
	if err := json.Unmarshal(body, &iresp); err != nil {
		return nil, err
	}
	return &iresp, nil
}
