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
	"time"

	"github.com/tektoncd/pipeline/pkg/apis/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/contexts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
)

var _ apis.Defaultable = (*TaskRun)(nil)

const managedByLabelKey = "app.kubernetes.io/managed-by"

func (tr *TaskRun) SetDefaults(ctx context.Context) {
	ctx = apis.WithinParent(ctx, tr.ObjectMeta)
	tr.Spec.SetDefaults(apis.WithinSpec(ctx))

	// If the TaskRun doesn't have a managed-by label, apply the default
	// specified in the config.
	cfg := config.FromContextOrDefaults(ctx)
	if tr.ObjectMeta.Labels == nil {
		tr.ObjectMeta.Labels = map[string]string{}
	}
	if _, found := tr.ObjectMeta.Labels[managedByLabelKey]; !found {
		tr.ObjectMeta.Labels[managedByLabelKey] = cfg.Defaults.DefaultManagedByLabelValue
	}
}

func (trs *TaskRunSpec) SetDefaults(ctx context.Context) {
	if contexts.IsUpgradeViaDefaulting(ctx) {
		v := v1beta1.TaskRunSpec{}
		if trs.ConvertTo(ctx, &v) == nil {
			alpha := TaskRunSpec{}
			if alpha.ConvertFrom(ctx, &v) == nil {
				*trs = alpha
			}
		}
	}

	cfg := config.FromContextOrDefaults(ctx)
	if trs.TaskRef != nil && trs.TaskRef.Kind == "" {
		trs.TaskRef.Kind = NamespacedTaskKind
	}

	if trs.Timeout == nil {
		trs.Timeout = &metav1.Duration{Duration: time.Duration(cfg.Defaults.DefaultTimeoutMinutes) * time.Minute}
	}

	defaultSA := cfg.Defaults.DefaultServiceAccount
	if trs.ServiceAccountName == "" && defaultSA != "" {
		trs.ServiceAccountName = defaultSA
	}

	defaultPodTemplate := cfg.Defaults.DefaultPodTemplate
	if trs.PodTemplate == nil {
		trs.PodTemplate = defaultPodTemplate
	}

	// If this taskrun has an embedded task, apply the usual task defaults
	if trs.TaskSpec != nil {
		trs.TaskSpec.SetDefaults(ctx)
	}
}
