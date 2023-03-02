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

package v1beta1_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_TriggerBindingValidate(t *testing.T) {
	tests := []struct {
		name string
		tb   *v1beta1.TriggerBinding
	}{{
		name: "multiple params",
		tb: &v1beta1.TriggerBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.TriggerBindingSpec{
				Params: []v1beta1.Param{{
					Name:  "param1",
					Value: "$(body.input1)",
				}, {
					Name:  "param2",
					Value: "$(body.input2)",
				}, {
					Name:  "param3",
					Value: "$(body.(input3))",
				}, {
					Name:  "param4",
					Value: "static-input",
				}},
			},
		},
	}, {
		name: "multiple params case sensitive",
		tb: &v1beta1.TriggerBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.TriggerBindingSpec{
				Params: []v1beta1.Param{{
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
	}, {
		name: "multiple expressions in one body",
		tb: &v1beta1.TriggerBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.TriggerBindingSpec{
				Params: []v1beta1.Param{{
					Name:  "param1",
					Value: "$(body.input1)-$(body.input2)",
				}},
			},
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.tb.Validate(context.Background()); err != nil {
				t.Errorf("TriggerBinding.Validate() returned error: %s", err)
			}
		})
	}
}

func Test_TriggerBindingValidate_error(t *testing.T) {
	tests := []struct {
		name   string
		tb     *v1beta1.TriggerBinding
		errMsg string
	}{{
		name: "empty",
		tb: &v1beta1.TriggerBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
		},
		errMsg: "missing field(s): spec",
	}, {
		name: "duplicate params",
		tb: &v1beta1.TriggerBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.TriggerBindingSpec{
				Params: []v1beta1.Param{{
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
		errMsg: "expected exactly one, got both: spec.params[1].name",
	}, {
		name: "invalid parameter",
		tb: &v1beta1.TriggerBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.TriggerBindingSpec{
				Params: []v1beta1.Param{{
					Name:  "param1",
					Value: "$($(body.param1))",
				},
				},
			},
		},
		errMsg: "invalid value: $($(body.param1)): spec.params[0].value",
	}, {
		name: "invalid parameter further nested",
		tb: &v1beta1.TriggerBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.TriggerBindingSpec{
				Params: []v1beta1.Param{{
					Name:  "param1",
					Value: "$(body.test-$(body.param1))",
				},
				},
			},
		},
		errMsg: "invalid value: $(body.test-$(body.param1)): spec.params[0].value",
	}, {
		name: "invalid parameter triple nested",
		tb: &v1beta1.TriggerBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: v1beta1.TriggerBindingSpec{
				Params: []v1beta1.Param{{
					Name:  "param1",
					Value: "$($($(body.param1)))",
				},
				},
			},
		},
		errMsg: "invalid value: $($($(body.param1))): spec.params[0].value",
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tb.Validate(context.Background())
			if err == nil {
				t.Errorf("TriggerBinding.Validate() expected error for TriggerBinding: %v", tt.tb)
			}
			if diff := cmp.Diff(tt.errMsg, err.Error()); diff != "" {
				t.Errorf("-want +got: %s", diff)
			}
		})
	}
}
