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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	reconcilersource "knative.dev/eventing/pkg/reconciler/source"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"
	"knative.dev/pkg/ptr"
)

func TestDeployment(t *testing.T) {
	t.Setenv("METRICS_PROMETHEUS_PORT", "9000")
	t.Setenv("SYSTEM_NAMESPACE", "tekton-pipelines")

	config := *MakeConfig()
	labels := map[string]string{
		"app.kubernetes.io/managed-by": "EventListener",
		"app.kubernetes.io/part-of":    "Triggers",
		"eventlistener":                eventListenerName,
	}

	tests := []struct {
		name string
		el   *v1beta1.EventListener
		want *appsv1.Deployment
	}{{
		name: "vanilla",
		el:   makeEL(),
		want: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "",
				Namespace:       namespace,
				Labels:          labels,
				OwnerReferences: []metav1.OwnerReference{*kmeta.NewControllerRef(makeEL())},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: labels,
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: labels,
					},
					Spec: corev1.PodSpec{
						ServiceAccountName: "sa",
						Containers: []corev1.Container{
							MakeContainer(makeEL(), &reconcilersource.EmptyVarsGenerator{}, config,
								mustAddDeployBits(t, makeEL(), config),
								addCertsForSecureConnection(config)),
						},
						SecurityContext: &strongerSecurityPolicy,
					},
				},
			},
		},
	}, {
		name: "with replicas",
		el: makeEL(func(el *v1beta1.EventListener) {
			el.Spec.Resources.KubernetesResource = &v1beta1.KubernetesResource{
				Replicas: ptr.Int32(5),
			}
		}),
		want: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "",
				Namespace:       namespace,
				Labels:          labels,
				OwnerReferences: []metav1.OwnerReference{*kmeta.NewControllerRef(makeEL())},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: ptr.Int32(5),
				Selector: &metav1.LabelSelector{
					MatchLabels: labels,
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: labels,
					},
					Spec: corev1.PodSpec{
						ServiceAccountName: "sa",
						Containers: []corev1.Container{
							MakeContainer(makeEL(), &reconcilersource.EmptyVarsGenerator{}, config,
								mustAddDeployBits(t, makeEL(), config),
								addCertsForSecureConnection(config)),
						},
						SecurityContext: &strongerSecurityPolicy,
					},
				},
			},
		},
	}, {
		name: "with tolerations",
		el: makeEL(func(el *v1beta1.EventListener) {
			el.Spec.Resources.KubernetesResource = &v1beta1.KubernetesResource{
				WithPodSpec: duckv1.WithPodSpec{
					Template: duckv1.PodSpecable{
						Spec: corev1.PodSpec{
							Tolerations: []corev1.Toleration{{
								Key:   "foo",
								Value: "bar",
							}},
						},
					},
				},
			}
		}),
		want: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "",
				Namespace:       namespace,
				Labels:          labels,
				OwnerReferences: []metav1.OwnerReference{*kmeta.NewControllerRef(makeEL())},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: labels,
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: labels,
					},
					Spec: corev1.PodSpec{
						ServiceAccountName: "sa",
						Containers: []corev1.Container{
							MakeContainer(makeEL(), &reconcilersource.EmptyVarsGenerator{}, config,
								mustAddDeployBits(t, makeEL(), config),
								addCertsForSecureConnection(config)),
						},
						SecurityContext: &strongerSecurityPolicy,
						Tolerations: []corev1.Toleration{{
							Key:   "foo",
							Value: "bar",
						}},
					},
				},
			},
		},
	}, {
		name: "with node selector",
		el: makeEL(func(el *v1beta1.EventListener) {
			el.Spec.Resources.KubernetesResource = &v1beta1.KubernetesResource{
				WithPodSpec: duckv1.WithPodSpec{
					Template: duckv1.PodSpecable{
						Spec: corev1.PodSpec{
							NodeSelector: map[string]string{
								"foo": "bar",
							},
						},
					},
				},
			}
		}),
		want: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "",
				Namespace:       namespace,
				Labels:          labels,
				OwnerReferences: []metav1.OwnerReference{*kmeta.NewControllerRef(makeEL())},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: labels,
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: labels,
					},
					Spec: corev1.PodSpec{
						ServiceAccountName: "sa",
						Containers: []corev1.Container{
							MakeContainer(makeEL(), &reconcilersource.EmptyVarsGenerator{}, config,
								mustAddDeployBits(t, makeEL(), config),
								addCertsForSecureConnection(config)),
						},
						SecurityContext: &strongerSecurityPolicy,
						NodeSelector: map[string]string{
							"foo": "bar",
						},
					},
				},
			},
		},
	}, {
		name: "with service account",
		el: makeEL(func(el *v1beta1.EventListener) {
			el.Spec.Resources.KubernetesResource = &v1beta1.KubernetesResource{
				WithPodSpec: duckv1.WithPodSpec{
					Template: duckv1.PodSpecable{
						Spec: corev1.PodSpec{
							ServiceAccountName: "bob",
						},
					},
				},
			}
		}),
		want: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "",
				Namespace:       namespace,
				Labels:          labels,
				OwnerReferences: []metav1.OwnerReference{*kmeta.NewControllerRef(makeEL())},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: labels,
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: labels,
					},
					Spec: corev1.PodSpec{
						ServiceAccountName: "bob",
						Containers: []corev1.Container{
							MakeContainer(makeEL(), &reconcilersource.EmptyVarsGenerator{}, config,
								mustAddDeployBits(t, makeEL(), config),
								addCertsForSecureConnection(config)),
						},
						SecurityContext: &strongerSecurityPolicy,
					},
				},
			},
		},
	}, {
		name: "with TLS",
		el:   makeEL(withTLSEnvFrom("Bill")),
		want: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "",
				Namespace:       namespace,
				Labels:          labels,
				OwnerReferences: []metav1.OwnerReference{*kmeta.NewControllerRef(makeEL())},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: labels,
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: labels,
					},
					Spec: corev1.PodSpec{
						ServiceAccountName: "sa",
						Containers: []corev1.Container{
							MakeContainer(makeEL(withTLSEnvFrom("Bill")), &reconcilersource.EmptyVarsGenerator{}, config,
								mustAddDeployBits(t, makeEL(withTLSEnvFrom("Bill")), config),
								addCertsForSecureConnection(config)),
						},
						Volumes: []corev1.Volume{{
							Name: "https-connection",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "Bill",
								},
							},
						}},
						SecurityContext: &strongerSecurityPolicy,
					},
				},
			},
		},
	}, {
		name: "with Affinity and TopologySpreadConstraints",
		el:   makeEL(withAffinityAndTopologySpreadConstraints()),
		want: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "",
				Namespace:       namespace,
				Labels:          labels,
				OwnerReferences: []metav1.OwnerReference{*kmeta.NewControllerRef(makeEL())},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: labels,
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: labels,
					},
					Spec: corev1.PodSpec{
						ServiceAccountName: "sa",
						Affinity: &corev1.Affinity{
							NodeAffinity: &corev1.NodeAffinity{
								RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
									NodeSelectorTerms: []corev1.NodeSelectorTerm{{
										MatchExpressions: []corev1.NodeSelectorRequirement{{
											Key:      "topology.kubernetes.io/zone",
											Operator: corev1.NodeSelectorOpIn,
											Values:   []string{"antarctica-east1"},
										}},
									}},
								},
							},
						},
						TopologySpreadConstraints: []corev1.TopologySpreadConstraint{{
							MaxSkew: 1,
						}},
						Containers: []corev1.Container{
							MakeContainer(makeEL(), &reconcilersource.EmptyVarsGenerator{}, config,
								mustAddDeployBits(t, makeEL(), config),
								addCertsForSecureConnection(config)),
						},
						SecurityContext: &strongerSecurityPolicy,
					},
				},
			},
		},
	}, {
		name: "with container probes",
		el:   makeEL(setProbes()),
		want: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "",
				Namespace:       namespace,
				Labels:          labels,
				OwnerReferences: []metav1.OwnerReference{*kmeta.NewControllerRef(makeEL())},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: labels,
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: labels,
					},
					Spec: corev1.PodSpec{
						ServiceAccountName: "sa",
						Containers: []corev1.Container{
							MakeContainer(makeEL(setProbes()), &reconcilersource.EmptyVarsGenerator{}, config,
								mustAddDeployBits(t, makeEL(setProbes()), config),
								addCertsForSecureConnection(config)),
						},
						SecurityContext: &strongerSecurityPolicy,
					},
				},
			},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MakeDeployment(context.Background(), tt.el, &reconcilersource.EmptyVarsGenerator{}, config)
			if err != nil {
				t.Fatalf("MakeDeployment() = %v", err)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("MakeDeployment() did not return expected. -want, +got: %s", diff)
			}
		})
	}
}

