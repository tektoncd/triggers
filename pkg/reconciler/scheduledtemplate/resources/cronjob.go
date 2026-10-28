/*
Copyright 2026 The Tekton Authors

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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/tektoncd/triggers/pkg/apis/config"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/kmeta"
	"knative.dev/pkg/ptr"
)

const (
	// GeneratedResourcePrefix is the name prefix for CronJobs owned by a ScheduledTemplate.
	GeneratedResourcePrefix = "st"

	// EnvScheduledTemplateName is the CronJob pod env var for the ScheduledTemplate name.
	EnvScheduledTemplateName = "SCHEDULED_TEMPLATE_NAME"
	// EnvScheduledTemplateNamespace is the CronJob pod env var for the ScheduledTemplate namespace.
	EnvScheduledTemplateNamespace = "SCHEDULED_TEMPLATE_NAMESPACE"

	// LabelScheduledTemplate names the owning ScheduledTemplate.
	LabelScheduledTemplate = "triggers.tekton.dev/scheduledtemplate"

	// maxCronJobNameLength is the Kubernetes limit for CronJob names (Jobs append a timestamp suffix).
	maxCronJobNameLength = 52

	// hashSuffixLen is the hex length appended when truncating long CronJob names.
	hashSuffixLen = 8
)

// StaticResourceLabels are labels applied to all CronJobs owned by ScheduledTemplates.
var StaticResourceLabels = map[string]string{
	"app.kubernetes.io/managed-by": "ScheduledTemplate",
	"app.kubernetes.io/part-of":    "Triggers",
}

// CronJobName returns the name of the CronJob owned by the given ScheduledTemplate.
// Short names use st-<name>. Names that would exceed the CronJob limit use
// st-<truncated>-<hash> so different long names do not collide.
// The truncated prefix is trimmed of trailing "." / "-" so the result remains a
// valid DNS subdomain (avoids forms like "st-foo.-deadbeef").
func CronJobName(stName string) string {
	name := fmt.Sprintf("%s-%s", GeneratedResourcePrefix, stName)
	if len(name) <= maxCronJobNameLength {
		return name
	}

	sum := sha256.Sum256([]byte(stName))
	hash := hex.EncodeToString(sum[:])[:hashSuffixLen]
	// st- + truncated + - + hash
	prefixBudget := maxCronJobNameLength - len(GeneratedResourcePrefix) - 1 - 1 - hashSuffixLen
	if prefixBudget < 1 {
		return fmt.Sprintf("%s-%s", GeneratedResourcePrefix, hash)
	}
	trimmed := stName
	if len(trimmed) > prefixBudget {
		trimmed = trimmed[:prefixBudget]
	}
	trimmed = strings.TrimRight(trimmed, ".-")
	if trimmed == "" {
		return fmt.Sprintf("%s-%s", GeneratedResourcePrefix, hash)
	}
	return fmt.Sprintf("%s-%s-%s", GeneratedResourcePrefix, trimmed, hash)
}

// MakeCronJob builds the desired CronJob for a ScheduledTemplate.
// defaults controls optional RunAsUser/RunAsGroup (same empty-value behavior as EventListener).
// If defaults is nil, Triggers config defaults are used.
func MakeCronJob(st *v1alpha1.ScheduledTemplate, image string, defaults *config.Defaults) *batchv1.CronJob {
	if defaults == nil {
		defaults, _ = config.NewDefaultsFromMap(map[string]string{})
	}
	name := CronJobName(st.Name)
	labels := kmeta.UnionMaps(StaticResourceLabels, map[string]string{
		LabelScheduledTemplate: st.Name,
	})

	backoffLimit := int32(0)
	podSecurityContext := &corev1.PodSecurityContext{
		RunAsNonRoot: ptr.Bool(defaults.DefaultRunAsNonRoot),
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
	}
	if !defaults.IsDefaultRunAsUserEmpty {
		podSecurityContext.RunAsUser = ptr.Int64(defaults.DefaultRunAsUser)
	}
	if !defaults.IsDefaultRunAsGroupEmpty {
		podSecurityContext.RunAsGroup = ptr.Int64(defaults.DefaultRunAsGroup)
	}

	containerSecurityContext := &corev1.SecurityContext{
		AllowPrivilegeEscalation: ptr.Bool(false),
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
		RunAsNonRoot: ptr.Bool(defaults.DefaultRunAsNonRoot),
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
	}
	if !defaults.IsDefaultRunAsUserEmpty {
		containerSecurityContext.RunAsUser = ptr.Int64(defaults.DefaultRunAsUser)
	}
	if !defaults.IsDefaultRunAsGroupEmpty {
		containerSecurityContext.RunAsGroup = ptr.Int64(defaults.DefaultRunAsGroup)
	}

	return &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       st.Namespace,
			Labels:          labels,
			OwnerReferences: []metav1.OwnerReference{*kmeta.NewControllerRef(st)},
		},
		Spec: batchv1.CronJobSpec{
			Schedule: st.Spec.Schedule,
			JobTemplate: batchv1.JobTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: batchv1.JobSpec{
					BackoffLimit: &backoffLimit,
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: labels,
						},
						Spec: corev1.PodSpec{
							ServiceAccountName: st.Spec.ServiceAccountName,
							RestartPolicy:      corev1.RestartPolicyNever,
							SecurityContext:    podSecurityContext,
							Containers: []corev1.Container{{
								Name:            "cronrunner",
								Image:           image,
								SecurityContext: containerSecurityContext,
								Env: []corev1.EnvVar{{
									Name:  EnvScheduledTemplateName,
									Value: st.Name,
								}, {
									Name:  EnvScheduledTemplateNamespace,
									Value: st.Namespace,
								}},
							}},
						},
					},
				},
			},
		},
	}
}
