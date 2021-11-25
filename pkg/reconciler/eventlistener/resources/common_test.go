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
	"fmt"

	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

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
	el.Status.SetAddress(ListenerHostname(el, *MakeConfig()))
}

func withTLSPort(el *v1beta1.EventListener) {
	el.Status.SetAddress(ListenerHostname(el, *MakeConfig(func(c *Config) {
		x := 8443
		c.Port = &x
	})))
}

func withServiceTypeLoadBalancer(el *v1beta1.EventListener) {
	el.Spec.Resources.KubernetesResource = &v1beta1.KubernetesResource{
		ServiceType: "LoadBalancer",
	}
}

func withServicePort80(el *v1beta1.EventListener) {
	port := int32(80)
	el.Spec.Resources.KubernetesResource = &v1beta1.KubernetesResource{
		ServicePort: &port,
	}
}
