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

func TestInterceptor_ExecuteTrigger_Changed_Files_Pull_Request(t *testing.T) {
	var secretToken = "secret"
	tests := []struct {
		name               string
		githubServerReply  string
		secret             *corev1.Secret
		interceptorRequest *triggersv1.InterceptorRequest
		wantResContinue    bool
		want               string
		wantStatusMessage  string
	}{
		{
			name:              "changed_files",
			githubServerReply: `[{"filename":"terraform/envs/dev/main.tf"},{"filename":"terraform/envs/prod/main.tf"},{"filename":"terraform/envs/qa/main.tf"}]`,
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"action":"opened","number":1,"pull_request":{"head":{"sha":"28911bbb5"}},"repository":{"full_name":"testowner/testrepo","clone_url":"https://github.com/testowner/testrepo.git"}}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"pull_request"}},
				Context: &triggersv1.TriggerContext{
					EventURL:  "https://testing.example.com",
					EventID:   "abcde",
					TriggerID: "namespaces/default/triggers/example-trigger",
				},
				InterceptorParams: map[string]interface{}{
					"eventTypes": []string{"pull_request", "push"},
					"addChangedFiles": &triggersv1.GithubAddChangedFiles{
						Enabled: true,
						PersonalAccessToken: &triggersv1.SecretRef{
							SecretName: "mysecret",
							SecretKey:  "token",
						},
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mysecret",
				},
				Data: map[string][]byte{
					"token": []byte(secretToken),
				},
			},
			wantResContinue:   true,
			want:              "terraform/envs/dev/main.tf,terraform/envs/prod/main.tf,terraform/envs/qa/main.tf",
			wantStatusMessage: "",
		},
		{
			name:              "empty body, failure",
			githubServerReply: `[{"filename":"terraform/envs/dev/main.tf"},{"filename":"terraform/envs/prod/main.tf"},{"filename":"terraform/envs/qa/main.tf"}]`,
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   "",
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"pull_request"}},
				Context: &triggersv1.TriggerContext{
					EventURL:  "https://testing.example.com",
					EventID:   "abcde",
					TriggerID: "namespaces/default/triggers/example-trigger",
				},
				InterceptorParams: map[string]interface{}{
					"eventTypes": []string{"pull_request", "push"},
					"addChangedFiles": &triggersv1.GithubAddChangedFiles{
						Enabled: true,
						PersonalAccessToken: &triggersv1.SecretRef{
							SecretName: "mysecret",
							SecretKey:  "token",
						},
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mysecret",
				},
				Data: map[string][]byte{
					"token": []byte(secretToken),
				},
			},
			wantResContinue:   false,
			want:              "",
			wantStatusMessage: "error parsing body: body is empty",
		},
		{
			name:              "non json body, failure",
			githubServerReply: `[{"filename":"terraform/envs/dev/main.tf"},{"filename":"terraform/envs/prod/main.tf"},{"filename":"terraform/envs/qa/main.tf"}]`,
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `this is not json`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"pull_request"}},
				Context: &triggersv1.TriggerContext{
					EventURL:  "https://testing.example.com",
					EventID:   "abcde",
					TriggerID: "namespaces/default/triggers/example-trigger",
				},
				InterceptorParams: map[string]interface{}{
					"eventTypes": []string{"pull_request", "push"},
					"addChangedFiles": &triggersv1.GithubAddChangedFiles{
						Enabled: true,
						PersonalAccessToken: &triggersv1.SecretRef{
							SecretName: "mysecret",
							SecretKey:  "token",
						},
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mysecret",
				},
				Data: map[string][]byte{
					"token": []byte(secretToken),
				},
			},
			wantResContinue:   false,
			want:              "",
			wantStatusMessage: "error parsing body: invalid character 'h' in literal true (expecting 'r')",
		},
		{
			name:              "pull request, missing 'number' json field",
			githubServerReply: `[{"filename":"terraform/envs/dev/main.tf"},{"filename":"terraform/envs/prod/main.tf"},{"filename":"terraform/envs/qa/main.tf"}]`,
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"action":"opened","pull_request":{"head":{"sha":"28911bbb5"}},"repository":{"full_name":"testowner/testrepo","clone_url":"https://github.com/testowner/testrepo.git"}}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"pull_request"}},
				Context: &triggersv1.TriggerContext{
					EventURL:  "https://testing.example.com",
					EventID:   "abcde",
					TriggerID: "namespaces/default/triggers/example-trigger",
				},
				InterceptorParams: map[string]interface{}{
					"eventTypes": []string{"pull_request", "push"},
					"addChangedFiles": &triggersv1.GithubAddChangedFiles{
						Enabled: true,
						PersonalAccessToken: &triggersv1.SecretRef{
							SecretName: "mysecret",
							SecretKey:  "token",
						},
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mysecret",
				},
				Data: map[string][]byte{
					"token": []byte(secretToken),
				},
			},
			wantResContinue:   false,
			want:              "",
			wantStatusMessage: "error parsing body: pull_request body missing 'number' field",
		},
		{
			name:              "missing repository json field, failure",
			githubServerReply: `[{"filename":"terraform/envs/dev/main.tf"},{"filename":"terraform/envs/prod/main.tf"},{"filename":"terraform/envs/qa/main.tf"}]`,
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"action":"opened","number":1,"pull_request":{"head":{"sha":"28911bbb5"}}}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"pull_request"}},
				Context: &triggersv1.TriggerContext{
					EventURL:  "https://testing.example.com",
					EventID:   "abcde",
					TriggerID: "namespaces/default/triggers/example-trigger",
				},
				InterceptorParams: map[string]interface{}{
					"eventTypes": []string{"pull_request", "push"},
					"addChangedFiles": &triggersv1.GithubAddChangedFiles{
						Enabled: true,
						PersonalAccessToken: &triggersv1.SecretRef{
							SecretName: "mysecret",
							SecretKey:  "token",
						},
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mysecret",
				},
				Data: map[string][]byte{
					"token": []byte(secretToken),
				},
			},
			wantResContinue:   false,
			want:              "",
			wantStatusMessage: "error parsing body: payload body missing 'repository' field",
		},
		{
			name:              "missing full_name json field, failure",
			githubServerReply: `[{"filename":"terraform/envs/dev/main.tf"},{"filename":"terraform/envs/prod/main.tf"},{"filename":"terraform/envs/qa/main.tf"}]`,
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"action":"opened","number":1,"pull_request":{"head":{"sha":"28911bbb5"}},"repository":{"clone_url":"https://github.com/testowner/testrepo.git"}}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"pull_request"}},
				Context: &triggersv1.TriggerContext{
					EventURL:  "https://testing.example.com",
					EventID:   "abcde",
					TriggerID: "namespaces/default/triggers/example-trigger",
				},
				InterceptorParams: map[string]interface{}{
					"eventTypes": []string{"pull_request", "push"},
					"addChangedFiles": &triggersv1.GithubAddChangedFiles{
						Enabled: true,
						PersonalAccessToken: &triggersv1.SecretRef{
							SecretName: "mysecret",
							SecretKey:  "token",
						},
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mysecret",
				},
				Data: map[string][]byte{
					"token": []byte(secretToken),
				},
			},
			wantResContinue:   false,
			want:              "",
			wantStatusMessage: "error parsing body: payload body missing 'repository.full_name' field",
		},
		{
			name:              "event type not push or pull_request, failure",
			githubServerReply: `[{"filename":"terraform/envs/dev/main.tf"},{"filename":"terraform/envs/prod/main.tf"},{"filename":"terraform/envs/qa/main.tf"}]`,
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"action":"opened","number":1,"pull_request":{"head":{"sha":"28911bbb5"}},"repository":{"clone_url":"https://github.com/testowner/testrepo.git"}}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"nothing"}},
				Context: &triggersv1.TriggerContext{
					EventURL:  "https://testing.example.com",
					EventID:   "abcde",
					TriggerID: "namespaces/default/triggers/example-trigger",
				},
				InterceptorParams: map[string]interface{}{
					"eventTypes": []string{"pull_request", "push", "nothing"},
					"addChangedFiles": &triggersv1.GithubAddChangedFiles{
						Enabled: true,
						PersonalAccessToken: &triggersv1.SecretRef{
							SecretName: "mysecret",
							SecretKey:  "token",
						},
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mysecret",
				},
				Data: map[string][]byte{
					"token": []byte(secretToken),
				},
			},
			wantResContinue:   true,
			want:              "",
			wantStatusMessage: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				writer.Write([]byte(tt.githubServerReply))
			}))
			ctx, _ := test.SetupFakeContext(t)

			ctx = context.WithValue(ctx, testURL, ts.URL)
			clientset := fakekubeclient.Get(ctx)
			if tt.secret != nil {
				tt.secret.Namespace = metav1.NamespaceDefault
				ctx, clientset = fakekubeclient.With(ctx, tt.secret)
			}

			w := &Interceptor{
				SecretGetter: interceptors.DefaultSecretGetter(clientset.CoreV1()),
			}
			res := w.Process(ctx, tt.interceptorRequest)

			if res.Continue != tt.wantResContinue {
				t.Fatalf("Interceptor.Process() expected res.Continue to be %t but got %t. \nStatus.Err(): %v", tt.wantResContinue, res.Continue, res.Status.Err())
			}

			if res.Status.Message != tt.wantStatusMessage {
				t.Fatalf("Interceptor.Process() expected res.Status.Message to be '%s' but got '%s'", tt.wantStatusMessage, res.Status.Message)
			}

			changedFilesExt := res.Extensions[changedFilesExtensionsKey]
			if changedFilesExt == nil {
				changedFilesExt = ""
			}
			if tt.want != changedFilesExt {
				t.Fatalf("Interceptor.Process() got %v '%v', want '%v'", changedFilesExtensionsKey, changedFilesExt, tt.want)
			}
		})
	}
}

