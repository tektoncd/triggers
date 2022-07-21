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

package interceptors_test

import (
	"context"
	"testing"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"github.com/tektoncd/triggers/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakekubeclient "knative.dev/pkg/client/injection/kube/client/fake"
)

func TestCacheSecrets(t *testing.T) {
	ctx, _ := test.SetupFakeContext(t)
	_, clientset := fakekubeclient.With(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "name",
			Namespace: "ns",
		},
		Data: map[string][]byte{
			"key": []byte("foobar"),
		},
	})
	getter := interceptors.DefaultSecretGetter(clientset.CoreV1())

	bin, err := getter.Get(context.Background(), "ns", &triggersv1.SecretRef{
		SecretKey:  "key",
		SecretName: "name",
	})
	if err != nil {
		t.Fatalf("Get() unexpected error: %s", err)
	}
	if string(bin) != "foobar" {
		t.Fatalf("Unexpected payload. Got: %s", string(bin))
	}

	if err := clientset.CoreV1().Secrets("ns").Delete(context.Background(), "name", metav1.DeleteOptions{}); err != nil {
		t.Fatalf("Cannot delete secret: %s", err)
	}

	bin, err = getter.Get(context.Background(), "ns", &triggersv1.SecretRef{
		SecretKey:  "key",
		SecretName: "name",
	})
	if err != nil {
		t.Fatalf("Get() unexpected error: %s", err)
	}
	if string(bin) != "foobar" {
		t.Fatalf("Unexpected payload. Got: %s", string(bin))
	}
}
