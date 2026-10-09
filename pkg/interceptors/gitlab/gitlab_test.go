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
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"github.com/tektoncd/triggers/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

const testWebhookID = "msg_test_1"

func testSigningKey() (whsec string, key []byte) {
	key = make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	return "whsec_" + base64.StdEncoding.EncodeToString(key), key
}

func signingSecret(whsec string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "signing-secret",
			Namespace: metav1.NamespaceDefault,
		},
		Data: map[string][]byte{
			"signingToken": []byte(whsec),
		},
	}
}

func webhookHeader(key []byte, body string, ts time.Time) http.Header {
	tsStr := strconv.FormatInt(ts.Unix(), 10)
	h := http.Header{}
	h.Set("Content-Type", "application/json")
	h.Set("Webhook-Id", testWebhookID)
	h.Set("Webhook-Timestamp", tsStr)
	h.Set("Webhook-Signature", computeSignature(key, testWebhookID, tsStr, body))
	return h
}

func freezeNow(t *testing.T, frozen time.Time) {
	t.Helper()
	orig := now
	now = func() time.Time { return frozen }
	t.Cleanup(func() { now = orig })
}

func processWithSigning(t *testing.T, params map[string]interface{}, header http.Header, body string, secrets ...*corev1.Secret) *triggersv1.InterceptorResponse {
	t.Helper()
	ctx, _ := test.SetupFakeContext(t)
	if len(secrets) > 0 {
		objs := make([]runtime.Object, 0, len(secrets))
		for _, s := range secrets {
			objs = append(objs, s)
		}
		ctx, _ = fakekubeclient.With(ctx, objs...)
	}
	req := &triggersv1.InterceptorRequest{
		Body:              body,
		Header:            header,
		InterceptorParams: params,
		Context: &triggersv1.TriggerContext{
			EventURL:  "https://testing.example.com",
			EventID:   "abcde",
			TriggerID: "namespaces/default/triggers/example-trigger",
		},
	}
	w := &InterceptorImpl{
		SecretGetter: interceptors.DefaultSecretGetter(fakekubeclient.Get(ctx).CoreV1()),
	}
	return w.Process(ctx, req)
}

