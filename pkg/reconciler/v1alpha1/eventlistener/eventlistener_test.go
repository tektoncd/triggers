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

package eventlistener

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"github.com/tektoncd/triggers/pkg/system"
	"github.com/tektoncd/triggers/test"
	bldr "github.com/tektoncd/triggers/test/builder"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8stest "k8s.io/client-go/testing"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	fakekubeclient "knative.dev/pkg/client/injection/kube/client/fake"
	cminformer "knative.dev/pkg/configmap/informer"
	"knative.dev/pkg/ptr"
	pkgreconciler "knative.dev/pkg/reconciler"
	rtesting "knative.dev/pkg/reconciler/testing"
)

var (
	eventListenerName     = "my-eventlistener"
	generatedResourceName = fmt.Sprintf("el-%s", eventListenerName)

	namespace         = "test-pipelines"
	namespaceResource = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}

	reconcilerNamespace         = "tekton-pipelines"
	reconcilerNamespaceResource = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: reconcilerNamespace,
		},
	}
	reconcileKey      = fmt.Sprintf("%s/%s", namespace, eventListenerName)
	updateLabel       = map[string]string{"update": "true"}
	updateAnnotation  = map[string]string{"update": "true"}
	updatedSa         = "updatedSa"
	updateTolerations = []corev1.Toleration{{
		Key:      "key",
		Operator: "Equal",
		Value:    "value",
		Effect:   "NoSchedule",
	}}
	updateNodeSelector           = map[string]string{"app": "test"}
	deploymentAvailableCondition = appsv1.DeploymentCondition{
		Type:    appsv1.DeploymentAvailable,
		Status:  corev1.ConditionTrue,
		Message: "Deployment has minimum availability",
		Reason:  "MinimumReplicasAvailable",
	}
	deploymentProgressingCondition = appsv1.DeploymentCondition{
		Type:    appsv1.DeploymentProgressing,
		Status:  corev1.ConditionTrue,
		Message: fmt.Sprintf("ReplicaSet \"%s\" has successfully progressed.", eventListenerName),
		Reason:  "NewReplicaSetAvailable",
	}

	// Standard labels added by EL reconciler to the underlying el-deployments/services
	generatedLabels = map[string]string{
		"app.kubernetes.io/managed-by": "EventListener",
		"app.kubernetes.io/part-of":    "Triggers",
		"eventlistener":                eventListenerName,
	}

	replicas int32 = 1
)

// getEventListenerTestAssets returns TestAssets that have been seeded with the
// given test.Resources r where r represents the state of the system
func getEventListenerTestAssets(t *testing.T, r test.Resources, c *Config) (test.Assets, context.CancelFunc) {
	t.Helper()
	ctx, _ := rtesting.SetupFakeContext(t)
	ctx, cancel := context.WithCancel(ctx)
	kubeClient := fakekubeclient.Get(ctx)
	// Fake client reactor chain ignores non handled reactors until v1.40.0
	// Test deployment/service resource should set their defaults
	// See: https://github.com/kubernetes/kubernetes/pull/73601
	kubeClient.PrependReactor("create", "deployments",
		func(action k8stest.Action) (bool, runtime.Object, error) {
			deployment := action.(k8stest.CreateActionImpl).GetObject().(*appsv1.Deployment)
			// Only add conditions when they don't exist
			// Test seeding expected resources "creates", which would create duplicates
			if len(deployment.Status.Conditions) == 0 {
				deployment.Status.Conditions = append(deployment.Status.Conditions, deploymentAvailableCondition)
				deployment.Status.Conditions = append(deployment.Status.Conditions, deploymentProgressingCondition)
			}
			// Pass modified resource and react using the default catch all reactor
			return kubeClient.ReactionChain[len(kubeClient.ReactionChain)-1].React(action)
		})
	clients := test.SeedResources(t, ctx, r)
	cmw := cminformer.NewInformedWatcher(clients.Kube, system.GetNamespace())
	if c == nil {
		c = makeConfig()
	}
	testAssets := test.Assets{
		Controller: NewController(*c)(ctx, cmw),
		Clients:    clients,
	}
	if la, ok := testAssets.Controller.Reconciler.(pkgreconciler.LeaderAware); ok {
		_ = la.Promote(pkgreconciler.UniversalBucket(), func(pkgreconciler.Bucket, types.NamespacedName) {})
	}
	return testAssets, cancel
}

// makeConfig is a helper to build a config that is consumed by an EventListener.
// It generates a default Config for the EventListener without any flags set and accepts functions for modification.
func makeConfig(ops ...func(d *Config)) *Config {
	c := Config{
		Image:              &DefaultImage,
		Port:               &DefaultPort,
		SetSecurityContext: &DefaultSetSecurityContext,
		ReadTimeOut:        &DefaultReadTimeout,
		WriteTimeOut:       &DefaultWriteTimeout,
		IdleTimeOut:        &DefaultIdleTimeout,
		TimeOutHandler:     &DefaultTimeOutHandler,
		PeriodSeconds:      &DefaultPeriodSeconds,
		FailureThreshold:   &DefaultFailureThreshold,

		StaticResourceLabels: DefaultStaticResourceLabels,
		SystemNamespace:      DefaultSystemNamespace,
	}

	for _, op := range ops {
		op(&c)
	}
	return &c
}

// makeEL is a helper to build an EventListener for tests.
// It generates a base EventListener that can then be modified by the passed in op function
// If no ops are specified, it generates a base EventListener with no triggers and no Status
func makeEL(ops ...func(el *v1alpha1.EventListener)) *v1alpha1.EventListener {
	e := bldr.EventListener(eventListenerName, namespace,
		bldr.EventListenerSpec(
			bldr.EventListenerServiceAccount("sa"),
		),
	)
	for _, op := range ops {
		op(e)
	}
	return e
}

