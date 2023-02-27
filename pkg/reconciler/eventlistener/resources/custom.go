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
	"context"
	"encoding/json"
	"os"
	"reflect"

	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	reconcilersource "knative.dev/eventing/pkg/reconciler/source"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"
)

func MakeCustomObject(ctx context.Context, el *v1beta1.EventListener, configAcc reconcilersource.ConfigAccessor, c Config) (*unstructured.Unstructured, error) {
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

	container := MakeContainer(el, configAcc, c, func(c *corev1.Container) {
		// handle env and resources for custom object
		if len(original.Spec.Template.Spec.Containers) == 1 {
			c.Env = append(c.Env, original.Spec.Template.Spec.Containers[0].Env...)
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
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/live",
					Scheme: corev1.URISchemeHTTP,
				},
			},
			SuccessThreshold: 1,
		}
	})

	podlabels := kmeta.UnionMaps(FilterLabels(ctx, el.Labels), GenerateLabels(el.Name, c.StaticResourceLabels))

	podlabels = kmeta.UnionMaps(podlabels, customObjectData.Labels)

	original.Labels = podlabels
	original.Annotations = customObjectData.Annotations
	original.Spec.Template.ObjectMeta = metav1.ObjectMeta{
		Name:        customObjectData.Spec.Template.Name,
		Labels:      customObjectData.Spec.Template.Labels,
		Annotations: customObjectData.Spec.Template.Annotations,
	}
	original.Spec.Template.Spec = corev1.PodSpec{
		Tolerations:               customObjectData.Spec.Template.Spec.Tolerations,
		NodeSelector:              customObjectData.Spec.Template.Spec.NodeSelector,
		ServiceAccountName:        customObjectData.Spec.Template.Spec.ServiceAccountName,
		Containers:                []corev1.Container{container},
		Affinity:                  customObjectData.Spec.Template.Spec.Affinity,
		TopologySpreadConstraints: customObjectData.Spec.Template.Spec.TopologySpreadConstraints,
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

func UpdateCustomObject(originalData, updatedCustomObject *unstructured.Unstructured) (bool, *duckv1.WithPod, error) {
	updated := false
	originalObject := &duckv1.WithPod{}
	existingObject := &duckv1.WithPod{}
	data, e := originalData.MarshalJSON()
	if e != nil {
		return false, nil, e
	}
	if e := json.Unmarshal(data, &originalObject); e != nil {
		return false, nil, e
	}
	updatedData, e := updatedCustomObject.MarshalJSON()
	if e != nil {
		return false, nil, e
	}
	if e := json.Unmarshal(updatedData, &existingObject); e != nil {
		return false, nil, e
	}

	// custom resource except few spec fields from user
	// added below checks in order to avoid unwanted updates on all spec changes.
	if !reflect.DeepEqual(existingObject.Spec.Template.Name, originalObject.Spec.Template.Name) {
		existingObject.Spec.Template.Name = originalObject.Spec.Template.Name
		updated = true
	}
	if !reflect.DeepEqual(existingObject.Spec.Template.Labels, originalObject.Spec.Template.Labels) {
		existingObject.Spec.Template.Labels = originalObject.Spec.Template.Labels
		updated = true
	}
	if !reflect.DeepEqual(existingObject.Spec.Template.Annotations, originalObject.Spec.Template.Annotations) {
		existingObject.Spec.Template.Annotations = originalObject.Spec.Template.Annotations
		updated = true
	}
	if existingObject.Spec.Template.Spec.ServiceAccountName != originalObject.Spec.Template.Spec.ServiceAccountName {
		existingObject.Spec.Template.Spec.ServiceAccountName = originalObject.Spec.Template.Spec.ServiceAccountName
		updated = true
	}
	if !reflect.DeepEqual(existingObject.Spec.Template.Spec.Tolerations, originalObject.Spec.Template.Spec.Tolerations) {
		existingObject.Spec.Template.Spec.Tolerations = originalObject.Spec.Template.Spec.Tolerations
		updated = true
	}
	if !reflect.DeepEqual(existingObject.Spec.Template.Spec.NodeSelector, originalObject.Spec.Template.Spec.NodeSelector) {
		existingObject.Spec.Template.Spec.NodeSelector = originalObject.Spec.Template.Spec.NodeSelector
		updated = true
	}
	if !reflect.DeepEqual(existingObject.Spec.Template.Spec.Affinity, originalObject.Spec.Template.Spec.Affinity) {
		existingObject.Spec.Template.Spec.Affinity = originalObject.Spec.Template.Spec.Affinity
		updated = true
	}
	if !reflect.DeepEqual(existingObject.Spec.Template.Spec.TopologySpreadConstraints, originalObject.Spec.Template.Spec.TopologySpreadConstraints) {
		existingObject.Spec.Template.Spec.TopologySpreadConstraints = originalObject.Spec.Template.Spec.TopologySpreadConstraints
		updated = true
	}
	if len(existingObject.Spec.Template.Spec.Containers) == 0 ||
		len(existingObject.Spec.Template.Spec.Containers) > 1 {
		existingObject.Spec.Template.Spec.Containers = originalObject.Spec.Template.Spec.Containers
		updated = true
	} else {
		if existingObject.Spec.Template.Spec.Containers[0].Name != originalObject.Spec.Template.Spec.Containers[0].Name {
			existingObject.Spec.Template.Spec.Containers[0].Name = originalObject.Spec.Template.Spec.Containers[0].Name
			updated = true
		}
		if existingObject.Spec.Template.Spec.Containers[0].Image != originalObject.Spec.Template.Spec.Containers[0].Image {
			existingObject.Spec.Template.Spec.Containers[0].Image = originalObject.Spec.Template.Spec.Containers[0].Image
			updated = true
		}
		if !reflect.DeepEqual(existingObject.Spec.Template.Spec.Containers[0].Ports, originalObject.Spec.Template.Spec.Containers[0].Ports) {
			existingObject.Spec.Template.Spec.Containers[0].Ports = originalObject.Spec.Template.Spec.Containers[0].Ports
			updated = true
		}
		if !reflect.DeepEqual(existingObject.Spec.Template.Spec.Containers[0].Args, originalObject.Spec.Template.Spec.Containers[0].Args) {
			existingObject.Spec.Template.Spec.Containers[0].Args = originalObject.Spec.Template.Spec.Containers[0].Args
			updated = true
		}
		if existingObject.Spec.Template.Spec.Containers[0].Command != nil {
			existingObject.Spec.Template.Spec.Containers[0].Command = nil
			updated = true
		}
		if !reflect.DeepEqual(existingObject.Spec.Template.Spec.Containers[0].Resources, originalObject.Spec.Template.Spec.Containers[0].Resources) {
			existingObject.Spec.Template.Spec.Containers[0].Resources = originalObject.Spec.Template.Spec.Containers[0].Resources
			updated = true
		}
		if !reflect.DeepEqual(existingObject.Spec.Template.Spec.Containers[0].Env, originalObject.Spec.Template.Spec.Containers[0].Env) {
			existingObject.Spec.Template.Spec.Containers[0].Env = originalObject.Spec.Template.Spec.Containers[0].Env
			updated = true
		}
		if !reflect.DeepEqual(existingObject.Spec.Template.Spec.Containers[0].ReadinessProbe, originalObject.Spec.Template.Spec.Containers[0].ReadinessProbe) {
			existingObject.Spec.Template.Spec.Containers[0].ReadinessProbe = originalObject.Spec.Template.Spec.Containers[0].ReadinessProbe
			updated = true
		}
		if !reflect.DeepEqual(existingObject.Spec.Template.Spec.Containers[0].VolumeMounts, originalObject.Spec.Template.Spec.Containers[0].VolumeMounts) {
			existingObject.Spec.Template.Spec.Containers[0].VolumeMounts = originalObject.Spec.Template.Spec.Containers[0].VolumeMounts
			updated = true
		}
		if !reflect.DeepEqual(existingObject.Spec.Template.Spec.Volumes, originalObject.Spec.Template.Spec.Volumes) {
			existingObject.Spec.Template.Spec.Volumes = originalObject.Spec.Template.Spec.Volumes
			updated = true
		}
	}

	return updated, existingObject, nil
}
