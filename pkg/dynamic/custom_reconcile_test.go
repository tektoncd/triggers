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

package dynamic

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	logger "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

func TestGetNestedFieldCopyData(t *testing.T) {
	original := &duckv1.WithPod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "serving.knative.dev/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Spec: duckv1.WithPodSpec{
			Template: duckv1.PodSpecable{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "event-listener",
						Image: "test",
						Ports: []corev1.ContainerPort{{
							ContainerPort: int32(8888),
							Protocol:      corev1.ProtocolTCP,
						}},
					}},
				},
			},
		},
	}
	marshaledData, err := json.Marshal(original)
	if err != nil {
		logger.Error("failed to marshal custom object", err)
	}
	existingData := new(unstructured.Unstructured)
	originalData := new(unstructured.Unstructured)
	if err := originalData.UnmarshalJSON(marshaledData); err != nil {
		logger.Error("failed to unmarshal to unstructured object", err)
	}
	_, equal := getNestedFieldCopyData(originalData, existingData)
	if diff := cmp.Diff(equal, false); diff != "" {
		t.Errorf("GetNestedFieldCopyData equality mismatch. Diff request body: -want +got: %s", diff)
	}
}

func TestGetConditions(t *testing.T) {
	tests := []struct {
		name    string
		objData *appsv1.Deployment
	}{{
		name: "No status",
		objData: &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Deployment",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
		},
	}, {
		name: "Status but no conditions",
		objData: &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Deployment",
				APIVersion: "v1",
			},
			Status: appsv1.DeploymentStatus{
				ObservedGeneration: 1,
			},
		},
	}, {
		name: "Status with conditions",
		objData: &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Deployment",
				APIVersion: "v1",
			},
			Status: appsv1.DeploymentStatus{
				ObservedGeneration: 1,
				Conditions: []appsv1.DeploymentCondition{{
					Type:    appsv1.DeploymentAvailable,
					Status:  corev1.ConditionTrue,
					Message: "deployment created",
				}},
			},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			marshaledData, err := json.Marshal(tt.objData)
			if err != nil {
				logger.Error("failed to marshal custom object", err)
			}
			originalData := new(unstructured.Unstructured)
			if err := originalData.UnmarshalJSON(marshaledData); err != nil {
				logger.Error("failed to unmarshal to unstructured object", err)
			}
			cond, url, err := GetConditions(originalData)
			if cond != nil && url != nil && err != nil {
				t.Error("GetConditions is not working as expected")
			}
		})
	}
}

func TestGetConditionsInvalidObj(t *testing.T) {
	objWithNoStatus := map[string]interface{}{
		"kind":       "test",
		"apiVersion": "v1",
		"bacon":      "delicious",
	}
	marshaledData, err := json.Marshal(objWithNoStatus)
	if err != nil {
		logger.Error("failed to marshal custom object", err)
	}
	originalData := new(unstructured.Unstructured)
	if err := originalData.UnmarshalJSON(marshaledData); err != nil {
		logger.Error("failed to unmarshal to unstructured object", err)
	}
	cond, url, err := GetConditions(originalData)
	if cond != nil && url != nil && err != nil {
		t.Error("GetConditions is not working as expected")
	}
}