func TestInterceptor_ExecuteTrigger_Changed_Files_Push(t *testing.T) {
	var secretToken = "secret"
	tests := []struct {
		name               string
		githubServerReply  string
		secret             *corev1.Secret
		interceptorRequest *triggersv1.InterceptorRequest
		wantResContinue    bool
		want               string
		wantStatusMessage  string
	}{
		{
			name:              "changed_files",
			githubServerReply: `[{"filename":"terraform/envs/dev/main.tf"},{"filename":"terraform/envs/prod/main.tf"},{"filename":"terraform/envs/qa/main.tf"}]`,
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"repository":{"full_name":"testowner/testrepo","clone_url":"https://github.com/testowner/testrepo.git"},"commits":[{"added":["api/v1beta1/tektonhelperconfig_types.go","config/crd/bases/tekton-helper..com_tektonhelperconfigs.yaml"],"removed":["config/samples/tektonhelperconfig-oomkillpipeline.yaml","config/samples/tektonhelperconfig-timeout.yaml"],"modified":["controllers/tektonhelperconfig_controller.go"]}]}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"push"}},
				Context: &triggersv1.TriggerContext{
					EventURL:  "https://testing.example.com",
					EventID:   "abcde",
					TriggerID: "namespaces/default/triggers/example-trigger",
				},
				InterceptorParams: map[string]interface{}{
					"eventTypes": []string{"pull_request", "push"},
					"addChangedFiles": &triggersv1.GithubAddChangedFiles{
						Enabled: true,
						PersonalAccessToken: &triggersv1.SecretRef{
							SecretName: "mysecret",
							SecretKey:  "token",
						},
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mysecret",
				},
				Data: map[string][]byte{
					"token": []byte(secretToken),
				},
			},
			wantResContinue:   true,
			want:              "api/v1beta1/tektonhelperconfig_types.go,config/crd/bases/tekton-helper..com_tektonhelperconfigs.yaml,controllers/tektonhelperconfig_controller.go,config/samples/tektonhelperconfig-oomkillpipeline.yaml,config/samples/tektonhelperconfig-timeout.yaml",
			wantStatusMessage: "",
		},
		{
			name:              "missing commits added json field, failure",
			githubServerReply: `[{"filename":"terraform/envs/dev/main.tf"},{"filename":"terraform/envs/prod/main.tf"},{"filename":"terraform/envs/qa/main.tf"}]`,
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"repository":{"full_name":"testowner/testrepo","clone_url":"https://github.com/testowner/testrepo.git"},"commits":[{"removed":["config/samples/tektonhelperconfig-oomkillpipeline.yaml","config/samples/tektonhelperconfig-timeout.yaml"],"modified":["controllers/tektonhelperconfig_controller.go"]}]}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"push"}},
				Context: &triggersv1.TriggerContext{
					EventURL:  "https://testing.example.com",
					EventID:   "abcde",
					TriggerID: "namespaces/default/triggers/example-trigger",
				},
				InterceptorParams: map[string]interface{}{
					"eventTypes": []string{"pull_request", "push"},
					"addChangedFiles": &triggersv1.GithubAddChangedFiles{
						Enabled: true,
						PersonalAccessToken: &triggersv1.SecretRef{
							SecretName: "mysecret",
							SecretKey:  "token",
						},
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mysecret",
				},
				Data: map[string][]byte{
					"token": []byte(secretToken),
				},
			},
			wantResContinue:   false,
			want:              "",
			wantStatusMessage: "error parsing body: payload body missing 'commits.*.added' field",
		},
		{
			name:              "missing commits removed json field, failure",
			githubServerReply: `[{"filename":"terraform/envs/dev/main.tf"},{"filename":"terraform/envs/prod/main.tf"},{"filename":"terraform/envs/qa/main.tf"}]`,
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"repository":{"full_name":"testowner/testrepo","clone_url":"https://github.com/testowner/testrepo.git"},"commits":[{"added":["api/v1beta1/tektonhelperconfig_types.go","config/crd/bases/tekton-helper..com_tektonhelperconfigs.yaml"],"modified":["controllers/tektonhelperconfig_controller.go"]}]}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"push"}},
				Context: &triggersv1.TriggerContext{
					EventURL:  "https://testing.example.com",
					EventID:   "abcde",
					TriggerID: "namespaces/default/triggers/example-trigger",
				},
				InterceptorParams: map[string]interface{}{
					"eventTypes": []string{"pull_request", "push"},
					"addChangedFiles": &triggersv1.GithubAddChangedFiles{
						Enabled: true,
						PersonalAccessToken: &triggersv1.SecretRef{
							SecretName: "mysecret",
							SecretKey:  "token",
						},
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mysecret",
				},
				Data: map[string][]byte{
					"token": []byte(secretToken),
				},
			},
			wantResContinue:   false,
			want:              "",
			wantStatusMessage: "error parsing body: payload body missing 'commits.*.removed' field",
		},
		{
			name:              "missing commits modified json field, failure",
			githubServerReply: `[{"filename":"terraform/envs/dev/main.tf"},{"filename":"terraform/envs/prod/main.tf"},{"filename":"terraform/envs/qa/main.tf"}]`,
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"repository":{"full_name":"testowner/testrepo","clone_url":"https://github.com/testowner/testrepo.git"},"commits":[{"added":["api/v1beta1/tektonhelperconfig_types.go","config/crd/bases/tekton-helper..com_tektonhelperconfigs.yaml"],"removed":["config/samples/tektonhelperconfig-oomkillpipeline.yaml","config/samples/tektonhelperconfig-timeout.yaml"]}]}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"push"}},
				Context: &triggersv1.TriggerContext{
					EventURL:  "https://testing.example.com",
					EventID:   "abcde",
					TriggerID: "namespaces/default/triggers/example-trigger",
				},
				InterceptorParams: map[string]interface{}{
					"eventTypes": []string{"pull_request", "push"},
					"addChangedFiles": &triggersv1.GithubAddChangedFiles{
						Enabled: true,
						PersonalAccessToken: &triggersv1.SecretRef{
							SecretName: "mysecret",
							SecretKey:  "token",
						},
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mysecret",
				},
				Data: map[string][]byte{
					"token": []byte(secretToken),
				},
			},
			wantResContinue:   false,
			want:              "",
			wantStatusMessage: "error parsing body: payload body missing 'commits.*.modified' field",
		},
		{
			name:              "no context with secretRef",
			githubServerReply: `[{"filename":"terraform/envs/dev/main.tf"},{"filename":"terraform/envs/prod/main.tf"},{"filename":"terraform/envs/qa/main.tf"}]`,
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"repository":{"full_name":"testowner/testrepo","clone_url":"https://github.com/testowner/testrepo.git"},"commits":[{"added":["api/v1beta1/tektonhelperconfig_types.go","config/crd/bases/tekton-helper..com_tektonhelperconfigs.yaml"],"removed":["config/samples/tektonhelperconfig-oomkillpipeline.yaml","config/samples/tektonhelperconfig-timeout.yaml"]}]}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"push"}},
				InterceptorParams: map[string]interface{}{
					"eventTypes": []string{"pull_request", "push"},
					"secretRef": &triggersv1.SecretRef{
						SecretName: "mysecret",
						SecretKey:  "token",
					},
					"addChangedFiles": &triggersv1.GithubAddChangedFiles{
						Enabled: true,
						PersonalAccessToken: &triggersv1.SecretRef{
							SecretName: "mysecret",
							SecretKey:  "token",
						},
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mysecret",
				},
				Data: map[string][]byte{
					"token": []byte(secretToken),
				},
			},
			wantResContinue:   false,
			want:              "",
			wantStatusMessage: "no request context passed",
		},
		{
			name:              "no context",
			githubServerReply: `[{"filename":"terraform/envs/dev/main.tf"},{"filename":"terraform/envs/prod/main.tf"},{"filename":"terraform/envs/qa/main.tf"}]`,
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"repository":{"full_name":"testowner/testrepo","clone_url":"https://github.com/testowner/testrepo.git"},"commits":[{"added":["api/v1beta1/tektonhelperconfig_types.go","config/crd/bases/tekton-helper..com_tektonhelperconfigs.yaml"],"removed":["config/samples/tektonhelperconfig-oomkillpipeline.yaml","config/samples/tektonhelperconfig-timeout.yaml"]}]}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"push"}},
				InterceptorParams: map[string]interface{}{
					"eventTypes": []string{"pull_request", "push"},
					"addChangedFiles": &triggersv1.GithubAddChangedFiles{
						Enabled: true,
						PersonalAccessToken: &triggersv1.SecretRef{
							SecretName: "mysecret",
							SecretKey:  "token",
						},
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mysecret",
				},
				Data: map[string][]byte{
					"token": []byte(secretToken),
				},
			},
			wantResContinue:   false,
			want:              "",
			wantStatusMessage: "no request context passed",
		},
		{
			name:              "invalid secret",
			githubServerReply: `[{"filename":"terraform/envs/dev/main.tf"},{"filename":"terraform/envs/prod/main.tf"},{"filename":"terraform/envs/qa/main.tf"}]`,
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"repository":{"full_name":"testowner/testrepo","clone_url":"https://github.com/testowner/testrepo.git"},"commits":[{"added":["api/v1beta1/tektonhelperconfig_types.go","config/crd/bases/tekton-helper..com_tektonhelperconfigs.yaml"],"removed":["config/samples/tektonhelperconfig-oomkillpipeline.yaml","config/samples/tektonhelperconfig-timeout.yaml"],"modified":["controllers/tektonhelperconfig_controller.go"]}]}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"push"}},
				Context: &triggersv1.TriggerContext{
					EventURL:  "https://testing.example.com",
					EventID:   "abcde",
					TriggerID: "namespaces/default/triggers/example-trigger",
				},
				InterceptorParams: map[string]interface{}{
					"eventTypes": []string{"pull_request", "push"},
					"addChangedFiles": &triggersv1.GithubAddChangedFiles{
						Enabled: true,
						PersonalAccessToken: &triggersv1.SecretRef{
							SecretName: "missingsecret",
							SecretKey:  "token",
						},
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mysecret",
				},
				Data: map[string][]byte{
					"token": []byte(secretToken),
				},
			},
			wantResContinue:   false,
			want:              "",
			wantStatusMessage: "error getting secret: secrets \"missingsecret\" not found",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				writer.Write([]byte(tt.githubServerReply))
			}))
			ctx, _ := test.SetupFakeContext(t)

			ctx = context.WithValue(ctx, testURL, ts.URL)
			clientset := fakekubeclient.Get(ctx)
			if tt.secret != nil {
				tt.secret.Namespace = metav1.NamespaceDefault
				ctx, clientset = fakekubeclient.With(ctx, tt.secret)
			}

			w := &Interceptor{
				SecretGetter: interceptors.DefaultSecretGetter(clientset.CoreV1()),
			}
			res := w.Process(ctx, tt.interceptorRequest)

			if res.Continue != tt.wantResContinue {
				t.Fatalf("Interceptor.Process() expected res.Continue to be %t but got %t. \nStatus.Err(): %v", tt.wantResContinue, res.Continue, res.Status.Err())
			}

			if res.Status.Message != tt.wantStatusMessage {
				t.Fatalf("Interceptor.Process() expected res.Status.Message to be '%s' but got '%s'", tt.wantStatusMessage, res.Status.Message)
			}

			changedFilesExt := res.Extensions[changedFilesExtensionsKey]
			if changedFilesExt == nil {
				changedFilesExt = ""
			}
			if tt.want != changedFilesExt {
				t.Fatalf("Interceptor.Process() got %v '%v', want '%v'", changedFilesExtensionsKey, changedFilesExt, tt.want)
			}
		})
	}
}

