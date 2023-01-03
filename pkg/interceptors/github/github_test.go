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
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"github.com/tektoncd/triggers/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakekubeclient "knative.dev/pkg/client/injection/kube/client/fake"
)

func TestInterceptor_ExecuteTrigger_Signature(t *testing.T) {
	var (
		emptyJSONBody = json.RawMessage(`{}`)
		secretToken   = "secret"
	)
	emptyBodySha1Header := map[string][]string{"X-Hub-Signature": {test.HMACHeader(t, secretToken, emptyJSONBody, "sha1")}}
	emptyBodySha256Header := map[string][]string{"X-Hub-Signature-256": {test.HMACHeader(t, secretToken, emptyJSONBody, "sha256")}}

	tests := []struct {
		name              string
		interceptorParams *triggersv1.GitHubInterceptor
		payload           []byte
		secret            *corev1.Secret
		headers           map[string][]string
		eventType         string
	}{{
		name:              "no secret",
		interceptorParams: &triggersv1.GitHubInterceptor{},
		payload:           emptyJSONBody,
		headers:           map[string][]string{"X-Hub-Signature": {"foo"}},
	}, {
		name: "valid header for secret",
		interceptorParams: &triggersv1.GitHubInterceptor{
			SecretRef: &triggersv1.SecretRef{
				SecretName: "mysecret",
				SecretKey:  "token",
			},
		},
		headers: emptyBodySha1Header,
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
		name: "valid sha-256 header for secret",
		interceptorParams: &triggersv1.GitHubInterceptor{
			SecretRef: &triggersv1.SecretRef{
				SecretName: "mysecret",
				SecretKey:  "token",
			},
		},
		headers: emptyBodySha256Header,
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

		headers: emptyBodySha1Header,
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
		headers:           map[string][]string{"X-Hub-Signature": {"foo"}},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _ := test.SetupFakeContext(t)
			headers := http.Header{"Content-Type": []string{"application/json"}}
			if tt.eventType != "" {
				headers["X-GITHUB-EVENT"] = []string{tt.eventType}
			}
			for k, v := range tt.headers {
				headers[k] = v
			}

			req := &triggersv1.InterceptorRequest{
				Body:   string(tt.payload),
				Header: headers,
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

			clientset := fakekubeclient.Get(ctx)
			if tt.secret != nil {
				tt.secret.Namespace = metav1.NamespaceDefault
				ctx, clientset = fakekubeclient.With(ctx, tt.secret)
			}

			w := &Interceptor{
				SecretGetter: interceptors.DefaultSecretGetter(clientset.CoreV1()),
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
	emptyBodyHMACSignature := test.HMACHeader(t, secretToken, emptyJSONBody, "sha1")

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

			clientset := fakekubeclient.Get(ctx)
			if tt.secret != nil {
				tt.secret.Namespace = metav1.NamespaceDefault
				ctx, clientset = fakekubeclient.With(ctx, tt.secret)
			}

			w := &Interceptor{
				SecretGetter: interceptors.DefaultSecretGetter(clientset.CoreV1()),
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
		SecretGetter: interceptors.DefaultSecretGetter(fakekubeclient.Get(ctx).CoreV1()),
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

	w := &Interceptor{
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

func TestInterceptor_ExecuteTrigger_owners_IssueComment(t *testing.T) {
	tests := []struct {
		name               string
		commentsReply      string
		secret             *corev1.Secret
		interceptorRequest *triggersv1.InterceptorRequest
		allowed            bool
		wantErr            bool
	}{
		{
			name:          "owner issue comment with ok-to-test event",
			commentsReply: `[{"body": "/ok-to-test", "sender": {"login": "owner"}}]`,
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"action": "created", "issue": {"number": 1}, "repository": {"full_name": "owner/repo"}, "sender": {"login": "nonowner"}}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"issue_comment"}},
			},
			allowed: true,
			wantErr: false,
		},
		{
			name:          "owner issue comment with other than ok-to-test event",
			commentsReply: `[{"body": "/something", "sender": {"login": "owner"}}]`,
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"action": "created", "issue": {"number": 1}, "repository": {"full_name": "owner/repo"}, "sender": {"login": "nonowner"}}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"issue_comment"}},
			},
			allowed: false,
			wantErr: false,
		},
		{
			name:          "nonowner issue comment event",
			commentsReply: `[{"body": "/ok-to-test", "sender": {"login": "nonowner"}}]`,
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"action": "created", "issue": {"number": 1}, "repository": {"full_name": "owner/repo"}, "sender": {"login": "nonowner"}}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"issue_comment"}},
			},
			allowed: false,
			wantErr: false,
		},
		{
			name:          "nonowner issue comment event",
			commentsReply: `[{"body": "/ok-to-test", "sender": {"login": "nonowner"}}]`,
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"action": "created", "issue": {"number": 1}, "repository": {"full_name": "owner/repo"}, "sender": {"login": "nonowner"}}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"issue_comment"}},
			},
			allowed: false,
			wantErr: false,
		},
		{
			name: "owner raising pull request",
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"action": "opened","number": 2,"repository":{"full_name": "owner/repo"}, "sender":{"login": "owner"}}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"pull_request"}},
			},
			allowed: true,
			wantErr: false,
		},
		{
			name: "nonowner raising pull request with no owner making a comment",
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"action": "opened","number": 2,"repository":{"full_name": "owner/repo"}, "sender":{"login": "nonowner"}}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"pull_request"}},
			},
			allowed: false,
			wantErr: false,
		},
		{
			name:          "nonowner raising pull request with a owner making a comment /ok-to-test",
			commentsReply: `[{"body": "/ok-to-test", "sender": {"login": "owner"}}]`,
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"action": "opened","number": 2,"repository":{"full_name": "owner/repo"}, "sender":{"login": "nonowner"}}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"pull_request"}},
			},
			allowed: true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				writer.Write([]byte(tt.commentsReply))
			}))
			ctx, _ := test.SetupFakeContext(t)
			type cfgKey struct{}
			ctx = context.WithValue(ctx, cfgKey{}, ts.URL)
			clientset := fakekubeclient.Get(ctx)
			if tt.secret != nil {
				tt.secret.Namespace = metav1.NamespaceDefault
				ctx, clientset = fakekubeclient.With(ctx, tt.secret)
			}

			w := &Interceptor{
				SecretGetter: interceptors.DefaultSecretGetter(clientset.CoreV1()),
			}
			res := w.Process(ctx, tt.interceptorRequest)

			if !res.Continue {
				t.Fatalf("Interceptor.Process() expected res.Continue to be true but got %t. \nStatus.Err(): %v", res.Continue, res.Status.Err())
			}
		})
	}
}

