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
	"fmt"

	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"knative.dev/pkg/network"
)

const (
	// eventListenerServicePortName defines service port name for EventListener Service
	eventListenerServicePortName = "http-listener"
	// eventListenerServiceTLSPortName defines service TLS port name for EventListener Service
	eventListenerServiceTLSPortName = "https-listener"
	// eventListenerMetricsPortName defines the metrics port name by the EventListener Container
	eventListenerMetricsPortName = "http-metrics"
	// eventListenerContainerPort defines service port for EventListener Service
	eventListenerContainerPort = 8080
	// eventListenerMetricsPort defines metrics port for EventListener Service
	eventListenerMetricsPort = 9000
)

var metricsPort = corev1.ServicePort{
	Name:     eventListenerMetricsPortName,
	Protocol: corev1.ProtocolTCP,
	Port:     int32(9000),
	TargetPort: intstr.IntOrString{
		IntVal: int32(eventListenerMetricsPort),
	},
}

func MakeService(ctx context.Context, el *v1beta1.EventListener, c Config) *corev1.Service {
	// for backward compatibility with original behavior
	var (
		serviceType corev1.ServiceType
		servicePort corev1.ServicePort
	)
	if el.Spec.Resources.KubernetesResource != nil && el.Spec.Resources.KubernetesResource.ServiceType != "" {
		serviceType = el.Spec.Resources.KubernetesResource.ServiceType
	}
	if el.Spec.Resources.KubernetesResource != nil && el.Spec.Resources.KubernetesResource.ServicePort != nil {
		port := int(*el.Spec.Resources.KubernetesResource.ServicePort)
		c.Port = &port
	}

	servicePort = ServicePort(el, c)

	return &corev1.Service{
		ObjectMeta: ObjectMeta(el, FilterLabels(ctx, el.Labels), c.StaticResourceLabels),
		Spec: corev1.ServiceSpec{
			Selector: GenerateLabels(el.Name, c.StaticResourceLabels),
			Type:     serviceType,
			Ports:    []corev1.ServicePort{servicePort, metricsPort}},
	}
}

func ServicePort(el *v1beta1.EventListener, c Config) corev1.ServicePort {
	var elCert, elKey string

	servicePortName := eventListenerServicePortName
	servicePort := *c.Port

	certEnv := map[string]*corev1.EnvVarSource{}
	if el.Spec.Resources.KubernetesResource != nil {
		if len(el.Spec.Resources.KubernetesResource.Template.Spec.Containers) != 0 {
			for i := range el.Spec.Resources.KubernetesResource.Template.Spec.Containers[0].Env {
				certEnv[el.Spec.Resources.KubernetesResource.Template.Spec.Containers[0].Env[i].Name] =
					el.Spec.Resources.KubernetesResource.Template.Spec.Containers[0].Env[i].ValueFrom
			}
		}
		if el.Spec.Resources.KubernetesResource.ServicePort != nil {
			servicePort = int(*el.Spec.Resources.KubernetesResource.ServicePort)
		}
	}

	if v, ok := certEnv["TLS_CERT"]; ok {
		elCert = v.SecretKeyRef.Key
	} else {
		elCert = ""
	}
	if v, ok := certEnv["TLS_KEY"]; ok {
		elKey = v.SecretKeyRef.Key
	} else {
		elKey = ""
	}

	if elCert != "" && elKey != "" {
		servicePortName = eventListenerServiceTLSPortName
		if *c.Port == DefaultPort {
			// We return port 8443 if TLS is enabled and the default HTTP port is set.
			// This effectively makes 8443 the default HTTPS port unless a user explicitly sets a different port.
			servicePort = 8443
		}
	}

	return corev1.ServicePort{
		Name:     servicePortName,
		Protocol: corev1.ProtocolTCP,
		Port:     int32(servicePort),
		TargetPort: intstr.IntOrString{
			IntVal: int32(eventListenerContainerPort),
		},
	}
}

// ListenerHostname returns the intended hostname for the EventListener service.
func ListenerHostname(el *v1beta1.EventListener, c Config) string {
	sp := ServicePort(el, c)
	return network.GetServiceHostname(el.Status.Configuration.GeneratedResourceName, el.Namespace) + fmt.Sprintf(":%d", sp.Port)
}
