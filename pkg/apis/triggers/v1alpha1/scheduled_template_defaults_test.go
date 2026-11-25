/*
Copyright 2024 The Tekton Authors

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
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/triggers/pkg/apis/config"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
)

func TestScheduledTemplateSetDefaults(t *testing.T) {
	tcs := []struct {
		name   string
		spec   v1alpha1.ScheduledTemplateSpec
		wantSA string
	}{{
		name:   "empty serviceAccountName defaults to default",
		spec:   v1alpha1.ScheduledTemplateSpec{},
		wantSA: config.DefaultServiceAccountValue,
	}, {
		name: "explicit serviceAccountName is preserved",
		spec: v1alpha1.ScheduledTemplateSpec{
			ServiceAccountName: "tekton-triggers-sa",
		},
		wantSA: "tekton-triggers-sa",
	}}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			st := &v1alpha1.ScheduledTemplate{Spec: tc.spec}
			st.SetDefaults(context.Background())
			if d := cmp.Diff(tc.wantSA, st.Spec.ServiceAccountName); d != "" {
				t.Errorf("ServiceAccountName mismatch (-want +got):\n%s", d)
			}
		})
	}
}