// makeDeployment is a helper to build a Deployment that is created by an EventListener.
// It generates a basic Deployment for the simplest EventListener and accepts functions for modification
func makeDeployment(ops ...func(d *appsv1.Deployment)) *appsv1.Deployment {
	ownerRefs := makeEL().GetOwnerReference()

	d := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      generatedResourceName,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				*ownerRefs,
			},
			Labels: generatedLabels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: generatedLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: generatedLabels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "sa",
					Containers: []corev1.Container{{
						Name:  "event-listener",
						Image: DefaultImage,
						Ports: []corev1.ContainerPort{{
							ContainerPort: int32(eventListenerContainerPort),
							Protocol:      corev1.ProtocolTCP,
						}},
						LivenessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Path:   "/live",
									Scheme: corev1.URISchemeHTTP,
									Port:   intstr.FromInt(eventListenerContainerPort),
								},
							},
							PeriodSeconds:    int32(DefaultPeriodSeconds),
							FailureThreshold: int32(DefaultFailureThreshold),
						},
						ReadinessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Path:   "/live",
									Scheme: corev1.URISchemeHTTP,
									Port:   intstr.FromInt(eventListenerContainerPort),
								},
							},
							PeriodSeconds:    int32(DefaultPeriodSeconds),
							FailureThreshold: int32(DefaultFailureThreshold),
						},
						Args: []string{
							"-el-name", eventListenerName,
							"-el-namespace", namespace,
							"-port", strconv.Itoa(eventListenerContainerPort),
							"readtimeout", strconv.FormatInt(DefaultReadTimeout, 10),
							"writetimeout", strconv.FormatInt(DefaultWriteTimeout, 10),
							"idletimeout", strconv.FormatInt(DefaultIdleTimeout, 10),
							"timeouthandler", strconv.FormatInt(DefaultTimeOutHandler, 10),
						},
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "config-logging",
							MountPath: "/etc/config-logging",
						}},
						Env: []corev1.EnvVar{{
							Name: "SYSTEM_NAMESPACE",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									FieldPath: "metadata.namespace",
								},
							},
						}},
					}},
					Volumes: []corev1.Volume{{
						Name: "config-logging",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: eventListenerConfigMapName,
								},
							},
						},
					}},
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: ptr.Bool(true),
						RunAsUser:    ptr.Int64(65532),
					},
				},
			},
		},
		Status: appsv1.DeploymentStatus{
			Conditions: []appsv1.DeploymentCondition{
				deploymentAvailableCondition,
				deploymentProgressingCondition,
			},
		},
	}

	for _, op := range ops {
		op(&d)
	}
	return &d
}

var withTLSConfig = func(d *appsv1.Deployment) {
	d.Spec.Template.Spec.Containers = []corev1.Container{{
		Name:  "event-listener",
		Image: DefaultImage,
		Ports: []corev1.ContainerPort{{
			ContainerPort: int32(eventListenerContainerPort),
			Protocol:      corev1.ProtocolTCP,
		}},
		LivenessProbe: &corev1.Probe{
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/live",
					Scheme: corev1.URISchemeHTTPS,
					Port:   intstr.FromInt(eventListenerContainerPort),
				},
			},
			PeriodSeconds:    int32(DefaultPeriodSeconds),
			FailureThreshold: int32(DefaultFailureThreshold),
		},
		ReadinessProbe: &corev1.Probe{
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/live",
					Scheme: corev1.URISchemeHTTPS,
					Port:   intstr.FromInt(eventListenerContainerPort),
				},
			},
			PeriodSeconds:    int32(DefaultPeriodSeconds),
			FailureThreshold: int32(DefaultFailureThreshold),
		},
		Args: []string{
			"-el-name", eventListenerName,
			"-el-namespace", namespace,
			"-port", strconv.Itoa(eventListenerContainerPort),
			"-tls-cert", "/etc/triggers/tls/tls.pem",
			"-tls-key", "/etc/triggers/tls/tls.key",
		},
		VolumeMounts: []corev1.VolumeMount{{
			Name:      "config-logging",
			MountPath: "/etc/config-logging",
		}, {
			Name:      "https-connection",
			MountPath: "/etc/triggers/tls",
			ReadOnly:  true,
		}},
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
		}, {
			Name: "SYSTEM_NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		}},
	}}
	d.Spec.Template.Spec.Volumes = []corev1.Volume{{
		Name: "config-logging",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: eventListenerConfigMapName,
				},
			},
		},
	}, {
		Name: "https-connection",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: "tls-secret-key",
			},
		},
	}}
}

// makeWithPod is a helper to build a Knative Service that is created by an EventListener.
// It generates a basic Knative Service for the simplest EventListener and accepts functions for modification
func makeWithPod(ops ...func(d *duckv1.WithPod)) *duckv1.WithPod {
	ownerRefs := makeEL().GetOwnerReference()

	d := duckv1.WithPod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "serving.knative.dev/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      generatedResourceName,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				*ownerRefs,
			},
			Labels: generatedLabels,
		},
		Spec: duckv1.WithPodSpec{
			Template: duckv1.PodSpecable{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
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
						},
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "config-logging",
							MountPath: "/etc/config-logging",
							ReadOnly:  true,
						}},
						Env: []corev1.EnvVar{{
							Name:  "SYSTEM_NAMESPACE",
							Value: "test-pipelines",
						}},
					}},
					Volumes: []corev1.Volume{{
						Name: "config-logging",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: eventListenerConfigMapName,
								},
							},
						},
					}},
				},
			},
		},
	}
	for _, op := range ops {
		op(&d)
	}
	return &d
}

// makeService is a helper to build a Service that is created by an EventListener.
// It generates a basic Service for the simplest EventListener and accepts functions for modification.
func makeService(ops ...func(*corev1.Service)) *corev1.Service {
	ownerRefs := makeEL().GetOwnerReference()
	s := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      generatedResourceName,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				*ownerRefs,
			},
			Labels: generatedLabels,
		},
		Spec: corev1.ServiceSpec{
			Selector: generatedLabels,
			Ports: []corev1.ServicePort{{
				Name:     eventListenerServicePortName,
				Protocol: corev1.ProtocolTCP,
				Port:     int32(DefaultPort),
				TargetPort: intstr.IntOrString{
					IntVal: int32(eventListenerContainerPort),
				},
			}},
		},
	}

	for _, op := range ops {
		op(&s)
	}

	return &s
}

func logConfig(ns string) *corev1.ConfigMap {
	lc := defaultLoggingConfigMap()
	lc.Namespace = ns
	return lc
}

var withTLSPort = bldr.EventListenerStatus(
	bldr.EventListenerAddress(listenerHostname(generatedResourceName, namespace, 8443)),
)

var withKnativeStatus = bldr.EventListenerStatus(
	bldr.EventListenerCondition(
		v1alpha1.ServiceExists,
		corev1.ConditionFalse,
		"", "",
	),
	bldr.EventListenerCondition(
		v1alpha1.DeploymentExists,
		corev1.ConditionFalse,
		"", "",
	),
)

var withStatus = bldr.EventListenerStatus(
	bldr.EventListenerConfig(generatedResourceName),
	bldr.EventListenerAddress(listenerHostname(generatedResourceName, namespace, DefaultPort)),
	bldr.EventListenerCondition(
		v1alpha1.ServiceExists,
		corev1.ConditionTrue,
		"Service exists", "",
	),
	bldr.EventListenerCondition(
		v1alpha1.DeploymentExists,
		corev1.ConditionTrue,
		"Deployment exists", "",
	),
	bldr.EventListenerCondition(
		apis.ConditionType(appsv1.DeploymentAvailable),
		corev1.ConditionTrue,
		"Deployment has minimum availability",
		"MinimumReplicasAvailable",
	),
	bldr.EventListenerCondition(
		apis.ConditionType(appsv1.DeploymentProgressing),
		corev1.ConditionTrue,
		fmt.Sprintf("ReplicaSet \"%s\" has successfully progressed.", eventListenerName),
		"NewReplicaSetAvailable",
	),
)

func withAddedLabels(el *v1alpha1.EventListener) {
	el.Labels = updateLabel
}

