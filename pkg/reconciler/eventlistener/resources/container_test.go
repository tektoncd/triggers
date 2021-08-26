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

package resources

import (
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/triggers/pkg/apis/triggers"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

func TestContainer(t *testing.T) {
	config := *MakeConfig()

	tests := []struct {
		name    string
		el      *v1beta1.EventListener
		want    corev1.Container
		withPod *duckv1.WithPod
	}{{
		name: "vanilla",
		el:   makeEL(),
		want: corev1.Container{
			Name:  "event-listener",
			Image: DefaultImage,
			Ports: []corev1.ContainerPort{{
				ContainerPort: int32(eventListenerContainerPort),
				Protocol:      corev1.ProtocolTCP,
			}},
			Args: []string{
				"--el-name=" + eventListenerName,
				"--el-namespace=" + namespace,
				"--port=" + strconv.Itoa(eventListenerContainerPort),
				"--readtimeout=" + strconv.FormatInt(DefaultReadTimeout, 10),
				"--writetimeout=" + strconv.FormatInt(DefaultWriteTimeout, 10),
				"--idletimeout=" + strconv.FormatInt(DefaultIdleTimeout, 10),
				"--timeouthandler=" + strconv.FormatInt(DefaultTimeOutHandler, 10),
				"--is-multi-ns=" + strconv.FormatBool(false),
				"--payload-validation=" + strconv.FormatBool(true),
			},
			Env: []corev1.EnvVar{},
		},
	}, {
		name: "with resources",
		el: makeEL(func(el *v1beta1.EventListener) {
			el.Spec.Resources.KubernetesResource = &v1beta1.KubernetesResource{
				WithPodSpec: duckv1.WithPodSpec{
					Template: duckv1.PodSpecable{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Resources: corev1.ResourceRequirements{
									Requests: map[corev1.ResourceName]resource.Quantity{
										corev1.ResourceCPU: resource.MustParse("100m"),
									},
								},
							}},
						},
					},
				},
			}
		}),
		want: corev1.Container{
			Name:  "event-listener",
			Image: DefaultImage,
			Ports: []corev1.ContainerPort{{
				ContainerPort: int32(eventListenerContainerPort),
				Protocol:      corev1.ProtocolTCP,
			}},
			Args: []string{
				"--el-name=" + eventListenerName,
				"--el-namespace=" + namespace,
				"--port=" + strconv.Itoa(eventListenerContainerPort),
				"--readtimeout=" + strconv.FormatInt(DefaultReadTimeout, 10),
				"--writetimeout=" + strconv.FormatInt(DefaultWriteTimeout, 10),
				"--idletimeout=" + strconv.FormatInt(DefaultIdleTimeout, 10),
				"--timeouthandler=" + strconv.FormatInt(DefaultTimeOutHandler, 10),
				"--is-multi-ns=" + strconv.FormatBool(false),
				"--payload-validation=" + strconv.FormatBool(true),
			},
			Resources: corev1.ResourceRequirements{
				Requests: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceCPU: resource.MustParse("100m"),
				},
			},
			Env: []corev1.EnvVar{},
		},
	}, {
		name: "with env",
		el: makeEL(func(el *v1beta1.EventListener) {
			el.Spec.Resources.KubernetesResource = &v1beta1.KubernetesResource{
				WithPodSpec: duckv1.WithPodSpec{
					Template: duckv1.PodSpecable{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Env: []corev1.EnvVar{{
									Name:  "FOO",
									Value: "bar",
								}},
							}},
						},
					},
				},
			}
		}),
		want: corev1.Container{
			Name:  "event-listener",
			Image: DefaultImage,
			Ports: []corev1.ContainerPort{{
				ContainerPort: int32(eventListenerContainerPort),
				Protocol:      corev1.ProtocolTCP,
			}},
			Args: []string{
				"--el-name=" + eventListenerName,
				"--el-namespace=" + namespace,
				"--port=" + strconv.Itoa(eventListenerContainerPort),
				"--readtimeout=" + strconv.FormatInt(DefaultReadTimeout, 10),
				"--writetimeout=" + strconv.FormatInt(DefaultWriteTimeout, 10),
				"--idletimeout=" + strconv.FormatInt(DefaultIdleTimeout, 10),
				"--timeouthandler=" + strconv.FormatInt(DefaultTimeOutHandler, 10),
				"--is-multi-ns=" + strconv.FormatBool(false),
				"--payload-validation=" + strconv.FormatBool(true),
			},
			Env: []corev1.EnvVar{{
				Name:  "FOO",
				Value: "bar",
			}},
		},
	}, {
		name: "with pod resources",
		el:   makeEL(),
		withPod: &duckv1.WithPod{
			Spec: duckv1.WithPodSpec{
				Template: duckv1.PodSpecable{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Resources: corev1.ResourceRequirements{
								Requests: map[corev1.ResourceName]resource.Quantity{
									corev1.ResourceCPU: resource.MustParse("200m"),
								},
							},
						}},
					},
				},
			},
		},
		want: corev1.Container{
			Name:  "event-listener",
			Image: DefaultImage,
			Ports: []corev1.ContainerPort{{
				ContainerPort: int32(eventListenerContainerPort),
				Protocol:      corev1.ProtocolTCP,
			}},
			Args: []string{
				"--el-name=" + eventListenerName,
				"--el-namespace=" + namespace,
				"--port=" + strconv.Itoa(eventListenerContainerPort),
				"--readtimeout=" + strconv.FormatInt(DefaultReadTimeout, 10),
				"--writetimeout=" + strconv.FormatInt(DefaultWriteTimeout, 10),
				"--idletimeout=" + strconv.FormatInt(DefaultIdleTimeout, 10),
				"--timeouthandler=" + strconv.FormatInt(DefaultTimeOutHandler, 10),
				"--is-multi-ns=" + strconv.FormatBool(false),
				"--payload-validation=" + strconv.FormatBool(true),
			},
			Resources: corev1.ResourceRequirements{
				Requests: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceCPU: resource.MustParse("200m"),
				},
			},
			Env: []corev1.EnvVar{},
		},
	}, {
		name: "with pod env",
		el:   makeEL(),
		withPod: &duckv1.WithPod{
			Spec: duckv1.WithPodSpec{
				Template: duckv1.PodSpecable{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Env: []corev1.EnvVar{{
								Name:  "BAR",
								Value: "food",
							}},
						}},
					},
				},
			},
		},
		want: corev1.Container{
			Name:  "event-listener",
			Image: DefaultImage,
			Ports: []corev1.ContainerPort{{
				ContainerPort: int32(eventListenerContainerPort),
				Protocol:      corev1.ProtocolTCP,
			}},
			Args: []string{
				"--el-name=" + eventListenerName,
				"--el-namespace=" + namespace,
				"--port=" + strconv.Itoa(eventListenerContainerPort),
				"--readtimeout=" + strconv.FormatInt(DefaultReadTimeout, 10),
				"--writetimeout=" + strconv.FormatInt(DefaultWriteTimeout, 10),
				"--idletimeout=" + strconv.FormatInt(DefaultIdleTimeout, 10),
				"--timeouthandler=" + strconv.FormatInt(DefaultTimeOutHandler, 10),
				"--is-multi-ns=" + strconv.FormatBool(false),
				"--payload-validation=" + strconv.FormatBool(true),
			},
			Env: []corev1.EnvVar{{
				Name:  "BAR",
				Value: "food",
			}},
		},
	}, {
		name: "with namespace selector",
		el: makeEL(func(el *v1beta1.EventListener) {
			el.Spec.NamespaceSelector.MatchNames = []string{"a", "b"}
		}),
		want: corev1.Container{
			Name:  "event-listener",
			Image: DefaultImage,
			Ports: []corev1.ContainerPort{{
				ContainerPort: int32(eventListenerContainerPort),
				Protocol:      corev1.ProtocolTCP,
			}},
			Args: []string{
				"--el-name=" + eventListenerName,
				"--el-namespace=" + namespace,
				"--port=" + strconv.Itoa(eventListenerContainerPort),
				"--readtimeout=" + strconv.FormatInt(DefaultReadTimeout, 10),
				"--writetimeout=" + strconv.FormatInt(DefaultWriteTimeout, 10),
				"--idletimeout=" + strconv.FormatInt(DefaultIdleTimeout, 10),
				"--timeouthandler=" + strconv.FormatInt(DefaultTimeOutHandler, 10),
				"--is-multi-ns=" + strconv.FormatBool(true),
				"--payload-validation=" + strconv.FormatBool(true),
			},
			Env: []corev1.EnvVar{},
		},
	}, {
		name: "without payload validation",
		el: makeEL(func(el *v1beta1.EventListener) {
			el.Annotations = map[string]string{
				triggers.PayloadValidationAnnotation: "false",
			}
		}),
		want: corev1.Container{
			Name:  "event-listener",
			Image: DefaultImage,
			Ports: []corev1.ContainerPort{{
				ContainerPort: int32(eventListenerContainerPort),
				Protocol:      corev1.ProtocolTCP,
			}},
			Args: []string{
				"--el-name=" + eventListenerName,
				"--el-namespace=" + namespace,
				"--port=" + strconv.Itoa(eventListenerContainerPort),
				"--readtimeout=" + strconv.FormatInt(DefaultReadTimeout, 10),
				"--writetimeout=" + strconv.FormatInt(DefaultWriteTimeout, 10),
				"--idletimeout=" + strconv.FormatInt(DefaultIdleTimeout, 10),
				"--timeouthandler=" + strconv.FormatInt(DefaultTimeOutHandler, 10),
				"--is-multi-ns=" + strconv.FormatBool(false),
				"--payload-validation=" + strconv.FormatBool(false),
			},
			Env: []corev1.EnvVar{},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MakeContainer(tt.el, config, tt.withPod)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("MakeContainer() did not return expected. -want, +got: %s", diff)
			}
		})
	}
}
