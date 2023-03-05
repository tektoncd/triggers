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
	"net/http"
	"testing"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"github.com/tektoncd/triggers/test"
	fakekubeclient "knative.dev/pkg/client/injection/kube/client/fake"
)

func TestInterceptor_ExecuteTrigger_ShouldContinue(t *testing.T) {
	tests := []struct {
		name              string
		interceptorParams *triggersv1.SlackInterceptor
		payload           []byte
		header            http.Header
	}{{
		name: "valid case",
		interceptorParams: &triggersv1.SlackInterceptor{
			RequestedFields: []string{"text"},
		},
		payload: []byte("token=EidhofDor5uIpqQ9RrtOVdnC&team_id=T04PK47EDS4&team_domain=demoworkspace-tid8978&channel_id=C04NET94NBH&channel_name=sample-app&user_id=U04NVDEF7R8&user_name=pint12&command=%2Fbuild&text=main+2222&api_app_id=A04NXU23QGL&is_enterprise_install=false&response_url=https%3A%2F%2Fhooks.slack.com%2Fcommands%2FT04PK47EDS4%2F4863712501879%2FdOMNffCDfTjlSskBrmB1bOtR&trigger_id=4890883491553.4801143489888.910b8eaae200b381834de25310583f74"),
		header: http.Header{
			"Content-Type":      []string{"application/x-www-form-urlencoded"},
			"X-Slack-Signature": []string{"1231231231"},
		},
	},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _ := test.SetupFakeContext(t)
			req := &triggersv1.InterceptorRequest{
				Body:   string(tt.payload),
				Header: tt.header,
				InterceptorParams: map[string]interface{}{
					"requestedFields": tt.interceptorParams.RequestedFields,
				},
				Context: &triggersv1.TriggerContext{
					EventURL:  "https://testing.example.com",
					EventID:   "abcde",
					TriggerID: "namespaces/default/triggers/example-trigger",
				},
			}

			clientset := fakekubeclient.Get(ctx)
			w := &Interceptor{
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
		interceptorParams *triggersv1.SlackInterceptor
		payload           []byte
		header            http.Header
	}{{
		name: "bad payload",
		interceptorParams: &triggersv1.SlackInterceptor{
			RequestedFields: []string{"text"},
		},
		payload: []byte("{token: tttt}"),
		header: http.Header{
			"Content-Type":      []string{"application/x-www-form-urlencoded"},
			"X-Slack-Signature": []string{"1231231231"},
		},
	}, {
		name: "skip params - no content type",
		interceptorParams: &triggersv1.SlackInterceptor{
			RequestedFields: []string{"text"},
		},
		payload: []byte("somepayload"),
		header: http.Header{
			"X-Slack-Signature": []string{"1231231231"},
		},
	},
		{
			name: "skip params - no slack signature",
			interceptorParams: &triggersv1.SlackInterceptor{
				RequestedFields: []string{"text"},
			},
			payload: []byte("somepayload"),
			header: http.Header{
				"Content-Type": []string{"application/x-www-form-urlencoded"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _ := test.SetupFakeContext(t)
			req := &triggersv1.InterceptorRequest{
				Body:   string(tt.payload),
				Header: tt.header,
				InterceptorParams: map[string]interface{}{
					"requestedFields": tt.interceptorParams.RequestedFields,
				},
				Context: &triggersv1.TriggerContext{
					EventURL:  "https://testing.example.com",
					EventID:   "abcde",
					TriggerID: "namespaces/default/triggers/example-trigger",
				},
			}
			clientset := fakekubeclient.Get(ctx)
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
