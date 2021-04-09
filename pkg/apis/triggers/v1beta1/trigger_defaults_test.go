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

package v1beta1_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"github.com/tektoncd/triggers/test"
)

func TestTriggerSetDefaults(t *testing.T) {
	tests := []struct {
		name string
		in   *v1beta1.Trigger
		want *v1beta1.Trigger
		wc   func(context.Context) context.Context
	}{{
		name: "default binding kind",
		in: &v1beta1.Trigger{
			Spec: v1beta1.TriggerSpec{
				Bindings: []*v1beta1.TriggerSpecBinding{{
					Ref: "binding",
				}, {
					Kind: v1beta1.NamespacedTriggerBindingKind,
					Ref:  "namespace-binding",
				}, {
					Kind: v1beta1.ClusterTriggerBindingKind,
					Ref:  "cluster-binding",
				}},
			},
		},
		wc: v1beta1.WithUpgradeViaDefaulting,
		want: &v1beta1.Trigger{
			Spec: v1beta1.TriggerSpec{
				Bindings: []*v1beta1.TriggerSpecBinding{{
					Kind: v1beta1.NamespacedTriggerBindingKind,
					Ref:  "binding",
				}, {
					Kind: v1beta1.NamespacedTriggerBindingKind,
					Ref:  "namespace-binding",
				}, {
					Kind: v1beta1.ClusterTriggerBindingKind,
					Ref:  "cluster-binding",
				}},
			},
		},
	}, {
		name: "upgrade context not set",
		in: &v1beta1.Trigger{
			Spec: v1beta1.TriggerSpec{
				Bindings: []*v1beta1.TriggerSpecBinding{{
					Ref: "binding",
				}},
			},
		},
		want: &v1beta1.Trigger{
			Spec: v1beta1.TriggerSpec{
				Bindings: []*v1beta1.TriggerSpecBinding{{
					Ref: "binding", // If upgrade context was set, Kind should have been added
				}},
			},
		},
	}, {
		name: "defaults interceptor ref",
		in: &v1beta1.Trigger{
			Spec: v1beta1.TriggerSpec{
				Interceptors: []*v1beta1.TriggerInterceptor{{
					Ref: v1beta1.InterceptorRef{
						Name: "cel",
					},
				}},
			},
		},
		wc: v1beta1.WithUpgradeViaDefaulting,
		want: &v1beta1.Trigger{
			Spec: v1beta1.TriggerSpec{
				Interceptors: []*v1beta1.TriggerInterceptor{{
					Ref: v1beta1.InterceptorRef{
						Name: "cel",
						Kind: v1beta1.ClusterInterceptorKind,
					},
				}},
			},
		},
	}, {
		name: "default deprecatedGithub to new ref/params",
		in: &v1beta1.Trigger{
			Spec: v1beta1.TriggerSpec{
				Interceptors: []*v1beta1.TriggerInterceptor{{
					DeprecatedGitHub: &v1beta1.GitHubInterceptor{
						SecretRef: &v1beta1.SecretRef{
							SecretKey:  "key",
							SecretName: "name",
						},
						EventTypes: []string{"push"},
					},
				}},
			},
		},
		wc: v1beta1.WithUpgradeViaDefaulting,
		want: &v1beta1.Trigger{
			Spec: v1beta1.TriggerSpec{
				Interceptors: []*v1beta1.TriggerInterceptor{{
					Ref: v1beta1.InterceptorRef{
						Name: "github",
						Kind: v1beta1.ClusterInterceptorKind,
					},
					Params: []v1beta1.InterceptorParams{{
						Name: "secretRef",
						Value: test.ToV1JSON(t, &v1beta1.SecretRef{
							SecretKey:  "key",
							SecretName: "name",
						}),
					}, {
						Name:  "eventTypes",
						Value: test.ToV1JSON(t, []string{"push"}),
					}},
				}},
			},
		},
	}, {
		name: "default deprecatedGitlab to new ref/params",
		in: &v1beta1.Trigger{
			Spec: v1beta1.TriggerSpec{
				Interceptors: []*v1beta1.TriggerInterceptor{{
					DeprecatedGitLab: &v1beta1.GitLabInterceptor{
						SecretRef: &v1beta1.SecretRef{
							SecretKey:  "key",
							SecretName: "name",
						},
						EventTypes: []string{"push"},
					},
				}},
			},
		},
		wc: v1beta1.WithUpgradeViaDefaulting,
		want: &v1beta1.Trigger{
			Spec: v1beta1.TriggerSpec{
				Interceptors: []*v1beta1.TriggerInterceptor{{
					Ref: v1beta1.InterceptorRef{
						Name: "gitlab",
						Kind: v1beta1.ClusterInterceptorKind,
					},
					Params: []v1beta1.InterceptorParams{{
						Name: "secretRef",
						Value: test.ToV1JSON(t, &v1beta1.SecretRef{
							SecretKey:  "key",
							SecretName: "name",
						}),
					}, {
						Name:  "eventTypes",
						Value: test.ToV1JSON(t, []string{"push"}),
					}},
				}},
			},
		},
	}, {
		name: "default deprecatedBitbucket to new ref/params",
		in: &v1beta1.Trigger{
			Spec: v1beta1.TriggerSpec{
				Interceptors: []*v1beta1.TriggerInterceptor{{
					DeprecatedBitbucket: &v1beta1.BitbucketInterceptor{
						SecretRef: &v1beta1.SecretRef{
							SecretKey:  "key",
							SecretName: "name",
						},
						EventTypes: []string{"push"},
					},
				}},
			},
		},
		wc: v1beta1.WithUpgradeViaDefaulting,
		want: &v1beta1.Trigger{
			Spec: v1beta1.TriggerSpec{
				Interceptors: []*v1beta1.TriggerInterceptor{{
					Ref: v1beta1.InterceptorRef{
						Name: "bitbucket",
						Kind: v1beta1.ClusterInterceptorKind,
					},
					Params: []v1beta1.InterceptorParams{{
						Name: "secretRef",
						Value: test.ToV1JSON(t, &v1beta1.SecretRef{
							SecretKey:  "key",
							SecretName: "name",
						}),
					}, {
						Name:  "eventTypes",
						Value: test.ToV1JSON(t, []string{"push"}),
					}},
				}},
			},
		},
	}, {
		name: "default deprecatedCEL to new ref/params",
		in: &v1beta1.Trigger{
			Spec: v1beta1.TriggerSpec{
				Interceptors: []*v1beta1.TriggerInterceptor{{
					DeprecatedCEL: &v1beta1.CELInterceptor{
						Filter: "body.foo == bar",
						Overlays: []v1beta1.CELOverlay{{
							Key:        "abc",
							Expression: "body.foo",
						}},
					},
				}},
			},
		},
		wc: v1beta1.WithUpgradeViaDefaulting,
		want: &v1beta1.Trigger{
			Spec: v1beta1.TriggerSpec{
				Interceptors: []*v1beta1.TriggerInterceptor{{
					Ref: v1beta1.InterceptorRef{
						Name: "cel",
						Kind: v1beta1.ClusterInterceptorKind,
					},
					Params: []v1beta1.InterceptorParams{{
						Name:  "filter",
						Value: test.ToV1JSON(t, "body.foo == bar"),
					}, {
						Name: "overlays",
						Value: test.ToV1JSON(t, []v1beta1.CELOverlay{{
							Key:        "abc",
							Expression: "body.foo",
						}}),
					}},
				}},
			},
		},
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.in
			ctx := context.Background()
			if tc.wc != nil {
				ctx = tc.wc(ctx)
			}
			got.SetDefaults(ctx)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("SetDefaults (-want, +got) = %v", diff)
			}
		})
	}
}
