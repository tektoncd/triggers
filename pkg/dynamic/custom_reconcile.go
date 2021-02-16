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
	"reflect"

	logger "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"knative.dev/pkg/apis/duck/v1beta1"
)

func ReconcileCustomObject(existing, desired *unstructured.Unstructured) (updated bool) {
	originalMetaLabel, change := getNestedFieldCopyData(existing, desired, "metadata", "labels")
	if !change {
		updated = true
		if err := unstructured.SetNestedField(existing.Object, originalMetaLabel, "metadata", "labels"); err != nil {
			logger.Error("failed to set metadata labels to existing object: ", err)
			updated = false
		}
	}

	originalMetaOwner, change := getNestedFieldCopyData(existing, desired, "metadata", "ownerReferences")
	if !change {
		updated = true
		if err := unstructured.SetNestedField(existing.Object, originalMetaOwner, "metadata", "ownerReferences"); err != nil {
			logger.Error("failed to set metadata ownerReferences to existing object: ", err)
			updated = false
		}
	}

	existingMetaAnno, _, _ := unstructured.NestedFieldCopy(existing.Object, "metadata", "annotations")
	originalMetaAnno, _, _ := unstructured.NestedFieldCopy(desired.Object, "metadata", "annotations")
	originalAnno, _ := originalMetaAnno.(map[string]string)
	existingAnno, _ := existingMetaAnno.(map[string]string)
	if !reflect.DeepEqual(existingMetaAnno, MergeMaps(existingAnno, originalAnno)) {
		updated = true
		if err := unstructured.SetNestedField(existing.Object, originalMetaAnno, "metadata", "annotations"); err != nil {
			logger.Error("failed to set metadata annotations to existing object: ", err)
			updated = false
		}
	}

	originalSpecMetaName, change := getNestedFieldCopyData(existing, desired, "spec", "template", "metadata", "name")
	if !change {
		updated = true
		if err := unstructured.SetNestedField(existing.Object, originalSpecMetaName, "spec", "template", "metadata", "name"); err != nil {
			logger.Error("failed to set metadata name for spec to existing object: ", err)
			updated = false
		}
	}

	originalSpecMetaLabel, change := getNestedFieldCopyData(existing, desired, "spec", "template", "metadata", "labels")
	if !change {
		updated = true
		if err := unstructured.SetNestedField(existing.Object, originalSpecMetaLabel, "spec", "template", "metadata", "labels"); err != nil {
			logger.Error("failed to set metadata labels for spec to existing object: ", err)
			updated = false
		}
	}

	originalSpecMetaAnno, change := getNestedFieldCopyData(existing, desired, "spec", "template", "metadata", "annotations")
	if !change {
		updated = true
		if err := unstructured.SetNestedField(existing.Object, originalSpecMetaAnno, "spec", "template", "metadata", "annotations"); err != nil {
			logger.Error("failed to set metadata annotations for spec to existing object: ", err)
			updated = false
		}
	}

	var (
		existingEnv, existingPorts, existingVolumeMount        []interface{}
		desiredEnv, desiredPorts, desiredVolumeMount           []interface{}
		desiredName, existingName, desiredImage, existingImage string
		desiredArgs, existingArgs                              []string
		existingResources, desiredResources                    interface{}
	)

	existingContainersData, _, _ := unstructured.NestedSlice(existing.Object, "spec", "template", "spec", "containers")
	for i := range existingContainersData {
		existingEnv, _, _ = unstructured.NestedSlice(existingContainersData[i].(map[string]interface{}), "env")
		existingArgs, _, _ = unstructured.NestedStringSlice(existingContainersData[i].(map[string]interface{}), "args")
		existingImage, _, _ = unstructured.NestedString(existingContainersData[i].(map[string]interface{}), "image")
		existingName, _, _ = unstructured.NestedString(existingContainersData[i].(map[string]interface{}), "name")
		existingPorts, _, _ = unstructured.NestedSlice(existingContainersData[i].(map[string]interface{}), "ports")
		existingVolumeMount, _, _ = unstructured.NestedSlice(existingContainersData[i].(map[string]interface{}), "volumeMounts")
		existingResources, _, _ = unstructured.NestedFieldCopy(existingContainersData[i].(map[string]interface{}), "resources")
	}

	desiredContainersData, _, _ := unstructured.NestedSlice(desired.Object, "spec", "template", "spec", "containers")
	for i := range desiredContainersData {
		desiredEnv, _, _ = unstructured.NestedSlice(desiredContainersData[i].(map[string]interface{}), "env")
		desiredArgs, _, _ = unstructured.NestedStringSlice(desiredContainersData[i].(map[string]interface{}), "args")
		desiredImage, _, _ = unstructured.NestedString(desiredContainersData[i].(map[string]interface{}), "image")
		desiredName, _, _ = unstructured.NestedString(desiredContainersData[i].(map[string]interface{}), "name")
		desiredPorts, _, _ = unstructured.NestedSlice(desiredContainersData[i].(map[string]interface{}), "ports")
		desiredVolumeMount, _, _ = unstructured.NestedSlice(desiredContainersData[i].(map[string]interface{}), "volumeMounts")
		desiredResources, _, _ = unstructured.NestedFieldCopy(desiredContainersData[i].(map[string]interface{}), "resources")
	}

	var cUpdated bool
	if !reflect.DeepEqual(existingEnv, desiredEnv) {
		cUpdated = true
		for _, c := range existingEnv {
			if err := unstructured.SetNestedSlice(c.(map[string]interface{}), desiredEnv, "env"); err != nil {
				logger.Error("failed to set container env to existing object: ", err)
				cUpdated = false
			}
		}
	}
	if !reflect.DeepEqual(existingArgs, desiredArgs) {
		res := make(map[string]interface{})
		cUpdated = true
		for _, c := range existingArgs {
			res[c] = c
		}
		if err := unstructured.SetNestedStringSlice(res, desiredArgs, "args"); err != nil {
			logger.Error("failed to set container args to existing object: ", err)
			cUpdated = false
		}
	}
	if !reflect.DeepEqual(existingImage, desiredImage) {
		cUpdated = true
		res := make(map[string]interface{})
		res[existingImage] = existingImage
		if err := unstructured.SetNestedField(res, desiredImage, "image"); err != nil {
			logger.Error("failed to set container image to existing object: ", err)
			cUpdated = false
		}
	}
	if !reflect.DeepEqual(existingName, desiredName) {
		cUpdated = true
		res := make(map[string]interface{})
		res[existingName] = existingName
		if err := unstructured.SetNestedField(res, desiredName, "name"); err != nil {
			logger.Error("failed to set container name to existing object: ", err)
			cUpdated = false
		}
	}
	if !reflect.DeepEqual(existingPorts, desiredPorts) {
		cUpdated = true
		for _, c := range existingPorts {
			if err := unstructured.SetNestedSlice(c.(map[string]interface{}), desiredPorts, "ports"); err != nil {
				logger.Error("failed to set container ports to existing object: ", err)
				cUpdated = false
			}
		}
	}
	if !reflect.DeepEqual(existingVolumeMount, desiredVolumeMount) {
		cUpdated = true
		for _, c := range existingVolumeMount {
			if err := unstructured.SetNestedSlice(c.(map[string]interface{}), desiredVolumeMount, "volumeMounts"); err != nil {
				logger.Error("failed to set container volumeMount to existing object: ", err)
				cUpdated = false
			}
		}
	}
	if !reflect.DeepEqual(existingResources, desiredResources) {
		cUpdated = true
		if err := unstructured.SetNestedField(existingResources.(map[string]interface{}), desiredResources, "resources"); err != nil {
			logger.Error("failed to set container resources to existing object: ", err)
			cUpdated = false
		}
	}
	if cUpdated {
		updated = true
		err := unstructured.SetNestedField(existing.Object, desiredContainersData, "spec", "template", "spec", "containers")
		if err != nil {
			updated = false
		}
	}
	desiredSA, change := getNestedFieldCopyData(existing, desired, "spec", "template", "spec", "serviceAccountName")
	if !change {
		updated = true
		if err := unstructured.SetNestedField(existing.Object, desiredSA, "spec", "template", "spec", "serviceAccountName"); err != nil {
			logger.Error("failed to set service account to existing object: ", err)
			updated = false
		}
	}
	desiredVol, change := getNestedFieldCopyData(existing, desired, "spec", "template", "spec", "volumes")
	if !change {
		updated = true
		if err := unstructured.SetNestedField(existing.Object, desiredVol, "spec", "template", "spec", "volumes"); err != nil {
			logger.Error("failed to set volumes to existing object: ", err)
			updated = false
		}
	}
	desiredTolerations, change := getNestedFieldCopyData(existing, desired, "spec", "template", "spec", "tolerations")
	if !change {
		updated = true
		if err := unstructured.SetNestedField(existing.Object, desiredTolerations, "spec", "template", "spec", "tolerations"); err != nil {
			logger.Error("failed to set tolerations to existing object: ", err)
			updated = false
		}
	}
	desiredNodeSelector, change := getNestedFieldCopyData(existing, desired, "spec", "template", "spec", "nodeSelector")
	if !change {
		updated = true
		if err := unstructured.SetNestedField(existing.Object, desiredNodeSelector, "spec", "template", "spec", "nodeSelector"); err != nil {
			logger.Error("failed to set nodeSelector to existing object: ", err)
			updated = false
		}
	}
	return
}

func getNestedFieldCopyData(existing, desired *unstructured.Unstructured, fields ...string) (interface{}, bool) {
	updated, _, _ := unstructured.NestedFieldCopy(existing.Object, fields...)
	original, _, _ := unstructured.NestedFieldCopy(desired.Object, fields...)
	return original, reflect.DeepEqual(original, updated)
}

// MergeMaps merges the values in the passed maps into a new map.
// Values within m2 potentially clobber m1 values.
func MergeMaps(m1, m2 map[string]string) map[string]string {
	merged := make(map[string]string, len(m1)+len(m2))
	for k, v := range m1 {
		merged[k] = v
	}
	for k, v := range m2 {
		merged[k] = v
	}
	return merged
}

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