func TestReconcileCustomObject(t *testing.T) {
	existing := &duckv1.WithPod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "serving.knative.dev/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test",
			Namespace:   "test",
			Labels:      map[string]string{"key": "value"},
			Annotations: map[string]string{"key": "value"},
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "v1",
				Kind:       "EventListener",
			}},
		},
		Spec: duckv1.WithPodSpec{
			Template: duckv1.PodSpecable{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test",
					Labels:      map[string]string{"key": "value"},
					Annotations: map[string]string{"key": "value"},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "sa",
					NodeSelector:       map[string]string{"node": "value"},
					Tolerations: []corev1.Toleration{{
						Key:   "key",
						Value: "value",
					}},
					Volumes: []corev1.Volume{{
						Name: "key",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					}},
					Containers: []corev1.Container{{
						Name:  "event-listener",
						Image: "test",
						Ports: []corev1.ContainerPort{{
							ContainerPort: int32(8888),
							Protocol:      corev1.ProtocolTCP,
						}},
						Args: []string{"key"},
						VolumeMounts: []corev1.VolumeMount{{
							Name: "testvolume",
						}},
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU: resource.Quantity{Format: resource.DecimalSI},
							},
						},
						Env: []corev1.EnvVar{{
							Name:  "key",
							Value: "value",
						}},
					}},
				},
			},
		},
	}
	desired := &duckv1.WithPod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "serving.knative.dev/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test1",
			Namespace:   "test1",
			Labels:      map[string]string{"key1": "value"},
			Annotations: map[string]string{"key1": "value"},
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "v1",
				Kind:       "Pod",
			}},
		},
		Spec: duckv1.WithPodSpec{
			Template: duckv1.PodSpecable{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test1",
					Labels:      map[string]string{"key1": "value"},
					Annotations: map[string]string{"key1": "value"},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "sa1",
					NodeSelector:       map[string]string{"node1": "value"},
					Tolerations: []corev1.Toleration{{
						Key:   "key",
						Value: "value1",
					}},
					Volumes: []corev1.Volume{{
						Name: "key1",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					}},
					Containers: []corev1.Container{{
						Name:  "event-listener1",
						Image: "test1",
						Ports: []corev1.ContainerPort{{
							ContainerPort: int32(8888),
							Protocol:      corev1.ProtocolUDP,
						}},
						Args: []string{"key1"},
						VolumeMounts: []corev1.VolumeMount{{
							Name: "testvolume1",
						}},
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceMemory: resource.Quantity{Format: resource.DecimalSI},
							},
						},
						Env: []corev1.EnvVar{{
							Name:  "key1",
							Value: "value1",
						}},
					}},
				},
			},
		},
	}
	existingMarshaledData, err := json.Marshal(existing)
	if err != nil {
		logger.Error("failed to marshal custom object", err)
	}
	desiredMarshaledData, err := json.Marshal(desired)
	if err != nil {
		logger.Error("failed to marshal custom object", err)
	}
	existingData := new(unstructured.Unstructured)
	desiredData := new(unstructured.Unstructured)
	if err := existingData.UnmarshalJSON(existingMarshaledData); err != nil {
		logger.Error("failed to unmarshal to unstructured object", err)
	}
	if err := desiredData.UnmarshalJSON(desiredMarshaledData); err != nil {
		logger.Error("failed to unmarshal to unstructured object", err)
	}
	ReconcileCustomObject(existingData, desiredData)
}

func Test_mergeMaps(t *testing.T) {
	tests := []struct {
		name           string
		l1, l2         map[string]string
		expectedLabels map[string]string
	}{{
		name:           "Both maps empty",
		l1:             nil,
		l2:             nil,
		expectedLabels: map[string]string{},
	}, {
		name:           "Map one empty",
		l1:             nil,
		l2:             map[string]string{"k": "v"},
		expectedLabels: map[string]string{"k": "v"},
	}, {
		name:           "Map two empty",
		l1:             map[string]string{"k": "v"},
		l2:             nil,
		expectedLabels: map[string]string{"k": "v"},
	}, {
		name:           "Both maps",
		l1:             map[string]string{"k1": "v1"},
		l2:             map[string]string{"k2": "v2"},
		expectedLabels: map[string]string{"k1": "v1", "k2": "v2"},
	}, {
		name:           "Both maps with clobber",
		l1:             map[string]string{"k1": "v1"},
		l2:             map[string]string{"k1": "v2"},
		expectedLabels: map[string]string{"k1": "v2"},
	}}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			actualLabels := MergeMaps(tests[i].l1, tests[i].l2)
			if diff := cmp.Diff(tests[i].expectedLabels, actualLabels); diff != "" {
				t.Errorf("mergeLabels() did not return expected. -want, +got: %s", diff)
			}
		})
	}
}
