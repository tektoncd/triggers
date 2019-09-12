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

package builder

import (
	"time"

	"github.com/tektoncd/pipeline/pkg/apis/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
)

// PipelineOp is an operation which modify a Pipeline struct.
type PipelineOp func(*v1alpha1.Pipeline)

// PipelineSpecOp is an operation which modify a PipelineSpec struct.
type PipelineSpecOp func(*v1alpha1.PipelineSpec)

// PipelineTaskOp is an operation which modify a PipelineTask struct.
type PipelineTaskOp func(*v1alpha1.PipelineTask)

// PipelineRunOp is an operation which modify a PipelineRun struct.
type PipelineRunOp func(*v1alpha1.PipelineRun)

// PipelineRunSpecOp is an operation which modify a PipelineRunSpec struct.
type PipelineRunSpecOp func(*v1alpha1.PipelineRunSpec)

// PipelineResourceOp is an operation which modify a PipelineResource struct.
type PipelineResourceOp func(*v1alpha1.PipelineResource)

// PipelineResourceBindingOp is an operation which modify a PipelineResourceBinding struct.
type PipelineResourceBindingOp func(*v1alpha1.PipelineResourceBinding)

// PipelineResourceSpecOp is an operation which modify a PipelineResourceSpec struct.
type PipelineResourceSpecOp func(*v1alpha1.PipelineResourceSpec)

// PipelineTaskInputResourceOp is an operation which modifies a PipelineTaskInputResource.
type PipelineTaskInputResourceOp func(*v1alpha1.PipelineTaskInputResource)

// PipelineRunStatusOp is an operation which modifies a PipelineRunStatus
type PipelineRunStatusOp func(*v1alpha1.PipelineRunStatus)

// PipelineTaskConditionOp is an operation which modifies a PipelineTaskCondition
type PipelineTaskConditionOp func(condition *v1alpha1.PipelineTaskCondition)

// Pipeline creates a Pipeline with default values.
// Any number of Pipeline modifier can be passed to transform it.
func Pipeline(name, namespace string, ops ...PipelineOp) *v1alpha1.Pipeline {
	p := &v1alpha1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}

	for _, op := range ops {
		op(p)
	}

	return p
}

// PipelineSpec sets the PipelineSpec to the Pipeline.
// Any number of PipelineSpec modifier can be passed to transform it.
func PipelineSpec(ops ...PipelineSpecOp) PipelineOp {
	return func(p *v1alpha1.Pipeline) {
		ps := &p.Spec

		for _, op := range ops {
			op(ps)
		}

		p.Spec = *ps
	}
}

// PipelineCreationTimestamp sets the creation time of the pipeline
func PipelineCreationTimestamp(t time.Time) PipelineOp {
	return func(p *v1alpha1.Pipeline) {
		p.CreationTimestamp = metav1.Time{Time: t}
	}
}

// PipelineRunCancelled sets the status to cancel to the TaskRunSpec.
func PipelineRunCancelled(spec *v1alpha1.PipelineRunSpec) {
	spec.Status = v1alpha1.PipelineRunSpecStatusCancelled
}

// PipelineDeclaredResource adds a resource declaration to the Pipeline Spec,
// with the specified name and type.
func PipelineDeclaredResource(name string, t v1alpha1.PipelineResourceType) PipelineSpecOp {
	return func(ps *v1alpha1.PipelineSpec) {
		r := v1alpha1.PipelineDeclaredResource{
			Name: name,
			Type: t,
		}
		ps.Resources = append(ps.Resources, r)
	}
}

// PipelineParamSpec adds a param, with specified name and type, to the PipelineSpec.
// Any number of PipelineParamSpec modifiers can be passed to transform it.
func PipelineParamSpec(name string, pt v1alpha1.ParamType, ops ...ParamSpecOp) PipelineSpecOp {
	return func(ps *v1alpha1.PipelineSpec) {
		pp := &v1alpha1.ParamSpec{Name: name, Type: pt}
		for _, op := range ops {
			op(pp)
		}
		ps.Params = append(ps.Params, *pp)
	}
}