func TestInterceptor_ExecuteTrigger_owners_PullRequest(t *testing.T) {
	tests := []struct {
		name               string
		commentsReply      string
		secret             *corev1.Secret
		interceptorRequest *triggersv1.InterceptorRequest
		allowed            bool
		wantErr            bool
	}{
		{
			name: "owner raising pull request",
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"action": "opened","number": 2,"repository":{"full_name": "owner/repo"}, "sender":{"login": "owner"}}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"pull_request"}},
			},
			allowed: true,
			wantErr: false,
		},
		{
			name: "nonowner raising pull request with no owner making a comment",
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"action": "opened","number": 2,"repository":{"full_name": "owner/repo"}, "sender":{"login": "nonowner"}}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"pull_request"}},
			},
			allowed: false,
			wantErr: false,
		},
		{
			name:          "nonowner raising pull request with a owner making a comment /ok-to-test",
			commentsReply: `[{"body": "/ok-to-test", "sender": {"login": "owner"}}]`,
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"action": "opened","number": 2,"repository":{"full_name": "owner/repo"}, "sender":{"login": "nonowner"}}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"pull_request"}},
			},
			allowed: true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				writer.Write([]byte(""))
			}))
			ctx, _ := test.SetupFakeContext(t)
			type cfgKey struct{}
			ctx = context.WithValue(ctx, cfgKey{}, ts.URL)
			clientset := fakekubeclient.Get(ctx)
			if tt.secret != nil {
				tt.secret.Namespace = metav1.NamespaceDefault
				ctx, clientset = fakekubeclient.With(ctx, tt.secret)
			}

			w := &Interceptor{
				SecretGetter: interceptors.DefaultSecretGetter(clientset.CoreV1()),
			}
			res := w.Process(ctx, tt.interceptorRequest)

			if !res.Continue {
				t.Fatalf("Interceptor.Process() expected res.Continue to be true but got %t. \nStatus.Err(): %v", res.Continue, res.Status.Err())
			}
		})
	}
}
