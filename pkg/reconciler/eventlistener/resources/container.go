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
	reconcilersource "knative.dev/eventing/pkg/reconciler/source"
)

type ContainerOption func(*corev1.Container)

// var (
// 	MetricsConfig = `{"Domain":"` + TriggersMetricsDomain + `","Component":"eventlistener","ConfigMap":{}}`
// 	zapConfig     = `{"level": "info","development": false,"sampling": {"initial": 100,"thereafter": 100},"outputPaths": ["stdout"],"errorOutputPaths": ["stderr"],"encoding": "json","encoderConfig": {"timeKey": "ts","levelKey": "level","nameKey": "logger","callerKey": "caller","messageKey": "msg","stacktraceKey": "stacktrace","lineEnding": "","levelEncoder": "","timeEncoder": "iso8601","durationEncoder": "","callerEncoder": ""}}`
// 	LoggingConfig = fmt.Sprintf(`{"loglevel.eventlistener": "info", "zap-logger-config": %q}`, zapConfig)
// )

func MakeContainer(el *v1beta1.EventListener, configAcc reconcilersource.ConfigAccessor, c Config, opts ...ContainerOption) corev1.Container {
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

	ev := configAcc.ToEnvVars()

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
			"--is-multi-ns=" + strconv.FormatBool(isMultiNS),
			"--payload-validation=" + strconv.FormatBool(payloadValidation),
		},
		Env: append(ev, []corev1.EnvVar{{
			Name:  "NAMESPACE",
			Value: el.Namespace,
		}, {
			Name:  "NAME",
			Value: el.Name,
		}}...),
	}

	for _, opt := range opts {
		opt(&container)
	}

	return container
}