func withAddedAnnotations(el *v1alpha1.EventListener) {
	el.Annotations = updateAnnotation
}
func withFinalizer(el *v1alpha1.EventListener) {
	el.Finalizers = []string{"eventlisteners.triggers.tekton.dev"}
}

func withFinalizerRemoved(el *v1alpha1.EventListener) {
	el.Finalizers = []string{}
}

func withControllerNamespace(el *v1alpha1.EventListener) {
	el.Namespace = reconcilerNamespace
}

func withDeletionTimestamp(el *v1alpha1.EventListener) {
	deletionTime := metav1.NewTime(time.Unix(1e9, 0))
	el.DeletionTimestamp = &deletionTime
}

func TestReconcile(t *testing.T) {
	err := os.Setenv("SYSTEM_NAMESPACE", "tekton-pipelines")
	if err != nil {
		t.Fatal(err)
	}

	customPort := 80

	configWithSetSecurityContextFalse := makeConfig(func(c *Config) {
		c.SetSecurityContext = ptr.Bool(false)
	})

	configWithPortSet := makeConfig(func(c *Config) {
		c.Port = &customPort
	})

	elWithStatus := makeEL(withStatus)

	elWithUpdatedSA := makeEL(withStatus, func(el *v1alpha1.EventListener) {
		el.Spec.ServiceAccountName = updatedSa
	})

	elWithNodePortServiceType := makeEL(withStatus, func(el *v1alpha1.EventListener) {
		el.Spec.Resources.KubernetesResource = &v1alpha1.KubernetesResource{
			ServiceType: corev1.ServiceTypeNodePort,
		}
	})

	elWithTolerations := makeEL(withStatus, func(el *v1alpha1.EventListener) {
		el.Spec.Resources.KubernetesResource = &v1alpha1.KubernetesResource{
			WithPodSpec: duckv1.WithPodSpec{
				Template: duckv1.PodSpecable{
					Spec: corev1.PodSpec{
						Tolerations: updateTolerations,
					},
				},
			},
		}
	})

	elWithNodeSelector := makeEL(withStatus, func(el *v1alpha1.EventListener) {
		el.Spec.Resources.KubernetesResource = &v1alpha1.KubernetesResource{
			WithPodSpec: duckv1.WithPodSpec{
				Template: duckv1.PodSpecable{
					Spec: corev1.PodSpec{
						NodeSelector: updateNodeSelector,
					},
				},
			},
		}
	})

	elWithReplicas := makeEL(withStatus, func(el *v1alpha1.EventListener) {
		el.Spec.Replicas = ptr.Int32(2)
	})

	elWithDeploymentReplicaFailure := makeEL(withStatus, func(el *v1alpha1.EventListener) {
		el.Status.SetCondition(&apis.Condition{
			Type: apis.ConditionType(appsv1.DeploymentReplicaFailure),
		})
	})

	elWithKubernetesResource := makeEL(withStatus, func(el *v1alpha1.EventListener) {
		el.Spec.Resources.KubernetesResource = &v1alpha1.KubernetesResource{
			ServiceType: corev1.ServiceTypeNodePort,
			WithPodSpec: duckv1.WithPodSpec{
				Template: duckv1.PodSpecable{
					Spec: corev1.PodSpec{
						NodeSelector:       map[string]string{"key": "value"},
						ServiceAccountName: "k8sresource",
						Containers: []corev1.Container{{
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.Quantity{Format: resource.DecimalSI},
									corev1.ResourceMemory: resource.Quantity{Format: resource.BinarySI},
								},
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.Quantity{Format: resource.DecimalSI},
									corev1.ResourceMemory: resource.Quantity{Format: resource.BinarySI},
								},
							},
						}},
					},
				},
			},
		}
	})

	elWithCustomResourceForEnv := makeEL(withStatus, withKnativeStatus, func(el *v1alpha1.EventListener) {
		el.Spec.Resources.CustomResource = &v1alpha1.CustomResource{
			RawExtension: test.RawExtension(t, duckv1.WithPod{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Service",
					APIVersion: "serving.knative.dev/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: generatedResourceName,
				},
				Spec: duckv1.WithPodSpec{Template: duckv1.PodSpecable{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Env: []corev1.EnvVar{{
								Name: "key",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{Name: "test"},
										Key:                  "a.crt",
									},
								},
							}},
						}},
					},
				}},
			}),
		}
	})
	elWithCustomResourceForNodeSelector := makeEL(withStatus, withKnativeStatus, func(el *v1alpha1.EventListener) {
		el.Spec.Resources.CustomResource = &v1alpha1.CustomResource{
			RawExtension: test.RawExtension(t, duckv1.WithPod{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Service",
					APIVersion: "serving.knative.dev/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: generatedResourceName,
				},
				Spec: duckv1.WithPodSpec{
					Template: duckv1.PodSpecable{
						Spec: corev1.PodSpec{
							NodeSelector: map[string]string{
								"hi": "hello",
							},
						},
					}},
			}),
		}
	})

	elWithCustomResourceForArgs := makeEL(withStatus, withKnativeStatus, func(el *v1alpha1.EventListener) {
		el.Spec.Resources.CustomResource = &v1alpha1.CustomResource{
			RawExtension: test.RawExtension(t, duckv1.WithPod{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Service",
					APIVersion: "serving.knative.dev/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: generatedResourceName,
				},
				Spec: duckv1.WithPodSpec{
					Template: duckv1.PodSpecable{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Args: []string{
									"--test" + "10",
								},
							}},
						},
					}},
			}),
		}
	})
	elWithCustomResourceForImage := makeEL(withStatus, withKnativeStatus, func(el *v1alpha1.EventListener) {
		el.Spec.Resources.CustomResource = &v1alpha1.CustomResource{
			RawExtension: test.RawExtension(t, duckv1.WithPod{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Service",
					APIVersion: "serving.knative.dev/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: generatedResourceName,
				},
				Spec: duckv1.WithPodSpec{
					Template: duckv1.PodSpecable{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Image: "test",
							}},
						},
					}},
			}),
		}
	})
	elWithCustomResourceForAnnotation := makeEL(withStatus, withKnativeStatus, func(el *v1alpha1.EventListener) {
		el.Spec.Resources.CustomResource = &v1alpha1.CustomResource{
			RawExtension: test.RawExtension(t, duckv1.WithPod{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Service",
					APIVersion: "serving.knative.dev/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: generatedResourceName,
					Annotations: map[string]string{
						"key": "value",
					},
				},
			}),
		}
	})

	elWithTLSConnection := makeEL(withStatus, withTLSPort, func(el *v1alpha1.EventListener) {
		el.Spec.Resources.KubernetesResource = &v1alpha1.KubernetesResource{
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
	})

	elWithKubernetesResourceForObjectMeta := makeEL(withStatus, func(el *v1alpha1.EventListener) {
		el.Spec.Resources.KubernetesResource = &v1alpha1.KubernetesResource{
			WithPodSpec: duckv1.WithPodSpec{
				Template: duckv1.PodSpecable{
					ObjectMeta: metav1.ObjectMeta{
						Labels:      map[string]string{"labelkey": "labelvalue"},
						Annotations: map[string]string{"annotationkey": "annotationvalue"},
					},
				},
			},
		}
	})

	elWithPortSet := makeEL(withStatus, bldr.EventListenerStatus(
		bldr.EventListenerAddress(listenerHostname(generatedResourceName, namespace, customPort)),
	))

	elDeployment := makeDeployment()
	elDeploymentWithLabels := makeDeployment(func(d *appsv1.Deployment) {
		d.Labels = mergeMaps(updateLabel, generatedLabels)
		d.Spec.Selector.MatchLabels = generatedLabels
		d.Spec.Template.Labels = mergeMaps(updateLabel, generatedLabels)
	})

	elDeploymentWithAnnotations := makeDeployment(func(d *appsv1.Deployment) {
		d.Annotations = updateAnnotation
	})

	elDeploymentWithUpdatedSA := makeDeployment(func(d *appsv1.Deployment) {
		d.Spec.Template.Spec.ServiceAccountName = updatedSa
	})

	elDeploymentWithTolerations := makeDeployment(func(d *appsv1.Deployment) {
		d.Spec.Template.Spec.Tolerations = updateTolerations
	})

	elDeploymentWithNodeSelector := makeDeployment(func(d *appsv1.Deployment) {
		d.Spec.Template.Spec.NodeSelector = updateNodeSelector
	})

	deploymentWithUpdatedReplicas := makeDeployment(func(d *appsv1.Deployment) {
		d.Spec.Replicas = ptr.Int32(5)
	})

	deploymentWithUpdatedReplicasNotConsidered := makeDeployment(func(d *appsv1.Deployment) {
		d.Spec.Replicas = ptr.Int32(2)
	})

	deploymentMissingVolumes := makeDeployment(func(d *appsv1.Deployment) {
		d.Spec.Template.Spec.Volumes = nil
		d.Spec.Template.Spec.Containers[0].VolumeMounts = nil
	})

	deploymentMissingSecurityContext := makeDeployment(func(d *appsv1.Deployment) {
		d.Spec.Template.Spec.Containers[0].SecurityContext = nil
	})

	deploymentForKubernetesResource := makeDeployment(func(d *appsv1.Deployment) {
		d.Spec.Template.Spec.ServiceAccountName = "k8sresource"
		d.Spec.Template.Spec.NodeSelector = map[string]string{"key": "value"}
		d.Spec.Template.Spec.Containers[0].Resources = corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.Quantity{Format: resource.DecimalSI},
				corev1.ResourceMemory: resource.Quantity{Format: resource.BinarySI},
			},
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.Quantity{Format: resource.DecimalSI},
				corev1.ResourceMemory: resource.Quantity{Format: resource.BinarySI},
			},
		}
	})

	deploymentWithTLSConnection := makeDeployment(withTLSConfig)

	deploymentForKubernetesResourceObjectMeta := makeDeployment(func(d *appsv1.Deployment) {
		d.Spec.Template.ObjectMeta.Labels = map[string]string{
			"app.kubernetes.io/managed-by": "EventListener",
			"app.kubernetes.io/part-of":    "Triggers",
			"eventlistener":                "my-eventlistener",
			"labelkey":                     "labelvalue"}
		d.Spec.Template.ObjectMeta.Annotations = map[string]string{"annotationkey": "annotationvalue"}
	})

	nodeSelectorForCustomResource := makeWithPod(func(data *duckv1.WithPod) {
		data.Spec.Template.Spec.NodeSelector = map[string]string{
			"hi": "hello",
		}
	})

	argsForCustomResource := makeWithPod(func(data *duckv1.WithPod) {
		data.Spec.Template.Spec.Containers[0].Args = []string{
			"--is-multi-ns=" + strconv.FormatBool(true),
		}
	})

	imageForCustomResource := makeWithPod(func(data *duckv1.WithPod) {
		data.Spec.Template.Spec.Containers[0].Image = DefaultImage
	})
	annotationForCustomResource := makeWithPod(func(data *duckv1.WithPod) {
		data.Annotations = map[string]string{
			"key": "value",
		}
	})
	envForCustomResource := makeWithPod(func(data *duckv1.WithPod) {
		data.Spec.Template.Spec.Containers[0].Env = []corev1.EnvVar{{
			Name: "key",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: "test"},
					Key:                  "a.crt",
				},
			},
		}, {
			Name:  "SYSTEM_NAMESPACE",
			Value: "test-pipelines",
		}}
	})

	elService := makeService()

	elServiceWithLabels := makeService(func(s *corev1.Service) {
		s.Labels = mergeMaps(updateLabel, generatedLabels)
		s.Spec.Selector = generatedLabels
	})

	elServiceWithAnnotation := makeService(func(s *corev1.Service) {
		s.Annotations = updateAnnotation
	})

	elServiceTypeNodePort := makeService(func(s *corev1.Service) {
		s.Spec.Type = corev1.ServiceTypeNodePort
	})

	elServiceWithUpdatedNodePort := makeService(func(s *corev1.Service) {
		s.Spec.Type = corev1.ServiceTypeNodePort
		s.Spec.Ports[0].NodePort = 30000
	})

	elServiceWithTLSConnection := makeService(func(s *corev1.Service) {
		s.Spec.Ports[0].Name = eventListenerServiceTLSPortName
		s.Spec.Ports[0].Port = int32(8443)
	})

	elServiceWithPortSet := makeService(func(s *corev1.Service) {
		s.Spec.Ports[0].Port = int32(customPort)
	})

	loggingConfigMap := defaultLoggingConfigMap()
	loggingConfigMap.ObjectMeta.Namespace = namespace
	reconcilerLoggingConfigMap := defaultLoggingConfigMap()
	reconcilerLoggingConfigMap.ObjectMeta.Namespace = reconcilerNamespace

	tests := []struct {
		name           string
		key            string
		config         *Config        // Config of the reconciler
		startResources test.Resources // State of the world before we call Reconcile
		endResources   test.Resources // Expected State of the world after calling Reconcile
	}{{
		name: "eventlistener creation",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{makeEL()},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{makeEL(withStatus)},
			Deployments:    []*appsv1.Deployment{makeDeployment()},
			Services:       []*corev1.Service{makeService()},
			ConfigMaps:     []*corev1.ConfigMap{loggingConfigMap},
		},
	}, {
		name: "eventlistener with additional label",
		key:  reconcileKey,
		// Resources before reconcile starts: EL has extra label that deployment/svc does not
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{makeEL(withStatus, withAddedLabels)},
			Deployments:    []*appsv1.Deployment{makeDeployment()},
			Services:       []*corev1.Service{makeService()},
		},
		// We expect the deployment and services to propagate the extra label
		// but the selectors in both Service and deployment should have the same
		// label
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{makeEL(withStatus, withAddedLabels)},
			Deployments:    []*appsv1.Deployment{elDeploymentWithLabels},
			Services:       []*corev1.Service{elServiceWithLabels},
			ConfigMaps:     []*corev1.ConfigMap{loggingConfigMap},
		},
	}, {
		name: "eventlistener with additional annotation",
		key:  reconcileKey,
		// Resources before reconcile starts: EL has annotation that deployment/svc does not
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{makeEL(withStatus, withAddedAnnotations)},
			Deployments:    []*appsv1.Deployment{makeDeployment()},
			Services:       []*corev1.Service{makeService()},
		},
		// We expect the deployment and services to propagate the annotations
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{makeEL(withStatus, withAddedAnnotations)},
			Deployments:    []*appsv1.Deployment{elDeploymentWithAnnotations},
			Services:       []*corev1.Service{elServiceWithAnnotation},
			ConfigMaps:     []*corev1.ConfigMap{loggingConfigMap},
		},
	}, {
		name: "eventlistener with updated service account",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithUpdatedSA},
			Deployments:    []*appsv1.Deployment{elDeploymentWithLabels},
			Services:       []*corev1.Service{elService},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithUpdatedSA},
			Deployments:    []*appsv1.Deployment{elDeploymentWithUpdatedSA},
			Services:       []*corev1.Service{elService},
			ConfigMaps:     []*corev1.ConfigMap{loggingConfigMap},
		},
	}, {
		name: "eventlistener with added tolerations",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithTolerations},
			Deployments:    []*appsv1.Deployment{elDeploymentWithLabels},
			Services:       []*corev1.Service{elService},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithTolerations},
			Deployments:    []*appsv1.Deployment{elDeploymentWithTolerations},
			Services:       []*corev1.Service{elService},
			ConfigMaps:     []*corev1.ConfigMap{loggingConfigMap},
		},
	}, {
		name: "eventlistener with added NodeSelector",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithNodeSelector},
			Deployments:    []*appsv1.Deployment{elDeploymentWithLabels},
			Services:       []*corev1.Service{elService},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithNodeSelector},
			Deployments:    []*appsv1.Deployment{elDeploymentWithNodeSelector},
			Services:       []*corev1.Service{elService},
			ConfigMaps:     []*corev1.ConfigMap{loggingConfigMap},
		},
	}, {
		name: "eventlistener with NodePort service",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithNodePortServiceType},
			Deployments:    []*appsv1.Deployment{elDeployment},
			Services:       []*corev1.Service{elServiceWithLabels},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithNodePortServiceType},
			Deployments:    []*appsv1.Deployment{elDeployment},
			Services:       []*corev1.Service{elServiceTypeNodePort},
			ConfigMaps:     []*corev1.ConfigMap{loggingConfigMap},
		},
	}, {
		// Check that if a user manually updates the labels for a service, we revert the change.
		name: "eventlistener with labels added to service",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithStatus},
			Services:       []*corev1.Service{elServiceWithLabels},
			Deployments:    []*appsv1.Deployment{elDeployment},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithStatus},
			Services:       []*corev1.Service{elService}, // We expect the service to drop the user added labels
			Deployments:    []*appsv1.Deployment{elDeployment},
			ConfigMaps:     []*corev1.ConfigMap{loggingConfigMap},
		},
	}, {
		// Check that if a user manually updates the annotations for a service, we do not revert the change.
		name: "eventlistener with annotations added to service",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithStatus},
			Services:       []*corev1.Service{elServiceWithAnnotation},
			Deployments:    []*appsv1.Deployment{elDeployment},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithStatus},
			Services:       []*corev1.Service{elServiceWithAnnotation},
			Deployments:    []*appsv1.Deployment{elDeployment},
			ConfigMaps:     []*corev1.ConfigMap{loggingConfigMap},
		},
	}, {
		// Checks that EL reconciler does not overwrite NodePort set by k8s (see #167)
		name: "eventlistener with updated NodePort service",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithNodePortServiceType},
			Deployments:    []*appsv1.Deployment{elDeployment},
			Services:       []*corev1.Service{elServiceWithUpdatedNodePort},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithNodePortServiceType},
			Deployments:    []*appsv1.Deployment{elDeployment},
			Services:       []*corev1.Service{elServiceWithUpdatedNodePort},
			ConfigMaps:     []*corev1.ConfigMap{loggingConfigMap},
		},
	}, {
		name: "eventlistener with labels applied to deployment",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithStatus},
			Deployments:    []*appsv1.Deployment{elDeploymentWithLabels},
			Services:       []*corev1.Service{elService},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithStatus},
			Deployments:    []*appsv1.Deployment{elDeployment},
			Services:       []*corev1.Service{elService},
			ConfigMaps:     []*corev1.ConfigMap{loggingConfigMap},
		},
	}, {
		// Check that if a user manually updates the annotations for a deployment, we do not revert the change.
		name: "eventlistener with annotations applied to deployment",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithStatus},
			Deployments:    []*appsv1.Deployment{elDeploymentWithAnnotations},
			Services:       []*corev1.Service{elService},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithStatus},
			Deployments:    []*appsv1.Deployment{elDeploymentWithAnnotations},
			Services:       []*corev1.Service{elService},
			ConfigMaps:     []*corev1.ConfigMap{loggingConfigMap},
		},
	}, {
		// Updating replicas on deployment itself is success because no replicas provided as part of eventlistener spec
		name: "eventlistener with updated replicas on deployment",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithStatus},
			Deployments:    []*appsv1.Deployment{deploymentWithUpdatedReplicas},
			Services:       []*corev1.Service{elService},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithStatus},
			Deployments:    []*appsv1.Deployment{deploymentWithUpdatedReplicas},
			Services:       []*corev1.Service{elService},
			ConfigMaps:     []*corev1.ConfigMap{loggingConfigMap},
		},
	}, {
		name: "eventlistener with failed update to deployment replicas",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithDeploymentReplicaFailure},
			Services:       []*corev1.Service{elService},
			Deployments:    []*appsv1.Deployment{elDeployment},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithStatus},
			Deployments:    []*appsv1.Deployment{elDeployment},
			Services:       []*corev1.Service{elService},
			ConfigMaps:     []*corev1.ConfigMap{loggingConfigMap},
		},
	}, {
		name: "eventlistener with updated config volumes",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{makeEL(withStatus)},
			Deployments:    []*appsv1.Deployment{deploymentMissingVolumes},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{makeEL(withStatus)},
			Deployments:    []*appsv1.Deployment{elDeployment},
			Services:       []*corev1.Service{elService},
			ConfigMaps:     []*corev1.ConfigMap{loggingConfigMap},
		},
	}, {
		// Checks that we do not overwrite replicas changed on the deployment itself when replicas provided as part of eventlistener spec
		name: "eventlistener with updated replicas",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithReplicas},
			Deployments:    []*appsv1.Deployment{deploymentWithUpdatedReplicas},
			Services:       []*corev1.Service{elService},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithReplicas},
			Deployments:    []*appsv1.Deployment{deploymentWithUpdatedReplicasNotConsidered},
			Services:       []*corev1.Service{elService},
			ConfigMaps:     []*corev1.ConfigMap{loggingConfigMap},
		},
	}, {
		name: "eventlistener with kubernetes resource",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithKubernetesResource},
			ConfigMaps:     []*corev1.ConfigMap{loggingConfigMap},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithKubernetesResource},
			ConfigMaps:     []*corev1.ConfigMap{loggingConfigMap},
			Deployments:    []*appsv1.Deployment{deploymentForKubernetesResource},
			Services:       []*corev1.Service{elServiceTypeNodePort},
		},
	}, {
		name: "eventlistener with kubernetes resource for podtemplate objectmeta",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithKubernetesResourceForObjectMeta},
			ConfigMaps:     []*corev1.ConfigMap{loggingConfigMap},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithKubernetesResourceForObjectMeta},
			ConfigMaps:     []*corev1.ConfigMap{loggingConfigMap},
			Deployments:    []*appsv1.Deployment{deploymentForKubernetesResourceObjectMeta},
			Services:       []*corev1.Service{elService},
		},
	}, {
		name: "eventlistener with TLS connection",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithTLSConnection},
			ConfigMaps:     []*corev1.ConfigMap{loggingConfigMap},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithTLSConnection},
			ConfigMaps:     []*corev1.ConfigMap{loggingConfigMap},
			Deployments:    []*appsv1.Deployment{deploymentWithTLSConnection},
			Services:       []*corev1.Service{elServiceWithTLSConnection},
		},
	}, {
		name: "eventlistener with security context",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithStatus},
			Deployments:    []*appsv1.Deployment{deploymentMissingSecurityContext},
			ConfigMaps:     []*corev1.ConfigMap{loggingConfigMap},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithStatus},
			Deployments:    []*appsv1.Deployment{elDeployment},
			Services:       []*corev1.Service{elService},
			ConfigMaps:     []*corev1.ConfigMap{loggingConfigMap},
		},
	}, {
		name:   "eventlistener with SetSecurityContext false",
		key:    reconcileKey,
		config: configWithSetSecurityContextFalse,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithStatus},
			Deployments:    []*appsv1.Deployment{deploymentMissingSecurityContext},
			ConfigMaps:     []*corev1.ConfigMap{loggingConfigMap},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithStatus},
			Deployments:    []*appsv1.Deployment{deploymentMissingSecurityContext},
			Services:       []*corev1.Service{elService},
			ConfigMaps:     []*corev1.ConfigMap{loggingConfigMap},
		},
	}, {
		name:   "eventlistener with port set in config",
		key:    reconcileKey,
		config: configWithPortSet,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithStatus},
			Deployments:    []*appsv1.Deployment{elDeployment},
			ConfigMaps:     []*corev1.ConfigMap{loggingConfigMap},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithPortSet},
			Deployments:    []*appsv1.Deployment{elDeployment},
			Services:       []*corev1.Service{elServiceWithPortSet},
			ConfigMaps:     []*corev1.ConfigMap{loggingConfigMap},
		},
	}, {
		name: "eventlistener with added env for custome resource",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithCustomResourceForEnv},
			ConfigMaps:     []*corev1.ConfigMap{loggingConfigMap},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithCustomResourceForEnv},
			ConfigMaps:     []*corev1.ConfigMap{loggingConfigMap},
			WithPod:        []*duckv1.WithPod{envForCustomResource},
		},
	}, {
		name: "eventlistener with added NodeSelector for custom resource",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithCustomResourceForNodeSelector},
			ConfigMaps:     []*corev1.ConfigMap{loggingConfigMap},
			WithPod:        []*duckv1.WithPod{nodeSelectorForCustomResource},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithCustomResourceForNodeSelector},
			ConfigMaps:     []*corev1.ConfigMap{loggingConfigMap},
			WithPod:        []*duckv1.WithPod{nodeSelectorForCustomResource},
		},
	}, {
		name: "eventlistener with added Args for custom resource",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithCustomResourceForArgs},
			ConfigMaps:     []*corev1.ConfigMap{loggingConfigMap},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithCustomResourceForArgs},
			ConfigMaps:     []*corev1.ConfigMap{loggingConfigMap},
			WithPod:        []*duckv1.WithPod{argsForCustomResource},
		},
	}, {
		name: "eventlistener with added Image for custom resource",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithCustomResourceForImage},
			ConfigMaps:     []*corev1.ConfigMap{loggingConfigMap},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithCustomResourceForImage},
			ConfigMaps:     []*corev1.ConfigMap{loggingConfigMap},
			WithPod:        []*duckv1.WithPod{imageForCustomResource},
		},
	}, {
		name: "eventlistener with annotation for custom resource",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithCustomResourceForAnnotation},
			ConfigMaps:     []*corev1.ConfigMap{loggingConfigMap},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{elWithCustomResourceForAnnotation},
			ConfigMaps:     []*corev1.ConfigMap{loggingConfigMap},
			WithPod:        []*duckv1.WithPod{annotationForCustomResource},
		},
	},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup with startResources
			testAssets, cancel := getEventListenerTestAssets(t, tt.startResources, tt.config)
			defer cancel()
			// Run Reconcile
			err := testAssets.Controller.Reconciler.Reconcile(context.Background(), tt.key)
			if err != nil {
				t.Errorf("eventlistener.Reconcile() returned error: %s", err)
				return
			}
			// Grab test resource results
			actualEndResources, err := test.GetResourcesFromClients(testAssets.Clients)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tt.endResources, *actualEndResources, cmpopts.IgnoreTypes(
				apis.Condition{}.LastTransitionTime.Inner.Time,
				metav1.ObjectMeta{}.Finalizers,
			)); diff != "" {
				t.Errorf("eventlistener.Reconcile() equality mismatch. Diff request body: -want +got: %s", diff)
			}
		})
	}
}

