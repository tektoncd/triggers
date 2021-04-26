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

	"github.com/google/go-cmp/cmp"
	"knative.dev/pkg/ptr"
)

func TestToEventListenerTrigger(t *testing.T) {
	tt := []struct {
		name string
		in   TriggerSpec
		out  EventListenerTrigger
	}{
		{
			name: "Convert Empty Object",
			in:   TriggerSpec{},
			out:  EventListenerTrigger{Template: &EventListenerTemplate{}},
		},
		{
			name: "Convert Partial Object",
			in: TriggerSpec{
				Name: "foo",
				Template: TriggerSpecTemplate{
					Ref: ptr.String("baz"),
				},
			},
			out: EventListenerTrigger{
				Name: "foo",
				Template: &EventListenerTemplate{
					Ref: ptr.String("baz"),
				},
			},
		},
		{
			name: "Convert Full Object",
			in: TriggerSpec{
				Name:               "a",
				ServiceAccountName: "a",
				Interceptors: []*TriggerInterceptor{{
					Webhook: &WebhookInterceptor{},
				}},
				Bindings: []*TriggerSpecBinding{{
					Name:       "a",
					APIVersion: "b",
					Kind:       "c",
					Ref:        "d",
				}},
				Template: TriggerSpecTemplate{
					Ref:        ptr.String("a"),
					APIVersion: "b",
				},
			},
			out: EventListenerTrigger{
				Name:               "a",
				ServiceAccountName: "a",
				Interceptors: []*TriggerInterceptor{{
					Webhook: &WebhookInterceptor{},
				}},
				Bindings: []*EventListenerBinding{{
					Name:       "a",
					APIVersion: "b",
					Kind:       "c",
					Ref:        "d",
				}},
				Template: &EventListenerTemplate{
					Ref:        ptr.String("a"),
					APIVersion: "b",
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ToEventListenerTrigger(tc.in)
			if err != nil {
				t.Fatalf("ToEventListenerTrigger: %v", err)
			}

			if diff := cmp.Diff(tc.out, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
