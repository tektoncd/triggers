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
	"os"
	"strconv"

	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	reconcilersource "knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/kmeta"
	"knative.dev/pkg/ptr"
)

const (
	TriggersMetricsDomain = "tekton.dev/triggers"
)

var (
	strongerSecurityPolicy = corev1.PodSecurityContext{
		RunAsNonRoot: ptr.Bool(true),
	}
)

func MakeDeployment(ctx context.Context, el *v1beta1.EventListener, configAcc reconcilersource.ConfigAccessor, c Config) (*appsv1.Deployment, error) {

	opt, err := addDeploymentBits(el, c)
	if err != nil {
		return nil, err
	}

	container := MakeContainer(el, configAcc, c, opt, addCertsForSecureConnection(c))

	filteredLabels := FilterLabels(ctx, el.Labels)

	var (
		podlabels                 = kmeta.UnionMaps(filteredLabels, GenerateLabels(el.Name, c.StaticResourceLabels))
		serviceAccountName        = el.Spec.ServiceAccountName
		replicas                  *int32
		vol                       []corev1.Volume
		tolerations               []corev1.Toleration
		nodeSelector, annotations map[string]string
		affinity                  *corev1.Affinity
		topologySpreadConstraints []corev1.TopologySpreadConstraint
	)

	for _, v := range container.Env {
		// If TLS related env are set then mount secret volume which will be used while starting the eventlistener.
		if v.Name == "TLS_CERT" {
			vol = append(vol, corev1.Volume{
				Name: "https-connection",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: v.ValueFrom.SecretKeyRef.Name,
					},
				},
			})
		}
	}

	if el.Spec.Resources.KubernetesResource != nil {
		if el.Spec.Resources.KubernetesResource.Replicas != nil {
			replicas = el.Spec.Resources.KubernetesResource.Replicas
		}
		if len(el.Spec.Resources.KubernetesResource.Template.Spec.Tolerations) != 0 {
			tolerations = el.Spec.Resources.KubernetesResource.Template.Spec.Tolerations
		}
		if len(el.Spec.Resources.KubernetesResource.Template.Spec.NodeSelector) != 0 {
			nodeSelector = el.Spec.Resources.KubernetesResource.Template.Spec.NodeSelector
		}
		if el.Spec.Resources.KubernetesResource.Template.Spec.ServiceAccountName != "" {
			serviceAccountName = el.Spec.Resources.KubernetesResource.Template.Spec.ServiceAccountName
		}
		if el.Spec.Resources.KubernetesResource.Template.Spec.Affinity != nil {
			affinity = el.Spec.Resources.KubernetesResource.Template.Spec.Affinity
		}
		if len(el.Spec.Resources.KubernetesResource.Template.Spec.TopologySpreadConstraints) != 0 {
			topologySpreadConstraints = el.Spec.Resources.KubernetesResource.Template.Spec.TopologySpreadConstraints
		}
		annotations = el.Spec.Resources.KubernetesResource.Template.Annotations
		podlabels = kmeta.UnionMaps(podlabels, el.Spec.Resources.KubernetesResource.Template.Labels)
	}

	var securityContext corev1.PodSecurityContext
	if *c.SetSecurityContext {
		securityContext = strongerSecurityPolicy
	}

	return &appsv1.Deployment{
		ObjectMeta: ObjectMeta(el, filteredLabels, c.StaticResourceLabels),
		Spec: appsv1.DeploymentSpec{
			Replicas: replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: GenerateLabels(el.Name, c.StaticResourceLabels),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      podlabels,
					Annotations: annotations,
				},
				Spec: corev1.PodSpec{
					Tolerations:               tolerations,
					NodeSelector:              nodeSelector,
					ServiceAccountName:        serviceAccountName,
					Containers:                []corev1.Container{container},
					Volumes:                   vol,
					SecurityContext:           &securityContext,
					Affinity:                  affinity,
					TopologySpreadConstraints: topologySpreadConstraints,
				},
			},
		},
	}, nil
}

