/*
Copyright 2024 The Tekton Authors

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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"
)

// Check that ScheduledTemplate may be validated and defaulted.
var _ apis.Validatable = (*ScheduledTemplate)(nil)
var _ apis.Defaultable = (*ScheduledTemplate)(nil)
var _ kmeta.OwnerRefable = (*ScheduledTemplate)(nil)

// ScheduledTemplateSpec holds the desired state of ScheduledTemplate
type ScheduledTemplateSpec struct {
	// Ref optionally names a TriggerTemplate in the same namespace.
	// Mutually exclusive with inline Params / ResourceTemplates.
	// +optional
	Ref *string `json:"ref,omitempty"`
	// +listType=atomic
	// Embed TriggerTemplateSpec to inherit Params and ResourceTemplates
	TriggerTemplateSpec `json:",inline"`
	// CloudEventSink is an optional destination for CloudEvents related to scheduled runs.
	// +optional
	CloudEventSink *apis.URL `json:"cloudEventSink,omitempty"`
	// Schedule is a Kubernetes CronJob-style cron expression (five fields:
	// minute hour day-of-month month day-of-week). Example: "0 0 * * *"
	Schedule string `json:"schedule,omitempty"`
	// ServiceAccountName is the name of the ServiceAccount to use for the
	// owned CronJob pod that creates resources on schedule.
	// Defaults to "default".
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
}

// ScheduledTemplateStatus describes the observed state of ScheduledTemplate.
// +k8s:deepcopy-gen=true
type ScheduledTemplateStatus struct {
	duckv1.Status `json:",inline"`

	// CronJobName is the name of the CronJob owned by this ScheduledTemplate.
	// +optional
	CronJobName string `json:"cronJobName,omitempty"`

	// ObservedGeneration is the 'Generation' of the ScheduledTemplate that
	// was last processed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// LastScheduleTime is the last time the schedule triggered a Job, when known.
	// +optional
	LastScheduleTime *metav1.Time `json:"lastScheduleTime,omitempty"`

	// LastSuccessfulTime is the last time a scheduled create succeeded, when known.
	// +optional
	LastSuccessfulTime *metav1.Time `json:"lastSuccessfulTime,omitempty"`

	// Active lists currently running Jobs created for this ScheduledTemplate, when known.
	// +optional
	// +listType=atomic
	Active []corev1.ObjectReference `json:"active,omitempty"`
}

// +genclient
// +genreconciler:krshapedlogic=false
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ScheduledTemplate takes parameters and uses them to create CRDs
// +k8s:openapi-gen=true
type ScheduledTemplate struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec holds the desired state of the ScheduledTemplate from the client
	// +optional
	Spec ScheduledTemplateSpec `json:"spec"`
	// +optional
	Status ScheduledTemplateStatus `json:"status,omitempty"`
}

// ScheduledTemplateList contains a list of ScheduledTemplate
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ScheduledTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ScheduledTemplate `json:"items"`
}

// The conditions that are internally resolved by the ScheduledTemplate reconciler
const (
	// CronJobExists is the ConditionType set on the ScheduledTemplate, which
	// specifies CronJob existence.
	CronJobExists apis.ConditionType = "CronJob"
)

var scheduledTemplateCondSet = apis.NewLivingConditionSet(CronJobExists)

// GetGroupVersionKind implements kmeta.OwnerRefable
func (st *ScheduledTemplate) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("ScheduledTemplate")
}

// GetCondition returns the Condition matching the given type.
func (sts *ScheduledTemplateStatus) GetCondition(t apis.ConditionType) *apis.Condition {
	return scheduledTemplateCondSet.Manage(sts).GetCondition(t)
}

// SetCondition sets the condition, unsetting previous conditions with the same
// type as necessary. This is a local change and needs to be persisted to the
// K8s API elsewhere.
func (sts *ScheduledTemplateStatus) SetCondition(newCond *apis.Condition) {
	if newCond != nil {
		scheduledTemplateCondSet.Manage(sts).SetCondition(*newCond)
	}
}

// SetExistsCondition simplifies setting the exists conditions on the
// ScheduledTemplateStatus.
func (sts *ScheduledTemplateStatus) SetExistsCondition(cond apis.ConditionType, err error) {
	if err != nil {
		sts.SetCondition(&apis.Condition{
			Type:    cond,
			Status:  corev1.ConditionFalse,
			Message: err.Error(),
		})
		return
	}
	sts.SetCondition(&apis.Condition{
		Type:    cond,
		Status:  corev1.ConditionTrue,
		Message: fmt.Sprintf("%s exists", cond),
	})
}

// InitializeConditions will set all conditions in scheduledTemplateCondSet to false
// for the ScheduledTemplate. This is a local change and needs to be persisted to
// the K8s API elsewhere.
func (sts *ScheduledTemplateStatus) InitializeConditions() {
	for _, condition := range []apis.ConditionType{
		CronJobExists,
		apis.ConditionReady,
	} {
		sts.SetCondition(&apis.Condition{
			Type:   condition,
			Status: corev1.ConditionFalse,
		})
	}
}

// SetReadyCondition sets Ready based on whether the owned CronJob exists.
func (sts *ScheduledTemplateStatus) SetReadyCondition() {
	if sc := sts.GetCondition(CronJobExists); sc != nil {
		if sc.Status != corev1.ConditionTrue {
			sts.SetCondition(&apis.Condition{
				Type:    apis.ConditionReady,
				Status:  corev1.ConditionFalse,
				Message: fmt.Sprintf("Condition %s has status: %s with message: %s", sc.Type, sc.Status, sc.Message),
			})
			return
		}
	}
	sts.SetCondition(&apis.Condition{
		Type:    apis.ConditionReady,
		Status:  corev1.ConditionTrue,
		Message: "ScheduledTemplate is ready",
	})
}
