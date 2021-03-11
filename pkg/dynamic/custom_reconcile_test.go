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

	logger "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

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
