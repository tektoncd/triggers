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

package template

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"github.com/tektoncd/triggers/test"
	bldr "github.com/tektoncd/triggers/test/builder"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/ptr"
)

func Test_MergeInDefaultParams(t *testing.T) {
	var (
		oneDefault   = "onedefault"
		twoDefault   = "twodefault"
		threeDefault = "threedefault"
		oneParam     = triggersv1.Param{
			Name:  "oneid",
			Value: "onevalue",
		}
		oneParamSpec = triggersv1.ParamSpec{
			Name:    "oneid",
			Default: &oneDefault,
		}
		wantDefaultOneParam = triggersv1.Param{
			Name:  "oneid",
			Value: "onedefault",
		}
		twoParamSpec = triggersv1.ParamSpec{
			Name:    "twoid",
			Default: &twoDefault,
		}
		wantDefaultTwoParam = triggersv1.Param{
			Name:  "twoid",
			Value: "twodefault",
		}
		threeParamSpec = triggersv1.ParamSpec{
			Name:    "threeid",
			Default: &threeDefault,
		}
		wantDefaultThreeParam = triggersv1.Param{
			Name:  "threeid",
			Value: "threedefault",
		}
		noDefaultParamSpec = triggersv1.ParamSpec{
			Name: "nodefault",
		}
	)
	type args struct {
		params     []triggersv1.Param
		paramSpecs []triggersv1.ParamSpec
	}
	tests := []struct {
		name string
		args args
		want []triggersv1.Param
	}{
		{
			name: "add one default param",
			args: args{
				params:     []triggersv1.Param{},
				paramSpecs: []triggersv1.ParamSpec{oneParamSpec},
			},
			want: []triggersv1.Param{wantDefaultOneParam},
		},
		{
			name: "add multiple default params",
			args: args{
				params:     []triggersv1.Param{},
				paramSpecs: []triggersv1.ParamSpec{oneParamSpec, twoParamSpec, threeParamSpec},
			},
			want: []triggersv1.Param{wantDefaultOneParam, wantDefaultTwoParam, wantDefaultThreeParam},
		},
		{
			name: "do not override existing value",
			args: args{
				params:     []triggersv1.Param{oneParam},
				paramSpecs: []triggersv1.ParamSpec{oneParamSpec},
			},
			want: []triggersv1.Param{oneParam},
		},
		{
			name: "add no default params",
			args: args{
				params:     []triggersv1.Param{},
				paramSpecs: []triggersv1.ParamSpec{noDefaultParamSpec},
			},
			want: []triggersv1.Param{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeInDefaultParams(tt.args.params, tt.args.paramSpecs)
			if diff := cmp.Diff(tt.want, got, cmpopts.SortSlices(test.CompareParams)); diff != "" {
				t.Errorf("mergeInDefaultParams(): -want +got: %s", diff)
			}
		})
	}
}

