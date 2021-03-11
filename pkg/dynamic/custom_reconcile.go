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

	logger "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"knative.dev/pkg/apis/duck/v1beta1"
)

func GetConditions(existingData *unstructured.Unstructured) (v1beta1.Conditions, interface{}, error) {
	statusData, ok, err := unstructured.NestedMap(existingData.Object, "status")
	if !ok || err != nil {
		// No status in the created object, it is weird but let's not fail
		logger.Warn("empty status for the created custom object")
		return nil, nil, err
	}
	conditionData, ok, err := unstructured.NestedFieldCopy(statusData, "conditions")
	if !ok || err != nil {
		// No conditions in the created object, it is weird but let's not fail
		logger.Warn("empty status conditions for the created custom object")
		return nil, nil, err
	}
	cMarshalledData, err := json.Marshal(conditionData)
	if err != nil {
		return nil, nil, err
	}
	var customConditions v1beta1.Conditions
	if err = json.Unmarshal(cMarshalledData, &customConditions); err != nil {
		return nil, nil, err
	}
	return customConditions, statusData["url"], nil
}
