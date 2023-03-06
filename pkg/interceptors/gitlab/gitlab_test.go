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

package gitlab

import (
	"net/http"
	"testing"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"github.com/tektoncd/triggers/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakekubeclient "knative.dev/pkg/client/injection/kube/client/fake"
)

func TestInterceptor_ExecuteTrigger_ShouldContinue(t *testing.T) {
	tests := []struct {
		name              string
		interceptorParams *InterceptorParams
		payload           []byte
		secret            *corev1.Secret
		token             string
		eventType         string
	}{{
		name:              "no secret",
		interceptorParams: &InterceptorParams{},

		payload: []byte("somepayload"),
		token:   "foo",
	}, {
		name: "valid header for secret",
		interceptorParams: &InterceptorParams{
			SecretRef: &triggersv1.SecretRef{
				SecretName: "mysecret",
				SecretKey:  "token",
			},
		},

		token: "secret",
		secret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mysecret",
			},
			Data: map[string][]byte{
				"token": []byte("secret"),
			},
		},
		payload: []byte("somepayload"),
	}, {
		name: "valid event",
		interceptorParams: &InterceptorParams{
			EventTypes: []string{"foo", "bar"},
		},

		eventType: "foo",
		payload:   []byte("somepayload"),
	}, {
		name: "valid event, valid secret",
		interceptorParams: &InterceptorParams{
			EventTypes: []string{"foo", "bar"},
			SecretRef: &triggersv1.SecretRef{
				SecretName: "mysecret",
				SecretKey:  "token",
			},
		},
		eventType: "bar",
		payload:   []byte("somepayload"),
		token:     "secrettoken",
		secret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mysecret",
			},
			Data: map[string][]byte{
				"token": []byte("secrettoken"),
			},
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _ := test.SetupFakeContext(t)
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
			if tt.token != "" {
				req.Header["X-GitLab-Token"] = []string{tt.token}
			}
			if tt.eventType != "" {
				req.Header["X-GitLab-Event"] = []string{tt.eventType}
			}
			clientset := fakekubeclient.Get(ctx)
			if tt.secret != nil {
				tt.secret.Namespace = metav1.NamespaceDefault
				ctx, clientset = fakekubeclient.With(ctx, tt.secret)
			}
			w := &InterceptorImpl{
				SecretGetter: interceptors.DefaultSecretGetter(clientset.CoreV1()),
			}
			res := w.Process(ctx, req)
			if !res.Continue {
				t.Fatalf("Interceptor.Process() expected res.Continue to be : true but got %t. \nStatus.Err(): %v", res.Continue, res.Status.Err())
			}

		})
	}
}

func TestInterceptor_ExecuteTrigger_ShouldNotContinue(t *testing.T) {
	tests := []struct {
		name              string
		interceptorParams *InterceptorParams
		payload           []byte
		secret            *corev1.Secret
		token             string
		eventType         string
	}{{
		name: "invalid header for secret",
		interceptorParams: &InterceptorParams{
			SecretRef: &triggersv1.SecretRef{
				SecretName: "mysecret",
				SecretKey:  "token",
			},
		},

		token: "foo",
		secret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mysecret",
			},
			Data: map[string][]byte{
				"token": []byte("secrettoken"),
			},
		},
		payload: []byte("somepayload"),
	}, {
		name: "missing header for secret",
		interceptorParams: &InterceptorParams{
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
		payload: []byte("somepayload"),
	}, {
		name: "invalid event",
		interceptorParams: &InterceptorParams{
			EventTypes: []string{"foo", "bar"},
		},

		eventType: "baz",
		payload:   []byte("somepayload"),
	}, {
		name: "valid event, invalid secret",
		interceptorParams: &InterceptorParams{
			EventTypes: []string{"foo", "bar"},
			SecretRef: &triggersv1.SecretRef{
				SecretName: "mysecret",
				SecretKey:  "token",
			},
		},

		eventType: "bar",
		payload:   []byte("somepayload"),
		token:     "foo",
		secret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mysecret",
			},
			Data: map[string][]byte{
				"token": []byte("secrettoken"),
			},
		},
	}, {
		name: "invalid event, valid secret",
		interceptorParams: &InterceptorParams{
			EventTypes: []string{"foo", "bar"},
			SecretRef: &triggersv1.SecretRef{
				SecretName: "mysecret",
				SecretKey:  "token",
			},
		},

		eventType: "baz",
		payload:   []byte("somepayload"),
		token:     "secrettoken",
		secret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mysecret",
			},
			Data: map[string][]byte{
				"token": []byte("secrettoken"),
			},
		},
	}, {
		name: "empty secret",
		interceptorParams: &InterceptorParams{
			SecretRef: &triggersv1.SecretRef{
				SecretName: "mysecret",
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
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _ := test.SetupFakeContext(t)
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
			if tt.token != "" {
				req.Header["X-GitLab-Token"] = []string{tt.token}
			}
			if tt.eventType != "" {
				req.Header["X-interceptorParams-Event"] = []string{tt.eventType}
			}
			clientset := fakekubeclient.Get(ctx)
			if tt.secret != nil {
				tt.secret.Namespace = metav1.NamespaceDefault
				ctx, clientset = fakekubeclient.With(ctx, tt.secret)
			}
			w := &InterceptorImpl{
				SecretGetter: interceptors.DefaultSecretGetter(clientset.CoreV1()),
			}
			res := w.Process(ctx, req)
			if res.Continue {
				t.Fatalf("Interceptor.Process() expected res.Continue to be false but got %t. \nStatus.Err(): %v", res.Continue, res.Status.Err())
			}
		})
	}
}

func TestInterceptor_Process_InvalidParams(t *testing.T) {
	ctx, _ := test.SetupFakeContext(t)

	w := &InterceptorImpl{
		SecretGetter: interceptors.DefaultSecretGetter(fakekubeclient.Get(ctx).CoreV1()),
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
