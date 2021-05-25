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

package v1beta1

import (
	"testing"
)

func TestGetName(t *testing.T) {
	for _, tc := range []struct {
		in   TriggerInterceptor
		want string
	}{{
		in: TriggerInterceptor{
			DeprecatedCEL: &CELInterceptor{},
		},
		want: "cel",
	}, {
		in: TriggerInterceptor{
			DeprecatedGitLab: &GitLabInterceptor{},
		},
		want: "gitlab",
	}, {
		in: TriggerInterceptor{
			DeprecatedGitHub: &GitHubInterceptor{},
		},
		want: "github",
	}, {
		in: TriggerInterceptor{
			DeprecatedBitbucket: &BitbucketInterceptor{},
		},
		want: "bitbucket",
	}, {
		in: TriggerInterceptor{
			Webhook: &WebhookInterceptor{},
		},
		want: "",
	}, {
		in: TriggerInterceptor{
			Ref: InterceptorRef{
				Name: "pluggable-interceptor",
			},
		},
		want: "pluggable-interceptor",
	}} {
		t.Run(tc.want, func(t *testing.T) {
			got := tc.in.GetName()
			if tc.want != got {
				t.Fatalf("GetName() want: %s; got: %s", tc.want, got)
			}
		})
	}
}

func TestUpdateCoreInterceptors_Error(t *testing.T) {
	var ti *TriggerInterceptor

	if err := ti.updateCoreInterceptors(); err != nil {
		t.Fatalf("updateCoreInterceptors() unexpected error: %s", err)
	}
}
