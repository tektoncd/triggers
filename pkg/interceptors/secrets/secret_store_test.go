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

// TODO make this work
import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	fakekubeclient "knative.dev/pkg/client/injection/kube/client/fake"
	fakesecretinformer "knative.dev/pkg/client/injection/kube/informers/core/v1/secret/fake"
	rtesting "knative.dev/pkg/reconciler/testing"
)

func AddToInformer(store cache.Store) func(ktesting.Action) (bool, runtime.Object, error) {
	return func(action ktesting.Action) (bool, runtime.Object, error) {
		switch a := action.(type) {
		case ktesting.CreateActionImpl:
			if err := store.Add(a.GetObject()); err != nil {
				return false, nil, err
			}

		case ktesting.UpdateActionImpl:
			objMeta, err := meta.Accessor(a.GetObject())
			if err != nil {
				return true, nil, err
			}

			// Look up the old copy of this resource and perform the optimistic concurrency check.
			old, exists, err := store.GetByKey(objMeta.GetNamespace() + "/" + objMeta.GetName())
			if err != nil {
				return true, nil, err
			} else if !exists {
				// Let the client return the error.
				return false, nil, nil
			}
			oldMeta, err := meta.Accessor(old)
			if err != nil {
				return true, nil, err
			}
			// If the resource version is mismatched, then fail with a conflict.
			if oldMeta.GetResourceVersion() != objMeta.GetResourceVersion() {
				return true, nil, apierrs.NewConflict(
					a.Resource.GroupResource(), objMeta.GetName(),
					fmt.Errorf("resourceVersion mismatch, got: %v, wanted: %v",
						objMeta.GetResourceVersion(), oldMeta.GetResourceVersion()))
			}

			// Update the store with the new object when it's fine.
			if err := store.Update(a.GetObject()); err != nil {
				return false, nil, err
			}
		}
		return false, nil, nil
	}
}

func SeedTestData(ctx context.Context, data []corev1.Secret) (*fakekubeclientset.Clientset, error) {
	kubeClient := fakekubeclient.Get(ctx)
	i := fakesecretinformer.Get(ctx)
	kubeClient.PrependReactor("*", "secrets", AddToInformer(i.Informer().GetIndexer()))
	for _, s := range data {
		s := s.DeepCopy()
		if _, err := kubeClient.CoreV1().Secrets(s.Namespace).Create(s); err != nil {
			return nil, err
		}
	}

	kubeClient.ClearActions()
	return kubeClient, nil
}

func TestSecretStoreManyNamespaces(t *testing.T) {
	ctx, _ := rtesting.SetupFakeContext(t)
	kubeClient, err := SeedTestData(ctx, []corev1.Secret{
		corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret-1",
				Namespace: "ns-1",
			},
			StringData: map[string]string{"pwd": "topsecret"},
		},
		corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret-2",
				Namespace: "ns-2",
			},
			StringData: map[string]string{"pwd": "stilltopsecret"},
		},
	})
	if err != nil {
		t.Fatalf("Could not create secrets: %s", err.Error())
	}

	stopCh := make(<-chan struct{})
	store := secretStore{
		mutex:                  &sync.Mutex{},
		store:                  storeSet{},
		resyncInterval:         time.Millisecond,
		stopCh:                 stopCh,
		kubeClient:             kubeClient,
		eventListenerNamespace: "ns-1",
	}

	secret1, err := store.Get(triggersv1.SecretRef{
		SecretKey:  "pwd",
		Namespace:  "ns-1",
		SecretName: "secret-1",
	})
	if err != nil {
		t.Fatalf("Could not retrieve secret: %s", err.Error())
	}
	if bytes.Equal(secret1, []byte("topsecret")) {
		t.Fatalf("Expected topsecret, got %s", string(secret1))
	}

	secret2, err := store.Get(triggersv1.SecretRef{
		SecretKey:  "pwd",
		Namespace:  "ns-2",
		SecretName: "secret-2",
	})
	if err != nil {
		t.Fatalf("Could not retrieve secret: %s", err.Error())
	}
	if bytes.Equal(secret2, []byte("stilltopsecret")) {
		t.Fatalf("Expected stilltopsecret, got %s", string(secret1))
	}
}