func Test_getGithubTokenSecret(t *testing.T) {

	ctx, _ := test.SetupFakeContext(t)
	var secretToken = "secret"

	type args struct {
		ctx context.Context
		r   *triggersv1.InterceptorRequest
		p   triggersv1.GitHubInterceptor
	}
	tests := []struct {
		name    string
		secret  *corev1.Secret
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "valid secret",
			args: args{
				ctx: ctx,
				r: &triggersv1.InterceptorRequest{
					Context: &triggersv1.TriggerContext{
						EventURL:  "https://testing.example.com",
						EventID:   "abcde",
						TriggerID: "namespaces/default/triggers/example-trigger",
					},
				},
				p: triggersv1.GitHubInterceptor{
					AddChangedFiles: triggersv1.GithubAddChangedFiles{
						PersonalAccessToken: &triggersv1.SecretRef{
							SecretName: "mysecret",
							SecretKey:  "token",
						},
					},

					EventTypes: []string{"pull_request", "push"},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mysecret",
				},
				Data: map[string][]byte{
					"token": []byte(secretToken),
				},
			},
			want:    "secret",
			wantErr: false,
		},
		{
			name: "nil secret reference",
			args: args{
				ctx: ctx,
				r: &triggersv1.InterceptorRequest{
					Context: &triggersv1.TriggerContext{
						EventURL:  "https://testing.example.com",
						EventID:   "abcde",
						TriggerID: "namespaces/default/triggers/example-trigger",
					},
				},
				p: triggersv1.GitHubInterceptor{
					EventTypes: []string{"pull_request", "push"},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mysecret",
				},
				Data: map[string][]byte{
					"token": []byte(secretToken),
				},
			},
			want:    "",
			wantErr: false,
		},
		{
			name: "missing secret key, failure",
			args: args{
				ctx: ctx,
				r: &triggersv1.InterceptorRequest{
					Context: &triggersv1.TriggerContext{
						EventURL:  "https://testing.example.com",
						EventID:   "abcde",
						TriggerID: "namespaces/default/triggers/example-trigger",
					},
				},
				p: triggersv1.GitHubInterceptor{
					AddChangedFiles: triggersv1.GithubAddChangedFiles{
						PersonalAccessToken: &triggersv1.SecretRef{
							SecretName: "mysecret",
						},
					},
					EventTypes: []string{"pull_request", "push"},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mysecret",
				},
				Data: map[string][]byte{
					"token": []byte(secretToken),
				},
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			clientset := fakekubeclient.Get(ctx)
			if tt.secret != nil {
				tt.secret.Namespace = metav1.NamespaceDefault
				ctx, clientset = fakekubeclient.With(ctx, tt.secret)
			}

			w := &Interceptor{
				SecretGetter: interceptors.DefaultSecretGetter(clientset.CoreV1()),
			}

			got, err := w.getGithubTokenSecret(tt.args.ctx, tt.args.r, tt.args.p)
			if (err != nil) != tt.wantErr {
				t.Errorf("Interceptor.getSecret() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Interceptor.getSecret() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_makeClient(t *testing.T) {
	ctx, _ := test.SetupFakeContext(t)
	type args struct {
		ctx               context.Context
		enterpriseBaseURL string
		token             string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "public github",
			args: args{
				ctx:               ctx,
				enterpriseBaseURL: "",
				token:             "",
			},
			want: "api.github.com",
		},
		{
			name: "enterprise github",
			args: args{
				ctx:               ctx,
				enterpriseBaseURL: "github.somecompany.com",
				token:             "1234567",
			},
			want: "github.somecompany.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := makeClient(tt.args.ctx, tt.args.enterpriseBaseURL, tt.args.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("makeClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got.BaseURL.Host != tt.want {
				t.Errorf("makeClient() = %v, want %v", got.BaseURL.Host, tt.want)
			}

		})
	}
}

func Test_getPersonalAccessTokenSecret(t *testing.T) {

	ctx, _ := test.SetupFakeContext(t)
	var secretToken = "secret"

	type args struct {
		ctx context.Context
		r   *triggersv1.InterceptorRequest
		p   triggersv1.GitHubInterceptor
	}
	tests := []struct {
		name    string
		secret  *corev1.Secret
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "valid secret",
			args: args{
				ctx: ctx,
				r: &triggersv1.InterceptorRequest{
					Context: &triggersv1.TriggerContext{
						EventURL:  "https://testing.example.com",
						EventID:   "abcde",
						TriggerID: "namespaces/default/triggers/example-trigger",
					},
				},
				p: triggersv1.GitHubInterceptor{
					GithubOwners: triggersv1.GithubOwners{
						PersonalAccessToken: &triggersv1.SecretRef{
							SecretName: "mysecret",
							SecretKey:  "token",
						},
					},
					EventTypes: []string{"pull_request", "issue_comment"},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mysecret",
				},
				Data: map[string][]byte{
					"token": []byte(secretToken),
				},
			},
			want:    "secret",
			wantErr: false,
		},
		{
			name: "nil secret reference",
			// w:    interceptor,
			args: args{
				ctx: ctx,
				r: &triggersv1.InterceptorRequest{
					Context: &triggersv1.TriggerContext{
						EventURL:  "https://testing.example.com",
						EventID:   "abcde",
						TriggerID: "namespaces/default/triggers/example-trigger",
					},
				},
				p: triggersv1.GitHubInterceptor{
					EventTypes: []string{"pull_request", "issue_comment"},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mysecret",
				},
				Data: map[string][]byte{
					"token": []byte(secretToken),
				},
			},
			want:    "",
			wantErr: false,
		},
		{
			name: "missing secret key, failure",
			// w:    interceptor,
			args: args{
				ctx: ctx,
				r: &triggersv1.InterceptorRequest{
					Context: &triggersv1.TriggerContext{
						EventURL:  "https://testing.example.com",
						EventID:   "abcde",
						TriggerID: "namespaces/default/triggers/example-trigger",
					},
				},
				p: triggersv1.GitHubInterceptor{
					GithubOwners: triggersv1.GithubOwners{
						PersonalAccessToken: &triggersv1.SecretRef{
							SecretName: "mysecret",
						},
					},
					EventTypes: []string{"pull_request", "issue_comment"},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mysecret",
				},
				Data: map[string][]byte{
					"token": []byte(secretToken),
				},
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			clientset := fakekubeclient.Get(ctx)
			if tt.secret != nil {
				tt.secret.Namespace = metav1.NamespaceDefault
				ctx, clientset = fakekubeclient.With(ctx, tt.secret)
			}

			w := &Interceptor{
				SecretGetter: interceptors.DefaultSecretGetter(clientset.CoreV1()),
			}

			got, err := w.getPersonalAccessTokenSecret(tt.args.ctx, tt.args.r, tt.args.p)
			if (err != nil) != tt.wantErr {
				t.Errorf("Interceptor.getSecret() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Interceptor.getSecret() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInterceptor_ExecuteTrigger_owners(t *testing.T) {
	secretToken := "secret"
	tests := []struct {
		name                  string
		issueCommentReply     string
		ownersFileReply       string
		collaboratorsReply    string
		orgPublicMembersReply string
		secret                *corev1.Secret
		interceptorRequest    *triggersv1.InterceptorRequest
		allowed               bool
		wantErr               bool
		want                  string
	}{
		{
			name:            "owners file have sender",
			ownersFileReply: `{"type": "file","encoding": "base64","content": "YXBwcm92ZXJzOg0KLSB0ZXN0X293bmVy"}`,
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"action": "created", "issue": {"number": 1}, "comment": {"body": "/ok-to-test"}, "repository": {"full_name": "owner/repo"}, "sender": {"login": "test_owner"}}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"issue_comment"}},
				InterceptorParams: map[string]interface{}{
					"eventTypes": []string{"pull_request", "issue_comment"},
					"githubOwners": &triggersv1.GithubOwners{
						Enabled: true,
						PersonalAccessToken: &triggersv1.SecretRef{
							SecretName: "mysecret",
							SecretKey:  "token",
						},
						CheckType: "none",
					},
				},
				Context: &triggersv1.TriggerContext{
					EventURL:  "https://testing.example.com",
					EventID:   "abcde",
					TriggerID: "namespaces/default/triggers/example-trigger",
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mysecret",
				},
				Data: map[string][]byte{
					"token": []byte(secretToken),
				},
			},
			allowed: true,
			wantErr: false,
		},
		{
			name:               "sender is a collaborator but PersonalAccessToken not supplied",
			ownersFileReply:    `{"type": "file","encoding": "base64","content": "YXBwcm92ZXJzOg0KLSB0ZXN0X293bmVy"}`,
			collaboratorsReply: `[{"login": "test_owner"}]`,
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"action": "created", "issue": {"number": 1}, "comment": {"body": "/ok-to-test"}, "repository": {"full_name": "owner/repo"}, "sender": {"login": "test_owner"}}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"issue_comment"}},
				InterceptorParams: map[string]interface{}{
					"eventTypes": []string{"pull_request", "issue_comment"},
					"githubOwners": &triggersv1.GithubOwners{
						Enabled:   true,
						CheckType: "repoMembers",
					},
				},
				Context: &triggersv1.TriggerContext{
					EventURL:  "https://testing.example.com",
					EventID:   "abcde",
					TriggerID: "namespaces/default/triggers/example-trigger",
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mysecret",
				},
				Data: map[string][]byte{
					"token": []byte(secretToken),
				},
			},
			allowed: false,
			wantErr: true,
			want:    "checkType is set to check org or repo members but no personalAccessToken was supplied",
		},
		{
			name:               "sender is a repository member",
			ownersFileReply:    `{"type": "file","encoding": "base64","content": "YXBwcm92ZXJzOg0KLSB0ZXN0X293bmVy"}`,
			collaboratorsReply: `[{"login": "test_owner"}]`,
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"action": "created", "issue": {"number": 1}, "comment": {"body": "/ok-to-test"}, "repository": {"full_name": "owner/repo"}, "sender": {"login": "test_owner"}}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"issue_comment"}},
				InterceptorParams: map[string]interface{}{
					"eventTypes": []string{"pull_request", "issue_comment"},
					"githubOwners": &triggersv1.GithubOwners{
						Enabled: true,
						PersonalAccessToken: &triggersv1.SecretRef{
							SecretName: "mysecret",
							SecretKey:  "token",
						},
						CheckType: "repoMembers",
					},
				},
				Context: &triggersv1.TriggerContext{
					EventURL:  "https://testing.example.com",
					EventID:   "abcde",
					TriggerID: "namespaces/default/triggers/example-trigger",
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mysecret",
				},
				Data: map[string][]byte{
					"token": []byte(secretToken),
				},
			},
			allowed: true,
			wantErr: false,
		},
		{
			name:               "sender is a organization member",
			ownersFileReply:    `{"type": "file","encoding": "base64","content": "YXBwcm92ZXJzOg0KLSB0ZXN0X293bmVy"}`,
			collaboratorsReply: `[{"login": "test_owner"}]`,
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"action": "opened", "number": 1, "repository":{"full_name": "owner/repo", "clone_url": "https://github.com/owner/repo.git"}, "sender":{"login": "test_owner"}}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"pull_request"}},
				InterceptorParams: map[string]interface{}{
					"eventTypes": []string{"pull_request", "issue_comment"},
					"githubOwners": &triggersv1.GithubOwners{
						Enabled: true,
						PersonalAccessToken: &triggersv1.SecretRef{
							SecretName: "mysecret",
							SecretKey:  "token",
						},
						CheckType: "orgMembers",
					},
				},
				Context: &triggersv1.TriggerContext{
					EventURL:  "https://testing.example.com",
					EventID:   "abcde",
					TriggerID: "namespaces/default/triggers/example-trigger",
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mysecret",
				},
				Data: map[string][]byte{
					"token": []byte(secretToken),
				},
			},
			allowed: true,
			wantErr: false,
		},
		{
			name:            "owners file does not have sender",
			ownersFileReply: `{"type": "file","encoding": "base64","content": "YXBwcm92ZXJzOg0KLSB0ZXN0X293bmVy"}`,
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"action": "created", "issue": {"number": 1}, "comment": {"body": "/ok-to-test"}, "repository": {"full_name": "owner/repo"}, "sender": {"login": "nonowner"}}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"issue_comment"}},
				InterceptorParams: map[string]interface{}{
					"eventTypes": []string{"pull_request", "issue_comment"},
					"githubOwners": &triggersv1.GithubOwners{
						Enabled:   true,
						CheckType: "none",
					},
				},
				Context: &triggersv1.TriggerContext{
					EventURL:  "https://testing.example.com",
					EventID:   "abcde",
					TriggerID: "namespaces/default/triggers/example-trigger",
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mysecret",
				},
				Data: map[string][]byte{
					"token": []byte(secretToken),
				},
			},
			allowed: false,
			wantErr: true,
			want:    "owners check requirements not met",
		},
		{
			name:              "owners file have sender and sender commented /ok-to-test",
			issueCommentReply: `[{"body": "/ok-to-test", "sender": {"login": "test_owner"}}]`,
			ownersFileReply:   `{"type": "file","encoding": "base64","content": "YXBwcm92ZXJzOg0KLSB0ZXN0X293bmVy"}`,
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"action": "created", "issue": {"number": 1}, "comment": {"body": "/ok-to-test"}, "repository": {"full_name": "owner/repo"}, "sender": {"login": "test_owner"}}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"issue_comment"}},
				InterceptorParams: map[string]interface{}{
					"eventTypes": []string{"pull_request", "issue_comment"},
					"githubOwners": &triggersv1.GithubOwners{
						Enabled:   true,
						CheckType: "none",
					},
				},
				Context: &triggersv1.TriggerContext{
					EventURL:  "https://testing.example.com",
					EventID:   "abcde",
					TriggerID: "namespaces/default/triggers/example-trigger",
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mysecret",
				},
				Data: map[string][]byte{
					"token": []byte(secretToken),
				},
			},
			allowed: true,
			wantErr: false,
		},
		{
			name:              "owners file have sender and owner commented /random",
			issueCommentReply: `[{"body": "/random", "sender": {"login": "test_owner"}}]`,
			ownersFileReply:   `{"type": "file","encoding": "base64","content": "YXBwcm92ZXJzOg0KLSB0ZXN0X293bmVy"}`,
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"action": "created", "issue": {"number": 1}, "comment": {"body": "/random"}, "repository": {"full_name": "owner/repo"}, "sender": {"login": "test_owner"}}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"issue_comment"}},
				InterceptorParams: map[string]interface{}{
					"eventTypes": []string{"pull_request", "issue_comment"},
					"githubOwners": &triggersv1.GithubOwners{
						Enabled:   true,
						CheckType: "none",
					},
				},
				Context: &triggersv1.TriggerContext{
					EventURL:  "https://testing.example.com",
					EventID:   "abcde",
					TriggerID: "namespaces/default/triggers/example-trigger",
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mysecret",
				},
				Data: map[string][]byte{
					"token": []byte(secretToken),
				},
			},
			allowed: false,
			wantErr: true,
			want:    "owners check requirements not met",
		},
		{
			name:              "owners file does not have sender and nonowner commented /ok-to-test",
			issueCommentReply: `[{"body": "/ok-to-test", "sender": {"login": "nonowner"}}]`,
			ownersFileReply:   `{"type": "file","encoding": "base64","content": "YXBwcm92ZXJzOg0KLSB0ZXN0X293bmVy"}`,
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"action": "created", "issue": {"number": 1}, "comment": {"body": "/ok-to-test"}, "repository": {"full_name": "owner/repo"}, "sender": {"login": "nonowner"}}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"issue_comment"}},
				InterceptorParams: map[string]interface{}{
					"eventTypes": []string{"pull_request", "issue_comment"},
					"githubOwners": &triggersv1.GithubOwners{
						Enabled:   true,
						CheckType: "none",
					},
				},
				Context: &triggersv1.TriggerContext{
					EventURL:  "https://testing.example.com",
					EventID:   "abcde",
					TriggerID: "namespaces/default/triggers/example-trigger",
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mysecret",
				},
				Data: map[string][]byte{
					"token": []byte(secretToken),
				},
			},
			allowed: false,
			wantErr: true,
			want:    "owners check requirements not met",
		},
		{
			name:               "pull request submitted by owner",
			ownersFileReply:    `{"type": "file","encoding": "base64","content": "YXBwcm92ZXJzOg0KLSB0ZXN0X293bmVy"}`,
			collaboratorsReply: `[{"login": "test_owner"}]`,
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"action": "opened", "number": 1, "repository":{"full_name": "owner/repo", "clone_url": "https://github.com/owner/repo.git"}, "sender":{"login": "test_owner"}}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"pull_request"}},
				InterceptorParams: map[string]interface{}{
					"eventTypes": []string{"pull_request", "issue_comment"},
					"githubOwners": &triggersv1.GithubOwners{
						Enabled: true,
						PersonalAccessToken: &triggersv1.SecretRef{
							SecretName: "mysecret",
							SecretKey:  "token",
						},
						CheckType: "none",
					},
				},
				Context: &triggersv1.TriggerContext{
					EventURL:  "https://testing.example.com",
					EventID:   "abcde",
					TriggerID: "namespaces/default/triggers/example-trigger",
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mysecret",
				},
				Data: map[string][]byte{
					"token": []byte(secretToken),
				},
			},
			allowed: true,
			wantErr: false,
		},
		{
			name:               "sender is not a repository member",
			ownersFileReply:    `{"type": "file","encoding": "base64","content": "YXBwcm92ZXJzOg0KLSB0ZXN0X293bmVy"}`,
			collaboratorsReply: `[{"login": "test_owner"}]`,
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"action": "opened", "number": 1, "repository":{"full_name": "owner/repo", "clone_url": "https://github.com/owner/repo.git"}, "sender":{"login": "nonowner"}}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"pull_request"}},
				InterceptorParams: map[string]interface{}{
					"eventTypes": []string{"pull_request", "issue_comment"},
					"githubOwners": &triggersv1.GithubOwners{
						Enabled: true,
						PersonalAccessToken: &triggersv1.SecretRef{
							SecretName: "mysecret",
							SecretKey:  "token",
						},
						CheckType: "repoMembers",
					},
				},
				Context: &triggersv1.TriggerContext{
					EventURL:  "https://testing.example.com",
					EventID:   "abcde",
					TriggerID: "namespaces/default/triggers/example-trigger",
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mysecret",
				},
				Data: map[string][]byte{
					"token": []byte(secretToken),
				},
			},
			allowed: false,
			wantErr: true,
			want:    "owners check requirements not met",
		},
		{
			name:               "sender is not a organization member",
			ownersFileReply:    `{"type": "file","encoding": "base64","content": "YXBwcm92ZXJzOg0KLSB0ZXN0X293bmVy"}`,
			collaboratorsReply: `[{"login": "test_owner"}]`,
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"action": "opened", "number": 1, "repository":{"full_name": "owner/repo", "clone_url": "https://github.com/owner/repo.git"}, "sender":{"login": "nonowner"}}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"pull_request"}},
				InterceptorParams: map[string]interface{}{
					"eventTypes": []string{"pull_request", "issue_comment"},
					"githubOwners": &triggersv1.GithubOwners{
						Enabled: true,
						PersonalAccessToken: &triggersv1.SecretRef{
							SecretName: "mysecret",
							SecretKey:  "token",
						},
						CheckType: "orgMembers",
					},
				},
				Context: &triggersv1.TriggerContext{
					EventURL:  "https://testing.example.com",
					EventID:   "abcde",
					TriggerID: "namespaces/default/triggers/example-trigger",
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mysecret",
				},
				Data: map[string][]byte{
					"token": []byte(secretToken),
				},
			},
			allowed: false,
			wantErr: true,
			want:    "owners check requirements not met",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if request.URL.Path == "api/v3/repos/owner/repo/issues/comments" {
					writer.Write([]byte(tt.issueCommentReply))
				}
				if request.URL.Path == "/api/v3/repos/owner/repo/contents/OWNERS" {
					writer.Write([]byte(tt.ownersFileReply))
				}
				if request.URL.Path == "/api/v3/repos/owner/repo/collaborators" {
					writer.Write([]byte(tt.collaboratorsReply))
				}
				if request.URL.Path == "/api/v3/orgs/owner/public_members" {
					writer.Write([]byte(tt.collaboratorsReply))
				}
			}))
			ctx, _ := test.SetupFakeContext(t)
			ctx = context.WithValue(ctx, testURL, ts.URL)
			clientset := fakekubeclient.Get(ctx)
			if tt.secret != nil {
				tt.secret.Namespace = metav1.NamespaceDefault
				ctx, clientset = fakekubeclient.With(ctx, tt.secret)
			}

			w := &Interceptor{
				SecretGetter: interceptors.DefaultSecretGetter(clientset.CoreV1()),
			}
			res := w.Process(ctx, tt.interceptorRequest)

			if (!res.Continue) && (tt.wantErr == true) {
				t.Logf("Interceptor.Process() = %v, want %v", res.Status.Message, tt.want)
			} else if !res.Continue && (tt.wantErr != true) {
				t.Fatalf("Interceptor.Process() expected res.Continue to be true but got %t. \nStatus.Err(): %v", res.Continue, res.Status.Err())
			}
		})
	}
}

