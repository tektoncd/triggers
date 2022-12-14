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
	reconcilersource "knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/ptr"
)

func TestContainer(t *testing.T) {
	config := *MakeConfig()

	tests := []struct {
		name string
		el   *v1beta1.EventListener
		want corev1.Container
		opts []ContainerOption
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
				"--httpclient-readtimeout=" + strconv.FormatInt(DefaultHTTPClientReadTimeOut, 10),
				"--httpclient-keep-alive=" + strconv.FormatInt(DefaultHTTPClientKeepAlive, 10),
				"--httpclient-tlshandshaketimeout=" + strconv.FormatInt(DefaultHTTPClientTLSHandshakeTimeout, 10),
				"--httpclient-responseheadertimeout=" + strconv.FormatInt(DefaultHTTPClientResponseHeaderTimeout, 10),
				"--httpclient-expectcontinuetimeout=" + strconv.FormatInt(DefaultHTTPClientExpectContinueTimeout, 10),
				"--is-multi-ns=" + strconv.FormatBool(false),
				"--payload-validation=" + strconv.FormatBool(true),
				"--cloudevent-uri=",
			},
			Env: []corev1.EnvVar{{
				Name: "K_LOGGING_CONFIG",
			}, {
				Name: "K_METRICS_CONFIG",
			}, {
				Name: "K_TRACING_CONFIG",
			}, {
				Name:  "NAMESPACE",
				Value: namespace,
			}, {
				Name:  "NAME",
				Value: eventListenerName,
			}, {
				Name:  "EL_EVENT",
				Value: "disable",
			}, {
				Name:  "K_SINK_TIMEOUT",
				Value: strconv.FormatInt(DefaultTimeOutHandler, 10),
			}},
			SecurityContext: &corev1.SecurityContext{
				AllowPrivilegeEscalation: ptr.Bool(false),
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{"ALL"},
				},
				// 65532 is the distroless nonroot user ID
				RunAsUser:    ptr.Int64(65532),
				RunAsGroup:   ptr.Int64(65532),
				RunAsNonRoot: ptr.Bool(true),
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
			},
		},
	}, {
		name: "with resources option",
		el:   makeEL(),
		opts: []ContainerOption{
			func(c *corev1.Container) {
				c.Resources = corev1.ResourceRequirements{
					Requests: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU: resource.MustParse("200m"),
					},
				}
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
				"--httpclient-readtimeout=" + strconv.FormatInt(DefaultHTTPClientReadTimeOut, 10),
				"--httpclient-keep-alive=" + strconv.FormatInt(DefaultHTTPClientKeepAlive, 10),
				"--httpclient-tlshandshaketimeout=" + strconv.FormatInt(DefaultHTTPClientTLSHandshakeTimeout, 10),
				"--httpclient-responseheadertimeout=" + strconv.FormatInt(DefaultHTTPClientResponseHeaderTimeout, 10),
				"--httpclient-expectcontinuetimeout=" + strconv.FormatInt(DefaultHTTPClientExpectContinueTimeout, 10),
				"--is-multi-ns=" + strconv.FormatBool(false),
				"--payload-validation=" + strconv.FormatBool(true),
				"--cloudevent-uri=",
			},
			Resources: corev1.ResourceRequirements{
				Requests: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceCPU: resource.MustParse("200m"),
				},
			},
			Env: []corev1.EnvVar{{
				Name: "K_LOGGING_CONFIG",
			}, {
				Name: "K_METRICS_CONFIG",
			}, {
				Name: "K_TRACING_CONFIG",
			}, {
				Name:  "NAMESPACE",
				Value: namespace,
			}, {
				Name:  "NAME",
				Value: eventListenerName,
			}, {
				Name:  "EL_EVENT",
				Value: "disable",
			}, {
				Name:  "K_SINK_TIMEOUT",
				Value: strconv.FormatInt(DefaultTimeOutHandler, 10),
			}},
			SecurityContext: &corev1.SecurityContext{
				AllowPrivilegeEscalation: ptr.Bool(false),
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{"ALL"},
				},
				// 65532 is the distroless nonroot user ID
				RunAsUser:    ptr.Int64(65532),
				RunAsGroup:   ptr.Int64(65532),
				RunAsNonRoot: ptr.Bool(true),
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
			},
		},
	}, {
		name: "with env option",
		el:   makeEL(),
		opts: []ContainerOption{
			func(c *corev1.Container) {
				c.Env = []corev1.EnvVar{{
					Name:  "BAR",
					Value: "food",
				}}
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
				"--httpclient-readtimeout=" + strconv.FormatInt(DefaultHTTPClientReadTimeOut, 10),
				"--httpclient-keep-alive=" + strconv.FormatInt(DefaultHTTPClientKeepAlive, 10),
				"--httpclient-tlshandshaketimeout=" + strconv.FormatInt(DefaultHTTPClientTLSHandshakeTimeout, 10),
				"--httpclient-responseheadertimeout=" + strconv.FormatInt(DefaultHTTPClientResponseHeaderTimeout, 10),
				"--httpclient-expectcontinuetimeout=" + strconv.FormatInt(DefaultHTTPClientExpectContinueTimeout, 10),
				"--is-multi-ns=" + strconv.FormatBool(false),
				"--payload-validation=" + strconv.FormatBool(true),
				"--cloudevent-uri=",
			},
			Env: []corev1.EnvVar{{
				Name:  "BAR",
				Value: "food",
			}},
			SecurityContext: &corev1.SecurityContext{
				AllowPrivilegeEscalation: ptr.Bool(false),
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{"ALL"},
				},
				// 65532 is the distroless nonroot user ID
				RunAsUser:    ptr.Int64(65532),
				RunAsGroup:   ptr.Int64(65532),
				RunAsNonRoot: ptr.Bool(true),
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
			},
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
				"--httpclient-readtimeout=" + strconv.FormatInt(DefaultHTTPClientReadTimeOut, 10),
				"--httpclient-keep-alive=" + strconv.FormatInt(DefaultHTTPClientKeepAlive, 10),
				"--httpclient-tlshandshaketimeout=" + strconv.FormatInt(DefaultHTTPClientTLSHandshakeTimeout, 10),
				"--httpclient-responseheadertimeout=" + strconv.FormatInt(DefaultHTTPClientResponseHeaderTimeout, 10),
				"--httpclient-expectcontinuetimeout=" + strconv.FormatInt(DefaultHTTPClientExpectContinueTimeout, 10),
				"--is-multi-ns=" + strconv.FormatBool(true),
				"--payload-validation=" + strconv.FormatBool(true),
				"--cloudevent-uri=",
			},
			Env: []corev1.EnvVar{{
				Name: "K_LOGGING_CONFIG",
			}, {
				Name: "K_METRICS_CONFIG",
			}, {
				Name: "K_TRACING_CONFIG",
			}, {
				Name:  "NAMESPACE",
				Value: namespace,
			}, {
				Name:  "NAME",
				Value: eventListenerName,
			}, {
				Name:  "EL_EVENT",
				Value: "disable",
			}, {
				Name:  "K_SINK_TIMEOUT",
				Value: strconv.FormatInt(DefaultTimeOutHandler, 10),
			}},
			SecurityContext: &corev1.SecurityContext{
				AllowPrivilegeEscalation: ptr.Bool(false),
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{"ALL"},
				},
				// 65532 is the distroless nonroot user ID
				RunAsUser:    ptr.Int64(65532),
				RunAsGroup:   ptr.Int64(65532),
				RunAsNonRoot: ptr.Bool(true),
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
			},
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
				"--httpclient-readtimeout=" + strconv.FormatInt(DefaultHTTPClientReadTimeOut, 10),
				"--httpclient-keep-alive=" + strconv.FormatInt(DefaultHTTPClientKeepAlive, 10),
				"--httpclient-tlshandshaketimeout=" + strconv.FormatInt(DefaultHTTPClientTLSHandshakeTimeout, 10),
				"--httpclient-responseheadertimeout=" + strconv.FormatInt(DefaultHTTPClientResponseHeaderTimeout, 10),
				"--httpclient-expectcontinuetimeout=" + strconv.FormatInt(DefaultHTTPClientExpectContinueTimeout, 10),
				"--is-multi-ns=" + strconv.FormatBool(false),
				"--payload-validation=" + strconv.FormatBool(false),
				"--cloudevent-uri=",
			},
			Env: []corev1.EnvVar{{
				Name: "K_LOGGING_CONFIG",
			}, {
				Name: "K_METRICS_CONFIG",
			}, {
				Name: "K_TRACING_CONFIG",
			}, {
				Name:  "NAMESPACE",
				Value: namespace,
			}, {
				Name:  "NAME",
				Value: eventListenerName,
			}, {
				Name:  "EL_EVENT",
				Value: "disable",
			}, {
				Name:  "K_SINK_TIMEOUT",
				Value: strconv.FormatInt(DefaultTimeOutHandler, 10),
			}},
			SecurityContext: &corev1.SecurityContext{
				AllowPrivilegeEscalation: ptr.Bool(false),
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{"ALL"},
				},
				// 65532 is the distroless nonroot user ID
				RunAsUser:    ptr.Int64(65532),
				RunAsGroup:   ptr.Int64(65532),
				RunAsNonRoot: ptr.Bool(true),
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
			},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MakeContainer(tt.el, &reconcilersource.EmptyVarsGenerator{}, config, tt.opts...)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("MakeContainer() did not return expected. -want, +got: %s", diff)
			}
		})
	}
}
