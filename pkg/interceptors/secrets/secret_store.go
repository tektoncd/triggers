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
	"sync"
	"time"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	fields "k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// SecretStore provides cached lookups of k8s secrets, backed by a Reflector.
type SecretStore interface {
	Get(sr triggersv1.SecretRef) ([]byte, error)
}

// We need one store per namespace, so we create this map
type storeSet map[string]cache.Store

type secretStore struct {
	mutex                  *sync.Mutex
	resyncInterval         time.Duration
	store                  storeSet
	stopCh                 <-chan struct{}
	kubeClient             kubernetes.Interface
	eventListenerNamespace string
}

// NewSecretStore provides cached lookups of k8s secrets, backed by a Reflector.
func NewSecretStore(cs kubernetes.Interface, ns string, resyncInterval time.Duration, stopCh <-chan struct{}) SecretStore {
	return secretStore{
		mutex:                  &sync.Mutex{},
		store:                  storeSet{},
		resyncInterval:         resyncInterval,
		stopCh:                 stopCh,
		kubeClient:             cs,
		eventListenerNamespace: ns,
	}
}

func (ws secretStore) AddStore(ns string) cache.Store {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()

	store := cache.NewStore(func(obj interface{}) (string, error) {
		secret, ok := obj.(*corev1.Secret)
		if !ok {
			return "", fmt.Errorf("object is not a secret; got %s", reflect.TypeOf(obj))
		}

		return secret.Name, nil
	})

	lw := cache.NewListWatchFromClient(ws.kubeClient.CoreV1().RESTClient(), "secrets", ns, fields.Everything())
	reflector := cache.NewReflector(lw, corev1.Secret{}, store, ws.resyncInterval)

	go reflector.Run(ws.stopCh)

	ws.store[ns] = store

	return store
}

// Get returns the secret value for a given SecretRef.
func (ws secretStore) Get(sr triggersv1.SecretRef) ([]byte, error) {
	ns := sr.Namespace
	if ns == "" {
		ns = ws.eventListenerNamespace
	}

	store, ok := ws.store[ns]
	if !ok {
		store = ws.AddStore(ns)
	}

	cachedObj, ok, _ := store.GetByKey(sr.SecretName)
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