// PipelineTask adds a PipelineTask, with specified name and task name, to the PipelineSpec.
// Any number of PipelineTask modifier can be passed to transform it.
func PipelineTask(name, taskName string, ops ...PipelineTaskOp) PipelineSpecOp {
	return func(ps *v1alpha1.PipelineSpec) {
		pTask := &v1alpha1.PipelineTask{
			Name: name,
			TaskRef: v1alpha1.TaskRef{
				Name: taskName,
			},
		}
		for _, op := range ops {
			op(pTask)
		}
		ps.Tasks = append(ps.Tasks, *pTask)
	}
}

func Retries(retries int) PipelineTaskOp {
	return func(pt *v1alpha1.PipelineTask) {
		pt.Retries = retries
	}
}

// RunAfter will update the provided Pipeline Task to indicate that it
// should be run after the provided list of Pipeline Task names.
func RunAfter(tasks ...string) PipelineTaskOp {
	return func(pt *v1alpha1.PipelineTask) {
		pt.RunAfter = tasks
	}
}

// PipelineTaskRefKind sets the TaskKind to the PipelineTaskRef.
func PipelineTaskRefKind(kind v1alpha1.TaskKind) PipelineTaskOp {
	return func(pt *v1alpha1.PipelineTask) {
		pt.TaskRef.Kind = kind
	}
}

// PipelineTaskParam adds a ResourceParam, with specified name and value, to the PipelineTask.
func PipelineTaskParam(name string, value string, additionalValues ...string) PipelineTaskOp {
	arrayOrString := ArrayOrString(value, additionalValues...)
	return func(pt *v1alpha1.PipelineTask) {
		pt.Params = append(pt.Params, v1alpha1.Param{
			Name:  name,
			Value: *arrayOrString,
		})
	}
}

// From will update the provided PipelineTaskInputResource to indicate that it
// should come from tasks.
func From(tasks ...string) PipelineTaskInputResourceOp {
	return func(r *v1alpha1.PipelineTaskInputResource) {
		r.From = tasks
	}
}

// PipelineTaskInputResource adds an input resource to the PipelineTask with the specified
// name, pointing at the declared resource.
// Any number of PipelineTaskInputResource modifies can be passed to transform it.
func PipelineTaskInputResource(name, resource string, ops ...PipelineTaskInputResourceOp) PipelineTaskOp {
	return func(pt *v1alpha1.PipelineTask) {
		r := v1alpha1.PipelineTaskInputResource{
			Name:     name,
			Resource: resource,
		}
		for _, op := range ops {
			op(&r)
		}
		if pt.Resources == nil {
			pt.Resources = &v1alpha1.PipelineTaskResources{}
		}
		pt.Resources.Inputs = append(pt.Resources.Inputs, r)
	}
}

// PipelineTaskOutputResource adds an output resource to the PipelineTask with the specified
// name, pointing at the declared resource.
func PipelineTaskOutputResource(name, resource string) PipelineTaskOp {
	return func(pt *v1alpha1.PipelineTask) {
		r := v1alpha1.PipelineTaskOutputResource{
			Name:     name,
			Resource: resource,
		}
		if pt.Resources == nil {
			pt.Resources = &v1alpha1.PipelineTaskResources{}
		}
		pt.Resources.Outputs = append(pt.Resources.Outputs, r)
	}
}

// PipelineTaskCondition adds a condition to the PipelineTask with the
// specified conditionRef. Any number of PipelineTaskCondition modifiers can be passed
// to transform it
func PipelineTaskCondition(conditionRef string, ops ...PipelineTaskConditionOp) PipelineTaskOp {
	return func(pt *v1alpha1.PipelineTask) {
		c := &v1alpha1.PipelineTaskCondition{
			ConditionRef: conditionRef,
		}
		for _, op := range ops {
			op(c)
		}
		pt.Conditions = append(pt.Conditions, *c)
	}
}

// PipelineTaskCondition adds a parameter to a PipelineTaskCondition
func PipelineTaskConditionParam(name, val string) PipelineTaskConditionOp {
	return func(condition *v1alpha1.PipelineTaskCondition) {
		if condition.Params == nil {
			condition.Params = []v1alpha1.Param{}
		}
		condition.Params = append(condition.Params, v1alpha1.Param{
			Name:  name,
			Value: *ArrayOrString(val),
		})
	}
}

// PipelineRun creates a PipelineRun with default values.
// Any number of PipelineRun modifier can be passed to transform it.
func PipelineRun(name, namespace string, ops ...PipelineRunOp) *v1alpha1.PipelineRun {
	pr := &v1alpha1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: v1alpha1.PipelineRunSpec{},
	}

	for _, op := range ops {
		op(pr)
	}

	return pr
}

