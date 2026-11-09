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

package slack

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"github.com/tektoncd/triggers/test"
	"google.golang.org/grpc/codes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	fakekubeclient "knative.dev/pkg/client/injection/kube/client/fake"
)

const signingSecret = "slack-signing-secret"

func TestInterceptor_Process_ValidSignature(t *testing.T) {
	ctx, clientset := testContextWithSigningSecret(t)
	rawBody := "team_id=T123&text=main+2222"
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	res := NewInterceptor(interceptors.DefaultSecretGetter(clientset.CoreV1())).Process(ctx, request(rawBody, timestamp))
	if !res.Continue {
		t.Fatalf("Interceptor.Process() expected res.Continue to be true but got %t. Status.Err(): %v", res.Continue, res.Status.Err())
	}
	if got := res.Extensions["text"]; fmt.Sprint(got) != "[main 2222]" {
		t.Errorf("text extension: got %v, want [main 2222]", got)
	}
}

func TestInterceptor_Process_InvalidRequest(t *testing.T) {
	rawBody := "team_id=T123&text=main+2222"
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	tests := []struct {
		name     string
		mutate   func(*triggersv1.InterceptorRequest)
		wantCode codes.Code
	}{
		{"wrong content type", func(r *triggersv1.InterceptorRequest) { http.Header(r.Header).Set("Content-Type", "application/json") }, codes.InvalidArgument},
		{"missing secretRef", func(r *triggersv1.InterceptorRequest) { r.InterceptorParams["secretRef"] = nil }, codes.FailedPrecondition},
		{"empty secret name", func(r *triggersv1.InterceptorRequest) {
			r.InterceptorParams["secretRef"] = &triggersv1.SecretRef{SecretKey: "signingSecret"}
		}, codes.FailedPrecondition},
		{"empty secret key", func(r *triggersv1.InterceptorRequest) {
			r.InterceptorParams["secretRef"] = &triggersv1.SecretRef{SecretName: "slack-secret"}
		}, codes.FailedPrecondition},
		{"missing signature", func(r *triggersv1.InterceptorRequest) { http.Header(r.Header).Del("X-Slack-Signature") }, codes.FailedPrecondition},
		{"invalid signature", func(r *triggersv1.InterceptorRequest) {
			http.Header(r.Header).Set("X-Slack-Signature", "v0=not-a-valid-signature")
		}, codes.FailedPrecondition},
		{"unsupported signature version", func(r *triggersv1.InterceptorRequest) {
			http.Header(r.Header).Set("X-Slack-Signature", "sha1=not-a-valid-signature")
		}, codes.FailedPrecondition},
		{"missing timestamp", func(r *triggersv1.InterceptorRequest) { http.Header(r.Header).Del("X-Slack-Request-Timestamp") }, codes.FailedPrecondition},
		{"invalid timestamp", func(r *triggersv1.InterceptorRequest) {
			http.Header(r.Header).Set("X-Slack-Request-Timestamp", "not-a-timestamp")
		}, codes.FailedPrecondition},
		{"stale timestamp", func(r *triggersv1.InterceptorRequest) {
			staleTimestamp := strconv.FormatInt(time.Now().Add(-maxRequestAge-time.Second).Unix(), 10)
			http.Header(r.Header).Set("X-Slack-Request-Timestamp", staleTimestamp)
			http.Header(r.Header).Set("X-Slack-Signature", slackSignature(signingSecret, staleTimestamp, rawBody))
		}, codes.FailedPrecondition},
		{"timestamp too far in the future", func(r *triggersv1.InterceptorRequest) {
			futureTimestamp := strconv.FormatInt(time.Now().Add(maxRequestAge+time.Second).Unix(), 10)
			http.Header(r.Header).Set("X-Slack-Request-Timestamp", futureTimestamp)
			http.Header(r.Header).Set("X-Slack-Signature", slackSignature(signingSecret, futureTimestamp, rawBody))
		}, codes.FailedPrecondition},
		{"missing request context", func(r *triggersv1.InterceptorRequest) { r.Context = nil }, codes.InvalidArgument},
		{"secret lookup failure", func(r *triggersv1.InterceptorRequest) {
			r.InterceptorParams["secretRef"] = &triggersv1.SecretRef{SecretName: "missing-secret", SecretKey: "signingSecret"}
		}, codes.FailedPrecondition},
		{"empty raw body", func(r *triggersv1.InterceptorRequest) { r.RawBody = "" }, codes.InvalidArgument},
		{"empty requested fields", func(r *triggersv1.InterceptorRequest) { r.InterceptorParams["requestedFields"] = []string{} }, codes.InvalidArgument},
		{"requested field missing from payload", func(r *triggersv1.InterceptorRequest) { r.InterceptorParams["requestedFields"] = []string{"missing"} }, codes.NotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, clientset := testContextWithSigningSecret(t)
			req := request(rawBody, timestamp)
			tt.mutate(req)

			res := NewInterceptor(interceptors.DefaultSecretGetter(clientset.CoreV1())).Process(ctx, req)
			if res.Continue {
				t.Fatalf("Interceptor.Process() expected res.Continue to be false; status: %v", res.Status.Err())
			}
			if res.Status.Code != tt.wantCode {
				t.Errorf("Interceptor.Process() status code: got %v, want %v; status: %v", res.Status.Code, tt.wantCode, res.Status.Err())
			}
		})
	}
}

func request(rawBody, timestamp string) *triggersv1.InterceptorRequest {
	return &triggersv1.InterceptorRequest{
		RawBody: rawBody,
		Header: http.Header{
			"Content-Type":              {"application/x-www-form-urlencoded"},
			"X-Slack-Request-Timestamp": {timestamp},
			"X-Slack-Signature":         {slackSignature(signingSecret, timestamp, rawBody)},
		},
		InterceptorParams: map[string]interface{}{
			"requestedFields": []string{"text"},
			"secretRef":       &triggersv1.SecretRef{SecretName: "slack-secret", SecretKey: "signingSecret"},
		},
		Context: &triggersv1.TriggerContext{EventURL: "https://testing.example.com", EventID: "abcde", TriggerID: "namespaces/default/triggers/example-trigger"},
	}
}

func testContextWithSigningSecret(t *testing.T) (context.Context, kubernetes.Interface) {
	t.Helper()
	ctx, _ := test.SetupFakeContext(t)
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "slack-secret", Namespace: metav1.NamespaceDefault}, Data: map[string][]byte{"signingSecret": []byte(signingSecret)}}
	ctx, clientset := fakekubeclient.With(ctx, secret)
	return ctx, clientset
}

func slackSignature(secret, timestamp, body string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(fmt.Sprintf("%s:%s:%s", signingVersion, timestamp, body)))
	return fmt.Sprintf("%s=%x", signingVersion, mac.Sum(nil))
}
