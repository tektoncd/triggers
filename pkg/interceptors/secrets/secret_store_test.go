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
// import (
//     "bytes"
//     "testing"
//     "time"

//     triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
//     corev1 "k8s.io/api/core/v1"
//     metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
//     fakekubeclient "knative.dev/pkg/client/injection/kube/client/fake"
//     rtesting "knative.dev/pkg/reconciler/testing"
// )

// func TestSecretStoreManyNamespaces(t *testing.T) {
//     ctx, _ := rtesting.SetupFakeContext(t)
//     kubeClient := fakekubeclient.Get(ctx)
//     stopCh := make(<-chan struct{})
//     store := NewSecretStore(kubeClient, metav1.NamespaceAll, time.Millisecond*1, stopCh)

//     _, err := kubeClient.CoreV1().Secrets("ns-1").Create(
//         &corev1.Secret{
//             ObjectMeta: metav1.ObjectMeta{
//                 Name: "secret-1",
//             },
//             StringData: map[string]string{"pwd": "topsecret"},
//         },
//     )
//     if err != nil {
//         t.Fatalf("Could not create secrets: %s", err.Error())
//     }

//     _, err = kubeClient.CoreV1().Secrets("ns-2").Create(
//         &corev1.Secret{
//             ObjectMeta: metav1.ObjectMeta{
//                 Name: "secret-2",
//             },
//             StringData: map[string]string{"pwd": "stilltopsecret"},
//         },
//     )
//     if err != nil {
//         t.Fatalf("Could not create secrets: %s", err.Error())
//     }

//     err = store.Resync()
//     if err != nil {
//         t.Fatalf("Could not resync store: %s", err.Error())
//     }

//     secret1, err := store.Get(triggersv1.SecretRef{
//         SecretKey:  "pwd",
//         Namespace:  "ns-1",
//         SecretName: "secret-1",
//     })
//     if err != nil {
//         t.Fatalf("Could not retrieve secret: %s", err.Error())
//     }
//     if bytes.Equal(secret1, []byte("topsecret")) {
//         t.Fatalf("Expected topsecret, got %s", string(secret1))
//     }

//     secret2, err := store.Get(triggersv1.SecretRef{
//         SecretKey:  "pwd",
//         Namespace:  "ns-2",
//         SecretName: "secret-2",
//     })
//     if err != nil {
//         t.Fatalf("Could not retrieve secret: %s", err.Error())
//     }
//     if bytes.Equal(secret2, []byte("stilltopsecret")) {
//         t.Fatalf("Expected stilltopsecret, got %s", string(secret1))
//     }
// }