func TestReconcile_InvalidForCustomResource(t *testing.T) {
	err := os.Setenv("SYSTEM_NAMESPACE", "tekton-pipelines")
	if err != nil {
		t.Fatal(err)
	}

	elWithCustomResource := makeEL(withStatus, withKnativeStatus, func(el *v1alpha1.EventListener) {
		el.Spec.Resources.CustomResource = &v1alpha1.CustomResource{
			RawExtension: test.RawExtension(t, duckv1.WithPod{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Service",
					APIVersion: "serving.knative.dev/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:        generatedResourceName,
					Labels:      map[string]string{"serving.knative.dev/visibility": "cluster-local"},
					Annotations: map[string]string{"key": "value"},
					OwnerReferences: []metav1.OwnerReference{{
						APIVersion: "v1",
					}},
				},
				Spec: duckv1.WithPodSpec{Template: duckv1.PodSpecable{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "rev1",
						Labels:      map[string]string{"key": "value"},
						Annotations: map[string]string{"key": "value"},
					},
					Spec: corev1.PodSpec{
						Tolerations: updateTolerations,
						NodeSelector: map[string]string{
							"hi1": "hello1",
						},
						ServiceAccountName: "sa",
						Containers: []corev1.Container{{
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU: resource.Quantity{Format: resource.DecimalSI},
								},
							},
							Env: []corev1.EnvVar{{
								Name: "key",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{Name: "test"},
										Key:                  "a.crt",
									},
								},
							}},
						}},
					},
				}},
			}),
		}
	})
	customResource := makeWithPod(func(data *duckv1.WithPod) {
		data.ObjectMeta.Labels = map[string]string{"serving.knative.dev/visibility": "cluster-local1"}
		data.ObjectMeta.Annotations = map[string]string{"key1": "value1"}
		data.ObjectMeta.OwnerReferences = []metav1.OwnerReference{{
			APIVersion: "v2",
		}}
		data.Spec.Template.ObjectMeta.Name = "rev"
		data.Spec.Template.ObjectMeta.Labels = map[string]string{"key1": "value1"}
		data.Spec.Template.ObjectMeta.Annotations = map[string]string{"key1": "value1"}
		data.Spec.Template.Spec.NodeSelector = map[string]string{"hi": "hello"}
		data.Spec.Template.Spec.Containers[0].Env = []corev1.EnvVar{{
			Name:  "SYSTEM_NAMESPACE",
			Value: "test-pipelines",
		}, {
			Name: "key1",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: "test"},
					Key:                  "a.crt",
				},
			},
		}}
		data.Spec.Template.Spec.Containers[0].Args = []string{
			"--is-multi-ns1=" + strconv.FormatBool(true),
			"--el-namespace=" + "test",
		}
		data.Spec.Template.Spec.Containers[0].Image = "test"
		data.Spec.Template.Spec.Containers[0].Name = "test"
		data.Spec.Template.Spec.Containers[0].Ports = []corev1.ContainerPort{{
			Protocol: corev1.ProtocolUDP,
		}}
		data.Spec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{{
			Name:      "config-logging-dummy",
			MountPath: "/etc/config-logging-dummy",
			ReadOnly:  true,
		}}
		data.Spec.Template.Spec.Containers[0].Resources = corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.Quantity{Format: resource.BinarySI},
			},
		}
		data.Spec.Template.Spec.Volumes = []corev1.Volume{{
			Name: "config-logging1",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: eventListenerConfigMapName,
					},
				},
			},
		}}
		data.Spec.Template.Spec.ServiceAccountName = "test"
		data.Spec.Template.Spec.Tolerations = []corev1.Toleration{{
			Key: "key1",
		}}
	})

	loggingConfigMap := defaultLoggingConfigMap()
	loggingConfigMap.ObjectMeta.Namespace = namespace
	reconcilerLoggingConfigMap := defaultLoggingConfigMap()
	reconcilerLoggingConfigMap.ObjectMeta.Namespace = reconcilerNamespace

	tests := []struct {
		name           string
		key            string
		config         *Config        // Config of the reconciler
		startResources test.Resources // State of the world before we call Reconcile
		endResources   test.Resources // Expected State of the world after calling Reconcile
	}{
		{
			name: "eventlistener with custome resource",
			key:  reconcileKey,
			startResources: test.Resources{
				Namespaces:     []*corev1.Namespace{namespaceResource},
				EventListeners: []*v1alpha1.EventListener{elWithCustomResource},
				ConfigMaps:     []*corev1.ConfigMap{loggingConfigMap},
				WithPod:        []*duckv1.WithPod{customResource},
			},
			endResources: test.Resources{
				Namespaces:     []*corev1.Namespace{namespaceResource},
				EventListeners: []*v1alpha1.EventListener{elWithCustomResource},
				ConfigMaps:     []*corev1.ConfigMap{loggingConfigMap},
				WithPod:        []*duckv1.WithPod{customResource},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup with startResources
			testAssets, cancel := getEventListenerTestAssets(t, tt.startResources, tt.config)
			defer cancel()
			// Run Reconcile
			err := testAssets.Controller.Reconciler.Reconcile(context.Background(), tt.key)
			if err != nil {
				t.Errorf("eventlistener.Reconcile() returned error: %s", err)
				return
			}
			// Grab test resource results
			actualEndResources, err := test.GetResourcesFromClients(testAssets.Clients)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tt.endResources, *actualEndResources, cmpopts.IgnoreTypes(
				apis.Condition{}.LastTransitionTime.Inner.Time,
				metav1.ObjectMeta{}.Finalizers,
			)); diff == "" {
				t.Errorf("eventlistener.Reconcile() equality mismatch. Diff request body: -want +got: %s", diff)
			}
		})
	}
}

