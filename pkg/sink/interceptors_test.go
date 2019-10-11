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

package sink

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	corev1 "k8s.io/api/core/v1"
)

func TestGetURI(t *testing.T) {
	var eventListenerNs = "default"
	tcs := []struct {
		name     string
		ref      corev1.ObjectReference
		expected string
		wantErr  bool
	}{{
		name: "namespace specified",
		ref: corev1.ObjectReference{
			Kind:       "Service",
			Name:       "foo",
			APIVersion: "v1",
			Namespace:  "bar",
		},
		expected: "http://foo.bar.svc/",
		wantErr:  false,
	}, {
		name: "no namespace",
		ref: corev1.ObjectReference{
			Kind:       "Service",
			Name:       "foo",
			APIVersion: "v1",
		},
		expected: "http://foo.default.svc/",
		wantErr:  false,
	}, {
		name: "non services",
		ref: corev1.ObjectReference{
			Kind:       "Blah",
			Name:       "foo",
			APIVersion: "v1",
		},
		expected: "",
		wantErr:  true,
	}}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			url, err := GetURI(&tc.ref, eventListenerNs)
			if err != nil {
				if !tc.wantErr {
					t.Errorf("Unexpected error: %v", err)
				}
			} else if diff := cmp.Diff(tc.expected, url.String()); diff != "" {
				t.Errorf("Did not get expected URL: %s", diff)
			}
		})
	}
}

func TestCreateOutgoingRequest(t *testing.T) {
	reqBody, _ := json.Marshal(map[string]string{
		"eventType": "push",
		"foo":       "bar",
	})

	req, _ := http.NewRequest(http.MethodPost, "http://event.listener.url", ioutil.NopCloser(bytes.NewBuffer(reqBody)))
	req.Header.Add("Content-type", "application/json")
	req.Header.Add("X-Event-Id", "blah")
	eventProcessorURL, _ := url.Parse("http://some.other.url")

	expectedReq, _ := http.NewRequest(http.MethodPost, "http://some.other.url", ioutil.NopCloser(bytes.NewBuffer(reqBody)))
	expectedReq.Header.Add("Content-type", "application/json")
	expectedReq.Header.Add("X-Event-Id", "blah")

	outgoing := createOutgoingRequest(context.Background(), req, eventProcessorURL, reqBody)

	respBody, err := ioutil.ReadAll(outgoing.Body)
	if err != nil {
		t.Errorf("Failed to parse outgoing request body: %q", err)
	}

	if outgoing.URL != eventProcessorURL {
		t.Errorf("Outgoing request has unexpected URL: %s", outgoing.URL)
	}

	if diff := cmp.Diff(expectedReq.Header, outgoing.Header); diff != "" {
		t.Errorf("Did not create request with expected headers: %s", diff)
	}

	if diff := cmp.Diff(reqBody, respBody, cmpopts.IgnoreUnexported()); diff != "" {
		t.Errorf("Did not create request with expected body: %s", diff)
	}
}

func TestMakeRequest(t *testing.T) {
	reqBody, _ := json.Marshal(map[string]string{
		"eventType": "push",
		"foo":       "bar",
	})

	tcs := []struct {
		name            string
		handler         http.HandlerFunc
		expectedPayload []byte
		wantErr         bool
	}{{
		name: "status 200",
		handler: func(w http.ResponseWriter, r *http.Request) {
			p, _ := ioutil.ReadAll(r.Body)
			_, _ = w.Write(p)
		},
		expectedPayload: reqBody,
		wantErr:         false,
	}, {
		name: "status 400",
		handler: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		},
		wantErr: true,
	}}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(tc.handler))
			defer ts.Close()

			req, err := http.NewRequest(http.MethodPost, ts.URL, bytes.NewBuffer(reqBody))
			if err != nil {
				t.Fatalf("Error trying to create request: %q", err)
			}
			payload, err := makeRequest(http.DefaultClient, req)
			expectedPayload := tc.expectedPayload
			if err != nil {
				if !tc.wantErr {
					t.Errorf("Unexpected error from make request: %s", err)
				}
			}
			if diff := cmp.Diff(expectedPayload, payload); diff != "" {
				t.Errorf("Did not get expected body back: %s", diff)
			}
		})
	}
}
