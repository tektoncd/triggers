/*
Copyright 2020 The Tekton Authors

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

package builder

import (
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TriggerOp is an operation which modifies the Trigger.
type TriggerOp func(*v1alpha1.Trigger)

// TriggerSpecOp is an operation which modifies the EventListenerSpec.
type TriggerSpecOp func(*v1alpha1.TriggerSpec)

// TriggerInterceptorOp is an operation which modifies the EventInterceptor.
type TriggerInterceptorOp func(*v1alpha1.TriggerInterceptor)

// Trigger creates an Trigger with default values.
// Any number of TriggerOp modifiers can be passed to transform it.
func Trigger(name, namespace string, ops ...TriggerOp) *v1alpha1.Trigger {
	e := &v1alpha1.Trigger{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	for _, op := range ops {
		op(e)
	}

	return e
}

// TriggerSpec sets the specified spec of the EventListener.
// Any number of EventListenerSpecOp modifiers can be passed to create/modify it.
func TriggerSpec(ops ...TriggerSpecOp) TriggerOp {
	return func(t *v1alpha1.Trigger) {
		for _, op := range ops {
			op(&t.Spec)
		}
	}
}

// TriggerSpecTemplate adds an TriggerTemplate to the TriggerSpec.
func TriggerSpecTemplate(ttName, apiVersion string) TriggerSpecOp {
	return func(spec *v1alpha1.TriggerSpec) {
		tt := v1alpha1.TriggerSpecTemplate{
			Ref:        &ttName,
			APIVersion: apiVersion,
		}
		spec.Template = tt
	}
}

// TriggerSpecName adds a Name to the Trigger in TriggerSpec.
func TriggerSpecName(name string) TriggerSpecOp {
	return func(spec *v1alpha1.TriggerSpec) {
		spec.Name = name
	}
}

// TriggerSpecServiceAccountName set the specified ServiceAccount of the Trigger.
func TriggerSpecServiceAccountName(saName string) TriggerSpecOp {
	return func(spec *v1alpha1.TriggerSpec) {
		spec.ServiceAccountName = saName
	}
}

// TriggerSpecBinding adds a Binding to the Trigger in TriggerSpec.
func TriggerSpecBinding(ref, kind, name, apiVersion string) TriggerSpecOp {
	return func(spec *v1alpha1.TriggerSpec) {
		binding := &v1alpha1.TriggerSpecBinding{
			Name:       name,
			APIVersion: apiVersion,
		}

		if len(ref) != 0 {
			binding.Ref = ref
			if kind == "ClusterTriggerBinding" {
				binding.Kind = v1alpha1.ClusterTriggerBindingKind
			}

			if kind == "TriggerBinding" || kind == "" {
				binding.Kind = v1alpha1.NamespacedTriggerBindingKind
			}
		}
		spec.Bindings = append(spec.Bindings, binding)
	}
}

// TriggerSpecInterceptor adds an objectRef to an interceptor Service to the TriggerSpec.
func TriggerSpecInterceptor(name, version, kind, namespace string, ops ...TriggerInterceptorOp) TriggerSpecOp {
	return func(spec *v1alpha1.TriggerSpec) {
		i := &v1alpha1.TriggerInterceptor{
			Webhook: &v1alpha1.WebhookInterceptor{
				ObjectRef: &corev1.ObjectReference{
					Kind:       kind,
					Name:       name,
					APIVersion: version,
					Namespace:  namespace,
				},
			},
		}
		for _, op := range ops {
			op(i)
		}
		spec.Interceptors = append(spec.Interceptors, i)
	}
}

// TriggerSpecInterceptorParam adds a parameter to the TriggerInterceptor.
func TriggerSpecInterceptorParam(name, value string) TriggerInterceptorOp {
	return func(i *v1alpha1.TriggerInterceptor) {
		if i.Webhook != nil {
			for _, param := range i.Webhook.Header {
				if param.Name == name {
					param.Value.ArrayVal = append(param.Value.ArrayVal, value)
					return
				}
			}
			i.Webhook.Header = append(i.Webhook.Header,
				pipelinev1.Param{
					Name: name,
					Value: pipelinev1.ArrayOrString{
						ArrayVal: []string{value},
						Type:     pipelinev1.ParamTypeArray,
					},
				})
		}
	}
}

// TriggerSpecCELInterceptor adds a DeprecatedCEL filter to the TriggerSpecTrigger.
func TriggerSpecCELInterceptor(filter string, ops ...TriggerInterceptorOp) TriggerSpecOp {
	return func(spec *v1alpha1.TriggerSpec) {
		i := &v1alpha1.TriggerInterceptor{
			DeprecatedCEL: &v1alpha1.CELInterceptor{
				Filter: filter,
			},
		}
		for _, op := range ops {
			op(i)
		}
		spec.Interceptors = append(spec.Interceptors, i)
	}
}

// TriggerSpecCELOverlay modifies DeprecatedCEL interceptor
func TriggerSpecCELOverlay(key, expression string) TriggerInterceptorOp {
	return func(i *v1alpha1.TriggerInterceptor) {
		if i.DeprecatedCEL != nil {
			i.DeprecatedCEL.Overlays = append(i.DeprecatedCEL.Overlays, v1alpha1.CELOverlay{
				Key:        key,
				Expression: expression,
			})
		}
	}
}
