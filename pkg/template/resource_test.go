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
	"encoding/json"
	"fmt"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
)

// Allows us to sort Params by Name
type ByName []pipelinev1.Param

func (n ByName) Len() int           { return len(n) }
func (n ByName) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }
func (n ByName) Less(i, j int) bool { return n[i].Name < n[j].Name }

func Test_MergeInDefaultParams(t *testing.T) {
	var (
		oneParam = pipelinev1.Param{
			Name:  "oneid",
			Value: pipelinev1.ArrayOrString{StringVal: "onevalue"},
		}
		oneParamSpec = pipelinev1.ParamSpec{
			Name:    "oneid",
			Default: &pipelinev1.ArrayOrString{StringVal: "onedefault"},
		}
		wantDefaultOneParam = pipelinev1.Param{
			Name:  "oneid",
			Value: pipelinev1.ArrayOrString{StringVal: "onedefault"},
		}
		twoParamSpec = pipelinev1.ParamSpec{
			Name:    "twoid",
			Default: &pipelinev1.ArrayOrString{StringVal: "twodefault"},
		}
		wantDefaultTwoParam = pipelinev1.Param{
			Name:  "twoid",
			Value: pipelinev1.ArrayOrString{StringVal: "twodefault"},
		}
		threeParamSpec = pipelinev1.ParamSpec{
			Name:    "threeid",
			Default: &pipelinev1.ArrayOrString{StringVal: "threedefault"},
		}
		wantDefaultThreeParam = pipelinev1.Param{
			Name:  "threeid",
			Value: pipelinev1.ArrayOrString{StringVal: "threedefault"},
		}
		noDefaultParamSpec = pipelinev1.ParamSpec{
			Name: "nodefault",
		}
	)
	type args struct {
		params     []pipelinev1.Param
		paramSpecs []pipelinev1.ParamSpec
	}
	tests := []struct {
		name string
		args args
		want []pipelinev1.Param
	}{
		{
			name: "add one default param",
			args: args{
				params:     []pipelinev1.Param{},
				paramSpecs: []pipelinev1.ParamSpec{oneParamSpec},
			},
			want: []pipelinev1.Param{wantDefaultOneParam},
		},
		{
			name: "add multiple default params",
			args: args{
				params:     []pipelinev1.Param{},
				paramSpecs: []pipelinev1.ParamSpec{oneParamSpec, twoParamSpec, threeParamSpec},
			},
			want: []pipelinev1.Param{wantDefaultOneParam, wantDefaultTwoParam, wantDefaultThreeParam},
		},
		{
			name: "do not override existing value",
			args: args{
				params:     []pipelinev1.Param{oneParam},
				paramSpecs: []pipelinev1.ParamSpec{oneParamSpec},
			},
			want: []pipelinev1.Param{oneParam},
		},
		{
			name: "add no default params",
			args: args{
				params:     []pipelinev1.Param{},
				paramSpecs: []pipelinev1.ParamSpec{noDefaultParamSpec},
			},
			want: []pipelinev1.Param{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeInDefaultParams(tt.args.params, tt.args.paramSpecs)
			sort.Sort(ByName(got))
			sort.Sort(ByName(tt.want))
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("MergeInDefaultParams(): -want +got: %s", diff)
			}
		})
	}
}

func Test_applyParamToResourceTemplate(t *testing.T) {
	var (
		oneParam = pipelinev1.Param{
			Name:  "oneid",
			Value: pipelinev1.ArrayOrString{StringVal: "onevalue"},
		}
		rtNoParamVars             = json.RawMessage(`{"foo": "bar"}`)
		wantRtNoParamVars         = json.RawMessage(`{"foo": "bar"}`)
		rtNoMatchingParamVars     = json.RawMessage(`{"foo": "$(params.no.matching.path)"}`)
		wantRtNoMatchingParamVars = json.RawMessage(`{"foo": "$(params.no.matching.path)"}`)
		rtOneParamVar             = json.RawMessage(`{"foo": "bar-$(params.oneid)-bar"}`)
		wantRtOneParamVar         = json.RawMessage(`{"foo": "bar-onevalue-bar"}`)
		rtMultipleParamVars       = json.RawMessage(`{"$(params.oneid)": "bar-$(params.oneid)-$(params.oneid)$(params.oneid)$(params.oneid)-$(params.oneid)-bar"}`)
		wantRtMultipleParamVars   = json.RawMessage(`{"onevalue": "bar-onevalue-onevalueonevalueonevalue-onevalue-bar"}`)
	)
	type args struct {
		param pipelinev1.Param
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
	rt := json.RawMessage(`{"oneparam": "$(params.oneid)", "twoparam": "$(params.twoid)", "threeparam": "$(params.threeid)"`)
	type args struct {
		params []pipelinev1.Param
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
				params: []pipelinev1.Param{},
				rt:     rt,
			},
			want: rt,
		},
		{
			name: "one param",
			args: args{
				params: []pipelinev1.Param{
					pipelinev1.Param{Name: "oneid", Value: pipelinev1.ArrayOrString{StringVal: "onevalue"}},
				},
				rt: rt,
			},
			want: json.RawMessage(`{"oneparam": "onevalue", "twoparam": "$(params.twoid)", "threeparam": "$(params.threeid)"`),
		},
		{
			name: "multiple params",
			args: args{
				params: []pipelinev1.Param{
					pipelinev1.Param{Name: "oneid", Value: pipelinev1.ArrayOrString{StringVal: "onevalue"}},
					pipelinev1.Param{Name: "twoid", Value: pipelinev1.ArrayOrString{StringVal: "twovalue"}},
					pipelinev1.Param{Name: "threeid", Value: pipelinev1.ArrayOrString{StringVal: "threevalue"}},
				},
				rt: rt,
			},
			want: json.RawMessage(`{"oneparam": "onevalue", "twoparam": "twovalue", "threeparam": "threevalue"`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ApplyParamsToResourceTemplate(tt.args.params, tt.args.rt)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ApplyParamsToResourceTemplate(): -want +got: %s", diff)
			}
		})
	}
}

