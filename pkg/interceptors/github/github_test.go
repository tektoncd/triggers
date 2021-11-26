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

package github

import (
	"encoding/json"
	"net/http"
	"testing"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"github.com/tektoncd/triggers/test"
	"go.uber.org/zap/zaptest"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakeSecretInformer "knative.dev/pkg/client/injection/kube/informers/core/v1/secret/fake"
)

func TestInterceptor_ExecuteTrigger_Signature(t *testing.T) {
	var (
		emptyJSONBody = json.RawMessage(`{}`)
		secretToken   = "secret"
	)
	emptyBodyHMACSignature := test.HMACHeader(t, secretToken, emptyJSONBody)

	tests := []struct {
		name              string
		interceptorParams *triggersv1.GitHubInterceptor
		payload           []byte
		secret            *corev1.Secret
		signature         string
		eventType         string
	}{{
		name:              "no secret",
		interceptorParams: &triggersv1.GitHubInterceptor{},
		payload:           emptyJSONBody,
		signature:         "foo",
	}, {
		name: "valid header for secret",
		interceptorParams: &triggersv1.GitHubInterceptor{
			SecretRef: &triggersv1.SecretRef{
				SecretName: "mysecret",
				SecretKey:  "token",
			},
		},
		// signature is based on the secret token below and the payload
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
		name: "no secret, matching event",
		interceptorParams: &triggersv1.GitHubInterceptor{
			EventTypes: []string{"MY_EVENT", "YOUR_EVENT"},
		},

		payload:   emptyJSONBody,
		eventType: "YOUR_EVENT",
	}, {
		name: "valid header for secret and matching event",
		interceptorParams: &triggersv1.GitHubInterceptor{
			SecretRef: &triggersv1.SecretRef{
				SecretName: "mysecret",
				SecretKey:  "token",
			},
			EventTypes: []string{"MY_EVENT", "YOUR_EVENT"},
		},

		// signature is based on the secret token below and the payload
		signature: emptyBodyHMACSignature,
		secret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mysecret",
			},
			Data: map[string][]byte{
				"token": []byte("secret"),
			},
		},
		eventType: "MY_EVENT",
		payload:   emptyJSONBody,
	}, {
		name:              "nil body does not panic",
		interceptorParams: &triggersv1.GitHubInterceptor{},
		payload:           nil,
		signature:         "foo",
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _ := test.SetupFakeContext(t)
			logger := zaptest.NewLogger(t)
			secretInformer := fakeSecretInformer.Get(ctx)

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
				req.Header["X-GITHUB-EVENT"] = []string{tt.eventType}
			}
			if tt.signature != "" {
				req.Header["X-Hub-Signature"] = []string{tt.signature}
			}
			if tt.secret != nil {
				tt.secret.Namespace = metav1.NamespaceDefault
				if err := secretInformer.Informer().GetIndexer().Add(tt.secret); err != nil {
					t.Fatal(err)
				}
			}

			w := &Interceptor{
				SecretLister: secretInformer.Lister(),
				Logger:       logger.Sugar(),
			}
			res := w.Process(ctx, req)

			if !res.Continue {
				t.Fatalf("Interceptor.Process() expected res.Continue to be true but got %t. \nStatus.Err(): %v", res.Continue, res.Status.Err())
			}
		})
	}
}