func Test_applyParamToResourceTemplate(t *testing.T) {
	var (
		oneParam = triggersv1.Param{
			Name:  "oneid",
			Value: "onevalue",
		}
		rtNoParamVars             = json.RawMessage(`{"foo": "bar"}`)
		wantRtNoParamVars         = json.RawMessage(`{"foo": "bar"}`)
		rtNoMatchingParamVars     = json.RawMessage(`{"foo": "$(tt.params.no.matching.path)"}`)
		wantRtNoMatchingParamVars = json.RawMessage(`{"foo": "$(tt.params.no.matching.path)"}`)
		rtOneParamVar             = json.RawMessage(`{"foo": "bar-$(tt.params.oneid)-bar"}`)
		wantRtOneParamVar         = json.RawMessage(`{"foo": "bar-onevalue-bar"}`)
		rtMultipleParamVars       = json.RawMessage(`{"$(tt.params.oneid)": "bar-$(tt.params.oneid)-$(tt.params.oneid)$(tt.params.oneid)$(tt.params.oneid)-$(tt.params.oneid)-bar"}`)
		wantRtMultipleParamVars   = json.RawMessage(`{"onevalue": "bar-onevalue-onevalueonevalueonevalue-onevalue-bar"}`)
	)
	type args struct {
		param triggersv1.Param
		rt    json.RawMessage
	}
	tests := []struct {
		name string
		args args
		want json.RawMessage
	}{
		{
			name: "replace no param vars",
			args: args{
				param: oneParam,
				rt:    rtNoParamVars,
			},
			want: wantRtNoParamVars,
		},
		{
			name: "replace no param vars with non match present",
			args: args{
				param: oneParam,
				rt:    rtNoMatchingParamVars,
			},
			want: wantRtNoMatchingParamVars,
		},
		{
			name: "replace one param var",
			args: args{
				param: oneParam,
				rt:    rtOneParamVar,
			},
			want: wantRtOneParamVar,
		},
		{
			name: "replace multiple param vars",
			args: args{
				param: oneParam,
				rt:    rtMultipleParamVars,
			},
			want: wantRtMultipleParamVars,
		}, {
			name: "espcae quotes in param val",
			args: args{
				param: triggersv1.Param{
					Name:  "p1",
					Value: `{"a":"b"}`,
				},
				rt: json.RawMessage(`{"foo": "$(tt.params.p1)"}`),
			},
			want: json.RawMessage(`{"foo": "{\"a\":\"b\"}"}`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyParamToResourceTemplate(tt.args.param, tt.args.rt)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("applyParamToResourceTemplate(): -want +got: %s", diff)
			}
		})
	}
}

func Test_ApplyParamsToResourceTemplate(t *testing.T) {
	rt := json.RawMessage(`{"oneparam": "$(tt.params.oneid)", "twoparam": "$(tt.params.twoid)", "threeparam": "$(tt.params.threeid)"`)
	rt3 := json.RawMessage(`{"actualParam": "$(tt.params.oneid)", "invalidParam": "$(tt.params1.invalidid)", "deprecatedParam": "$(params.twoid)"`)
	type args struct {
		params []triggersv1.Param
		rt     json.RawMessage
	}
	tests := []struct {
		name string
		args args
		want json.RawMessage
	}{
		{
			name: "no params",
			args: args{
				params: []triggersv1.Param{},
				rt:     rt,
			},
			want: rt,
		},
		{
			name: "one param",
			args: args{
				params: []triggersv1.Param{
					{Name: "oneid", Value: "onevalue"},
				},
				rt: rt,
			},
			want: json.RawMessage(`{"oneparam": "onevalue", "twoparam": "$(tt.params.twoid)", "threeparam": "$(tt.params.threeid)"`),
		},
		{
			name: "multiple params",
			args: args{
				params: []triggersv1.Param{
					{Name: "oneid", Value: "onevalue"},
					{Name: "twoid", Value: "twovalue"},
					{Name: "threeid", Value: "threevalue"},
				},
				rt: rt,
			},
			want: json.RawMessage(`{"oneparam": "onevalue", "twoparam": "twovalue", "threeparam": "threevalue"`),
		},
		{
			name: "valid and invalid params together",
			args: args{
				params: []triggersv1.Param{
					{Name: "oneid", Value: "actualValue"},
					{Name: "invalidid", Value: "invalidValue"},
					{Name: "twoid", Value: "deprecatedParamValue"},
				},
				rt: rt3,
			},
			want: json.RawMessage(`{"actualParam": "actualValue", "invalidParam": "$(tt.params1.invalidid)", "deprecatedParam": "$(params.twoid)"`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyParamsToResourceTemplate(tt.args.params, tt.args.rt)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("applyParamsToResourceTemplate(): -want +got: %s", diff)
			}
		})
	}
}

