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
	"strconv"
	"testing"
	"time"

	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"github.com/tektoncd/triggers/pkg/reconciler/eventlistener/resources"
	"github.com/tektoncd/triggers/pkg/reconciler/metrics"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"github.com/tektoncd/triggers/pkg/system"
	"github.com/tektoncd/triggers/test"
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
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
	fakekubeclient "knative.dev/pkg/client/injection/kube/client/fake"
	cminformer "knative.dev/pkg/configmap/informer"
	"knative.dev/pkg/kmeta"
	"knative.dev/pkg/ptr"
	pkgreconciler "knative.dev/pkg/reconciler"
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
)

// compareCondition compares two conditions based on their Type field.
func compareCondition(x, y apis.Condition) bool {
	return x.Type < y.Type
}

// compareEnv compares two conditions based on their Type field.
func compareEnv(x, y corev1.EnvVar) bool {
	return x.Name < y.Name
}

// getEventListenerTestAssets returns TestAssets that have been seeded with the
// given test.Resources r where r represents the state of the system
func getEventListenerTestAssets(t *testing.T, r test.Resources, c *resources.Config) (test.Assets, context.CancelFunc) {
	t.Helper()
	ctx, _ := test.SetupFakeContext(t)
	ctx = metrics.WithClient(ctx)
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
		c = resources.MakeConfig()
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

// makeEL is a helper to build an EventListener for tests.
// It generates a base EventListener that can then be modified by the passed in op function
// If no ops are specified, it generates a base EventListener with no triggers and no Status
func makeEL(ops ...func(el *v1beta1.EventListener)) *v1beta1.EventListener {
	e := &v1beta1.EventListener{
		ObjectMeta: metav1.ObjectMeta{
			Name:      eventListenerName,
			Namespace: namespace,
		},
		Spec: v1beta1.EventListenerSpec{
			ServiceAccountName: "sa",
		},
	}
	for _, op := range ops {
		op(e)
	}
	return e
}

// makeDeployment is a helper to build a Deployment that is created by an EventListener.
// It generates a basic Deployment for the simplest EventListener and accepts functions for modification
func makeDeployment(ops ...func(d *appsv1.Deployment)) *appsv1.Deployment {
	ownerRefs := kmeta.NewControllerRef(makeEL())

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
						Image: resources.DefaultImage,
						Ports: []corev1.ContainerPort{{
							ContainerPort: int32(eventListenerContainerPort),
							Protocol:      corev1.ProtocolTCP,
						}, {
							ContainerPort: int32(9000),
							Protocol:      corev1.ProtocolTCP,
						}},
						LivenessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{
									Path:   "/live",
									Scheme: corev1.URISchemeHTTP,
									Port:   intstr.FromInt(eventListenerContainerPort),
								},
							},
							PeriodSeconds:    int32(resources.DefaultPeriodSeconds),
							FailureThreshold: int32(resources.DefaultFailureThreshold),
						},
						ReadinessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{
									Path:   "/live",
									Scheme: corev1.URISchemeHTTP,
									Port:   intstr.FromInt(eventListenerContainerPort),
								},
							},
							PeriodSeconds:    int32(resources.DefaultPeriodSeconds),
							FailureThreshold: int32(resources.DefaultFailureThreshold),
						},
						Args: []string{
							"--el-name=" + eventListenerName,
							"--el-namespace=" + namespace,
							"--port=" + strconv.Itoa(eventListenerContainerPort),
							"--readtimeout=" + strconv.FormatInt(resources.DefaultReadTimeout, 10),
							"--writetimeout=" + strconv.FormatInt(resources.DefaultWriteTimeout, 10),
							"--idletimeout=" + strconv.FormatInt(resources.DefaultIdleTimeout, 10),
							"--timeouthandler=" + strconv.FormatInt(resources.DefaultTimeOutHandler, 10),
							"--httpclient-readtimeout=" + strconv.FormatInt(resources.DefaultHTTPClientReadTimeOut, 10),
							"--httpclient-keep-alive=" + strconv.FormatInt(resources.DefaultHTTPClientKeepAlive, 10),
							"--httpclient-tlshandshaketimeout=" + strconv.FormatInt(resources.DefaultHTTPClientTLSHandshakeTimeout, 10),
							"--httpclient-responseheadertimeout=" + strconv.FormatInt(resources.DefaultHTTPClientResponseHeaderTimeout, 10),
							"--httpclient-expectcontinuetimeout=" + strconv.FormatInt(resources.DefaultHTTPClientExpectContinueTimeout, 10),
							"--is-multi-ns=false",
							"--payload-validation=true",
							"--cloudevent-uri=",
							"--tls-cert=",
							"--tls-key=",
						},
						Env: []corev1.EnvVar{{
							Name: "K_LOGGING_CONFIG",
						}, {
							Name:  "K_METRICS_CONFIG",
							Value: `{"Domain":"","Component":"","PrometheusPort":0,"PrometheusHost":"","ConfigMap":null}`,
						}, {
							Name:  "K_TRACING_CONFIG",
							Value: `{"backend":"","debug":"false","sample-rate":"0"}`,
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
							Value: strconv.FormatInt(resources.DefaultTimeOutHandler, 10),
						}, {
							Name: "SYSTEM_NAMESPACE",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									APIVersion: "v1",
									FieldPath:  "metadata.namespace",
								},
							},
						}, {
							Name:  "METRICS_PROMETHEUS_PORT",
							Value: "9000",
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
					}},
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: ptr.Bool(true),
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
	// Replace the 2 TLS args with the right values
	container := &d.Spec.Template.Spec.Containers[0]

	// Set probes to use HTTPS
	container.LivenessProbe.HTTPGet.Scheme = corev1.URISchemeHTTPS
	container.ReadinessProbe.HTTPGet.Scheme = corev1.URISchemeHTTPS

	// Pass keys as container args
	for i, arg := range container.Args {
		if arg == "--tls-key=" {
			container.Args[i] = "--tls-key=/etc/triggers/tls/tls.key"
		}
		if arg == "--tls-cert=" {
			container.Args[i] = "--tls-cert=/etc/triggers/tls/tls.crt"
		}
	}
	container.VolumeMounts = []corev1.VolumeMount{{
		Name:      "https-connection",
		MountPath: "/etc/triggers/tls",
		ReadOnly:  true,
	}}

	container.Env = append(container.Env, corev1.EnvVar{
		Name: "TLS_CERT",
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "tls-secret-key",
				},
				Key: "tls.crt",
			},
		},
	}, corev1.EnvVar{
		Name: "TLS_KEY",
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "tls-secret-key",
				},
				Key: "tls.key",
			},
		},
	})
	d.Spec.Template.Spec.Volumes = []corev1.Volume{{
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
	ownerRefs := kmeta.NewControllerRef(makeEL())

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
			Labels:          generatedLabels,
			ResourceVersion: "testresourceversion",
		},
		Spec: duckv1.WithPodSpec{
			Template: duckv1.PodSpecable{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "event-listener",
						Image: resources.DefaultImage,
						Ports: []corev1.ContainerPort{{
							ContainerPort: int32(eventListenerContainerPort),
							Protocol:      corev1.ProtocolTCP,
						}},
						Args: []string{
							"--el-name=" + eventListenerName,
							"--el-namespace=" + namespace,
							"--port=" + strconv.Itoa(eventListenerContainerPort),
							"--readtimeout=" + strconv.FormatInt(resources.DefaultReadTimeout, 10),
							"--writetimeout=" + strconv.FormatInt(resources.DefaultWriteTimeout, 10),
							"--idletimeout=" + strconv.FormatInt(resources.DefaultIdleTimeout, 10),
							"--timeouthandler=" + strconv.FormatInt(resources.DefaultTimeOutHandler, 10),
							"--httpclient-readtimeout=" + strconv.FormatInt(resources.DefaultHTTPClientReadTimeOut, 10),
							"--httpclient-keep-alive=" + strconv.FormatInt(resources.DefaultHTTPClientKeepAlive, 10),
							"--httpclient-tlshandshaketimeout=" + strconv.FormatInt(resources.DefaultHTTPClientTLSHandshakeTimeout, 10),
							"--httpclient-responseheadertimeout=" + strconv.FormatInt(resources.DefaultHTTPClientResponseHeaderTimeout, 10),
							"--httpclient-expectcontinuetimeout=" + strconv.FormatInt(resources.DefaultHTTPClientExpectContinueTimeout, 10),
							"--is-multi-ns=" + strconv.FormatBool(false),
							"--payload-validation=" + strconv.FormatBool(true),
							"--cloudevent-uri=",
						},
						Env: []corev1.EnvVar{{
							Name: "K_LOGGING_CONFIG",
						}, {
							Name:  "K_METRICS_CONFIG",
							Value: `{"Domain":"","Component":"","PrometheusPort":0,"PrometheusHost":"","ConfigMap":null}`,
						}, {
							Name:  "K_TRACING_CONFIG",
							Value: `{"backend":"","debug":"false","sample-rate":"0"}`,
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
							Value: strconv.FormatInt(resources.DefaultTimeOutHandler, 10),
						}, {
							Name:  "SYSTEM_NAMESPACE",
							Value: namespace,
						}, {
							Name:  "METRICS_PROMETHEUS_PORT",
							Value: "9000",
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
						ReadinessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{
									Path:   "/live",
									Scheme: corev1.URISchemeHTTP,
								},
							},
							SuccessThreshold: 1,
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
	ownerRefs := kmeta.NewControllerRef(makeEL())
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
				Port:     int32(resources.DefaultPort),
				TargetPort: intstr.IntOrString{
					IntVal: int32(eventListenerContainerPort),
				},
			}, {
				Name:     eventListenerMetricsPortName,
				Protocol: corev1.ProtocolTCP,
				Port:     int32(9000),
				TargetPort: intstr.IntOrString{
					IntVal: int32(eventListenerMetricsPort),
				},
			}},
		},
	}

	for _, op := range ops {
		op(&s)
	}

	return &s
}

