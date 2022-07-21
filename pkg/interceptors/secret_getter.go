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

package interceptors

import (
	"context"
	"fmt"
	"time"

	triggersv1beta1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/cache"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	// cacheSize is the size of the LRU secrets cache
	cacheSize = 1024
	// ttl is the time to live for a cache entry
	ttl = 5 * time.Second
)

type SecretGetter interface {
	Get(ctx context.Context, triggerNS string, sr *triggersv1beta1.SecretRef) ([]byte, error)
}

type kubeclientSecretGetter struct {
	getter corev1.SecretsGetter
	cache  *cache.LRUExpireCache
	ttl    time.Duration
}

type cacheKey struct {
	triggerNS string
	sr        triggersv1beta1.SecretRef
}

func DefaultSecretGetter(getter corev1.SecretsGetter) SecretGetter {
	return &kubeclientSecretGetter{
		getter: getter,
		cache:  cache.NewLRUExpireCache(cacheSize),
		ttl:    ttl,
	}
}

// Get queries Kubernetes for the given secret reference. We use this function
// to resolve secret material like GitHub webhook secrets, and call it once for every
// trigger that references it.
//
// As we may have many triggers that all use the same secret, we cache the secret values
// in the request cache.
func (g *kubeclientSecretGetter) Get(ctx context.Context, triggerNS string, sr *triggersv1beta1.SecretRef) ([]byte, error) {
	key := cacheKey{
		triggerNS: triggerNS,
		sr:        *sr,
	}
	val, ok := g.cache.Get(key)
	if ok {
		return val.([]byte), nil
	}
	secret, err := g.getter.Secrets(triggerNS).Get(ctx, sr.SecretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	secretValue, ok := secret.Data[sr.SecretKey]
	if !ok {
		return nil, fmt.Errorf("cannot find %s key in secret %s/%s", sr.SecretKey, triggerNS, sr.SecretName)
	}
	g.cache.Add(key, secretValue, g.ttl)
	return secretValue, nil
}
