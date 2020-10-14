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

package v1alpha1

import (
	"context"

	"knative.dev/pkg/ptr"
)

type triggerSpecBindingArray []*TriggerSpecBinding

// SetDefaults sets the defaults on the object.
func (t *Trigger) SetDefaults(ctx context.Context) {
	if !IsUpgradeViaDefaulting(ctx) {
		return
	}
	triggerSpecBindingArray(t.Spec.Bindings).defaultBindings()

	// Upgrade old style embedded bindings to new concise syntax
	bindings := []*TriggerSpecBinding{}
	for _, b := range t.Spec.Bindings {
		if b.Spec == nil {
			bindings = append(bindings, b)
		} else {
			for _, p := range b.Spec.Params {
				bindings = append(bindings, &TriggerSpecBinding{
					Name:  p.Name,
					Value: ptr.String(p.Value),
				})
			}
		}
	}
	t.Spec.Bindings = bindings
	templateNameToRef(&t.Spec.Template)
}

// set default TriggerBinding kind for Bindings in TriggerSpec
func (t triggerSpecBindingArray) defaultBindings() {
	if len(t) > 0 {
		for _, b := range t {
			if b.Kind == "" {
				b.Kind = NamespacedTriggerBindingKind
			}
		}
	}
}

func templateNameToRef(template *TriggerSpecTemplate) {
	name := template.Name
	if name != "" && template.Ref == nil {
		template.Ref = &name
		template.Name = ""
	}
}