var (
	triggerBindings = map[string]*triggersv1.TriggerBinding{
		"my-triggerbinding": {
			ObjectMeta: metav1.ObjectMeta{Name: "my-triggerbinding"},
		},
		"tb-params": {
			ObjectMeta: metav1.ObjectMeta{Name: "tb-params"},
			Spec: triggersv1.TriggerBindingSpec{
				Params: []triggersv1.Param{{
					Name:  "foo",
					Value: "bar",
				}},
			},
		},
	}
	tt = triggersv1.TriggerTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "my-triggertemplate"},
	}
	clusterTriggerBindings = map[string]*triggersv1.ClusterTriggerBinding{
		"my-clustertriggerbinding": {
			ObjectMeta: metav1.ObjectMeta{Name: "my-clustertriggerbinding"},
		},
		"ctb-params": {
			ObjectMeta: metav1.ObjectMeta{Name: "ctb-params"},
			Spec: triggersv1.TriggerBindingSpec{
				Params: []triggersv1.Param{{
					Name:  "foo-ctb",
					Value: "bar-ctb",
				}},
			},
		},
	}
	getTB = func(ctx context.Context, name string, options metav1.GetOptions) (*triggersv1.TriggerBinding, error) {
		if v, ok := triggerBindings[name]; ok {
			return v, nil
		}
		return nil, fmt.Errorf("error invalid name: %s", name)
	}
	getCTB = func(ctx context.Context, name string, options metav1.GetOptions) (*triggersv1.ClusterTriggerBinding, error) {
		if v, ok := clusterTriggerBindings[name]; ok {
			return v, nil
		}
		return nil, fmt.Errorf("error invalid name: %s", name)
	}
	getTT = func(ctx context.Context, name string, options metav1.GetOptions) (*triggersv1.TriggerTemplate, error) {
		if name == "my-triggertemplate" {
			return &tt, nil
		}
		return nil, fmt.Errorf("error invalid name: %s", name)
	}
)