func TestInterceptor_ExecuteTrigger_ShouldNotContinue(t *testing.T) {
	var (
		emptyJSONBody = json.RawMessage(`{}`)
		secretToken   = "secret"
	)
	emptyBodyHMACSignature := test.HMACHeader(t, secretToken, emptyJSONBody)

	tests := []struct {
		name              string
		interceptorParams *triggersv1.GitHubInterceptor
		payload           []byte
		secret            *corev1.Secret
		signature         string
		eventType         string
	}{{
		name: "invalid signature header",
		interceptorParams: &triggersv1.GitHubInterceptor{
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
		name: "missing signature header",
		interceptorParams: &triggersv1.GitHubInterceptor{
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
		name: "no secret, failing event",
		interceptorParams: &triggersv1.GitHubInterceptor{
			EventTypes: []string{"MY_EVENT", "YOUR_EVENT"},
		},

		payload:   emptyJSONBody,
		eventType: "OTHER_EVENT",
	}, {
		name: "valid header for secret, failing event",
		interceptorParams: &triggersv1.GitHubInterceptor{
			SecretRef: &triggersv1.SecretRef{
				SecretName: "mysecret",
				SecretKey:  "token",
			},
			EventTypes: []string{"MY_EVENT", "YOUR_EVENT"},
		},
		signature: emptyBodyHMACSignature,
		secret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mysecret",
			},
			Data: map[string][]byte{
				"token": []byte("secret"),
			},
		},
		eventType: "OTHER_EVENT",
		payload:   emptyJSONBody,
	}, {
		name: "invalid header for secret, matching event",
		interceptorParams: &triggersv1.GitHubInterceptor{
			SecretRef: &triggersv1.SecretRef{
				SecretName: "mysecret",
				SecretKey:  "token",
			},
			EventTypes: []string{"MY_EVENT", "YOUR_EVENT"},
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
		eventType: "MY_EVENT",
		payload:   emptyJSONBody,
	}, {
		name: "empty secret",
		interceptorParams: &triggersv1.GitHubInterceptor{
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
			ctx, _ := test.SetupFakeContext(t)
			logger := zaptest.NewLogger(t)
			secretInformer := fakeSecretInformer.Get(ctx)

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
				req.Header["X-GITHUB-EVENT"] = []string{tt.eventType}
			}
			if tt.signature != "" {
				req.Header["X-Hub-Signature"] = []string{tt.signature}
			}
			if tt.secret != nil {
				tt.secret.Namespace = metav1.NamespaceDefault
				if err := secretInformer.Informer().GetIndexer().Add(tt.secret); err != nil {
					t.Fatal(err)
				}
			}
			w := &Interceptor{
				SecretLister: secretInformer.Lister(),
				Logger:       logger.Sugar(),
			}
			res := w.Process(ctx, req)

			if res.Continue {
				t.Fatalf("Interceptor.Process() expected res.Continue to be false but got %t. \nStatus.Err(): %v", res.Continue, res.Status.Err())
			}
		})
	}
}

func TestInterceptor_ExecuteTrigger_with_invalid_content_type(t *testing.T) {
	ctx, _ := test.SetupFakeContext(t)
	logger := zaptest.NewLogger(t)
	secretInformer := fakeSecretInformer.Get(ctx)
	req := &triggersv1.InterceptorRequest{
		Body: `{}`,
		Header: http.Header{
			"Content-Type":    []string{"application/x-www-form-urlencoded"},
			"X-Hub-Signature": []string{"foo"},
		},
		InterceptorParams: map[string]interface{}{},
		Context: &triggersv1.TriggerContext{
			EventURL:  "https://testing.example.com",
			EventID:   "abcde",
			TriggerID: "namespaces/default/triggers/example-trigger",
		},
	}
	w := &Interceptor{
		SecretLister: secretInformer.Lister(),
		Logger:       logger.Sugar(),
	}
	res := w.Process(ctx, req)
	if res.Continue {
		t.Fatalf("Interceptor.Process() expected res.Continue to be : %t but got %t.\n Status.Err(): %v", false, res.Continue, res.Status.Err())
	}
	if res.Status.Message != ErrInvalidContentType.Error() {
		t.Fatalf("got error %v, want %v", res.Status.Err(), ErrInvalidContentType)
	}
}

func TestInterceptor_Process_InvalidParams(t *testing.T) {
	ctx, _ := test.SetupFakeContext(t)
	logger := zaptest.NewLogger(t)
	secretInformer := fakeSecretInformer.Get(ctx)

	w := &Interceptor{
		SecretLister: secretInformer.Lister(),
		Logger:       logger.Sugar(),
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
