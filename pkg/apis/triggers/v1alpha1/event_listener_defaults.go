/*
Copyright 2019 The Tekton Authors

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

package v1alpha1

import (
	"context"

	"github.com/tektoncd/triggers/pkg/apis/config"
	"github.com/tektoncd/triggers/pkg/apis/triggers/contexts"
	"knative.dev/pkg/logging"
)

// SetDefaults sets the defaults on the object.
func (el *EventListener) SetDefaults(ctx context.Context) {
	cfg := config.FromContextOrDefaults(ctx)
	if contexts.IsUpgradeViaDefaulting(ctx) {
		defaultSA := cfg.Defaults.DefaultServiceAccount
		if el.Spec.ServiceAccountName == "" && defaultSA != "" {
			el.Spec.ServiceAccountName = defaultSA
		}
		// set defaults
		if el.Spec.Resources.KubernetesResource != nil {
			if el.Spec.Resources.KubernetesResource.Replicas != nil && *el.Spec.Resources.KubernetesResource.Replicas == 0 {
				*el.Spec.Resources.KubernetesResource.Replicas = 1
			}
		}

		for i, t := range el.Spec.Triggers {
			triggerSpecBindingArray(el.Spec.Triggers[i].Bindings).defaultBindings()
			for _, ti := range t.Interceptors {
				if ti != nil {
					ti.defaultInterceptorKind()
					if err := ti.updateCoreInterceptors(); err != nil {
						// The err only happens due to malformed JSON and should never really happen
						// We can't return an error here, so print out the error
						logger := logging.FromContext(ctx)
						logger.Errorf("failed to setDefaults for trigger: %s; err: %s", t.Name, err)
					}
				}
			}
		}
	}
}