func TestInterceptor_ExecuteTrigger_owners_parseBodyForOwners(t *testing.T) {
	tests := []struct {
		name               string
		eventType          string
		interceptorRequest *triggersv1.InterceptorRequest
		allowed            bool
		wantErr            bool
		want               string
	}{
		{
			name:      "No payload supplied",
			eventType: "pull_request",
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   ``,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"pull_request"}},
			},
			allowed: false,
			wantErr: true,
			want:    "payload body is empty",
		},
		{
			name:      "PR number missing on the payload",
			eventType: "pull_request",
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"action": "opened","repository":{"full_name": "owner/repo", "clone_url": "https://github.com/owner/repo.git"}, "sender":{"login": "test_owner"}}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"pull_request"}},
			},
			allowed: false,
			wantErr: true,
			want:    "pull_request body missing 'number' field",
		},
		{
			name:      "Issue comment missing issue on the payload",
			eventType: "issue_comment",
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"action": "created", "comment": {"body": "/ok-to-test"}, "repository": {"full_name": "owner/repo"}, "sender": {"login": "test_owner"}}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"issue_comment"}},
			},
			allowed: false,
			wantErr: true,
			want:    "issue_comment body missing 'issue' section",
		},
		{
			name:      "Issue comment missing issue on the payload",
			eventType: "issue_comment",
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"action": "created", "issue": {}, "comment": {"body": "/ok-to-test"}, "repository": {"full_name": "owner/repo"}, "sender": {"login": "test_owner"}}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"issue_comment"}},
			},
			allowed: false,
			wantErr: true,
			want:    "'number' field missing in the issue section of issue_comment body",
		},
		{
			name:      "Payload body missing repository field",
			eventType: "issue_comment",
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"action": "created", "issue": {"number": 1}, "comment": {"body": "/ok-to-test"}, "sender": {"login": "test_owner"}}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"issue_comment"}},
			},
			allowed: false,
			wantErr: true,
			want:    "payload body missing 'repository' field",
		},
		{
			name:      "Payload body missing full_name in repository field",
			eventType: "issue_comment",
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"action": "created", "issue": {"number": 1}, "comment": {"body": "/ok-to-test"}, "repository": {}, "sender": {"login": "test_owner"}}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"issue_comment"}},
			},
			allowed: false,
			wantErr: true,
			want:    "payload body missing 'repository.full_name' field",
		},
		{
			name:      "Payload body missing sender field",
			eventType: "issue_comment",
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"action": "created", "issue": {"number": 1}, "comment": {"body": "/ok-to-test"}, "repository": {"full_name": "owner/repo"}}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"issue_comment"}},
			},
			allowed: false,
			wantErr: true,
			want:    "payload body missing 'sender' field",
		},
		{
			name:      "Issue_comment missing 'comment' section",
			eventType: "issue_comment",
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"action": "created", "issue": {"number": 1}, "repository": {"full_name": "owner/repo"}, "sender": {"login": "test_owner"}}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"issue_comment"}},
			},
			allowed: false,
			wantErr: true,
			want:    "issue_comment body missing 'comment' section",
		},
		{
			name:      "Issue_comment missing body field in comment section",
			eventType: "issue_comment",
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"action": "created", "issue": {"number": 1}, "comment": {}, "repository": {"full_name": "owner/repo"}, "sender": {"login": "test_owner"}}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"issue_comment"}},
			},
			allowed: false,
			wantErr: true,
			want:    "'body' field missing in the comment section of issue_comment body",
		},
		{
			name:      "Payload return with all details issue_comment",
			eventType: "issue_comment",
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"action": "created", "issue": {"number": 1}, "comment": {"body": "/ok-to-test"}, "repository": {"full_name": "owner/repo"}, "sender": {"login": "test_owner"}}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"issue_comment"}},
			},
			allowed: true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseBodyForOwners(tt.interceptorRequest.Body, tt.eventType)
			if (err != nil) != tt.wantErr {
				t.Errorf("Interceptor.parseBodyForOwners() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInterceptor_ExecuteTrigger_owners_data_validation(t *testing.T) {
	secretToken := "secret"
	tests := []struct {
		name                    string
		issueCommentReply       string
		ownersFileReply         string
		collaboratorsReply      string
		ownersAPICallStatusCode int
		secret                  *corev1.Secret
		interceptorRequest      *triggersv1.InterceptorRequest
		allowed                 bool
		wantErr                 bool
		want                    string
	}{
		{
			name:            "personalAccessToken secretKey is empty",
			ownersFileReply: `{"type": "file","encoding": "base64","content": "YXBwcm92ZXJzOg0KLSB0ZXN0X293bmVy"}`,
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"action": "created", "issue": {"number": 1}, "comment": {"body": "/ok-to-test"}, "repository": {"full_name": "owner/repo"}, "sender": {"login": "test_owner"}}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"issue_comment"}},
				InterceptorParams: map[string]interface{}{
					"eventTypes": []string{"pull_request", "issue_comment"},
					"githubOwners": &triggersv1.GithubOwners{
						Enabled: true,
						PersonalAccessToken: &triggersv1.SecretRef{
							SecretName: "mysecret",
							SecretKey:  "",
						},
						CheckType: "none",
					},
				},
				Context: &triggersv1.TriggerContext{
					EventURL:  "https://testing.example.com",
					EventID:   "abcde",
					TriggerID: "namespaces/default/triggers/example-trigger",
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mysecret",
				},
				Data: map[string][]byte{
					"token": []byte(secretToken),
				},
			},
			allowed: false,
			wantErr: true,
			want:    "error getting github token: github interceptor personalAccessToken.secretKey is empty",
		},
		{
			name: "error parsing payload body",
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"action": "opened","repository":{"full_name": "owner/repo", "clone_url": "https://github.com/owner/repo.git"}, "sender":{"login": "test_owner"}}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"pull_request"}},
				InterceptorParams: map[string]interface{}{
					"eventTypes": []string{"pull_request", "issue_comment"},
					"githubOwners": &triggersv1.GithubOwners{
						Enabled:   true,
						CheckType: "none",
					},
				},
				Context: &triggersv1.TriggerContext{
					EventURL:  "https://testing.example.com",
					EventID:   "abcde",
					TriggerID: "namespaces/default/triggers/example-trigger",
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mysecret",
				},
				Data: map[string][]byte{
					"token": []byte(secretToken),
				},
			},
			allowed: false,
			wantErr: true,
			want:    "error parsing body: pull_request body missing 'number' field",
		},
		{
			name:                    "no owners file",
			ownersFileReply:         `{"statusCode": 404}`,
			ownersAPICallStatusCode: 404,
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"action": "created", "issue": {"number": 1}, "comment": {"body": "/ok-to-test"}, "repository": {"full_name": "owner/repo"}, "sender": {"login": "test_owner"}}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"issue_comment"}},
				InterceptorParams: map[string]interface{}{
					"eventTypes": []string{"pull_request", "issue_comment"},
					"githubOwners": &triggersv1.GithubOwners{
						Enabled: true,
						PersonalAccessToken: &triggersv1.SecretRef{
							SecretName: "mysecret",
							SecretKey:  "token",
						},
						CheckType: "none",
					},
				},
				Context: &triggersv1.TriggerContext{
					EventURL:  "https://testing.example.com",
					EventID:   "abcde",
					TriggerID: "namespaces/default/triggers/example-trigger",
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mysecret",
				},
				Data: map[string][]byte{
					"token": []byte(secretToken),
				},
			},
			allowed: false,
			wantErr: true,
			want:    "owners check requirements not met",
		},
		{
			name:                    "error checking owner file validation",
			ownersFileReply:         `{"statusCode": 500}`,
			ownersAPICallStatusCode: 500,
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"action": "created", "issue": {"number": 1}, "comment": {"body": "/ok-to-test"}, "repository": {"full_name": "owner/repo"}, "sender": {"login": "test_owner"}}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"issue_comment"}},
				InterceptorParams: map[string]interface{}{
					"eventTypes": []string{"pull_request", "issue_comment"},
					"githubOwners": &triggersv1.GithubOwners{
						Enabled: true,
						PersonalAccessToken: &triggersv1.SecretRef{
							SecretName: "mysecret",
							SecretKey:  "token",
						},
						CheckType: "none",
					},
				},
				Context: &triggersv1.TriggerContext{
					EventURL:  "https://testing.example.com",
					EventID:   "abcde",
					TriggerID: "namespaces/default/triggers/example-trigger",
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mysecret",
				},
				Data: map[string][]byte{
					"token": []byte(secretToken),
				},
			},
			allowed: false,
			wantErr: true,
		},
		{
			name:               "no context",
			ownersFileReply:    `{"type": "file","encoding": "base64","content": "YXBwcm92ZXJzOg0KLSB0ZXN0X293bmVy"}`,
			collaboratorsReply: `[{"login": "test_owner"}]`,
			interceptorRequest: &triggersv1.InterceptorRequest{
				Body:   `{"action": "opened", "number": 1, "repository":{"full_name": "owner/repo", "clone_url": "https://github.com/owner/repo.git"}, "sender":{"login": "nonowner"}}`,
				Header: map[string][]string{"X-Hub-Signature": {"foo"}, "X-GitHub-Event": {"pull_request"}},
				InterceptorParams: map[string]interface{}{
					"eventTypes": []string{"pull_request", "issue_comment"},
					"githubOwners": &triggersv1.GithubOwners{
						Enabled: true,
						PersonalAccessToken: &triggersv1.SecretRef{
							SecretName: "mysecret",
							SecretKey:  "token",
						},
						CheckType: "orgMembers",
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mysecret",
				},
				Data: map[string][]byte{
					"token": []byte(secretToken),
				},
			},
			allowed: false,
			wantErr: true,
			want:    "error getting github token: no request context passed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if request.URL.Path == "api/v3/repos/owner/repo/issues/comments" {
					writer.Write([]byte(tt.issueCommentReply))
				}
				if request.URL.Path == "/api/v3/repos/owner/repo/contents/OWNERS" {
					if tt.ownersAPICallStatusCode != 0 {
						writer.WriteHeader(tt.ownersAPICallStatusCode)
					} else {
						writer.Write([]byte(tt.ownersFileReply))
					}
				}
				if request.URL.Path == "/api/v3/repos/owner/repo/collaborators" {
					writer.Write([]byte(tt.collaboratorsReply))
				}
				if request.URL.Path == "/api/v3/orgs/owner/public_members" {
					writer.Write([]byte(tt.collaboratorsReply))
				}
			}))
			ctx, _ := test.SetupFakeContext(t)
			ctx = context.WithValue(ctx, testURL, ts.URL)
			clientset := fakekubeclient.Get(ctx)
			if tt.secret != nil {
				tt.secret.Namespace = metav1.NamespaceDefault
				ctx, clientset = fakekubeclient.With(ctx, tt.secret)
			}

			w := &Interceptor{
				SecretGetter: interceptors.DefaultSecretGetter(clientset.CoreV1()),
			}
			res := w.Process(ctx, tt.interceptorRequest)

			if (!res.Continue) && (tt.wantErr == true) {
				t.Logf("Interceptor.Process() = %v, want %v", res.Status.Message, tt.want)
			} else if !res.Continue && (tt.wantErr != true) {
				t.Fatalf("Interceptor.Process() expected res.Continue to be true but got %t. \nStatus.Err(): %v", res.Continue, res.Status.Err())
			}
		})
	}
}