// revive:disable:unused-parameter

func addDeploymentBits(el *v1beta1.EventListener, c Config) (ContainerOption, error) {
	// METRICS_PROMETHEUS_PORT defines the port exposed by the EventListener metrics endpoint
	// env METRICS_PROMETHEUS_PORT set by controller
	metricsPort, err := strconv.ParseInt(os.Getenv("METRICS_PROMETHEUS_PORT"), 10, 64)
	if err != nil {
		return nil, err
	}

	return func(container *corev1.Container) {
		if el.Spec.Resources.KubernetesResource != nil {
			if len(el.Spec.Resources.KubernetesResource.Template.Spec.Containers) != 0 {
				container.Resources = el.Spec.Resources.KubernetesResource.Template.Spec.Containers[0].Resources
				container.Env = append(container.Env, el.Spec.Resources.KubernetesResource.Template.Spec.Containers[0].Env...)
				container.ReadinessProbe = el.Spec.Resources.KubernetesResource.Template.Spec.Containers[0].ReadinessProbe
				container.LivenessProbe = el.Spec.Resources.KubernetesResource.Template.Spec.Containers[0].LivenessProbe
				container.StartupProbe = el.Spec.Resources.KubernetesResource.Template.Spec.Containers[0].StartupProbe
			}
		}

		container.Ports = append(container.Ports, corev1.ContainerPort{
			ContainerPort: int32(metricsPort),
			Protocol:      corev1.ProtocolTCP,
		})

		container.Env = append(container.Env, corev1.EnvVar{
			Name: "SYSTEM_NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					APIVersion: "v1",
					FieldPath:  "metadata.namespace",
				}},
		}, corev1.EnvVar{
			// METRICS_PROMETHEUS_PORT defines the port exposed by the EventListener metrics endpoint
			// env METRICS_PROMETHEUS_PORT set by controller
			Name:  "METRICS_PROMETHEUS_PORT",
			Value: os.Getenv("METRICS_PROMETHEUS_PORT"),
		})
	}, nil
}

func addCertsForSecureConnection(c Config) ContainerOption {
	return func(container *corev1.Container) {
		var elCert, elKey string
		certEnv := map[string]*corev1.EnvVarSource{}
		for i := range container.Env {
			certEnv[container.Env[i].Name] = container.Env[i].ValueFrom
		}
		var scheme corev1.URIScheme
		if v, ok := certEnv["TLS_CERT"]; ok {
			elCert = "/etc/triggers/tls/" + v.SecretKeyRef.Key
		} else {
			elCert = ""
		}
		if v, ok := certEnv["TLS_KEY"]; ok {
			elKey = "/etc/triggers/tls/" + v.SecretKeyRef.Key
		} else {
			elKey = ""
		}

		if elCert != "" && elKey != "" {
			scheme = corev1.URISchemeHTTPS
			container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
				Name:      "https-connection",
				ReadOnly:  true,
				MountPath: "/etc/triggers/tls",
			})
		} else {
			scheme = corev1.URISchemeHTTP
		}
		if container.LivenessProbe == nil {
			container.LivenessProbe = &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path:   "/live",
						Scheme: scheme,
						Port:   intstr.FromInt(eventListenerContainerPort),
					},
				},
				PeriodSeconds:    int32(*c.PeriodSeconds),
				FailureThreshold: int32(*c.FailureThreshold),
			}
		}
		if container.ReadinessProbe == nil {
			container.ReadinessProbe = &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path:   "/live",
						Scheme: scheme,
						Port:   intstr.FromInt(eventListenerContainerPort),
					},
				},
				PeriodSeconds:    int32(*c.PeriodSeconds),
				FailureThreshold: int32(*c.FailureThreshold),
			}
		}
		container.Args = append(container.Args, "--tls-cert="+elCert, "--tls-key="+elKey)
	}
}
