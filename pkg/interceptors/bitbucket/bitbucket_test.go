/*
Copyright 2020 The Tekton Authors

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

package bitbucket

import (
	"bytes"
	"errors"
	"io"
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

func TestInterceptor_ExecuteTrigger_Signature(t *testing.T) {
	type args struct {
		payload   io.ReadCloser
		secret    string
		signature string
		eventType string
	}
	tests := []struct {
		name      string
		Bitbucket *triggersv1.BitbucketInterceptor
		args      args
		want      []byte
		wantErr   bool
	}{
		{
			name:      "no secret",
			Bitbucket: &triggersv1.BitbucketInterceptor{},
			args: args{
				payload:   ioutil.NopCloser(bytes.NewBufferString("somepayload")),
				signature: "foo",
			},
			want:    []byte("somepayload"),
			wantErr: false,
		},
		{
			name: "invalid header for secret",
			Bitbucket: &triggersv1.BitbucketInterceptor{
				SecretRef: &triggersv1.SecretRef{
					SecretName: "mysecret",
					SecretKey:  "token",
				},
			},
			args: args{
				signature: "foo",
				secret:    "secrettoken",
				payload:   ioutil.NopCloser(bytes.NewBufferString("somepayload")),
			},
			wantErr: true,
		},
		{
			name: "valid header for secret",
			Bitbucket: &triggersv1.BitbucketInterceptor{
				SecretRef: &triggersv1.SecretRef{
					SecretName: "mysecret",
					SecretKey:  "token",
				},
			},
			args: args{
				// This was generated by using SHA1 and hmac from go stdlib on secret and payload.
				// https://play.golang.org/p/otp1o_cJTd7 for a sample.
				signature: "sha1=38e005ef7dd3faee13204505532011257023654e",
				secret:    "secret",
				payload:   ioutil.NopCloser(bytes.NewBufferString("somepayload")),
			},
			wantErr: false,
			want:    []byte("somepayload"),
		},
		{
			name: "matching event",
			Bitbucket: &triggersv1.BitbucketInterceptor{
				EventTypes: []string{"pr:opened", "repo:refs_changed"},
			},
			args: args{
				payload:   ioutil.NopCloser(bytes.NewBufferString("somepayload")),
				eventType: "repo:refs_changed",
			},
			wantErr: false,
			want:    []byte("somepayload"),
		},
		{
			name: "no matching event",
			Bitbucket: &triggersv1.BitbucketInterceptor{
				EventTypes: []string{"pr:opened", "repo:refs_changed"},
			},
			args: args{
				payload:   ioutil.NopCloser(bytes.NewBufferString("somepayload")),
				eventType: "event",
			},
			wantErr: true,
		},
		{
			name: "valid header for secret and matching event",
			Bitbucket: &triggersv1.BitbucketInterceptor{
				SecretRef: &triggersv1.SecretRef{
					SecretName: "mysecret",
					SecretKey:  "token",
				},
				EventTypes: []string{"pr:opened", "repo:refs_changed"},
			},
			args: args{
				// This was generated by using SHA1 and hmac from go stdlib on secret and payload.
				// https://play.golang.org/p/otp1o_cJTd7 for a sample.
				signature: "sha1=38e005ef7dd3faee13204505532011257023654e",
				secret:    "secret",
				eventType: "repo:refs_changed",
				payload:   ioutil.NopCloser(bytes.NewBufferString("somepayload")),
			},
			wantErr: false,
			want:    []byte("somepayload"),
		},
		{
			name: "valid header for secret, but no matching event",
			Bitbucket: &triggersv1.BitbucketInterceptor{
				SecretRef: &triggersv1.SecretRef{
					SecretName: "mysecret",
					SecretKey:  "token",
				},
				EventTypes: []string{"pr:opened", "repo:refs_changed"},
			},
			args: args{
				// This was generated by using SHA1 and hmac from go stdlib on secret and payload.
				// https://play.golang.org/p/otp1o_cJTd7 for a sample.
				signature: "sha1=38e005ef7dd3faee13204505532011257023654e",
				secret:    "secret",
				eventType: "event",
				payload:   ioutil.NopCloser(bytes.NewBufferString("somepayload")),
			},
			wantErr: true,
		},
		{
			name: "invalid header for secret, but matching event",
			Bitbucket: &triggersv1.BitbucketInterceptor{
				SecretRef: &triggersv1.SecretRef{
					SecretName: "mysecret",
					SecretKey:  "token",
				},
				EventTypes: []string{"pr:opened", "repo:refs_changed"},
			},
			args: args{
				signature: "foo",
				secret:    "secret",
				eventType: "pr:opened",
				payload:   ioutil.NopCloser(bytes.NewBufferString("somepayload")),
			},
			wantErr: true,
		}, {
			name:      "nil body does not panic",
			Bitbucket: &triggersv1.BitbucketInterceptor{},
			args: args{
				payload:   nil,
				signature: "foo",
			},
			want:    []byte{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := logging.NewLogger("", "")
			secretStore := fakeSecretStore{text: tt.args.secret}
			request := &http.Request{
				Body: tt.args.payload,
				Header: http.Header{
					"Content-Type": []string{"application/json"},
				},
			}
			if tt.args.eventType != "" {
				request.Header.Add("X-Event-Key", tt.args.eventType)
			}
			if tt.args.signature != "" {
				request.Header.Add("X-Hub-Signature", tt.args.signature)
			}
			w := &Interceptor{
				secretStore: secretStore,
				Bitbucket:   tt.Bitbucket,
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
				t.Fatalf("error reading response body %v", err)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Interceptor.ExecuteTrigger (-want, +got) = %s", diff)
			}
		})
	}
}
