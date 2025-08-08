/*
Copyright 2025 The Knative Authors

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

package attributekey

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"go.opentelemetry.io/otel/attribute"
)

func TestWith(t *testing.T) {
	cases := []struct {
		name      string
		got       attribute.KeyValue
		want      attribute.KeyValue
		wantPanic bool
	}{{
		name: "string",
		got:  String("key").With("val"),
		want: attribute.String("key", "val"),
	}, {
		name: "bool",
		got:  Bool("key").With(true),
		want: attribute.Bool("key", true),
	}, {
		name: "int",
		got:  Int("key").With(1),
		want: attribute.Int("key", 1),
	}, {
		name: "int64",
		got:  Int64("key").With(1),
		want: attribute.Int64("key", 1),
	}, {
		name: "float64",
		got:  Float64("key").With(1),
		want: attribute.Float64("key", 1),
	}, {
		name: "string slice",
		got:  Type[[]string]("key").With([]string{"1"}),
		want: attribute.StringSlice("key", []string{"1"}),
	}, {
		name: "bool slice",
		got:  Type[[]bool]("key").With([]bool{true}),
		want: attribute.BoolSlice("key", []bool{true}),
	}, {
		name: "float64 slice",
		got:  Type[[]float64]("key").With([]float64{1}),
		want: attribute.Float64Slice("key", []float64{1}),
	}, {
		name: "int64 slice",
		got:  Type[[]int64]("key").With([]int64{1}),
		want: attribute.Int64Slice("key", []int64{1}),
	}, {
		name: "int slice",
		got:  Type[[]int]("key").With([]int{1}),
		want: attribute.IntSlice("key", []int{1}),
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if diff := diff(tc.want, tc.got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func diff(want, got any) string {
	return cmp.Diff(want, got, cmpopts.EquateComparable(attribute.KeyValue{}))
}