func Test_ResolveTrigger(t *testing.T) {
	tests := []struct {
		name    string
		trigger triggersv1.EventListenerTrigger
		want    ResolvedTrigger
	}{{
		name: "1 binding",
		trigger: triggersv1.EventListenerTrigger{
			Bindings: []*triggersv1.EventListenerBinding{{
				Ref:  "my-triggerbinding",
				Kind: triggersv1.NamespacedTriggerBindingKind,
			}},
			Template: &triggersv1.EventListenerTemplate{
				Name:       "my-triggertemplate",
				APIVersion: "v1alpha1",
			},
		},
		want: ResolvedTrigger{
			TriggerTemplate: &tt,
			BindingParams:   []triggersv1.Param{},
		},
	}, {
		name: "1 clustertype binding",
		trigger: triggersv1.EventListenerTrigger{
			Bindings: []*triggersv1.EventListenerBinding{{
				Ref:  "my-clustertriggerbinding",
				Kind: triggersv1.ClusterTriggerBindingKind,
			}},
			Template: &triggersv1.EventListenerTemplate{
				Name:       "my-triggertemplate",
				APIVersion: "v1alpha1",
			},
		},
		want: ResolvedTrigger{
			TriggerTemplate: &tt,
			BindingParams:   []triggersv1.Param{},
		},
	}, {
		name: "1 embed binding",
		trigger: triggersv1.EventListenerTrigger{
			Bindings: []*triggersv1.EventListenerBinding{{
				Name: "my-embed-binding",
				Spec: &triggersv1.TriggerBindingSpec{
					Params: []triggersv1.Param{{
						Name:  "key",
						Value: "value",
					}},
				},
			}},
			Template: &triggersv1.EventListenerTemplate{
				Name:       "my-triggertemplate",
				APIVersion: "v1alpha1",
			},
		},
		want: ResolvedTrigger{
			BindingParams: []triggersv1.Param{{
				Name:  "key",
				Value: "value",
			}},
			TriggerTemplate: &tt,
		},
	}, {
		name: "no binding",
		trigger: triggersv1.EventListenerTrigger{
			Template: &triggersv1.EventListenerTemplate{
				Name:       "my-triggertemplate",
				APIVersion: "v1alpha1",
			},
		},
		want: ResolvedTrigger{BindingParams: []triggersv1.Param{}, TriggerTemplate: &tt},
	}, {
		name: "concise bindings",
		trigger: triggersv1.EventListenerTrigger{
			Template: &triggersv1.EventListenerTemplate{
				Name: "my-triggertemplate",
			},
			Bindings: []*triggersv1.EventListenerBinding{{
				Name:  "p1",
				Value: ptr.String("v1"),
			}, {
				Name:  "p2",
				Value: ptr.String("v2"),
			}},
		},
		want: ResolvedTrigger{
			TriggerTemplate: &tt,
			BindingParams: []triggersv1.Param{{
				Name:  "p1",
				Value: "v1",
			}, {
				Name:  "p2",
				Value: "v2",
			}},
		},
	}, {
		name: "multiple binding params are merged",
		trigger: triggersv1.EventListenerTrigger{
			Bindings: []*triggersv1.EventListenerBinding{{
				Kind: triggersv1.NamespacedTriggerBindingKind,
				Ref:  "my-triggerbinding",
			}, {
				Kind: triggersv1.NamespacedTriggerBindingKind,
				Ref:  "tb-params",
			}, {
				Kind: triggersv1.ClusterTriggerBindingKind,
				Ref:  "my-clustertriggerbinding",
			}, {
				Kind: triggersv1.ClusterTriggerBindingKind,
				Ref:  "ctb-params",
			}, {
				Name:  "p1",
				Value: ptr.String("v1"),
			}, {
				Name:  "p2",
				Value: ptr.String("v2"),
			}},
			Template: &triggersv1.EventListenerTemplate{
				Name:       "my-triggertemplate",
				APIVersion: "v1alpha1",
			},
		},
		want: ResolvedTrigger{
			TriggerTemplate: &tt,
			BindingParams: []triggersv1.Param{{
				Name:  "foo",
				Value: "bar",
			}, {
				Name:  "foo-ctb",
				Value: "bar-ctb",
			}, {
				Name:  "p1",
				Value: "v1",
			}, {
				Name:  "p2",
				Value: "v2",
			}},
		},
	}, {
		name: "missing kind implies namespacedTriggerBinding",
		trigger: triggersv1.EventListenerTrigger{
			Bindings: []*triggersv1.EventListenerBinding{{
				Name:       "my-triggerbinding",
				APIVersion: "v1alpha1",
				Ref:        "my-triggerbinding",
			}},
			Template: &triggersv1.EventListenerTemplate{
				Name:       "my-triggertemplate",
				APIVersion: "v1alpha1",
			},
		},
		want: ResolvedTrigger{
			BindingParams:   []triggersv1.Param{},
			TriggerTemplate: &tt,
		},
	},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveTrigger(tc.trigger, getTB, getCTB, getTT)
			if err != nil {
				t.Errorf("ResolveTrigger() returned unexpected error: %s", err)
			} else if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ResolveTrigger(): -want +got: %s", diff)
			}
		})
	}
}