func TestDeploymentError(t *testing.T) {
	t.Setenv("METRICS_PROMETHEUS_PORT", "bad")
	got, err := MakeDeployment(context.Background(), makeEL(), &reconcilersource.EmptyVarsGenerator{}, *MakeConfig())
	if err == nil {
		t.Fatalf("MakeDeployment() = %v, wanted error", got)
	}
}

func withTLSEnvFrom(name string) func(*v1beta1.EventListener) {
	return func(el *v1beta1.EventListener) {
		el.Spec.Resources.KubernetesResource = &v1beta1.KubernetesResource{
			WithPodSpec: duckv1.WithPodSpec{
				Template: duckv1.PodSpecable{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Env: []corev1.EnvVar{{
								Name: "TLS_CERT",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										Key: "cert",
										LocalObjectReference: corev1.LocalObjectReference{
											Name: name,
										},
									},
								},
							}, {
								Name: "TLS_KEY",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										Key: "key",
										LocalObjectReference: corev1.LocalObjectReference{
											Name: name,
										},
									},
								},
							}},
						}},
					},
				},
			},
		}
	}
}

func withAffinityAndTopologySpreadConstraints() func(*v1beta1.EventListener) {
	return func(el *v1beta1.EventListener) {
		el.Spec.Resources.KubernetesResource = &v1beta1.KubernetesResource{
			WithPodSpec: duckv1.WithPodSpec{
				Template: duckv1.PodSpecable{
					Spec: corev1.PodSpec{
						Affinity: &corev1.Affinity{
							NodeAffinity: &corev1.NodeAffinity{
								RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
									NodeSelectorTerms: []corev1.NodeSelectorTerm{{
										MatchExpressions: []corev1.NodeSelectorRequirement{{
											Key:      "topology.kubernetes.io/zone",
											Operator: corev1.NodeSelectorOpIn,
											Values:   []string{"antarctica-east1"},
										}},
									}},
								},
							},
						},
						TopologySpreadConstraints: []corev1.TopologySpreadConstraint{{
							MaxSkew: 1,
						}},
					},
				},
			},
		}
	}
}

func mustAddDeployBits(t *testing.T, el *v1beta1.EventListener, c Config) ContainerOption {
	opt, err := addDeploymentBits(el, c)
	if err != nil {
		t.Fatalf("addDeploymentBits() = %v", err)
	}
	return opt
}

func setProbes() func(*v1beta1.EventListener) {
	return func(el *v1beta1.EventListener) {
		el.Spec.Resources.KubernetesResource = &v1beta1.KubernetesResource{
			WithPodSpec: duckv1.WithPodSpec{
				Template: duckv1.PodSpecable{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{
							ReadinessProbe: &corev1.Probe{
								InitialDelaySeconds: 10,
							},
							LivenessProbe: &corev1.Probe{
								InitialDelaySeconds: 10,
							},
							StartupProbe: &corev1.Probe{
								InitialDelaySeconds: 10,
							},
						}},
					},
				},
			},
		}
	}
}
