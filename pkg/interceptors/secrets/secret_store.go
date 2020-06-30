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

package secrets

import (
	"fmt"
	"reflect"
	"time"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fields "k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// SecretStore provides cached lookups of k8s secrets, backed by a Reflector.
type SecretStore interface {
	Get(sr triggersv1.SecretRef) ([]byte, error)
}

type secretStore struct {
	store                  cache.Store
	eventListenerNamespace string
}

// NewSecretStore provides cached lookups of k8s secrets, backed by a Reflector.
func NewSecretStore(cs kubernetes.Interface, ns string, resyncInterval time.Duration, stopCh <-chan struct{}) SecretStore {
	store := cache.NewStore(func(obj interface{}) (string, error) {
		secret, ok := obj.(*corev1.Secret)
		if !ok {
			return "", fmt.Errorf("object is not a secret; got %s", reflect.TypeOf(obj))
		}

		return fmt.Sprintf("%s/%s", secret.Namespace, secret.Name), nil
	})

	secretStore := secretStore{
		store:                  store,
		eventListenerNamespace: ns,
	}

	lw := cache.NewListWatchFromClient(cs.CoreV1().RESTClient(), "secrets", metav1.NamespaceAll, fields.Everything())
	reflector := cache.NewReflector(lw, corev1.Secret{}, store, resyncInterval)

	go reflector.Run(stopCh)

	return secretStore
}

// Get returns the secret value for a given SecretRef.
func (ws secretStore) Get(sr triggersv1.SecretRef) ([]byte, error) {
	cachedObj, ok, _ := ws.store.GetByKey(ws.getKey(sr))
	if !ok {
		return nil, fmt.Errorf("secret not found: %s", sr.SecretName)
	}

	secret, ok := cachedObj.(*corev1.Secret)
	if !ok {
		return nil, fmt.Errorf("cached object is not a secret: %s", sr.SecretName)
	}

	value, ok := secret.Data[sr.SecretKey]
	if !ok {
		return nil, fmt.Errorf("key not found: %s", sr.SecretKey)
	}

	return value, nil
}

func (ws *secretStore) getKey(sr triggersv1.SecretRef) string {
	var namespace string
	if sr.Namespace == "" {
		namespace = ws.eventListenerNamespace
	} else {
		namespace = sr.Namespace
	}
	return fmt.Sprintf("%s/%s", namespace, sr.SecretName)
}