var (
	tb = triggersv1.TriggerBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "my-triggerbinding"},
	}
	tt = triggersv1.TriggerTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "my-triggertemplate"},
	}
	getTB = func(name string, options metav1.GetOptions) (*triggersv1.TriggerBinding, error) {
		if name == "my-triggerbinding" {
			return &tb, nil
		}
		return nil, fmt.Errorf("Error invalid name: %s", name)
	}
	getTT = func(name string, options metav1.GetOptions) (*triggersv1.TriggerTemplate, error) {
		if name == "my-triggertemplate" {
			return &tt, nil
		}
		return nil, fmt.Errorf("Error invalid name: %s", name)
	}
)

func Test_ResolveBinding(t *testing.T) {
	trigger := triggersv1.Trigger{
		TriggerBinding:  triggersv1.TriggerBindingRef{Name: "my-triggerbinding"},
		TriggerTemplate: triggersv1.TriggerTemplateRef{Name: "my-triggertemplate"},
	}
	want := ResolvedBinding{TriggerBinding: &tb, TriggerTemplate: &tt}
	got, err := ResolveBinding(trigger, getTB, getTT)
	if err != nil {
		t.Errorf("ResolveBinding() returned unexpected error: %s", err)
	} else if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("ResolveBinding(): -want +got: %s", diff)
	}
}
func Test_ResolveBinding_error(t *testing.T) {
	tests := []struct {
		name    string
		trigger triggersv1.Trigger
		getTB   getTriggerBinding
		getTT   getTriggerTemplate
	}{
		{
			name: "error triggerbinding",
			trigger: triggersv1.Trigger{
				TriggerBinding:  triggersv1.TriggerBindingRef{Name: "invalid-name"},
				TriggerTemplate: triggersv1.TriggerTemplateRef{Name: "my-triggertemplate"},
			},
			getTB: getTB,
			getTT: getTT,
		},
		{
			name: "error triggertemplate",
			trigger: triggersv1.Trigger{
				TriggerBinding:  triggersv1.TriggerBindingRef{Name: "my-triggerbinding"},
				TriggerTemplate: triggersv1.TriggerTemplateRef{Name: "invalid-name"},
			},
			getTB: getTB,
			getTT: getTT,
		},
		{
			name: "error triggerbinding and triggertemplate",
			trigger: triggersv1.Trigger{
				TriggerBinding:  triggersv1.TriggerBindingRef{Name: "invalid-name"},
				TriggerTemplate: triggersv1.TriggerTemplateRef{Name: "invalid-name"},
			},
			getTB: getTB,
			getTT: getTT,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ResolveBinding(tt.trigger, tt.getTB, tt.getTT)
			if err == nil {
				t.Error("ResolveBinding() did not return error when expected")
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
			expectedRt: json.RawMessage(`{"rt": "uid is cbhtc"}`),
		},
		{
			name:       "Three uid",
			rt:         json.RawMessage(`{"rt1": "uid is $(uid)", "rt2": "nothing", "rt3": "$(uid)-center-$(uid)"}`),
			expectedRt: json.RawMessage(`{"rt1": "uid is cbhtc", "rt2": "nothing", "rt3": "cbhtc-center-cbhtc"}`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This seeds Uid() to return 'cbhtc'
			rand.Seed(0)
			actualRt := ApplyUIDToResourceTemplate(tt.rt, Uid())
			if diff := cmp.Diff(string(tt.expectedRt), string(actualRt)); diff != "" {
				t.Errorf("ApplyUIDToResourceTemplate(): -want +got: %s", diff)
			}
		})
	}
}
