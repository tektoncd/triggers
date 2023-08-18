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
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"
)

func TestService(t *testing.T) {
	config := *MakeConfig()

	tests := []struct {
		name string
		el   *v1beta1.EventListener
		want *corev1.Service
	}{{
		name: "EventListener with status",
		el:   makeEL(withStatus),
		want: &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      generatedResourceName,
				Namespace: namespace,
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": "EventListener",
					"app.kubernetes.io/part-of":    "Triggers",
					"eventlistener":                eventListenerName,
				},
				OwnerReferences: []metav1.OwnerReference{*kmeta.NewControllerRef(makeEL())},
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{{
					Name:     eventListenerServicePortName,
					Protocol: corev1.ProtocolTCP,
					Port:     int32(*config.Port),
					TargetPort: intstr.IntOrString{
						IntVal: int32(eventListenerContainerPort),
					},
				}, metricsPort},
				Selector: map[string]string{
					"app.kubernetes.io/managed-by": "EventListener",
					"app.kubernetes.io/part-of":    "Triggers",
					"eventlistener":                eventListenerName,
				},
			},
		},
	}, {
		name: "EventListener with type: LoadBalancer",
		el:   makeEL(withServiceTypeLoadBalancer, withStatus),
		want: &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      generatedResourceName,
				Namespace: namespace,
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": "EventListener",
					"app.kubernetes.io/part-of":    "Triggers",
					"eventlistener":                eventListenerName,
				},
				OwnerReferences: []metav1.OwnerReference{*kmeta.NewControllerRef(makeEL())},
			},
			Spec: corev1.ServiceSpec{
				Type: "LoadBalancer",
				Ports: []corev1.ServicePort{{
					Name:     eventListenerServicePortName,
					Protocol: corev1.ProtocolTCP,
					Port:     int32(*config.Port),
					TargetPort: intstr.IntOrString{
						IntVal: int32(eventListenerContainerPort),
					},
				}, metricsPort},
				Selector: map[string]string{
					"app.kubernetes.io/managed-by": "EventListener",
					"app.kubernetes.io/part-of":    "Triggers",
					"eventlistener":                eventListenerName,
				},
			},
		}}, {
		name: "EventListener with service port: 80",
		el:   makeEL(withServicePort80, withStatus),
		want: &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      generatedResourceName,
				Namespace: namespace,
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": "EventListener",
					"app.kubernetes.io/part-of":    "Triggers",
					"eventlistener":                eventListenerName,
				},
				OwnerReferences: []metav1.OwnerReference{*kmeta.NewControllerRef(makeEL())},
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{{
					Name:     eventListenerServicePortName,
					Protocol: corev1.ProtocolTCP,
					Port:     int32(80),
					TargetPort: intstr.IntOrString{
						IntVal: int32(eventListenerContainerPort),
					},
				}, metricsPort},
				Selector: map[string]string{
					"app.kubernetes.io/managed-by": "EventListener",
					"app.kubernetes.io/part-of":    "Triggers",
					"eventlistener":                eventListenerName,
				},
			},
		}}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MakeService(context.Background(), tt.el, config)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("MakeService() did not return expected. -want, +got: %s", diff)
			}
		})
	}
}

func TestServicePort(t *testing.T) {

	k8sResSvcPort := int32(80)

	tests := []struct {
		name                string
		el                  *v1beta1.EventListener
		config              Config
		expectedServicePort corev1.ServicePort
		expectedServiceType corev1.ServiceType
	}{{
		name:   "EventListener with status",
		el:     makeEL(withStatus),
		config: *MakeConfig(),
		expectedServicePort: corev1.ServicePort{
			Name:     eventListenerServicePortName,
			Protocol: corev1.ProtocolTCP,
			Port:     int32(DefaultPort),
			TargetPort: intstr.IntOrString{
				IntVal: int32(eventListenerContainerPort),
			},
		},
	}, {
		name: "EventListener with TLS configuration",
		el: makeEL(withStatus, withTLSPort, func(el *v1beta1.EventListener) {
			el.Spec.Resources.KubernetesResource = &v1beta1.KubernetesResource{
				WithPodSpec: duckv1.WithPodSpec{
					Template: duckv1.PodSpecable{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Env: []corev1.EnvVar{{
									Name: "TLS_CERT",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "tls-secret-key",
											},
											Key: "tls.crt",
										},
									},
								}, {
									Name: "TLS_KEY",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "tls-secret-key",
											},
											Key: "tls.key",
										},
									},
								}},
							}},
						},
					},
				},
			}
		}),
		config: *MakeConfig(),
		expectedServicePort: corev1.ServicePort{
			Name:     eventListenerServiceTLSPortName,
			Protocol: corev1.ProtocolTCP,
			Port:     int32(8443),
			TargetPort: intstr.IntOrString{
				IntVal: int32(eventListenerContainerPort),
			},
		},
	}, {
		name: "EventListener with ServicePort 80 in KubernetesResource",
		el: makeEL(withStatus, withTLSPort, func(el *v1beta1.EventListener) {
			el.Spec.Resources.KubernetesResource = &v1beta1.KubernetesResource{
				ServicePort: &k8sResSvcPort,
			}
		}),
		config: *MakeConfig(),
		expectedServicePort: corev1.ServicePort{
			Name:     eventListenerServicePortName,
			Protocol: corev1.ProtocolTCP,
			Port:     k8sResSvcPort,
			TargetPort: intstr.IntOrString{
				IntVal: int32(eventListenerContainerPort),
			},
		},
	}, {
		name: "EventListener with ServicePort: 80",
		el:   makeEL(withStatus, withServicePort80),
		config: *MakeConfig(func(d *Config) {
			p := 80
			d.Port = &p
		}),
		expectedServicePort: corev1.ServicePort{
			Name:     eventListenerServicePortName,
			Protocol: corev1.ProtocolTCP,
			Port:     int32(80),
			TargetPort: intstr.IntOrString{
				IntVal: int32(eventListenerContainerPort),
			},
		},
		expectedServiceType: "LoadBalancer",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualPort := ServicePort(tt.el, tt.config)
			if diff := cmp.Diff(tt.expectedServicePort, actualPort); diff != "" {
				t.Errorf("ServicePort() did not return expected. -want, +got: %s", diff)
			}
		})
	}
}