func TestReconcile_Delete(t *testing.T) {
	tests := []struct {
		name           string
		key            string
		config         *Config        // Config of the reconciler
		startResources test.Resources // State of the world before we call Reconcile
		endResources   test.Resources // Expected State of the world after calling Reconcile
	}{{
		name: "delete eventlistener with remaining eventlisteners",
		key:  fmt.Sprintf("%s/%s", namespace, "el-2"),
		startResources: test.Resources{
			Namespaces: []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{
				makeEL(),
				// TODO: makeEL take name, ns as args
				makeEL(withFinalizer, withDeletionTimestamp, func(el *v1alpha1.EventListener) { el.Name = "el-2" })},
			ConfigMaps: []*corev1.ConfigMap{logConfig(namespace)},
		},
		endResources: test.Resources{
			Namespaces: []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{
				makeEL(withFinalizerRemoved, withDeletionTimestamp, func(el *v1alpha1.EventListener) { el.Name = "el-2" }),
				makeEL(),
			},
			ConfigMaps: []*corev1.ConfigMap{logConfig(namespace)},
		},
	}, {
		name: "delete last eventlistener in reconciler namespace",
		key:  fmt.Sprintf("%s/%s", reconcilerNamespace, eventListenerName),
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{reconcilerNamespaceResource},
			EventListeners: []*v1alpha1.EventListener{makeEL(withControllerNamespace, withFinalizer, withDeletionTimestamp)},
			ConfigMaps:     []*corev1.ConfigMap{logConfig(reconcilerNamespace)},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{reconcilerNamespaceResource},
			EventListeners: []*v1alpha1.EventListener{makeEL(withControllerNamespace, withDeletionTimestamp, withFinalizerRemoved)},
			ConfigMaps:     []*corev1.ConfigMap{logConfig(reconcilerNamespace)}, // We should not delete the logging configMap
		},
	}, {
		name: "delete last eventlistener in namespace",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{makeEL(withFinalizer, withDeletionTimestamp)},
			ConfigMaps:     []*corev1.ConfigMap{logConfig(namespace)},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{makeEL(withDeletionTimestamp, withFinalizerRemoved)},
		},
	}, {
		name: "delete last eventlistener in namespace with no logging config",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{makeEL(withFinalizer, withDeletionTimestamp)},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1alpha1.EventListener{makeEL(withDeletionTimestamp, withFinalizerRemoved)},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup with startResources
			testAssets, cancel := getEventListenerTestAssets(t, tt.startResources, tt.config)
			defer cancel()
			// Run Reconcile
			err := testAssets.Controller.Reconciler.Reconcile(context.Background(), tt.key)
			if err != nil {
				t.Errorf("eventlistener.Reconcile() returned error: %s", err)
				return
			}
			// Grab test resource results
			actualEndResources, err := test.GetResourcesFromClients(testAssets.Clients)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tt.endResources, *actualEndResources, cmpopts.IgnoreTypes(apis.Condition{}.LastTransitionTime.Inner.Time)); diff != "" {
				t.Errorf("eventlistener.Reconcile() equality mismatch. Diff request body: -want +got: %s", diff)
			}
		})
	}
}

