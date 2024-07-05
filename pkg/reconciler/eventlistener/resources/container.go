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
	"errors"
	"strconv"

	"github.com/tektoncd/triggers/pkg/apis/config"
	"github.com/tektoncd/triggers/pkg/apis/triggers"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	corev1 "k8s.io/api/core/v1"
	reconcilersource "knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/ptr"
)

type ContainerOption func(*corev1.Container)

func MakeContainer(el *v1beta1.EventListener, configAcc reconcilersource.ConfigAccessor, c Config, cfg *config.Config, opts ...ContainerOption) (corev1.Container, error) {
	isMultiNS := false
	if len(el.Spec.NamespaceSelector.MatchNames) != 0 {
		isMultiNS = true
	}
	for _, triggerGroup := range el.Spec.TriggerGroups {
		if len(triggerGroup.TriggerSelector.NamespaceSelector.MatchNames) != 0 {
			isMultiNS = true
			break
		}
	}

	payloadValidation := true
	if value, ok := el.GetAnnotations()[triggers.PayloadValidationAnnotation]; ok {
		if value == "false" {
			payloadValidation = false
		}
	}

	ev := configAcc.ToEnvVars()

	containerSecurityContext := corev1.SecurityContext{}
	if *c.SetSecurityContext {
		containerSecurityContext = corev1.SecurityContext{
			AllowPrivilegeEscalation: ptr.Bool(false),
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{"ALL"},
			},
			RunAsNonRoot: ptr.Bool(true),
			SeccompProfile: &corev1.SeccompProfile{
				Type: corev1.SeccompProfileTypeRuntimeDefault,
			},
		}
	}

	if cfg.Defaults.DefaultRunAsUser != "" {
		runAsUser, err := strconv.ParseInt(cfg.Defaults.DefaultRunAsUser, 10, 0)
		if err != nil {
			return corev1.Container{}, errors.New("failed parsing runAsUser config default-run-as-user")
		}
		containerSecurityContext.RunAsUser = ptr.Int64(runAsUser)
	}
	if cfg.Defaults.DefaultRunAsGroup != "" {
		runAsGroup, err := strconv.ParseInt(cfg.Defaults.DefaultRunAsGroup, 10, 0)
		if err != nil {
			return corev1.Container{}, errors.New("failed parsing runAsGroup config default-run-as-group")
		}
		containerSecurityContext.RunAsGroup = ptr.Int64(runAsGroup)
	}

	container := corev1.Container{
		Name:  "event-listener",
		Image: *c.Image,
		Ports: []corev1.ContainerPort{{
			ContainerPort: int32(eventListenerContainerPort),
			Protocol:      corev1.ProtocolTCP,
		}},
		Args: []string{
			"--el-name=" + el.Name,
			"--el-namespace=" + el.Namespace,
			"--port=" + strconv.Itoa(eventListenerContainerPort),
			"--readtimeout=" + strconv.FormatInt(*c.ReadTimeOut, 10),
			"--writetimeout=" + strconv.FormatInt(*c.WriteTimeOut, 10),
			"--idletimeout=" + strconv.FormatInt(*c.IdleTimeOut, 10),
			"--timeouthandler=" + strconv.FormatInt(*c.TimeOutHandler, 10),
			"--httpclient-readtimeout=" + strconv.FormatInt(*c.HTTPClientReadTimeOut, 10),
			"--httpclient-keep-alive=" + strconv.FormatInt(*c.HTTPClientKeepAlive, 10),
			"--httpclient-tlshandshaketimeout=" + strconv.FormatInt(*c.HTTPClientTLSHandshakeTimeout, 10),
			"--httpclient-responseheadertimeout=" + strconv.FormatInt(*c.HTTPClientResponseHeaderTimeout, 10),
			"--httpclient-expectcontinuetimeout=" + strconv.FormatInt(*c.HTTPClientExpectContinueTimeout, 10),
			"--is-multi-ns=" + strconv.FormatBool(isMultiNS),
			"--payload-validation=" + strconv.FormatBool(payloadValidation),
			"--cloudevent-uri=" + el.Spec.CloudEventURI,
		},
		Env: append(ev, []corev1.EnvVar{{
			Name:  "NAMESPACE",
			Value: el.Namespace,
		}, {
			Name:  "NAME",
			Value: el.Name,
		}, {
			Name:  "EL_EVENT",
			Value: *c.SetEventListenerEvent,
		}, {
			Name:  "K_SINK_TIMEOUT",
			Value: strconv.FormatInt(*c.TimeOutHandler, 10),
		}}...),
		SecurityContext: &containerSecurityContext,
	}

	for _, opt := range opts {
		opt(&container)
	}

	return container, nil
}
