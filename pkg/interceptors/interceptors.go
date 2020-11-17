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
	"context"
	"net/http"
	"path"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Interceptor is the interface that all interceptors implement.
type Interceptor interface {
	ExecuteTrigger(req *http.Request) (*http.Response, error)
}

type key string

const requestCacheKey key = "interceptors.RequestCache"

func getCache(req *http.Request) map[string]interface{} {
	if cache, ok := req.Context().Value(requestCacheKey).(map[string]interface{}); ok {
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
func GetSecretToken(req *http.Request, cs kubernetes.Interface, sr *triggersv1.SecretRef, eventListenerNamespace string) ([]byte, error) {
	var cache map[string]interface{}

	cacheKey := path.Join("secret", eventListenerNamespace, sr.SecretName, sr.SecretKey)
	if req != nil {
		cache = getCache(req)
		if secretValue, ok := cache[cacheKey]; ok {
			return secretValue.([]byte), nil
		}
	}

	secret, err := cs.CoreV1().Secrets(eventListenerNamespace).Get(context.Background(), sr.SecretName, metav1.GetOptions{})
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
	case i.GitHub != nil:
		if i.GitHub.EventTypes != nil {
			ip["eventTypes"] = i.GitHub.EventTypes
		}
		if i.GitHub.SecretRef != nil {
			ip["secretRef"] = i.GitHub.SecretRef
		}
	case i.GitLab != nil:
		if i.GitLab.EventTypes != nil {
			ip["eventTypes"] = i.GitLab.EventTypes
		}
		if i.GitLab.SecretRef != nil {
			ip["secretRef"] = i.GitLab.SecretRef
		}
	case i.CEL != nil:
		if i.CEL.Filter != "" {
			ip["filter"] = i.CEL.Filter
		}

		if i.CEL.Overlays != nil {
			ip["overlays"] = i.CEL.Overlays
		}

	case i.Bitbucket != nil:
		if i.Bitbucket.EventTypes != nil {
			ip["eventTypes"] = i.Bitbucket.EventTypes
		}
		if i.Bitbucket.SecretRef != nil {
			ip["secretRef"] = i.Bitbucket.SecretRef
		}
	}

	return ip
}
