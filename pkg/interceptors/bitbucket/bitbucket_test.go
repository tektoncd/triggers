/*
Copyright 2020 The Tekton Authors

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

package bitbucket

import (
	"encoding/json"
	"net/http"
	"testing"

	"go.uber.org/zap/zaptest"

	"github.com/tektoncd/triggers/test"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakekubeclient "knative.dev/pkg/client/injection/kube/client/fake"
	rtesting "knative.dev/pkg/reconciler/testing"
)

func TestInterceptor_Process_ShouldContinue(t *testing.T) {
	var (
		emptyJSONBody = json.RawMessage(`{}`)
		secretToken   = "secret"
	)
	emptyBodyHMACSignature := test.HMACHeader(t, secretToken, emptyJSONBody)

	tests := []struct {
		name              string
		interceptorParams *triggersv1.BitbucketInterceptor
		payload           []byte
		secret            *corev1.Secret
		signature         string
		eventType         string
	}{{
		name:              "no secret",
		interceptorParams: &triggersv1.BitbucketInterceptor{},
		payload:           emptyJSONBody,
		signature:         "foo",
	}, {
		name: "valid header for secret",
		interceptorParams: &triggersv1.BitbucketInterceptor{
			SecretRef: &triggersv1.SecretRef{
				SecretName: "mysecret",
				SecretKey:  "token",
			},
		},
		signature: emptyBodyHMACSignature,
		secret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mysecret",
			},
			Data: map[string][]byte{
				"token": []byte(secretToken),
			},
		},
		payload: emptyJSONBody,
	}, {
		name: "matching event",
		interceptorParams: &triggersv1.BitbucketInterceptor{
			EventTypes: []string{"pr:opened", "repo:refs_changed"},
		},
		payload:   emptyJSONBody,
		eventType: "repo:refs_changed",
	}, {
		name: "valid header for secret and matching event",
		interceptorParams: &triggersv1.BitbucketInterceptor{
			SecretRef: &triggersv1.SecretRef{
				SecretName: "mysecret",
				SecretKey:  "token",
			},
			EventTypes: []string{"pr:opened", "repo:refs_changed"},
		},
		signature: emptyBodyHMACSignature,
		secret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mysecret",
			},
			Data: map[string][]byte{
				"token": []byte(secretToken),
			},
		},
		eventType: "repo:refs_changed",
		payload:   emptyJSONBody,
	}, {
		name:              "nil body does not panic",
		interceptorParams: &triggersv1.BitbucketInterceptor{},
		payload:           nil,
		signature:         "foo",
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _ := rtesting.SetupFakeContext(t)
			logger := zaptest.NewLogger(t)
			kubeClient := fakekubeclient.Get(ctx)

			w := &Interceptor{
				KubeClientSet: kubeClient,
				Logger:        logger.Sugar(),
			}

			req := &triggersv1.InterceptorRequest{
				Body: string(tt.payload),
				Header: http.Header{
					"Content-Type": []string{"application/json"},
				},
				InterceptorParams: map[string]interface{}{
					"eventTypes": tt.interceptorParams.EventTypes,
					"secretRef":  tt.interceptorParams.SecretRef,
				},
				Context: &triggersv1.TriggerContext{
					EventURL:  "https://testing.example.com",
					EventID:   "abcde",
					TriggerID: "namespaces/default/triggers/example-trigger",
				},
			}

			if tt.eventType != "" {
				req.Header["X-Event-Key"] = []string{tt.eventType}
			}
			if tt.signature != "" {
				req.Header["X-Hub-Signature"] = []string{tt.signature}
			}
			if tt.secret != nil {
				if _, err := kubeClient.CoreV1().Secrets(metav1.NamespaceDefault).Create(ctx, tt.secret, metav1.CreateOptions{}); err != nil {
					t.Error(err)
				}
			}
			res := w.Process(ctx, req)
			if !res.Continue {
				t.Fatalf("Interceptor.Process() expected res.Continue to be true but got %t. \nStatus.Err(): %v", res.Continue, res.Status.Err())
			}
		})
	}
}

func TestInterceptor_Process_ShouldNotContinue(t *testing.T) {
	var (
		emptyJSONBody = json.RawMessage(`{}`)
		secretToken   = "secret"
	)
	emptyBodyHMACSignature := test.HMACHeader(t, secretToken, emptyJSONBody)

	tests := []struct {
		name              string
		interceptorParams *triggersv1.BitbucketInterceptor
		payload           []byte
		secret            *corev1.Secret
		signature         string
		eventType         string
	}{{
		name: "invalid header for secret",
		interceptorParams: &triggersv1.BitbucketInterceptor{
			SecretRef: &triggersv1.SecretRef{
				SecretName: "mysecret",
				SecretKey:  "token",
			},
		},
		signature: "foo",
		secret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mysecret",
			},
			Data: map[string][]byte{
				"token": []byte("secrettoken"),
			},
		},
		payload: emptyJSONBody,
	}, {
		name: "no X-Hub-Signature header for secret",
		interceptorParams: &triggersv1.BitbucketInterceptor{
			SecretRef: &triggersv1.SecretRef{
				SecretName: "mysecret",
				SecretKey:  "token",
			},
		},
		secret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mysecret",
			},
			Data: map[string][]byte{
				"token": []byte("secrettoken"),
			},
		},
		payload: emptyJSONBody,
	}, {
		name: "no matching event",
		interceptorParams: &triggersv1.BitbucketInterceptor{
			EventTypes: []string{"pr:opened", "repo:refs_changed"},
		},
		payload:   emptyJSONBody,
		eventType: "event",
	}, {
		name: "invalid header for secret, but matching event",
		interceptorParams: &triggersv1.BitbucketInterceptor{
			SecretRef: &triggersv1.SecretRef{
				SecretName: "mysecret",
				SecretKey:  "token",
			},
			EventTypes: []string{"pr:opened", "repo:refs_changed"},
		},
		signature: "foo",
		secret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mysecret",
			},
			Data: map[string][]byte{
				"token": []byte("secrettoken"),
			},
		},
		eventType: "pr:opened",
		payload:   emptyJSONBody,
	}, {
		name: "valid header for secret, but no matching event",
		interceptorParams: &triggersv1.BitbucketInterceptor{
			SecretRef: &triggersv1.SecretRef{
				SecretName: "mysecret",
				SecretKey:  "token",
			},
			EventTypes: []string{"pr:opened", "repo:refs_changed"},
		},
		signature: emptyBodyHMACSignature,
		secret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mysecret",
			},
			Data: map[string][]byte{
				"token": []byte(secretToken),
			},
		},
		eventType: "event",
		payload:   emptyJSONBody,
	}, {
		name: "empty secret",
		interceptorParams: &triggersv1.BitbucketInterceptor{
			SecretRef: &triggersv1.SecretRef{
				SecretName: "mysecret",
			},
		},
		signature: emptyBodyHMACSignature,
		secret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mysecret",
			},
			Data: map[string][]byte{
				"token": []byte(secretToken),
			},
		},
		payload: emptyJSONBody,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _ := rtesting.SetupFakeContext(t)
			logger := zaptest.NewLogger(t)
			kubeClient := fakekubeclient.Get(ctx)

			w := &Interceptor{
				KubeClientSet: kubeClient,
				Logger:        logger.Sugar(),
			}

			req := &triggersv1.InterceptorRequest{
				Body: string(tt.payload),
				Header: http.Header{
					"Content-Type": []string{"application/json"},
				},
				InterceptorParams: map[string]interface{}{
					"eventTypes": tt.interceptorParams.EventTypes,
					"secretRef":  tt.interceptorParams.SecretRef,
				},
				Context: &triggersv1.TriggerContext{
					EventURL:  "https://testing.example.com",
					EventID:   "abcde",
					TriggerID: "namespaces/default/triggers/example-trigger",
				},
			}

			if tt.eventType != "" {
				req.Header["X-Event-Key"] = []string{tt.eventType}
			}
			if tt.signature != "" {
				req.Header["X-Hub-Signature"] = []string{tt.signature}
			}
			if tt.secret != nil {
				if _, err := kubeClient.CoreV1().Secrets(metav1.NamespaceDefault).Create(ctx, tt.secret, metav1.CreateOptions{}); err != nil {
					t.Error(err)
				}
			}
			res := w.Process(ctx, req)
			if res.Continue {
				t.Fatalf("Interceptor.Process() expected res.Continue to be false but got %t. \nStatus.Err(): %v", res.Continue, res.Status.Err())
			}
		})
	}
}

func TestInterceptor_Process_InvalidParams(t *testing.T) {
	ctx, _ := rtesting.SetupFakeContext(t)
	logger := zaptest.NewLogger(t)
	kubeClient := fakekubeclient.Get(ctx)

	w := &Interceptor{
		KubeClientSet: kubeClient,
		Logger:        logger.Sugar(),
	}

	req := &triggersv1.InterceptorRequest{
		Body: `{}`,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		InterceptorParams: map[string]interface{}{
			"blah": func() {},
		},
		Context: &triggersv1.TriggerContext{
			EventURL:  "https://testing.example.com",
			EventID:   "abcde",
			TriggerID: "namespaces/default/triggers/example-trigger",
		},
	}

	res := w.Process(ctx, req)
	if res.Continue {
		t.Fatalf("Interceptor.Process() expected res.Continue to be false but got %t. \nStatus.Err(): %v", res.Continue, res.Status.Err())
	}
}