// PipelineRunSpec sets the PipelineRunSpec, references Pipeline with specified name, to the PipelineRun.
// Any number of PipelineRunSpec modifier can be passed to transform it.
func PipelineRunSpec(name string, ops ...PipelineRunSpecOp) PipelineRunOp {
	return func(pr *v1alpha1.PipelineRun) {
		prs := &pr.Spec

		prs.PipelineRef.Name = name
		// Set a default timeout
		prs.Timeout = &metav1.Duration{Duration: config.DefaultTimeoutMinutes * time.Minute}

		for _, op := range ops {
			op(prs)
		}

		pr.Spec = *prs
	}
}

// PipelineRunLabels adds a label to the PipelineRun.
func PipelineRunLabel(key, value string) PipelineRunOp {
	return func(pr *v1alpha1.PipelineRun) {
		if pr.ObjectMeta.Labels == nil {
			pr.ObjectMeta.Labels = map[string]string{}
		}
		pr.ObjectMeta.Labels[key] = value
	}
}

// PipelineRunAnnotations adds a annotation to the PipelineRun.
func PipelineRunAnnotation(key, value string) PipelineRunOp {
	return func(pr *v1alpha1.PipelineRun) {
		if pr.ObjectMeta.Annotations == nil {
			pr.ObjectMeta.Annotations = map[string]string{}
		}
		pr.ObjectMeta.Annotations[key] = value
	}
}

// PipelineRunResourceBinding adds bindings from actual instances to a Pipeline's declared resources.
func PipelineRunResourceBinding(name string, ops ...PipelineResourceBindingOp) PipelineRunSpecOp {
	return func(prs *v1alpha1.PipelineRunSpec) {
		r := &v1alpha1.PipelineResourceBinding{
			Name: name,
			ResourceRef: v1alpha1.PipelineResourceRef{
				Name: name,
			},
		}
		for _, op := range ops {
			op(r)
		}
		prs.Resources = append(prs.Resources, *r)
	}
}

// PipelineResourceBindingRef set the ResourceRef name to the Resource called Name.
func PipelineResourceBindingRef(name string) PipelineResourceBindingOp {
	return func(b *v1alpha1.PipelineResourceBinding) {
		b.ResourceRef.Name = name
	}
}

// PipelineRunServiceAccount sets the service account to the PipelineRunSpec.
func PipelineRunServiceAccount(sa string) PipelineRunSpecOp {
	return func(prs *v1alpha1.PipelineRunSpec) {
		prs.ServiceAccount = sa
	}
}

// PipelineRunServiceAccountTask configures the service account for given Task in PipelineRun.
func PipelineRunServiceAccountTask(taskName, sa string) PipelineRunSpecOp {
	return func(prs *v1alpha1.PipelineRunSpec) {
		prs.ServiceAccounts = append(prs.ServiceAccounts, v1alpha1.PipelineRunSpecServiceAccount{
			TaskName:       taskName,
			ServiceAccount: sa,
		})
	}
}

// PipelineRunParam add a param, with specified name and value, to the PipelineRunSpec.
func PipelineRunParam(name string, value string, additionalValues ...string) PipelineRunSpecOp {
	arrayOrString := ArrayOrString(value, additionalValues...)
	return func(prs *v1alpha1.PipelineRunSpec) {
		prs.Params = append(prs.Params, v1alpha1.Param{
			Name:  name,
			Value: *arrayOrString,
		})
	}
}

// PipelineRunTimeout sets the timeout to the PipelineRunSpec.
func PipelineRunTimeout(duration time.Duration) PipelineRunSpecOp {
	return func(prs *v1alpha1.PipelineRunSpec) {
		prs.Timeout = &metav1.Duration{Duration: duration}
	}
}

// PipelineRunNilTimeout sets the timeout to nil on the PipelineRunSpec
func PipelineRunNilTimeout(prs *v1alpha1.PipelineRunSpec) {
	prs.Timeout = nil
}

// PipelineRunNodeSelector sets the Node selector to the PipelineSpec.
func PipelineRunNodeSelector(values map[string]string) PipelineRunSpecOp {
	return func(prs *v1alpha1.PipelineRunSpec) {
		prs.NodeSelector = values
	}
}

