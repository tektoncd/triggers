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
	"fmt"
	"time"

	"github.com/tektoncd/pipeline/pkg/apis/config"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/apis"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
)

var (
	pipelineRunControllerName = "PipelineRun"
	groupVersionKind          = schema.GroupVersionKind{
		Group:   SchemeGroupVersion.Group,
		Version: SchemeGroupVersion.Version,
		Kind:    pipelineRunControllerName,
	}
)

// PipelineRunSpec defines the desired state of PipelineRun
type PipelineRunSpec struct {
	// +optional
	PipelineRef *PipelineRef `json:"pipelineRef,omitempty"`
	// +optional
	PipelineSpec *PipelineSpec `json:"pipelineSpec,omitempty"`
	// Resources is a list of bindings specifying which actual instances of
	// PipelineResources to use for the resources the Pipeline has declared
	// it needs.
	Resources []PipelineResourceBinding `json:"resources,omitempty"`
	// Params is a list of parameter names and values.
	Params []Param `json:"params,omitempty"`
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
	// +optional
	ServiceAccountNames []PipelineRunSpecServiceAccountName `json:"serviceAccountNames,omitempty"`
	// Used for cancelling a pipelinerun (and maybe more later on)
	// +optional
	Status PipelineRunSpecStatus `json:"status,omitempty"`
	// Time after which the Pipeline times out. Defaults to never.
	// Refer to Go's ParseDuration documentation for expected format: https://golang.org/pkg/time/#ParseDuration
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`

	// PodTemplate holds pod specific configuration
	PodTemplate PodTemplate `json:"podTemplate,omitempty"`
}

// PipelineRunSpecStatus defines the pipelinerun spec status the user can provide
type PipelineRunSpecStatus string

const (
	// PipelineRunSpecStatusCancelled indicates that the user wants to cancel the task,
	// if not already cancelled or terminated
	PipelineRunSpecStatusCancelled = "PipelineRunCancelled"
)

// PipelineResourceRef can be used to refer to a specific instance of a Resource
type PipelineResourceRef struct {
	// Name of the referent; More info: http://kubernetes.io/docs/user-guide/identifiers#names
	Name string `json:"name,omitempty"`
	// API version of the referent
	// +optional
	APIVersion string `json:"apiVersion,omitempty"`
}

// PipelineRef can be used to refer to a specific instance of a Pipeline.
// Copied from CrossVersionObjectReference: https://github.com/kubernetes/kubernetes/blob/169df7434155cbbc22f1532cba8e0a9588e29ad8/pkg/apis/autoscaling/types.go#L64
type PipelineRef struct {
	// Name of the referent; More info: http://kubernetes.io/docs/user-guide/identifiers#names
	Name string `json:"name,omitempty"`
	// API version of the referent
	// +optional
	APIVersion string `json:"apiVersion,omitempty"`
}

// PipelineRunStatus defines the observed state of PipelineRun
type PipelineRunStatus struct {
	duckv1beta1.Status `json:",inline"`

	// StartTime is the time the PipelineRun is actually started.
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime is the time the PipelineRun completed.
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// map of PipelineRunTaskRunStatus with the taskRun name as the key
	// +optional
	TaskRuns map[string]*PipelineRunTaskRunStatus `json:"taskRuns,omitempty"`
}

// PipelineRunTaskRunStatus contains the name of the PipelineTask for this TaskRun and the TaskRun's Status
type PipelineRunTaskRunStatus struct {
	// PipelineTaskName is the name of the PipelineTask.
	PipelineTaskName string `json:"pipelineTaskName,omitempty"`
	// Status is the TaskRunStatus for the corresponding TaskRun
	// +optional
	Status *TaskRunStatus `json:"status,omitempty"`
	// ConditionChecks maps the name of a condition check to its Status
	// +optional
	ConditionChecks map[string]*PipelineRunConditionCheckStatus `json:"conditionChecks,omitempty"`
}

type PipelineRunConditionCheckStatus struct {
	// ConditionName is the name of the Condition
	ConditionName string `json:"conditionName,omitempty"`
	// Status is the ConditionCheckStatus for the corresponding ConditionCheck
	// +optional
	Status *ConditionCheckStatus `json:"status,omitempty"`
}

var pipelineRunCondSet = apis.NewBatchConditionSet()

// GetCondition returns the Condition matching the given type.
func (pr *PipelineRunStatus) GetCondition(t apis.ConditionType) *apis.Condition {
	return pipelineRunCondSet.Manage(pr).GetCondition(t)
}

// InitializeConditions will set all conditions in pipelineRunCondSet to unknown for the PipelineRun
// and set the started time to the current time
func (pr *PipelineRunStatus) InitializeConditions() {
	if pr.TaskRuns == nil {
		pr.TaskRuns = make(map[string]*PipelineRunTaskRunStatus)
	}
	if pr.StartTime.IsZero() {
		pr.StartTime = &metav1.Time{Time: time.Now()}
	}
	pipelineRunCondSet.Manage(pr).InitializeConditions()
}

// PipelineRunSpecServiceAccountName can be used to configure specific
// ServiceAccountName for a concrete Task
type PipelineRunSpecServiceAccountName struct {
	TaskName           string `json:"taskName,omitempty"`
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
}

// SetCondition sets the condition, unsetting previous conditions with the same
// type as necessary.
func (pr *PipelineRunStatus) SetCondition(newCond *apis.Condition) {
	if newCond != nil {
		pipelineRunCondSet.Manage(pr).SetCondition(*newCond)
	}
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PipelineRun represents a single execution of a Pipeline. PipelineRuns are how
// the graph of Tasks declared in a Pipeline are executed; they specify inputs
// to Pipelines such as parameter values and capture operational aspects of the
// Tasks execution such as service account and tolerations. Creating a
// PipelineRun creates TaskRuns for Tasks in the referenced Pipeline.
//
// +k8s:openapi-gen=true
type PipelineRun struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +optional
	Spec PipelineRunSpec `json:"spec,omitempty"`
	// +optional
	Status PipelineRunStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PipelineRunList contains a list of PipelineRun
type PipelineRunList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PipelineRun `json:"items,omitempty"`
}

// PipelineTaskRun reports the results of running a step in the Task. Each
// task has the potential to succeed or fail (based on the exit code)
// and produces logs.
type PipelineTaskRun struct {
	Name string `json:"name,omitempty"`
}

// GetTaskRunRef for pipelinerun
func (pr *PipelineRun) GetTaskRunRef() corev1.ObjectReference {
	return corev1.ObjectReference{
		APIVersion: SchemeGroupVersion.String(),
		Kind:       "TaskRun",
		Namespace:  pr.Namespace,
		Name:       pr.Name,
	}
}

// GetOwnerReference gets the pipeline run as owner reference for any related objects
func (pr *PipelineRun) GetOwnerReference() []metav1.OwnerReference {
	return []metav1.OwnerReference{
		*metav1.NewControllerRef(pr, groupVersionKind),
	}
}

// IsDone returns true if the PipelineRun's status indicates that it is done.
func (pr *PipelineRun) IsDone() bool {
	return !pr.Status.GetCondition(apis.ConditionSucceeded).IsUnknown()
}

// HasStarted function check whether pipelinerun has valid start time set in its status
func (pr *PipelineRun) HasStarted() bool {
	return pr.Status.StartTime != nil && !pr.Status.StartTime.IsZero()
}

// IsCancelled returns true if the PipelineRun's spec status is set to Cancelled state
func (pr *PipelineRun) IsCancelled() bool {
	return pr.Spec.Status == PipelineRunSpecStatusCancelled
}

// GetRunKey return the pipelinerun key for timeout handler map
func (pr *PipelineRun) GetRunKey() string {
	// The address of the pointer is a threadsafe unique identifier for the pipelinerun
	return fmt.Sprintf("%s/%p", pipelineRunControllerName, pr)
}

// IsTimedOut returns true if a pipelinerun has exceeded its spec.Timeout based on its status.Timeout
func (pr *PipelineRun) IsTimedOut() bool {
	pipelineTimeout := pr.Spec.Timeout
	startTime := pr.Status.StartTime

	if !startTime.IsZero() && pipelineTimeout != nil {
		timeout := pipelineTimeout.Duration
		if timeout == config.NoTimeoutDuration {
			return false
		}
		runtime := time.Since(startTime.Time)
		if runtime > timeout {
			return true
		}
	}
	return false
}

// GetServiceAccountName returns the service account name for a given
// PipelineTask if configured, otherwise it returns the PipelineRun's serviceAccountName.
func (pr *PipelineRun) GetServiceAccountName(pipelineTaskName string) string {
	serviceAccountName := pr.Spec.ServiceAccountName
	for _, sa := range pr.Spec.ServiceAccountNames {
		if sa.TaskName == pipelineTaskName {
			serviceAccountName = sa.ServiceAccountName
		}
	}
	return serviceAccountName
}