func Test_ResolveTrigger_error(t *testing.T) {
	tests := []struct {
		name    string
		trigger triggersv1.EventListenerTrigger
		getTB   getTriggerBinding
		getTT   getTriggerTemplate
		getCTB  getClusterTriggerBinding
	}{{
		name: "triggerbinding not found",
		trigger: triggersv1.EventListenerTrigger{
			Bindings: []*triggersv1.EventListenerBinding{{
				Ref:  "invalid-tb-name",
				Kind: triggersv1.NamespacedTriggerBindingKind,
			}},
			Template: &triggersv1.EventListenerTemplate{
				Name:       "my-triggertemplate",
				APIVersion: "v1alpha1",
			},
		},
		getTB:  getTB,
		getCTB: getCTB,
		getTT:  getTT,
	}, {
		name: "clustertriggerbinding not found",
		trigger: triggersv1.EventListenerTrigger{
			Bindings: []*triggersv1.EventListenerBinding{{
				Ref:  "invalid-ctb-name",
				Kind: triggersv1.ClusterTriggerBindingKind,
			}},
			Template: &triggersv1.EventListenerTemplate{
				Name:       "my-triggertemplate",
				APIVersion: "v1alpha1",
			},
		},
		getTB:  getTB,
		getCTB: getCTB,
		getTT:  getTT,
	}, {
		name: "triggertemplate not found",
		trigger: triggersv1.EventListenerTrigger{
			Bindings: []*triggersv1.EventListenerBinding{{
				Ref:  "my-triggerbinding",
				Kind: triggersv1.NamespacedTriggerBindingKind,
			}},
			Template: &triggersv1.EventListenerTemplate{
				Name:       "invalid-tt-name",
				APIVersion: "v1alpha1",
			},
		},
		getTB:  getTB,
		getCTB: getCTB,
		getTT:  getTT,
	}, {
		name: "triggerbinding and triggertemplate not found",
		trigger: triggersv1.EventListenerTrigger{
			Bindings: []*triggersv1.EventListenerBinding{{
				Ref:  "invalid-tb-name",
				Kind: triggersv1.NamespacedTriggerBindingKind,
			}},
			Template: &triggersv1.EventListenerTemplate{
				Name:       "invalid-tt-name",
				APIVersion: "v1alpha1",
			},
		},
		getTB:  getTB,
		getCTB: getCTB,
		getTT:  getTT,
	}, {
		name: "trigger template missing name",
		trigger: triggersv1.EventListenerTrigger{
			Template: &triggersv1.EventListenerTemplate{
				Name: "",
			},
		},
		getTT: getTT,
	}, {
		name: "invalid trigger binding",
		trigger: triggersv1.EventListenerTrigger{
			Template: &triggersv1.EventListenerTemplate{
				Name: "my-triggertemplate",
			},
			Bindings: []*triggersv1.EventListenerBinding{{
				Value: ptr.String("only-val"),
			}},
		},
		getTT: getTT,
	}, {
		name: "same param name across multiple bindings",
		trigger: triggersv1.EventListenerTrigger{
			Bindings: []*triggersv1.EventListenerBinding{{
				Ref:  "tb-params",
				Kind: triggersv1.NamespacedTriggerBindingKind,
			}, {
				Name:  "foo",
				Value: ptr.String("bar"),
			}},
		},
		getTB: getTB,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ResolveTrigger(tt.trigger, tt.getTB, tt.getCTB, tt.getTT); err == nil {
				t.Error("ResolveTrigger() did not return error when expected")
			}
		})
	}
}