func withTLSPort(el *v1beta1.EventListener) {
	el.Status.SetAddress(resources.ListenerHostname(el, *resources.MakeConfig(func(c *resources.Config) {
		x := 8443
		c.Port = &x
	})))
}

func withKnativeStatus(el *v1beta1.EventListener) {
	el.Status.Status = duckv1.Status{
		Conditions: []apis.Condition{{
			Type:   v1alpha1.ServiceExists,
			Status: corev1.ConditionFalse,
		}, {
			Type:   v1alpha1.DeploymentExists,
			Status: corev1.ConditionFalse,
		}, {
			Type:   apis.ConditionReady,
			Status: corev1.ConditionFalse,
		}},
	}
}

func withStatus(el *v1beta1.EventListener) {
	el.Status = v1beta1.EventListenerStatus{
		Status: duckv1.Status{
			Conditions: []apis.Condition{{
				Type:    apis.ConditionType(appsv1.DeploymentAvailable),
				Status:  corev1.ConditionTrue,
				Message: "Deployment has minimum availability",
				Reason:  "MinimumReplicasAvailable",
			}, {
				Type:    v1beta1.DeploymentExists,
				Status:  corev1.ConditionTrue,
				Message: "Deployment exists",
			}, {
				Type:    apis.ConditionType(appsv1.DeploymentProgressing),
				Status:  corev1.ConditionTrue,
				Message: fmt.Sprintf("ReplicaSet \"%s\" has successfully progressed.", eventListenerName),
				Reason:  "NewReplicaSetAvailable",
			}, {
				Type:    apis.ConditionReady,
				Status:  corev1.ConditionTrue,
				Message: "EventListener is ready",
			}, {
				Type:    v1alpha1.ServiceExists,
				Status:  corev1.ConditionTrue,
				Message: "Service exists",
			}},
		},
		Configuration: v1beta1.EventListenerConfig{
			GeneratedResourceName: generatedResourceName,
		},
	}

	el.Status.SetAddress(resources.ListenerHostname(el, *resources.MakeConfig()))
}

