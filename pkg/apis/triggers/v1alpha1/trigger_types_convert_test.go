/*
Copyright 2020 The Tekton Authors

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

package v1alpha1

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "k8s.io/api/core/v1"
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
			out:  EventListenerTrigger{},
		},
		{
			name: "Convert Partial Object",
			in: TriggerSpec{
				Name: "foo",
				Template: TriggerSpecTemplate{
					Name: "baz",
				},
			},
			out: EventListenerTrigger{
				Name: "foo",
				Template: EventListenerTemplate{
					Name: "baz",
				},
			},
		},
		{
			name: "Convert Full Object",
			in: TriggerSpec{
				Name: "a",
				ServiceAccount: &v1.ObjectReference{
					Name: "a",
				},
				Interceptors: []*TriggerInterceptor{{
					Webhook: &WebhookInterceptor{},
				}},
				Bindings: []*TriggerSpecBinding{{
					Name:       "a",
					APIVersion: "b",
					Kind:       "c",
					Ref:        "d",
					Spec: &TriggerBindingSpec{
						Params: []Param{{
							Name:  "a",
							Value: "b",
						}},
					},
				}},
				Template: TriggerSpecTemplate{
					Name:       "a",
					APIVersion: "b",
				},
			},
			out: EventListenerTrigger{
				Name: "a",
				ServiceAccount: &v1.ObjectReference{
					Name: "a",
				},
				Interceptors: []*TriggerInterceptor{{
					Webhook: &WebhookInterceptor{},
				}},
				Bindings: []*EventListenerBinding{{
					Name:       "a",
					APIVersion: "b",
					Kind:       "c",
					Ref:        "d",
					Spec: &TriggerBindingSpec{
						Params: []Param{{
							Name:  "a",
							Value: "b",
						}},
					},
				}},
				Template: EventListenerTemplate{
					Name:       "a",
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