// PipelineRunTolerations sets the Node selector to the PipelineSpec.
func PipelineRunTolerations(values []corev1.Toleration) PipelineRunSpecOp {
	return func(prs *v1alpha1.PipelineRunSpec) {
		prs.Tolerations = values
	}
}

// PipelineRunAffinity sets the affinity to the PipelineSpec.
func PipelineRunAffinity(affinity *corev1.Affinity) PipelineRunSpecOp {
	return func(prs *v1alpha1.PipelineRunSpec) {
		prs.Affinity = affinity
	}
}

// PipelineRunStatus sets the PipelineRunStatus to the PipelineRun.
// Any number of PipelineRunStatus modifier can be passed to transform it.
func PipelineRunStatus(ops ...PipelineRunStatusOp) PipelineRunOp {
	return func(pr *v1alpha1.PipelineRun) {
		s := &v1alpha1.PipelineRunStatus{}
		for _, op := range ops {
			op(s)
		}
		pr.Status = *s
	}
}

// PipelineRunStatusCondition adds a StatusCondition to the TaskRunStatus.
func PipelineRunStatusCondition(condition apis.Condition) PipelineRunStatusOp {
	return func(s *v1alpha1.PipelineRunStatus) {
		s.Conditions = append(s.Conditions, condition)
	}
}

// PipelineRunStartTime sets the start time to the PipelineRunStatus.
func PipelineRunStartTime(startTime time.Time) PipelineRunStatusOp {
	return func(s *v1alpha1.PipelineRunStatus) {
		s.StartTime = &metav1.Time{Time: startTime}
	}
}

// PipelineRunCompletionTime sets the completion time  to the PipelineRunStatus.
func PipelineRunCompletionTime(t time.Time) PipelineRunStatusOp {
	return func(s *v1alpha1.PipelineRunStatus) {
		s.CompletionTime = &metav1.Time{Time: t}
	}
}

// PipelineRunTaskRunsStatus sets the status of TaskRun to the PipelineRunStatus.
func PipelineRunTaskRunsStatus(taskRunName string, status *v1alpha1.PipelineRunTaskRunStatus) PipelineRunStatusOp {
	return func(s *v1alpha1.PipelineRunStatus) {
		if s.TaskRuns == nil {
			s.TaskRuns = make(map[string]*v1alpha1.PipelineRunTaskRunStatus)
		}
		s.TaskRuns[taskRunName] = status
	}
}

// PipelineResource creates a PipelineResource with default values.
// Any number of PipelineResource modifier can be passed to transform it.
func PipelineResource(name, namespace string, ops ...PipelineResourceOp) *v1alpha1.PipelineResource {
	resource := &v1alpha1.PipelineResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	for _, op := range ops {
		op(resource)
	}
	return resource
}

// PipelineResourceSpec set the PipelineResourceSpec, with specified type, to the PipelineResource.
// Any number of PipelineResourceSpec modifier can be passed to transform it.
func PipelineResourceSpec(resourceType v1alpha1.PipelineResourceType, ops ...PipelineResourceSpecOp) PipelineResourceOp {
	return func(r *v1alpha1.PipelineResource) {
		spec := &r.Spec
		spec.Type = resourceType
		for _, op := range ops {
			op(spec)
		}
		r.Spec = *spec
	}
}

// PipelineResourceSpecParam adds a ResourceParam, with specified name and value, to the PipelineResourceSpec.
func PipelineResourceSpecParam(name, value string) PipelineResourceSpecOp {
	return func(spec *v1alpha1.PipelineResourceSpec) {
		spec.Params = append(spec.Params, v1alpha1.ResourceParam{
			Name:  name,
			Value: value,
		})
	}
}

// PipelineResourceSpecSecretParam adds a SecretParam, with specified fieldname, secretKey and secretName, to the PipelineResourceSpec.
func PipelineResourceSpecSecretParam(fieldname, secretName, secretKey string) PipelineResourceSpecOp {
	return func(spec *v1alpha1.PipelineResourceSpec) {
		spec.SecretParams = append(spec.SecretParams, v1alpha1.SecretParam{
			FieldName:  fieldname,
			SecretKey:  secretKey,
			SecretName: secretName,
		})
	}
}