func withAddedLabels(el *v1beta1.EventListener) {
	el.Labels = updateLabel
}

func withAddedAnnotations(el *v1beta1.EventListener) {
	el.Annotations = updateAnnotation
}

func withControllerNamespace(el *v1beta1.EventListener) {
	el.Namespace = reconcilerNamespace
}

func withDeletionTimestamp(el *v1beta1.EventListener) {
	deletionTime := metav1.NewTime(time.Unix(1e9, 0))
	el.DeletionTimestamp = &deletionTime
}

func TestReconcile(t *testing.T) {
	t.Setenv("METRICS_PROMETHEUS_PORT", "9000")
	t.Setenv("SYSTEM_NAMESPACE", "tekton-pipelines")

	customPort := 80

	configWithSetSecurityContextFalse := resources.MakeConfig(func(c *resources.Config) {
		c.SetSecurityContext = ptr.Bool(false)
	})

	configWithSetEventListenerEventEnable := resources.MakeConfig(func(c *resources.Config) {
		c.SetEventListenerEvent = ptr.String("enable")
	})

	configWithPortSet := resources.MakeConfig(func(c *resources.Config) {
		c.Port = &customPort
	})

	elWithStatus := makeEL(withStatus)

	elWithUpdatedSA := makeEL(withStatus, func(el *v1beta1.EventListener) {
		el.Spec.ServiceAccountName = updatedSa
	})

	elWithNodePortServiceType := makeEL(withStatus, func(el *v1beta1.EventListener) {
		el.Spec.Resources.KubernetesResource = &v1beta1.KubernetesResource{
			ServiceType: corev1.ServiceTypeNodePort,
		}
	})

	elWithTolerations := makeEL(withStatus, func(el *v1beta1.EventListener) {
		el.Spec.Resources.KubernetesResource = &v1beta1.KubernetesResource{
			WithPodSpec: duckv1.WithPodSpec{
				Template: duckv1.PodSpecable{
					Spec: corev1.PodSpec{
						Tolerations: updateTolerations,
					},
				},
			},
		}
	})

	elWithNodeSelector := makeEL(withStatus, func(el *v1beta1.EventListener) {
		el.Spec.Resources.KubernetesResource = &v1beta1.KubernetesResource{
			WithPodSpec: duckv1.WithPodSpec{
				Template: duckv1.PodSpecable{
					Spec: corev1.PodSpec{
						NodeSelector: updateNodeSelector,
					},
				},
			},
		}
	})

	elWithReplicas := makeEL(withStatus, func(el *v1beta1.EventListener) {
		el.Spec.Resources.KubernetesResource = &v1beta1.KubernetesResource{
			Replicas: ptr.Int32(2),
		}
	})

	elWithDeploymentReplicaFailure := makeEL(withStatus, func(el *v1beta1.EventListener) {
		el.Status.SetCondition(&apis.Condition{
			Type: apis.ConditionType(appsv1.DeploymentReplicaFailure),
		})
	})

	elWithKubernetesResource := makeEL(withStatus, func(el *v1beta1.EventListener) {
		el.Spec.Resources.KubernetesResource = &v1beta1.KubernetesResource{
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

	elWithCustomResourceForEnv := makeEL(withStatus, withKnativeStatus, func(el *v1beta1.EventListener) {
		el.Spec.Resources.CustomResource = &v1beta1.CustomResource{
			RawExtension: test.RawExtension(t, duckv1.WithPod{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Service",
					APIVersion: "serving.knative.dev/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            generatedResourceName,
					ResourceVersion: "testresourceversion",
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
	elWithCustomResourceForNodeSelector := makeEL(withStatus, withKnativeStatus, func(el *v1beta1.EventListener) {
		el.Spec.Resources.CustomResource = &v1beta1.CustomResource{
			RawExtension: test.RawExtension(t, duckv1.WithPod{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Service",
					APIVersion: "serving.knative.dev/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            generatedResourceName,
					ResourceVersion: "testresourceversion",
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

	elWithCustomResourceForArgs := makeEL(withStatus, withKnativeStatus, func(el *v1beta1.EventListener) {
		el.Spec.Resources.CustomResource = &v1beta1.CustomResource{
			RawExtension: test.RawExtension(t, duckv1.WithPod{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Service",
					APIVersion: "serving.knative.dev/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            generatedResourceName,
					ResourceVersion: "testresourceversion",
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
	elWithCustomResourceForImage := makeEL(withStatus, withKnativeStatus, func(el *v1beta1.EventListener) {
		el.Spec.Resources.CustomResource = &v1beta1.CustomResource{
			RawExtension: test.RawExtension(t, duckv1.WithPod{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Service",
					APIVersion: "serving.knative.dev/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            generatedResourceName,
					ResourceVersion: "testresourceversion",
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
	elWithCustomResourceForAnnotation := makeEL(withStatus, withKnativeStatus, func(el *v1beta1.EventListener) {
		el.Spec.Resources.CustomResource = &v1beta1.CustomResource{
			RawExtension: test.RawExtension(t, duckv1.WithPod{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Service",
					APIVersion: "serving.knative.dev/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            generatedResourceName,
					ResourceVersion: "testresourceversion",
					Annotations: map[string]string{
						"key": "value",
					},
				},
			}),
		}
	})

	elWithTLSConnection := makeEL(withStatus, withTLSPort, func(el *v1beta1.EventListener) {
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
	})

	elWithKubernetesResourceForObjectMeta := makeEL(withStatus, func(el *v1beta1.EventListener) {
		el.Spec.Resources.KubernetesResource = &v1beta1.KubernetesResource{
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

	elWithPortSet := makeEL(withStatus, func(el *v1beta1.EventListener) {
		el.Status.Address = &duckv1beta1.Addressable{
			URL: &apis.URL{
				Scheme: "http",
				Host: resources.ListenerHostname(el, *resources.MakeConfig(func(c *resources.Config) {
					c.Port = &customPort
				})),
			},
		}
	})

	elDeployment := makeDeployment()
	elDeploymentWithLabels := makeDeployment(func(d *appsv1.Deployment) {
		d.Labels = kmeta.UnionMaps(updateLabel, generatedLabels)
		d.Spec.Selector.MatchLabels = generatedLabels
		d.Spec.Template.Labels = kmeta.UnionMaps(updateLabel, generatedLabels)
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
		d.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{}
		d.Spec.Template.Spec.Containers[0].SecurityContext = &corev1.SecurityContext{}
	})

	deploymentEventListenerEvent := makeDeployment(func(d *appsv1.Deployment) {
		d.Spec.Template.Spec.Containers[0].Env = []corev1.EnvVar{{
			Name: "K_LOGGING_CONFIG",
		}, {
			Name:  "K_METRICS_CONFIG",
			Value: `{"Domain":"","Component":"","PrometheusPort":0,"PrometheusHost":"","ConfigMap":null}`,
		}, {
			Name:  "K_TRACING_CONFIG",
			Value: `{"backend":"","debug":"false","sample-rate":"0"}`,
		}, {
			Name:  "NAMESPACE",
			Value: namespace,
		}, {
			Name:  "NAME",
			Value: eventListenerName,
		}, {
			Name:  "EL_EVENT",
			Value: "enable",
		}, {
			Name:  "K_SINK_TIMEOUT",
			Value: strconv.FormatInt(resources.DefaultTimeOutHandler, 10),
		}, {
			Name: "SYSTEM_NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					APIVersion: "v1",
					FieldPath:  "metadata.namespace",
				},
			},
		}, {
			Name:  "METRICS_PROMETHEUS_PORT",
			Value: "9000",
		}}
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

	imageForCustomResource := makeWithPod(func(data *duckv1.WithPod) {
		data.Spec.Template.Spec.Containers[0].Image = resources.DefaultImage
	})
	annotationForCustomResource := makeWithPod(func(data *duckv1.WithPod) {
		data.Annotations = map[string]string{
			"key": "value",
		}
	})
	envForCustomResource := makeWithPod(func(data *duckv1.WithPod) {
		data.Spec.Template.Spec.Containers[0].Env = []corev1.EnvVar{{
			Name: "K_LOGGING_CONFIG",
		}, {
			Name:  "K_METRICS_CONFIG",
			Value: `{"Domain":"","Component":"","PrometheusPort":0,"PrometheusHost":"","ConfigMap":null}`,
		}, {
			Name:  "K_TRACING_CONFIG",
			Value: `{"backend":"","debug":"false","sample-rate":"0"}`,
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
			Value: strconv.FormatInt(resources.DefaultTimeOutHandler, 10),
		}, {
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
		}, {
			Name:  "METRICS_PROMETHEUS_PORT",
			Value: "9000",
		}}
	})

	elService := makeService()

	elServiceWithLabels := makeService(func(s *corev1.Service) {
		s.Labels = kmeta.UnionMaps(updateLabel, generatedLabels)
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

	tests := []struct {
		name           string
		key            string
		config         *resources.Config // Config of the reconciler
		startResources test.Resources    // State of the world before we call Reconcile
		endResources   test.Resources    // Expected State of the world after calling Reconcile
	}{{
		name: "eventlistener creation",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{makeEL()},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{makeEL(withStatus)},
			Deployments:    []*appsv1.Deployment{makeDeployment()},
			Services:       []*corev1.Service{makeService()},
		},
	}, {
		name: "eventlistener with additional label",
		key:  reconcileKey,
		// Resources before reconcile starts: EL has extra label that deployment/svc does not
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{makeEL(withStatus, withAddedLabels)},
			Deployments:    []*appsv1.Deployment{makeDeployment()},
			Services:       []*corev1.Service{makeService()},
		},
		// We expect the deployment and services to propagate the extra label
		// but the selectors in both Service and deployment should have the same
		// label
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{makeEL(withStatus, withAddedLabels)},
			Deployments:    []*appsv1.Deployment{elDeploymentWithLabels},
			Services:       []*corev1.Service{elServiceWithLabels},
		},
	}, {
		name: "eventlistener with additional annotation",
		key:  reconcileKey,
		// Resources before reconcile starts: EL has annotation that deployment/svc does not
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{makeEL(withStatus, withAddedAnnotations)},
			Deployments:    []*appsv1.Deployment{makeDeployment()},
			Services:       []*corev1.Service{makeService()},
		},
		// We expect the deployment and services to propagate the annotations
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{makeEL(withStatus, withAddedAnnotations)},
			Deployments:    []*appsv1.Deployment{elDeploymentWithAnnotations},
			Services:       []*corev1.Service{elServiceWithAnnotation},
		},
	}, {
		name: "eventlistener with updated service account",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithUpdatedSA},
			Deployments:    []*appsv1.Deployment{elDeploymentWithLabels},
			Services:       []*corev1.Service{elService},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithUpdatedSA},
			Deployments:    []*appsv1.Deployment{elDeploymentWithUpdatedSA},
			Services:       []*corev1.Service{elService},
		},
	}, {
		name: "eventlistener with added tolerations",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithTolerations},
			Deployments:    []*appsv1.Deployment{elDeploymentWithLabels},
			Services:       []*corev1.Service{elService},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithTolerations},
			Deployments:    []*appsv1.Deployment{elDeploymentWithTolerations},
			Services:       []*corev1.Service{elService},
		},
	}, {
		name: "eventlistener with added NodeSelector",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithNodeSelector},
			Deployments:    []*appsv1.Deployment{elDeploymentWithLabels},
			Services:       []*corev1.Service{elService},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithNodeSelector},
			Deployments:    []*appsv1.Deployment{elDeploymentWithNodeSelector},
			Services:       []*corev1.Service{elService},
		},
	}, {
		name: "eventlistener with NodePort service",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithNodePortServiceType},
			Deployments:    []*appsv1.Deployment{elDeployment},
			Services:       []*corev1.Service{elServiceWithLabels},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithNodePortServiceType},
			Deployments:    []*appsv1.Deployment{elDeployment},
			Services:       []*corev1.Service{elServiceTypeNodePort},
		},
	}, {
		// Check that if a user manually updates the labels for a service, we revert the change.
		name: "eventlistener with labels added to service",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithStatus},
			Services:       []*corev1.Service{elServiceWithLabels},
			Deployments:    []*appsv1.Deployment{elDeployment},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithStatus},
			Services:       []*corev1.Service{elService}, // We expect the service to drop the user added labels
			Deployments:    []*appsv1.Deployment{elDeployment},
		},
	}, {
		// Check that if a user manually updates the annotations for a service, we do not revert the change.
		name: "eventlistener with annotations added to service",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithStatus},
			Services:       []*corev1.Service{elServiceWithAnnotation},
			Deployments:    []*appsv1.Deployment{elDeployment},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithStatus},
			Services:       []*corev1.Service{elServiceWithAnnotation},
			Deployments:    []*appsv1.Deployment{elDeployment},
		},
	}, {
		// Checks that EL reconciler does not overwrite NodePort set by k8s (see #167)
		name: "eventlistener with updated NodePort service",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithNodePortServiceType},
			Deployments:    []*appsv1.Deployment{elDeployment},
			Services:       []*corev1.Service{elServiceWithUpdatedNodePort},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithNodePortServiceType},
			Deployments:    []*appsv1.Deployment{elDeployment},
			Services:       []*corev1.Service{elServiceWithUpdatedNodePort},
		},
	}, {
		name: "eventlistener with labels applied to deployment",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithStatus},
			Deployments:    []*appsv1.Deployment{elDeploymentWithLabels},
			Services:       []*corev1.Service{elService},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithStatus},
			Deployments:    []*appsv1.Deployment{elDeployment},
			Services:       []*corev1.Service{elService},
		},
	}, {
		// Check that if a user manually updates the annotations for a deployment, we do not revert the change.
		name: "eventlistener with annotations applied to deployment",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithStatus},
			Deployments:    []*appsv1.Deployment{elDeploymentWithAnnotations},
			Services:       []*corev1.Service{elService},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithStatus},
			Deployments:    []*appsv1.Deployment{elDeploymentWithAnnotations},
			Services:       []*corev1.Service{elService},
		},
	}, {
		// Updating replicas on deployment itself is success because no replicas provided as part of eventlistener spec
		name: "eventlistener with updated replicas on deployment",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithStatus},
			Deployments:    []*appsv1.Deployment{deploymentWithUpdatedReplicas},
			Services:       []*corev1.Service{elService},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithStatus},
			Deployments:    []*appsv1.Deployment{deploymentWithUpdatedReplicas},
			Services:       []*corev1.Service{elService},
		},
	}, {
		name: "eventlistener with failed update to deployment replicas",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithDeploymentReplicaFailure},
			Services:       []*corev1.Service{elService},
			Deployments:    []*appsv1.Deployment{elDeployment},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithStatus},
			Deployments:    []*appsv1.Deployment{elDeployment},
			Services:       []*corev1.Service{elService},
		},
	}, {
		name: "eventlistener with updated config volumes",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{makeEL(withStatus)},
			Deployments:    []*appsv1.Deployment{deploymentMissingVolumes},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{makeEL(withStatus)},
			Deployments:    []*appsv1.Deployment{elDeployment},
			Services:       []*corev1.Service{elService},
		},
	}, {
		// Checks that we do not overwrite replicas changed on the deployment itself when replicas provided as part of eventlistener spec
		name: "eventlistener with updated replicas",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithReplicas},
			Deployments:    []*appsv1.Deployment{deploymentWithUpdatedReplicas},
			Services:       []*corev1.Service{elService},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithReplicas},
			Deployments:    []*appsv1.Deployment{deploymentWithUpdatedReplicasNotConsidered},
			Services:       []*corev1.Service{elService},
		},
	}, {
		name: "eventlistener with kubernetes resource",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithKubernetesResource},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithKubernetesResource},
			Deployments:    []*appsv1.Deployment{deploymentForKubernetesResource},
			Services:       []*corev1.Service{elServiceTypeNodePort},
		},
	}, {
		name: "eventlistener with kubernetes resource for podtemplate objectmeta",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithKubernetesResourceForObjectMeta},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithKubernetesResourceForObjectMeta},
			Deployments:    []*appsv1.Deployment{deploymentForKubernetesResourceObjectMeta},
			Services:       []*corev1.Service{elService},
		},
	}, {
		name: "eventlistener with TLS connection",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithTLSConnection},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithTLSConnection},
			Deployments:    []*appsv1.Deployment{deploymentWithTLSConnection},
			Services:       []*corev1.Service{elServiceWithTLSConnection},
		},
	}, {
		name: "eventlistener with security context",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithStatus},
			Deployments:    []*appsv1.Deployment{deploymentMissingSecurityContext},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithStatus},
			Deployments:    []*appsv1.Deployment{elDeployment},
			Services:       []*corev1.Service{elService},
		},
	}, {
		name:   "eventlistener with SetSecurityContext false",
		key:    reconcileKey,
		config: configWithSetSecurityContextFalse,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithStatus},
			Deployments:    []*appsv1.Deployment{elDeployment},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithStatus},
			Deployments:    []*appsv1.Deployment{deploymentMissingSecurityContext}, // SecurityContext is cleared
			Services:       []*corev1.Service{elService},
		},
	}, {
		name:   "eventlistener with SetEventListenerEvent enable",
		key:    reconcileKey,
		config: configWithSetEventListenerEventEnable,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithStatus},
			Deployments:    []*appsv1.Deployment{elDeployment},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithStatus},
			Deployments:    []*appsv1.Deployment{deploymentEventListenerEvent}, // SecurityContext is cleared
			Services:       []*corev1.Service{elService},
		},
	}, {
		name:   "eventlistener with port set in config",
		key:    reconcileKey,
		config: configWithPortSet,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithStatus},
			Deployments:    []*appsv1.Deployment{elDeployment},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithPortSet},
			Deployments:    []*appsv1.Deployment{elDeployment},
			Services:       []*corev1.Service{elServiceWithPortSet},
		},
	}, {
		name: "eventlistener with added env for custom resource",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithCustomResourceForEnv},
			WithPod:        []*duckv1.WithPod{envForCustomResource},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithCustomResourceForEnv},
			WithPod:        []*duckv1.WithPod{envForCustomResource},
		},
	}, {
		name: "eventlistener with added NodeSelector for custom resource",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithCustomResourceForNodeSelector},
			WithPod:        []*duckv1.WithPod{nodeSelectorForCustomResource},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithCustomResourceForNodeSelector},
			WithPod:        []*duckv1.WithPod{nodeSelectorForCustomResource},
		},
	}, {
		// If a user provides custom args, we ignore them and create the EL with the standard set of args
		name: "CustomResource EventListener with user provided custom args",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithCustomResourceForArgs},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithCustomResourceForArgs},
			WithPod:        []*duckv1.WithPod{makeWithPod()},
		},
	}, {
		name: "eventlistener with added Image for custom resource",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithCustomResourceForImage},
			WithPod:        []*duckv1.WithPod{imageForCustomResource},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithCustomResourceForImage},
			WithPod:        []*duckv1.WithPod{imageForCustomResource},
		},
	}, {
		name: "eventlistener with annotation for custom resource",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithCustomResourceForAnnotation},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithCustomResourceForAnnotation},
			WithPod:        []*duckv1.WithPod{annotationForCustomResource},
		},
	}, {
		name: "eventlistener with cleanup test to ensure no k8s resource exist after upgrading to customresource",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithCustomResourceForNodeSelector},
			WithPod:        []*duckv1.WithPod{nodeSelectorForCustomResource},
			Deployments:    []*appsv1.Deployment{makeDeployment()},
			Services:       []*corev1.Service{makeService()},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithCustomResourceForNodeSelector},
			WithPod:        []*duckv1.WithPod{nodeSelectorForCustomResource},
		},
	}, {
		name: "reconcile removes old finalizers", // See #1243
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces: []*corev1.Namespace{namespaceResource},
			// The initial EL needs to have a Status set else the update from the generated Reconcile
			// overwrites the update from removeFinalizer
			EventListeners: []*v1beta1.EventListener{makeEL(withStatus, func(el *v1beta1.EventListener) {
				el.Finalizers = append(el.Finalizers, "eventlisteners.triggers.tekton.dev")
			})},
		},
		endResources: test.Resources{
			Namespaces: []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{makeEL(withStatus, func(el *v1beta1.EventListener) {
				el.Finalizers = []string{} // Finalizer should be removed
			})},
			Deployments: []*appsv1.Deployment{makeDeployment()},
			Services:    []*corev1.Service{makeService()},
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
			if diff := cmp.Diff(tt.endResources, *actualEndResources,
				cmpopts.IgnoreFields(apis.Condition{}, "LastTransitionTime.Inner.Time"),
				cmpopts.SortSlices(compareCondition),
				cmpopts.SortSlices(compareEnv)); diff != "" {
				t.Errorf("eventlistener.Reconcile() equality mismatch. Diff request body: -want +got: %s", diff)
			}
		})
	}
}

