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
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/tektoncd/pipeline/pkg/logging"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
)

type fakeSecretStore struct {
	text string
}

func (s fakeSecretStore) Get(sr triggersv1.SecretRef) ([]byte, error) {
	if s.text == "" {
		return []byte(""), errors.New("Not found")
	}

	return []byte(s.text), nil
}

func TestInterceptor_ExecuteTrigger(t *testing.T) {
	type args struct {
		payload   []byte
		secret    string
		token     string
		eventType string
	}
	tests := []struct {
		name    string
		GitLab  *triggersv1.GitLabInterceptor
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name:   "no secret",
			GitLab: &triggersv1.GitLabInterceptor{},
			args: args{
				payload: []byte("somepayload"),
				token:   "foo",
			},
			want:    []byte("somepayload"),
			wantErr: false,
		},
		{
			name: "invalid header for secret",
			GitLab: &triggersv1.GitLabInterceptor{
				SecretRef: &triggersv1.SecretRef{
					SecretName: "mysecret",
					SecretKey:  "token",
				},
			},
			args: args{
				token:   "foo",
				secret:  "secrettoken",
				payload: []byte("somepayload"),
			},
			wantErr: true,
		},
		{
			name: "valid header for secret",
			GitLab: &triggersv1.GitLabInterceptor{
				SecretRef: &triggersv1.SecretRef{
					SecretName: "mysecret",
					SecretKey:  "token",
				},
			},
			args: args{
				token:   "secret",
				secret:  "secret",
				payload: []byte("somepayload"),
			},
			wantErr: false,
			want:    []byte("somepayload"),
		},
		{
			name: "valid event",
			GitLab: &triggersv1.GitLabInterceptor{
				EventTypes: []string{"foo", "bar"},
			},
			args: args{
				eventType: "foo",
				payload:   []byte("somepayload"),
			},
			wantErr: false,
			want:    []byte("somepayload"),
		},
		{
			name: "invalid event",
			GitLab: &triggersv1.GitLabInterceptor{
				EventTypes: []string{"foo", "bar"},
			},
			args: args{
				eventType: "baz",
				payload:   []byte("somepayload"),
			},
			wantErr: true,
		},
		{
			name: "valid event, invalid secret",
			GitLab: &triggersv1.GitLabInterceptor{
				EventTypes: []string{"foo", "bar"},
				SecretRef: &triggersv1.SecretRef{
					SecretName: "mysecret",
					SecretKey:  "token",
				},
			},
			args: args{
				eventType: "bar",
				payload:   []byte("somepayload"),
				token:     "foo",
				secret:    "mysecret",
			},
			wantErr: true,
		},
		{
			name: "invalid event, valid secret",
			GitLab: &triggersv1.GitLabInterceptor{
				EventTypes: []string{"foo", "bar"},
				SecretRef: &triggersv1.SecretRef{
					SecretName: "mysecret",
					SecretKey:  "token",
				},
			},
			args: args{
				eventType: "baz",
				payload:   []byte("somepayload"),
				token:     "secrettoken",
				secret:    "secrettoken",
			},
			wantErr: true,
		},
		{
			name: "valid event, valid secret",
			GitLab: &triggersv1.GitLabInterceptor{
				EventTypes: []string{"foo", "bar"},
				SecretRef: &triggersv1.SecretRef{
					SecretName: "mysecret",
					SecretKey:  "token",
				},
			},
			args: args{
				eventType: "bar",
				payload:   []byte("somepayload"),
				token:     "secrettoken",
				secret:    "secrettoken",
			},
			want: []byte("somepayload"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := logging.NewLogger("", "")
			secretStore := fakeSecretStore{text: tt.args.secret}
			request := &http.Request{
				Body: ioutil.NopCloser(bytes.NewReader(tt.args.payload)),
				Header: http.Header{
					"Content-Type": []string{"application/json"},
				},
			}
			if tt.args.token != "" {
				request.Header.Add("X-GitLab-Token", tt.args.token)
			}
			if tt.args.eventType != "" {
				request.Header.Add("X-GitLab-Event", tt.args.eventType)
			}
			w := &Interceptor{
				secretStore: secretStore,
				GitLab:      tt.GitLab,
				Logger:      logger,
			}
			resp, err := w.ExecuteTrigger(request)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("Interceptor.ExecuteTrigger() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			got, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("error reading response: %v", err)
			}
			defer resp.Body.Close()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Interceptor.ExecuteTrigger (-want, +got) = %s", diff)
			}
		})
	}
}