func Test_ApplyUIDToResourceTemplate(t *testing.T) {
	tests := []struct {
		name       string
		rt         json.RawMessage
		expectedRt json.RawMessage
	}{
		{
			name:       "No uid",
			rt:         json.RawMessage(`{"rt": "nothing to see here"}`),
			expectedRt: json.RawMessage(`{"rt": "nothing to see here"}`),
		},
		{
			name:       "One uid",
			rt:         json.RawMessage(`{"rt": "uid is $(uid)"}`),
			expectedRt: json.RawMessage(`{"rt": "uid is abcde"}`),
		},
		{
			name:       "Three uid",
			rt:         json.RawMessage(`{"rt1": "uid is $(uid)", "rt2": "nothing", "rt3": "$(uid)-center-$(uid)"}`),
			expectedRt: json.RawMessage(`{"rt1": "uid is abcde", "rt2": "nothing", "rt3": "abcde-center-abcde"}`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Always resolve uid to abcde for easier testing
			actualRt := applyUIDToResourceTemplate(tt.rt, "abcde")
			if diff := cmp.Diff(string(tt.expectedRt), string(actualRt)); diff != "" {
				t.Errorf("applyUIDToResourceTemplate(): -want +got: %s", diff)
			}
		})
	}
}
func TestMergeBindingParams(t *testing.T) {
	tests := []struct {
		name            string
		bindings        []*triggersv1.TriggerBinding
		clusterBindings []*triggersv1.ClusterTriggerBinding
		want            []triggersv1.Param
		wantErr         bool
	}{{
		name:            "empty bindings",
		clusterBindings: []*triggersv1.ClusterTriggerBinding{},
		bindings: []*triggersv1.TriggerBinding{
			bldr.TriggerBinding("", "", bldr.TriggerBindingSpec()),
			bldr.TriggerBinding("", "", bldr.TriggerBindingSpec()),
		},
		want: []triggersv1.Param{},
	}, {
		name:            "single binding with multiple params",
		clusterBindings: []*triggersv1.ClusterTriggerBinding{},
		bindings: []*triggersv1.TriggerBinding{
			bldr.TriggerBinding("", "", bldr.TriggerBindingSpec(
				bldr.TriggerBindingParam("param1", "value1"),
				bldr.TriggerBindingParam("param2", "value2"),
			)),
		},
		want: []triggersv1.Param{{
			Name:  "param1",
			Value: "value1",
		}, {
			Name:  "param2",
			Value: "value2",
		}},
	}, {
		name: "single cluster type binding with multiple params",
		clusterBindings: []*triggersv1.ClusterTriggerBinding{
			bldr.ClusterTriggerBinding("", bldr.ClusterTriggerBindingSpec(
				bldr.TriggerBindingParam("param1", "value1"),
				bldr.TriggerBindingParam("param2", "value2"),
			)),
		},
		bindings: []*triggersv1.TriggerBinding{},
		want: []triggersv1.Param{{
			Name:  "param1",
			Value: "value1",
		}, {
			Name:  "param2",
			Value: "value2",
		}},
	}, {
		name: "multiple bindings each with multiple params",
		clusterBindings: []*triggersv1.ClusterTriggerBinding{
			bldr.ClusterTriggerBinding("", bldr.ClusterTriggerBindingSpec(
				bldr.TriggerBindingParam("param5", "value1"),
				bldr.TriggerBindingParam("param6", "value2"),
			)),
		},
		bindings: []*triggersv1.TriggerBinding{
			bldr.TriggerBinding("", "", bldr.TriggerBindingSpec(
				bldr.TriggerBindingParam("param1", "value1"),
				bldr.TriggerBindingParam("param2", "value2"),
			)),
			bldr.TriggerBinding("", "", bldr.TriggerBindingSpec(
				bldr.TriggerBindingParam("param3", "value3"),
				bldr.TriggerBindingParam("param4", "value4"),
			)),
		},
		want: []triggersv1.Param{{
			Name:  "param1",
			Value: "value1",
		}, {
			Name:  "param2",
			Value: "value2",
		}, {
			Name:  "param3",
			Value: "value3",
		}, {
			Name:  "param4",
			Value: "value4",
		}, {
			Name:  "param5",
			Value: "value1",
		}, {
			Name:  "param6",
			Value: "value2",
		}},
	}, {
		name:            "multiple bindings with duplicate params",
		clusterBindings: []*triggersv1.ClusterTriggerBinding{},
		bindings: []*triggersv1.TriggerBinding{
			bldr.TriggerBinding("", "", bldr.TriggerBindingSpec(
				bldr.TriggerBindingParam("param1", "value1"),
			)),
			bldr.TriggerBinding("", "", bldr.TriggerBindingSpec(
				bldr.TriggerBindingParam("param1", "value3"),
				bldr.TriggerBindingParam("param4", "value4"),
			)),
		},
		wantErr: true,
	}, {
		name:            "single binding with duplicate params",
		clusterBindings: []*triggersv1.ClusterTriggerBinding{},
		bindings: []*triggersv1.TriggerBinding{
			bldr.TriggerBinding("", "", bldr.TriggerBindingSpec(
				bldr.TriggerBindingParam("param1", "value1"),
				bldr.TriggerBindingParam("param1", "value3"),
			)),
		},
		wantErr: true,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mergeBindingParams(tt.bindings, tt.clusterBindings)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unexpected error : %q", err)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Unexpected output(-want +got): %s", diff)
			}
		})
	}
}