func TestInterceptor_SigningToken_ShouldContinue(t *testing.T) {
	frozen := time.Date(2026, 9, 23, 16, 0, 0, 0, time.UTC)
	freezeNow(t, frozen)
	whsec, key := testSigningKey()
	body := "somepayload"
	signingRef := &triggersv1.SecretRef{SecretName: "signing-secret", SecretKey: "signingToken"}

	t.Run("valid HMAC", func(t *testing.T) {
		res := processWithSigning(t,
			map[string]interface{}{"signingSecretRef": signingRef},
			webhookHeader(key, body, frozen),
			body,
			signingSecret(whsec),
		)
		if !res.Continue {
			t.Fatalf("expected Continue=true: %v", res.Status.Err())
		}
	})

	t.Run("token without whsec_ prefix", func(t *testing.T) {
		raw := base64.StdEncoding.EncodeToString(key)
		res := processWithSigning(t,
			map[string]interface{}{"signingSecretRef": signingRef},
			webhookHeader(key, body, frozen),
			body,
			signingSecret(raw),
		)
		if !res.Continue {
			t.Fatalf("expected Continue=true: %v", res.Status.Err())
		}
	})

	t.Run("multiple signatures with valid v1", func(t *testing.T) {
		h := webhookHeader(key, body, frozen)
		h.Set("Webhook-Signature", "v2,AAAA "+h.Get("Webhook-Signature"))
		res := processWithSigning(t,
			map[string]interface{}{"signingSecretRef": signingRef},
			h,
			body,
			signingSecret(whsec),
		)
		if !res.Continue {
			t.Fatalf("expected Continue=true: %v", res.Status.Err())
		}
	})

	t.Run("eventTypes and valid signing", func(t *testing.T) {
		h := webhookHeader(key, body, frozen)
		h.Set("X-Gitlab-Event", "foo")
		res := processWithSigning(t,
			map[string]interface{}{
				"eventTypes":       []string{"foo", "bar"},
				"signingSecretRef": signingRef,
			},
			h,
			body,
			signingSecret(whsec),
		)
		if !res.Continue {
			t.Fatalf("expected Continue=true: %v", res.Status.Err())
		}
	})

	t.Run("dual mode valid HMAC only", func(t *testing.T) {
		tokenSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "mysecret", Namespace: metav1.NamespaceDefault},
			Data:       map[string][]byte{"token": []byte("secrettoken")},
		}
		res := processWithSigning(t,
			map[string]interface{}{
				"secretRef":        &triggersv1.SecretRef{SecretName: "mysecret", SecretKey: "token"},
				"signingSecretRef": signingRef,
			},
			webhookHeader(key, body, frozen),
			body,
			signingSecret(whsec),
			tokenSecret,
		)
		if !res.Continue {
			t.Fatalf("expected Continue=true: %v", res.Status.Err())
		}
	})

	t.Run("dual mode valid token only", func(t *testing.T) {
		h := http.Header{}
		h.Set("Content-Type", "application/json")
		h.Set("X-Gitlab-Token", "secrettoken")
		tokenSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "mysecret", Namespace: metav1.NamespaceDefault},
			Data:       map[string][]byte{"token": []byte("secrettoken")},
		}
		res := processWithSigning(t,
			map[string]interface{}{
				"secretRef":        &triggersv1.SecretRef{SecretName: "mysecret", SecretKey: "token"},
				"signingSecretRef": signingRef,
			},
			h,
			body,
			signingSecret(whsec),
			tokenSecret,
		)
		if !res.Continue {
			t.Fatalf("expected Continue=true: %v", res.Status.Err())
		}
	})

	t.Run("dual secretRef and signingSecretRef", func(t *testing.T) {
		h := webhookHeader(key, body, frozen)
		h.Set("X-Gitlab-Token", "secrettoken")
		tokenSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "mysecret", Namespace: metav1.NamespaceDefault},
			Data:       map[string][]byte{"token": []byte("secrettoken")},
		}
		res := processWithSigning(t,
			map[string]interface{}{
				"secretRef":        &triggersv1.SecretRef{SecretName: "mysecret", SecretKey: "token"},
				"signingSecretRef": signingRef,
			},
			h,
			body,
			signingSecret(whsec),
			tokenSecret,
		)
		if !res.Continue {
			t.Fatalf("expected Continue=true: %v", res.Status.Err())
		}
	})
}

