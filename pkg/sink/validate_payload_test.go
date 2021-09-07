/*
Copyright 2021 The Tekton Authors

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

package sink

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"github.com/tektoncd/triggers/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
)

func TestSink_IsValidPayload(t *testing.T) {
	const defaultELName = "test-el"
	for _, tc := range []struct {
		name           string
		testResources  test.Resources
		eventBody      []byte
		wantStatusCode int
	}{{
		name: "event with Json Body",
		testResources: test.Resources{
			EventListeners: []*triggersv1.EventListener{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultELName,
					Namespace: namespace,
				},
				Spec: triggersv1.EventListenerSpec{
					Triggers: []triggersv1.EventListenerTrigger{{
						TriggerRef: "test",
					}},
				},
			}},
		},
		eventBody:      json.RawMessage(`{"head_commit": {"id": "testrevision"}, "repository": {"url": "testurl"}}`),
		wantStatusCode: http.StatusAccepted,
	}, {
		name: "event with non Json Body",
		testResources: test.Resources{
			EventListeners: []*triggersv1.EventListener{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultELName,
					Namespace: namespace,
				},
				Spec: triggersv1.EventListenerSpec{
					Triggers: []triggersv1.EventListenerTrigger{{
						TriggerRef: "test",
					}},
				},
			}},
		},
		eventBody:      []byte(`<test>xml</test>`),
		wantStatusCode: http.StatusBadRequest,
	}} {
		t.Run(tc.name, func(t *testing.T) {
			elName := defaultELName
			if len(tc.testResources.EventListeners) > 0 {
				elName = tc.testResources.EventListeners[0].Name
			}
			sink, _ := getSinkAssets(t, tc.testResources, elName, nil)

			for _, el := range tc.testResources.EventListeners {
				el.Status.SetCondition(&apis.Condition{
					Type:    apis.ConditionReady,
					Status:  corev1.ConditionTrue,
					Message: "EventListener is Ready",
				})
			}

			ts := httptest.NewServer(sink.IsValidPayload(http.HandlerFunc(sink.HandleEvent)))
			defer ts.Close()

			resp, err := http.Post(ts.URL, "application/json", bytes.NewReader(tc.eventBody))
			if err != nil {
				t.Fatalf("error making request to eventListener: %s", err)
			}
			if resp.StatusCode != tc.wantStatusCode {
				t.Fatalf("Status code mismatch: got %d, want %d", resp.StatusCode, http.StatusInternalServerError)
			}
		})
	}
}

func TestSink_IsValidPayload_PayloadValidation(t *testing.T) {
	const defaultELName = "test-el"
	for _, tc := range []struct {
		name           string
		testResources  test.Resources
		eventBody      []byte
		wantStatusCode int
	}{{
		name: "event with Json Body",
		testResources: test.Resources{
			EventListeners: []*triggersv1.EventListener{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultELName,
					Namespace: namespace,
				},
				Spec: triggersv1.EventListenerSpec{
					Triggers: []triggersv1.EventListenerTrigger{{
						TriggerRef: "test",
					}},
				},
			}},
		},
		eventBody:      json.RawMessage(`{"head_commit": {"id": "testrevision"}, "repository": {"url": "testurl"}}`),
		wantStatusCode: http.StatusAccepted,
	}, {
		name: "event with non Json Body",
		testResources: test.Resources{
			EventListeners: []*triggersv1.EventListener{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultELName,
					Namespace: namespace,
				},
				Spec: triggersv1.EventListenerSpec{
					Triggers: []triggersv1.EventListenerTrigger{{
						TriggerRef: "test",
					}},
				},
			}},
		},
		eventBody:      []byte(`<test>xml</test>`),
		wantStatusCode: http.StatusAccepted,
	}} {
		t.Run(tc.name, func(t *testing.T) {
			elName := defaultELName
			if len(tc.testResources.EventListeners) > 0 {
				elName = tc.testResources.EventListeners[0].Name
			}
			sink, _ := getSinkAssets(t, tc.testResources, elName, nil)

			for _, el := range tc.testResources.EventListeners {
				el.Status.SetCondition(&apis.Condition{
					Type:    apis.ConditionReady,
					Status:  corev1.ConditionTrue,
					Message: "EventListener is Ready",
				})
			}
			// Disabling payload validation
			sink.PayloadValidation = false

			ts := httptest.NewServer(sink.IsValidPayload(http.HandlerFunc(sink.HandleEvent)))
			defer ts.Close()

			resp, err := http.Post(ts.URL, "application/json", bytes.NewReader(tc.eventBody))
			if err != nil {
				t.Fatalf("error making request to eventListener: %s", err)
			}
			if resp.StatusCode != tc.wantStatusCode {
				t.Fatalf("Status code mismatch: got %d, want %d", resp.StatusCode, http.StatusInternalServerError)
			}
		})
	}
}