func Test_getServicePort(t *testing.T) {
	tests := []struct {
		name                string
		el                  *v1alpha1.EventListener
		config              Config
		expectedServicePort corev1.ServicePort
	}{{
		name:   "EventListener with status",
		el:     makeEL(withStatus),
		config: *makeConfig(),
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
		el: makeEL(withStatus, withTLSPort, func(el *v1alpha1.EventListener) {
			el.Spec.Resources.KubernetesResource = &v1alpha1.KubernetesResource{
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
		config: *makeConfig(),
		expectedServicePort: corev1.ServicePort{
			Name:     eventListenerServiceTLSPortName,
			Protocol: corev1.ProtocolTCP,
			Port:     int32(8443),
			TargetPort: intstr.IntOrString{
				IntVal: int32(eventListenerContainerPort),
			},
		},
	},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualPort := getServicePort(tt.el, tt.config)
			if diff := cmp.Diff(tt.expectedServicePort, actualPort); diff != "" {
				t.Errorf("getServicePort() did not return expected. -want, +got: %s", diff)
			}
		})
	}
}

func Test_wrapError(t *testing.T) {
	tests := []struct {
		name           string
		error1, error2 error
		expectedError  error
	}{{
		name:          "Both error empty",
		error1:        nil,
		error2:        nil,
		expectedError: nil,
	}, {
		name:          "Error one empty",
		error1:        nil,
		error2:        fmt.Errorf("error"),
		expectedError: fmt.Errorf("error"),
	}, {
		name:          "Error two empty",
		error1:        fmt.Errorf("error"),
		error2:        nil,
		expectedError: fmt.Errorf("error"),
	}, {
		name:          "Both errors",
		error1:        fmt.Errorf("error1"),
		error2:        fmt.Errorf("error2"),
		expectedError: fmt.Errorf("error1 : error2"),
	}}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			actualError := wrapError(tests[i].error1, tests[i].error2)
			// Compare strings since errors have unexported fields that panic
			var expectedErrorString string
			var actualErrorString string
			if tests[i].expectedError != nil {
				expectedErrorString = tests[i].expectedError.Error()
			}
			if actualError != nil {
				actualErrorString = actualError.Error()
			}
			if diff := cmp.Diff(expectedErrorString, actualErrorString); diff != "" {
				t.Errorf("wrapError() did not return expected. -want, +got: %s", diff)
			}
		})
	}
}