func TestInterceptor_SigningToken_ShouldNotContinue(t *testing.T) {
	frozen := time.Date(2026, 9, 23, 16, 0, 0, 0, time.UTC)
	freezeNow(t, frozen)
	whsec, key := testSigningKey()
	body := "somepayload"
	signingRef := &triggersv1.SecretRef{SecretName: "signing-secret", SecretKey: "signingToken"}

	tests := []struct {
		name    string
		params  map[string]interface{}
		header  http.Header
		secrets []*corev1.Secret
		wantErr string
	}{{
		name:   "missing webhook-signature",
		params: map[string]interface{}{"signingSecretRef": signingRef},
		header: func() http.Header {
			h := webhookHeader(key, body, frozen)
			h.Del("Webhook-Signature")
			return h
		}(),
		secrets: []*corev1.Secret{signingSecret(whsec)},
		wantErr: "no webhook-signature header set",
	}, {
		name:   "missing webhook-id",
		params: map[string]interface{}{"signingSecretRef": signingRef},
		header: func() http.Header {
			h := webhookHeader(key, body, frozen)
			h.Del("Webhook-Id")
			return h
		}(),
		secrets: []*corev1.Secret{signingSecret(whsec)},
		wantErr: "no webhook-id header set",
	}, {
		name:   "missing webhook-timestamp",
		params: map[string]interface{}{"signingSecretRef": signingRef},
		header: func() http.Header {
			h := webhookHeader(key, body, frozen)
			h.Del("Webhook-Timestamp")
			return h
		}(),
		secrets: []*corev1.Secret{signingSecret(whsec)},
		wantErr: "no webhook-timestamp header set",
	}, {
		name:    "empty signingSecretRef.secretKey",
		params:  map[string]interface{}{"signingSecretRef": &triggersv1.SecretRef{SecretName: "signing-secret"}},
		header:  webhookHeader(key, body, frozen),
		secrets: []*corev1.Secret{signingSecret(whsec)},
		wantErr: "signingSecretRef.secretKey is empty",
	}, {
		name:    "missing secret",
		params:  map[string]interface{}{"signingSecretRef": signingRef},
		header:  webhookHeader(key, body, frozen),
		wantErr: "error getting secret",
	}, {
		name:    "wrong HMAC",
		params:  map[string]interface{}{"signingSecretRef": signingRef},
		header:  webhookHeader(key, "otherpayload", frozen),
		secrets: []*corev1.Secret{signingSecret(whsec)},
		wantErr: "Invalid webhook-signature",
	}, {
		name:   "only non-v1 signature",
		params: map[string]interface{}{"signingSecretRef": signingRef},
		header: func() http.Header {
			h := webhookHeader(key, body, frozen)
			h.Set("Webhook-Signature", "v2,"+base64.StdEncoding.EncodeToString([]byte("nope")))
			return h
		}(),
		secrets: []*corev1.Secret{signingSecret(whsec)},
		wantErr: "Invalid webhook-signature",
	}, {
		name:    "stale timestamp",
		params:  map[string]interface{}{"signingSecretRef": signingRef},
		header:  webhookHeader(key, body, frozen.Add(-6*time.Minute)),
		secrets: []*corev1.Secret{signingSecret(whsec)},
		wantErr: "outside the allowed tolerance",
	}, {
		name:    "future timestamp",
		params:  map[string]interface{}{"signingSecretRef": signingRef},
		header:  webhookHeader(key, body, frozen.Add(6*time.Minute)),
		secrets: []*corev1.Secret{signingSecret(whsec)},
		wantErr: "outside the allowed tolerance",
	}, {
		name:    "invalid signing token encoding",
		params:  map[string]interface{}{"signingSecretRef": signingRef},
		header:  webhookHeader(key, body, frozen),
		secrets: []*corev1.Secret{signingSecret("whsec_not-valid-base64!!!")},
		wantErr: "error decoding signing token",
	}, {
		name: "dual mode no signature and no token",
		params: map[string]interface{}{
			"secretRef":        &triggersv1.SecretRef{SecretName: "mysecret", SecretKey: "token"},
			"signingSecretRef": signingRef,
		},
		header: http.Header{"Content-Type": []string{"application/json"}},
		secrets: []*corev1.Secret{
			signingSecret(whsec),
			{
				ObjectMeta: metav1.ObjectMeta{Name: "mysecret", Namespace: metav1.NamespaceDefault},
				Data:       map[string][]byte{"token": []byte("secrettoken")},
			},
		},
		wantErr: "gitlab webhook authentication failed",
	}, {
		name: "dual mode invalid signature and invalid token",
		params: map[string]interface{}{
			"secretRef":        &triggersv1.SecretRef{SecretName: "mysecret", SecretKey: "token"},
			"signingSecretRef": signingRef,
		},
		header: func() http.Header {
			h := webhookHeader(key, "otherpayload", frozen)
			h.Set("X-Gitlab-Token", "wrong")
			return h
		}(),
		secrets: []*corev1.Secret{
			signingSecret(whsec),
			{
				ObjectMeta: metav1.ObjectMeta{Name: "mysecret", Namespace: metav1.NamespaceDefault},
				Data:       map[string][]byte{"token": []byte("secrettoken")},
			},
		},
		wantErr: "gitlab webhook authentication failed",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := processWithSigning(t, tt.params, tt.header, body, tt.secrets...)
			if res.Continue {
				t.Fatalf("expected Continue=false")
			}
			if tt.wantErr != "" && !strings.Contains(res.Status.Err().Error(), tt.wantErr) {
				t.Fatalf("error %q does not contain %q", res.Status.Err(), tt.wantErr)
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
