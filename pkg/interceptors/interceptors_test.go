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
	"fmt"
	"net/http"
	"testing"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakekubeclient "knative.dev/pkg/client/injection/kube/client/fake"
	rtesting "knative.dev/pkg/reconciler/testing"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
)

const testNS = "testing-ns"

func TestGetInterceptorParams(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   triggersv1.EventInterceptor
		want map[string]interface{}
	}{{
		name: "cel",
		in: triggersv1.EventInterceptor{
			CEL: &triggersv1.CELInterceptor{
				Filter: `header.match("foo", "bar")`,
				Overlays: []triggersv1.CELOverlay{{
					Key:        "short_sha",
					Expression: "body.ref.truncate(7)",
				}},
			},
		},
		want: map[string]interface{}{
			"filter": `header.match("foo", "bar")`,
			"overlays": []triggersv1.CELOverlay{{
				Key:        "short_sha",
				Expression: "body.ref.truncate(7)",
			}},
		},
	}, {
		name: "gitlab",
		in: triggersv1.EventInterceptor{
			GitLab: &triggersv1.GitLabInterceptor{
				SecretRef: &triggersv1.SecretRef{
					SecretKey:  "test-secret",
					SecretName: "token",
				},
				EventTypes: []string{"push"},
			},
		},
		want: map[string]interface{}{
			"eventTypes": []string{"push"},
			"secretRef": &triggersv1.SecretRef{
				SecretKey:  "test-secret",
				SecretName: "token",
			},
		},
	}, {
		name: "github",
		in: triggersv1.EventInterceptor{
			GitHub: &triggersv1.GitHubInterceptor{
				SecretRef: &triggersv1.SecretRef{
					SecretKey:  "test-secret",
					SecretName: "token",
				},
				EventTypes: []string{"push"},
			},
		},
		want: map[string]interface{}{
			"eventTypes": []string{"push"},
			"secretRef": &triggersv1.SecretRef{
				SecretKey:  "test-secret",
				SecretName: "token",
			},
		},
	}, {
		name: "bitbucket",
		in: triggersv1.EventInterceptor{
			Bitbucket: &triggersv1.BitbucketInterceptor{
				SecretRef: &triggersv1.SecretRef{
					SecretKey:  "test-secret",
					SecretName: "token",
				},
				EventTypes: []string{"push"},
			},
		},
		want: map[string]interface{}{
			"eventTypes": []string{"push"},
			"secretRef": &triggersv1.SecretRef{
				SecretKey:  "test-secret",
				SecretName: "token",
			},
		},
	}, {
		name: "webhook",
		in: triggersv1.EventInterceptor{
			Webhook: &triggersv1.WebhookInterceptor{
				ObjectRef: &corev1.ObjectReference{
					Kind:       "Service",
					APIVersion: "v1",
					Namespace:  "default",
					Name:       "foo",
				},
				Header: []pipelinev1.Param{{
					Name: "p1",
					Value: pipelinev1.ArrayOrString{
						Type:     pipelinev1.ParamTypeArray,
						ArrayVal: []string{"v1", "v2"},
					},
				}},
			},
		},
		want: map[string]interface{}{
			"objectRef": &corev1.ObjectReference{
				Kind:       "Service",
				APIVersion: "v1",
				Namespace:  "default",
				Name:       "foo",
			},
			"header": []pipelinev1.Param{{
				Name: "p1",
				Value: pipelinev1.ArrayOrString{
					Type:     pipelinev1.ParamTypeArray,
					ArrayVal: []string{"v1", "v2"},
				},
			}},
		},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			got := GetInterceptorParams(&tc.in)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("GetInterceptorParams() failed. Diff (-want/+got): %s", diff)
			}
		})
	}
}

func Test_GetSecretToken(t *testing.T) {
	tests := []struct {
		name   string
		cache  map[string]interface{}
		wanted []byte
	}{
		{
			name:   "no matching cache entry exists",
			cache:  make(map[string]interface{}),
			wanted: []byte("secret from API"),
		},
		{
			name: "a matching cache entry exists",
			cache: map[string]interface{}{
				fmt.Sprintf("secret/%s/test-secret/token", testNS): []byte("secret from cache"),
			},
			wanted: []byte("secret from cache"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(rt *testing.T) {
			req := setCache(&http.Request{}, tt.cache)

			ctx, _ := rtesting.SetupFakeContext(t)
			kubeClient := fakekubeclient.Get(ctx)
			secretRef := makeSecretRef()

			if _, err := kubeClient.CoreV1().Secrets(testNS).Create(context.Background(), makeSecret("secret from API"), metav1.CreateOptions{}); err != nil {
				rt.Error(err)
			}

			secret, err := GetSecretToken(req, kubeClient, &secretRef, testNS)
			if err != nil {
				rt.Error(err)
			}

			if diff := cmp.Diff(tt.wanted, secret); diff != "" {
				rt.Errorf("secret value (-want, +got) = %s", diff)
			}
		})
	}
}

func makeSecret(secretText string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNS,
			Name:      "test-secret",
		},
		Data: map[string][]byte{
			"token": []byte(secretText),
		},
	}
}

func makeSecretRef() triggersv1.SecretRef {
	return triggersv1.SecretRef{
		SecretKey:  "token",
		SecretName: "test-secret",
	}
}

func setCache(req *http.Request, vals map[string]interface{}) *http.Request {
	return req.WithContext(context.WithValue(req.Context(), requestCacheKey, vals))
}