func TestGenerateResourceLabels(t *testing.T) {
	staticResourceLabels := map[string]string{
		"app.kubernetes.io/managed-by": "EventListener",
		"app.kubernetes.io/part-of":    "Triggers",
	}

	expectedLabels := mergeMaps(staticResourceLabels, map[string]string{"eventlistener": eventListenerName})
	actualLabels := GenerateResourceLabels(eventListenerName, staticResourceLabels)
	if diff := cmp.Diff(expectedLabels, actualLabels); diff != "" {
		t.Errorf("mergeLabels() did not return expected. -want, +got: %s", diff)
	}
}

func Test_generateObjectMeta(t *testing.T) {
	blockOwnerDeletion := true
	isController := true
	tests := []struct {
		name               string
		el                 *v1alpha1.EventListener
		expectedObjectMeta metav1.ObjectMeta
	}{{
		name: "Empty EventListener",
		el:   bldr.EventListener(eventListenerName, ""),
		expectedObjectMeta: metav1.ObjectMeta{
			Namespace: "",
			Name:      "",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion:         "triggers.tekton.dev/v1alpha1",
				Kind:               "EventListener",
				Name:               eventListenerName,
				UID:                "",
				Controller:         &isController,
				BlockOwnerDeletion: &blockOwnerDeletion,
			}},
			Labels: generatedLabels,
		},
	}, {
		name: "EventListener with Configuration",
		el: bldr.EventListener(eventListenerName, "",
			bldr.EventListenerStatus(
				bldr.EventListenerConfig("generatedName"),
			),
		),
		expectedObjectMeta: metav1.ObjectMeta{
			Namespace: "",
			Name:      "generatedName",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion:         "triggers.tekton.dev/v1alpha1",
				Kind:               "EventListener",
				Name:               eventListenerName,
				UID:                "",
				Controller:         &isController,
				BlockOwnerDeletion: &blockOwnerDeletion,
			}},
			Labels: generatedLabels,
		},
	}, {
		name: "EventListener with Labels",
		el: bldr.EventListener(eventListenerName, "",
			bldr.EventListenerMeta(
				bldr.Label("k", "v"),
			),
		),
		expectedObjectMeta: metav1.ObjectMeta{
			Namespace: "",
			Name:      "",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion:         "triggers.tekton.dev/v1alpha1",
				Kind:               "EventListener",
				Name:               eventListenerName,
				UID:                "",
				Controller:         &isController,
				BlockOwnerDeletion: &blockOwnerDeletion,
			}},
			Labels: mergeMaps(map[string]string{"k": "v"}, generatedLabels),
		},
	}, {
		name: "EventListener with Annotation",
		el: bldr.EventListener(eventListenerName, "",
			bldr.EventListenerMeta(
				bldr.Annotation("k", "v"),
			),
		),
		expectedObjectMeta: metav1.ObjectMeta{
			Namespace: "",
			Name:      "",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion:         "triggers.tekton.dev/v1alpha1",
				Kind:               "EventListener",
				Name:               eventListenerName,
				UID:                "",
				Controller:         &isController,
				BlockOwnerDeletion: &blockOwnerDeletion,
			}},
			Labels:      generatedLabels,
			Annotations: map[string]string{"k": "v"},
		},
	}}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			actualObjectMeta := generateObjectMeta(tests[i].el, DefaultStaticResourceLabels)
			if diff := cmp.Diff(tests[i].expectedObjectMeta, actualObjectMeta); diff != "" {
				t.Errorf("generateObjectMeta() did not return expected. -want, +got: %s", diff)
			}
		})
	}
}