func TestReconcile_InvalidForCustomResource(t *testing.T) {
	t.Setenv("SYSTEM_NAMESPACE", "tekton-pipelines")

	elWithCustomResource := makeEL(withStatus, withKnativeStatus, func(el *v1beta1.EventListener) {
		el.Spec.Resources.CustomResource = &v1beta1.CustomResource{
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
		data.Spec.Template.Spec.Containers[0].Resources = corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.Quantity{Format: resource.BinarySI},
			},
		}
		data.Spec.Template.Spec.ServiceAccountName = "test"
		data.Spec.Template.Spec.Tolerations = []corev1.Toleration{{
			Key: "key1",
		}}
	})

	tests := []struct {
		name           string
		key            string
		config         *resources.Config // Config of the reconciler
		startResources test.Resources    // State of the world before we call Reconcile
		endResources   test.Resources    // Expected State of the world after calling Reconcile
	}{{
		name: "eventlistener with custome resource",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithCustomResource},
			WithPod:        []*duckv1.WithPod{customResource},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{elWithCustomResource},
			WithPod:        []*duckv1.WithPod{customResource},
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
			if diff := cmp.Diff(tt.endResources, *actualEndResources,
				cmpopts.IgnoreFields(apis.Condition{}, "LastTransitionTime.Inner.Time"),
				cmpopts.SortSlices(compareCondition)); diff == "" {
				t.Errorf("eventlistener.Reconcile() equality mismatch. Diff request body: -want +got: %s", diff)
			}
		})
	}
}

