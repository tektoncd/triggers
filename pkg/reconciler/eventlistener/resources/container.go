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
	"strconv"

	"github.com/tektoncd/triggers/pkg/apis/triggers"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	corev1 "k8s.io/api/core/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

func MakeContainer(el *v1beta1.EventListener, c Config, pod *duckv1.WithPod) corev1.Container {
	var resources corev1.ResourceRequirements
	env := []corev1.EnvVar{}
	if el.Spec.Resources.KubernetesResource != nil {
		if len(el.Spec.Resources.KubernetesResource.Template.Spec.Containers) != 0 {
			resources = el.Spec.Resources.KubernetesResource.Template.Spec.Containers[0].Resources
			env = append(env, el.Spec.Resources.KubernetesResource.Template.Spec.Containers[0].Env...)
		}
	}

	// handle env and resources for custom object
	// TODO(mattmoor): Consider replacing this with a functional option,
	// to avoid leaking caller details into here?
	if pod != nil {
		if len(pod.Spec.Template.Spec.Containers) == 1 {
			for i := range pod.Spec.Template.Spec.Containers[0].Env {
				env = append(env, pod.Spec.Template.Spec.Containers[0].Env[i])
			}
			resources = pod.Spec.Template.Spec.Containers[0].Resources
		}
	}

	isMultiNS := false
	if len(el.Spec.NamespaceSelector.MatchNames) != 0 {
		isMultiNS = true
	}

	payloadValidation := true
	if value, ok := el.GetAnnotations()[triggers.PayloadValidationAnnotation]; ok {
		if value == "false" {
			payloadValidation = false
		}
	}

	return corev1.Container{
		Name:  "event-listener",
		Image: *c.Image,
		Ports: []corev1.ContainerPort{{
			ContainerPort: int32(eventListenerContainerPort),
			Protocol:      corev1.ProtocolTCP,
		}},
		Resources: resources,
		Args: []string{
			"--el-name=" + el.Name,
			"--el-namespace=" + el.Namespace,
			"--port=" + strconv.Itoa(eventListenerContainerPort),
			"--readtimeout=" + strconv.FormatInt(*c.ReadTimeOut, 10),
			"--writetimeout=" + strconv.FormatInt(*c.WriteTimeOut, 10),
			"--idletimeout=" + strconv.FormatInt(*c.IdleTimeOut, 10),
			"--timeouthandler=" + strconv.FormatInt(*c.TimeOutHandler, 10),
			"--is-multi-ns=" + strconv.FormatBool(isMultiNS),
			"--payload-validation=" + strconv.FormatBool(payloadValidation),
		},
		Env: env,
	}
}
