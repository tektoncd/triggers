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

package v1alpha1_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
)

func Test_ClusterTriggerBindingValidate_OnDelete(t *testing.T) {
	tb := &v1alpha1.ClusterTriggerBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: strings.Repeat("foo", 64), // Length should be lower than 63
		},
		Spec: v1alpha1.TriggerBindingSpec{
			Params: []v1alpha1.Param{{
				Name:  "param1",
				Value: "$(body.input1)",
			}, {
				Name:  "param2",
				Value: "$(body.input2)",
			}},
		},
	}
	err := tb.Validate(apis.WithinDelete(context.Background()))
	if err != nil {
		t.Errorf("TriggerBinding.Validate() on Delete expected no error, but got one, TriggerBinding: %v, error: %v", tb, err)
	}
}

func Test_ClusterTriggerBindingValidate(t *testing.T) {
	tests := []struct {
		name string
		tb   *v1alpha1.ClusterTriggerBinding
	}{{
		name: "multiple params",
		tb: &v1alpha1.ClusterTriggerBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "name",
			},
			Spec: v1alpha1.TriggerBindingSpec{
				Params: []v1alpha1.Param{{
					Name:  "param1",
					Value: "$(body.input1)",
				}, {
					Name:  "param2",
					Value: "$(body.input2)",
				}, {
					Name:  "param3",
					Value: "$(body.input3)",
				}},
			},
		},
	}, {
		name: "multiple params case sensitive",
		tb: &v1alpha1.ClusterTriggerBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "name",
			},
			Spec: v1alpha1.TriggerBindingSpec{
				Params: []v1alpha1.Param{{
					Name:  "param1",
					Value: "$(body.input1)",
				}, {
					Name:  "PARAM1",
					Value: "$(body.input2)",
				}, {
					Name:  "param3",
					Value: "$(body.input3)",
				}},
			},
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.tb.Validate(context.Background()); err != nil {
				t.Errorf("ClusterTriggerBinding.Validate() returned error: %s", err)
			}
		})
	}
}

func Test_ClusterTriggerBindingValidate_error(t *testing.T) {
	tests := []struct {
		name string
		tb   *v1alpha1.ClusterTriggerBinding
	}{{
		name: "empty",
		tb: &v1alpha1.ClusterTriggerBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "name",
			},
		},
	}, {
		name: "duplicate params",
		tb: &v1alpha1.ClusterTriggerBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "name",
			},
			Spec: v1alpha1.TriggerBindingSpec{
				Params: []v1alpha1.Param{{
					Name:  "param1",
					Value: "$(body.input1)",
				}, {
					Name:  "param1",
					Value: "$(body.input2)",
				}, {
					Name:  "param3",
					Value: "$(body.input3)",
				}},
			},
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.tb.Validate(context.Background()); err == nil {
				t.Errorf("ClusterTriggerBinding.Validate() expected error for ClusterTriggerBinding: %v", tt.tb)
			}
		})
	}
}
