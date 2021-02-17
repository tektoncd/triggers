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

package v1alpha1_test

import (
	"testing"

	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
)

func TestGetName(t *testing.T) {
	for _, tc := range []struct {
		in   v1alpha1.EventInterceptor
		want string
	}{{
		in: v1alpha1.EventInterceptor{
			CEL: &v1alpha1.CELInterceptor{},
		},
		want: "cel",
	}, {
		in: v1alpha1.EventInterceptor{
			GitLab: &v1alpha1.GitLabInterceptor{},
		},
		want: "gitlab",
	}, {
		in: v1alpha1.EventInterceptor{
			GitHub: &v1alpha1.GitHubInterceptor{},
		},
		want: "github",
	}, {
		in: v1alpha1.EventInterceptor{
			Bitbucket: &v1alpha1.BitbucketInterceptor{},
		},
		want: "bitbucket",
	}, {
		in: v1alpha1.EventInterceptor{
			Webhook: &v1alpha1.WebhookInterceptor{},
		},
		want: "",
	}} {
		t.Run(tc.want, func(t *testing.T) {
			got := tc.in.GetName()
			if tc.want != got {
				t.Fatalf("GetName() want: %s; got: %s", tc.want, got)
			}
		})
	}
}