func TestReconcile_Delete(t *testing.T) {
	tests := []struct {
		name           string
		key            string
		config         *resources.Config // Config of the reconciler
		startResources test.Resources    // State of the world before we call Reconcile
		endResources   test.Resources    // Expected State of the world after calling Reconcile
	}{{
		name: "delete eventlistener with remaining eventlisteners",
		key:  fmt.Sprintf("%s/%s", namespace, "el-2"),
		startResources: test.Resources{
			Namespaces: []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{
				makeEL(),
				// TODO: makeEL take name, ns as args
				makeEL(withDeletionTimestamp, func(el *v1beta1.EventListener) { el.Name = "el-2" })},
		},
		endResources: test.Resources{
			Namespaces: []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{
				makeEL(withDeletionTimestamp, func(el *v1beta1.EventListener) { el.Name = "el-2" }),
				makeEL(),
			},
		},
	}, {
		name: "delete last eventlistener in reconciler namespace",
		key:  fmt.Sprintf("%s/%s", reconcilerNamespace, eventListenerName),
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{reconcilerNamespaceResource},
			EventListeners: []*v1beta1.EventListener{makeEL(withControllerNamespace, withDeletionTimestamp)},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{reconcilerNamespaceResource},
			EventListeners: []*v1beta1.EventListener{makeEL(withControllerNamespace, withDeletionTimestamp)},
		},
	}, {
		name: "delete last eventlistener in namespace",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{makeEL(withDeletionTimestamp)},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{makeEL(withDeletionTimestamp)},
		},
	}, {
		name: "delete last eventlistener in namespace with no logging config",
		key:  reconcileKey,
		startResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{makeEL(withDeletionTimestamp)},
		},
		endResources: test.Resources{
			Namespaces:     []*corev1.Namespace{namespaceResource},
			EventListeners: []*v1beta1.EventListener{makeEL(withDeletionTimestamp)},
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
			if diff := cmp.Diff(tt.endResources, *actualEndResources, cmpopts.IgnoreFields(apis.Condition{}, "LastTransitionTime.Inner.Time")); diff != "" {
				t.Errorf("eventlistener.Reconcile() equality mismatch. Diff request body: -want +got: %s", diff)
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
