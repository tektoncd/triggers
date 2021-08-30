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
	"bytes"
	"encoding/json"
	"os"

	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"
)

func MakeCustomObject(el *v1beta1.EventListener, c Config) (*unstructured.Unstructured, error) {
	original := &duckv1.WithPod{}
	decoder := json.NewDecoder(bytes.NewBuffer(el.Spec.Resources.CustomResource.Raw))
	if err := decoder.Decode(&original); err != nil {
		return nil, err
	}

	customObjectData := original.DeepCopy()

	namespace := original.GetNamespace()
	// Default the resource creation to the EventListenerNamespace if not found in the resource object
	if namespace == "" {
		namespace = el.GetNamespace()
	}

	container := MakeContainer(el, c, func(c *corev1.Container) {
		// handle env and resources for custom object
		if len(original.Spec.Template.Spec.Containers) == 1 {
			for i := range original.Spec.Template.Spec.Containers[0].Env {
				c.Env = append(c.Env, original.Spec.Template.Spec.Containers[0].Env[i])
			}
			c.Resources = original.Spec.Template.Spec.Containers[0].Resources
		}

		c.Env = append(c.Env, corev1.EnvVar{
			Name: "SYSTEM_NAMESPACE",
			// Cannot use FieldRef here because Knative Serving mask that field under feature gate
			// https://github.com/knative/serving/blob/master/pkg/apis/config/features.go#L48
			Value: el.Namespace,
		}, corev1.EnvVar{
			// METRICS_PROMETHEUS_PORT defines the port exposed by the EventListener metrics endpoint
			// env METRICS_PROMETHEUS_PORT set by controller
			Name:  "METRICS_PROMETHEUS_PORT",
			Value: os.Getenv("METRICS_PROMETHEUS_PORT"),
		})

		c.ReadinessProbe = &corev1.Probe{
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/live",
					Scheme: corev1.URISchemeHTTP,
				},
			},
			SuccessThreshold: 1,
		}
	})

	podlabels := kmeta.UnionMaps(el.Labels, GenerateLabels(el.Name, c.StaticResourceLabels))

	podlabels = kmeta.UnionMaps(podlabels, customObjectData.Labels)

	original.Labels = podlabels
	original.Annotations = customObjectData.Annotations
	original.Spec.Template.ObjectMeta = metav1.ObjectMeta{
		Name:        customObjectData.Spec.Template.Name,
		Labels:      customObjectData.Spec.Template.Labels,
		Annotations: customObjectData.Spec.Template.Annotations,
	}
	original.Spec.Template.Spec = corev1.PodSpec{
		Tolerations:        customObjectData.Spec.Template.Spec.Tolerations,
		NodeSelector:       customObjectData.Spec.Template.Spec.NodeSelector,
		ServiceAccountName: customObjectData.Spec.Template.Spec.ServiceAccountName,
		Containers:         []corev1.Container{container},
	}
	marshaledData, err := json.Marshal(original)
	if err != nil {
		return nil, err
	}
	data := new(unstructured.Unstructured)
	if err := data.UnmarshalJSON(marshaledData); err != nil {
		return nil, err
	}

	if data.GetName() == "" {
		data.SetName(el.Status.Configuration.GeneratedResourceName)
	}
	data.SetNamespace(namespace)
	data.SetOwnerReferences([]metav1.OwnerReference{*kmeta.NewControllerRef(el)})

	return data, nil
}
