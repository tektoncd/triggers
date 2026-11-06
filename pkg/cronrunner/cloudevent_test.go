/*
Copyright 2026 The Tekton Authors

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

package cronrunner

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"github.com/tektoncd/triggers/pkg/reconciler/events"
	"go.uber.org/zap/zaptest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
)

func TestSendCloudEventToSink(t *testing.T) {
	var (
		mu      sync.Mutex
		gotBody []byte
		gotType string
	)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		gotBody = body
		gotType = r.Header.Get("Ce-Type")
		mu.Unlock()
		w.WriteHeader(http.StatusAccepted)
	}))
	defer ts.Close()

	sink, err := apis.ParseURL(ts.URL)
	if err != nil {
		t.Fatalf("ParseURL: %v", err)
	}
	st := &v1alpha1.ScheduledTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "daily", Namespace: "default"},
		Spec:       v1alpha1.ScheduledTemplateSpec{CloudEventSink: sink},
	}

	sendCloudEventToSink(context.Background(), zaptest.NewLogger(t).Sugar(), sink, st, "exec-1",
		events.ScheduledTemplateSuccessfulV1,
		cloudEventPayload{
			ScheduledTemplate: "daily",
			Namespace:         "default",
			ExecutionID:       "exec-1",
			ResourceCount:     2,
		}, nil)

	mu.Lock()
	defer mu.Unlock()
	if gotType != events.ScheduledTemplateSuccessfulV1 {
		t.Errorf("Ce-Type = %q, want %q", gotType, events.ScheduledTemplateSuccessfulV1)
	}
	var payload cloudEventPayload
	if err := json.Unmarshal(gotBody, &payload); err != nil {
		t.Fatalf("unmarshal body: %v\nbody=%s", err, string(gotBody))
	}
	if payload.ScheduledTemplate != "daily" || payload.ExecutionID != "exec-1" || payload.ResourceCount != 2 {
		t.Errorf("payload = %+v", payload)
	}
}

func TestSendCloudEventToSink_NilSink(t *testing.T) {
	// Should no-op without panicking.
	sendCloudEventToSink(context.Background(), zaptest.NewLogger(t).Sugar(), nil,
		&v1alpha1.ScheduledTemplate{ObjectMeta: metav1.ObjectMeta{Name: "x", Namespace: "ns"}},
		"id", events.ScheduledTemplateFailedV1, cloudEventPayload{}, nil)
}

func TestSendCloudEventToSink_StalledSinkTimesOut(t *testing.T) {
	prev := cloudEventDeliveryTimeout
	cloudEventDeliveryTimeout = 200 * time.Millisecond
	t.Cleanup(func() { cloudEventDeliveryTimeout = prev })

	started := make(chan struct{})
	release := make(chan struct{})
	handlerFinished := make(chan struct{})
	// Accept the request but never write response headers until the test releases.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		close(handlerFinished)
	}))
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
		ts.CloseClientConnections()
		ts.Close()
	})

	sink, err := apis.ParseURL(ts.URL)
	if err != nil {
		t.Fatalf("ParseURL: %v", err)
	}
	st := &v1alpha1.ScheduledTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "daily", Namespace: "default"},
		Spec:       v1alpha1.ScheduledTemplateSpec{CloudEventSink: sink},
	}

	done := make(chan struct{})
	go func() {
		sendCloudEventToSink(context.Background(), zaptest.NewLogger(t).Sugar(), sink, st, "exec-timeout",
			events.ScheduledTemplateSuccessfulV1,
			cloudEventPayload{ScheduledTemplate: "daily", Namespace: "default", ExecutionID: "exec-timeout"},
			nil)
		close(done)
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("stalled sink never received the request")
	}

	// Must return while the handler is still blocked on release.
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("sendCloudEventToSink did not return before the stalled sink was released")
	}
	select {
	case <-handlerFinished:
		t.Fatal("handler finished before release; delivery timeout was not what unblocked the client")
	default:
	}
}
