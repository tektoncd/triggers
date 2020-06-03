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
	"errors"
	"fmt"
	"net/http"
	"time"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// WebhookSecretStore provides cached lookups of k8s secrets, backed by a Reflector.
type WebhookSecretStore interface {
	Get(sr triggersv1.SecretRef) ([]byte, error)
}

type webhookSecretStore struct {
	store                  cache.Store
	eventListenerNamespace string
}

// NewWebhookSecretStore provides cached lookups of k8s secrets, backed by a Reflector.
func NewWebhookSecretStore(cs kubernetes.Interface, ns string, resyncInterval time.Duration, stopCh <-chan struct{}) WebhookSecretStore {
	secretsClient := cs.CoreV1().Secrets(metav1.NamespaceAll)
	store := cache.NewStore(func(obj interface{}) (string, error) {
		secret, ok := obj.(corev1.Secret)
		if !ok {
			return "", errors.New("object is not a secret")
		}

		return fmt.Sprintf("%s/%s", secret.Namespace, secret.Name), nil
	})

	secretStore := webhookSecretStore{
		store:                  store,
		eventListenerNamespace: ns,
	}

	reflector := cache.NewReflector(secretsClient, corev1.Secret{}, store, resyncInterval)

	go reflector.Run(stopCh)

	return secretStore
}

// Get returns the secret value for a given SecretRef.
func (ws *WebhookSecretStore) Get(sr triggersv1.SecretRef) ([]byte, error) {
	cachedObj, ok, _ := ws.store.GetByKey(getKey(sr))
	if !ok {
		return nil, fmt.Errorf("secret not found: %s", sr.SecretName)
	}

	secret, ok := cachedObj.(corev1.Secret)
	if !ok {
		return nil, fmt.Errorf("cached object is not a secret: %s", sr.SecretName)
	}

	value, ok := secret.Data[sr.SecretKey]
	if !ok {
		return nil, fmt.Errorf("key not found: %s", sr.SecretKey)
	}

	return value, nil
}

<<<<<<< HEAD
// Interceptor is the interface that all interceptors implement.
type Interceptor interface {
	ExecuteTrigger(req *http.Request) (*http.Response, error)
}

type key string

const requestCacheKey key = "interceptors.RequestCache"

// WithCache clones the given request and sets the request context to include a cache.
// This allows us to cache results from expensive operations and perform them just once
// per each trigger.
//
// Each request should have its own cache, and those caches should expire once the request
// is processed. For this reason, it's appropriate to store the cache on the request
// context.
func WithCache(req *http.Request) *http.Request {
	return req.WithContext(context.WithValue(req.Context(), requestCacheKey, make(map[string]interface{})))
}

func getCache(req *http.Request) map[string]interface{} {
	if cache, ok := req.Context().Value(requestCacheKey).(map[string]interface{}); ok {
		return cache
=======
func (ws *WebhookSecretStore) getKey(sr triggersv1.SecretRef) string {
	var namespace string
	if sr.Namespace == "" {
		namespace = ws.eventListenerNamespace
	} else {
		namespace = sr.Namespace
>>>>>>> 68edb108... use webhook secret store
	}
	return fmt.Sprintf("%s/%s", namespace, sr.Name)
}

// Interceptor is the interface that all interceptors implement.
type Interceptor interface {
	ExecuteTrigger(req *http.Request) (*http.Response, error)
}
