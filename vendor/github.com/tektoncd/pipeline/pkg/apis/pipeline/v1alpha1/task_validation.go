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
	"fmt"
	"strings"

	"github.com/tektoncd/pipeline/pkg/apis/validate"
	"github.com/tektoncd/pipeline/pkg/substitution"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/util/validation"
	"knative.dev/pkg/apis"
)

var _ apis.Validatable = (*Task)(nil)

func (t *Task) Validate(ctx context.Context) *apis.FieldError {
	if err := validate.ObjectMetadata(t.GetObjectMeta()); err != nil {
		return err.ViaField("metadata")
	}
	return t.Spec.Validate(ctx)
}

func (ts *TaskSpec) Validate(ctx context.Context) *apis.FieldError {
	if equality.Semantic.DeepEqual(ts, &TaskSpec{}) {
		return apis.ErrMissingField(apis.CurrentField)
	}

	if len(ts.Steps) == 0 {
		return apis.ErrMissingField("steps")
	}
	if err := ValidateVolumes(ts.Volumes).ViaField("volumes"); err != nil {
		return err
	}
	mergedSteps, err := MergeStepsWithStepTemplate(ts.StepTemplate, ts.Steps)
	if err != nil {
		return &apis.FieldError{
			Message: fmt.Sprintf("error merging step template and steps: %s", err),
			Paths:   []string{"stepTemplate"},
		}
	}

	if err := validateSteps(mergedSteps).ViaField("steps"); err != nil {
		return err
	}

	// A task doesn't have to have inputs or outputs, but if it does they must be valid.
	// A task can't duplicate input or output names.

	if ts.Inputs != nil {
		for _, resource := range ts.Inputs.Resources {
			if err := validateResourceType(resource, fmt.Sprintf("taskspec.Inputs.Resources.%s.Type", resource.Name)); err != nil {
				return err
			}
		}
		if err := checkForDuplicates(ts.Inputs.Resources, "taskspec.Inputs.Resources.Name"); err != nil {
			return err
		}
		if err := validateInputParameterTypes(ts.Inputs); err != nil {
			return err
		}
	}
	if ts.Outputs != nil {
		for _, resource := range ts.Outputs.Resources {
			if err := validateResourceType(resource, fmt.Sprintf("taskspec.Outputs.Resources.%s.Type", resource.Name)); err != nil {
				return err
			}
		}
		if err := checkForDuplicates(ts.Outputs.Resources, "taskspec.Outputs.Resources.Name"); err != nil {
			return err
		}
	}

	// Validate task step names
	for _, step := range ts.Steps {
		if errs := validation.IsDNS1123Label(step.Name); step.Name != "" && len(errs) > 0 {
			return &apis.FieldError{
				Message: fmt.Sprintf("invalid value %q", step.Name),
				Paths:   []string{"taskspec.steps.name"},
				Details: "Task step name must be a valid DNS Label, For more info refer to https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
			}
		}
	}

	if err := validateInputParameterVariables(ts.Steps, ts.Inputs); err != nil {
		return err
	}
	if err := validateResourceVariables(ts.Steps, ts.Inputs, ts.Outputs); err != nil {
		return err
	}
	return nil
}

func ValidateVolumes(volumes []corev1.Volume) *apis.FieldError {
	// Task must not have duplicate volume names.
	vols := map[string]struct{}{}
	for _, v := range volumes {
		if _, ok := vols[v.Name]; ok {
			return &apis.FieldError{
				Message: fmt.Sprintf("multiple volumes with same name %q", v.Name),
				Paths:   []string{"name"},
			}
		}
		vols[v.Name] = struct{}{}
	}
	return nil
}

func validateSteps(steps []Step) *apis.FieldError {
	// Task must not have duplicate step names.
	names := map[string]struct{}{}
	for _, s := range steps {
		if s.Image == "" {
			return apis.ErrMissingField("Image")
		}

		if s.Script != "" {
			if len(s.Args) > 0 || len(s.Command) > 0 {
				return &apis.FieldError{
					Message: "script cannot be used with args or command",
					Paths:   []string{"script"},
				}
			}
			if !strings.HasPrefix(strings.TrimSpace(s.Script), "#!") {
				return &apis.FieldError{
					Message: "script must start with a shebang (#!)",
					Paths:   []string{"script"},
				}
			}
		}

		if s.Name == "" {
			continue
		}
		if _, ok := names[s.Name]; ok {
			return apis.ErrInvalidValue(s.Name, "name")
		}
		names[s.Name] = struct{}{}
	}
	return nil
}

func validateInputParameterTypes(inputs *Inputs) *apis.FieldError {
	for _, p := range inputs.Params {
		// Ensure param has a valid type.
		validType := false
		for _, allowedType := range AllParamTypes {
			if p.Type == allowedType {
				validType = true
			}
		}
		if !validType {
			return apis.ErrInvalidValue(p.Type, fmt.Sprintf("taskspec.inputs.params.%s.type", p.Name))
		}

		// If a default value is provided, ensure its type matches param's declared type.
		if (p.Default != nil) && (p.Default.Type != p.Type) {
			return &apis.FieldError{
				Message: fmt.Sprintf(
					"\"%v\" type does not match default value's type: \"%v\"", p.Type, p.Default.Type),
				Paths: []string{
					fmt.Sprintf("taskspec.inputs.params.%s.type", p.Name),
					fmt.Sprintf("taskspec.inputs.params.%s.default.type", p.Name),
				},
			}
		}
	}
	return nil
}

func validateInputParameterVariables(steps []Step, inputs *Inputs) *apis.FieldError {
	parameterNames := map[string]struct{}{}
	arrayParameterNames := map[string]struct{}{}

	if inputs != nil {
		for _, p := range inputs.Params {
			parameterNames[p.Name] = struct{}{}
			if p.Type == ParamTypeArray {
				arrayParameterNames[p.Name] = struct{}{}
			}
		}
	}

	if err := validateVariables(steps, "params", parameterNames); err != nil {
		return err
	}
	return validateArrayUsage(steps, "params", arrayParameterNames)
}

func validateResourceVariables(steps []Step, inputs *Inputs, outputs *Outputs) *apis.FieldError {
	resourceNames := map[string]struct{}{}
	if inputs != nil {
		for _, r := range inputs.Resources {
			resourceNames[r.Name] = struct{}{}
		}
	}
	if outputs != nil {
		for _, r := range outputs.Resources {
			resourceNames[r.Name] = struct{}{}
		}
	}
	return validateVariables(steps, "resources", resourceNames)
}

func validateArrayUsage(steps []Step, prefix string, vars map[string]struct{}) *apis.FieldError {
	for _, step := range steps {
		if err := validateTaskNoArrayReferenced("name", step.Name, prefix, vars); err != nil {
			return err
		}
		if err := validateTaskNoArrayReferenced("image", step.Image, prefix, vars); err != nil {
			return err
		}
		if err := validateTaskNoArrayReferenced("workingDir", step.WorkingDir, prefix, vars); err != nil {
			return err
		}
		for i, cmd := range step.Command {
			if err := validateTaskArraysIsolated(fmt.Sprintf("command[%d]", i), cmd, prefix, vars); err != nil {
				return err
			}
		}
		for i, arg := range step.Args {
			if err := validateTaskArraysIsolated(fmt.Sprintf("arg[%d]", i), arg, prefix, vars); err != nil {
				return err
			}
		}
		for _, env := range step.Env {
			if err := validateTaskNoArrayReferenced(fmt.Sprintf("env[%s]", env.Name), env.Value, prefix, vars); err != nil {
				return err
			}
		}
		for i, v := range step.VolumeMounts {
			if err := validateTaskNoArrayReferenced(fmt.Sprintf("volumeMount[%d].Name", i), v.Name, prefix, vars); err != nil {
				return err
			}
			if err := validateTaskNoArrayReferenced(fmt.Sprintf("volumeMount[%d].MountPath", i), v.MountPath, prefix, vars); err != nil {
				return err
			}
			if err := validateTaskNoArrayReferenced(fmt.Sprintf("volumeMount[%d].SubPath", i), v.SubPath, prefix, vars); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateVariables(steps []Step, prefix string, vars map[string]struct{}) *apis.FieldError {
	for _, step := range steps {
		if err := validateTaskVariable("name", step.Name, prefix, vars); err != nil {
			return err
		}
		if err := validateTaskVariable("image", step.Image, prefix, vars); err != nil {
			return err
		}
		if err := validateTaskVariable("workingDir", step.WorkingDir, prefix, vars); err != nil {
			return err
		}
		for i, cmd := range step.Command {
			if err := validateTaskVariable(fmt.Sprintf("command[%d]", i), cmd, prefix, vars); err != nil {
				return err
			}
		}
		for i, arg := range step.Args {
			if err := validateTaskVariable(fmt.Sprintf("arg[%d]", i), arg, prefix, vars); err != nil {
				return err
			}
		}
		for _, env := range step.Env {
			if err := validateTaskVariable(fmt.Sprintf("env[%s]", env.Name), env.Value, prefix, vars); err != nil {
				return err
			}
		}
		for i, v := range step.VolumeMounts {
			if err := validateTaskVariable(fmt.Sprintf("volumeMount[%d].Name", i), v.Name, prefix, vars); err != nil {
				return err
			}
			if err := validateTaskVariable(fmt.Sprintf("volumeMount[%d].MountPath", i), v.MountPath, prefix, vars); err != nil {
				return err
			}
			if err := validateTaskVariable(fmt.Sprintf("volumeMount[%d].SubPath", i), v.SubPath, prefix, vars); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateTaskVariable(name, value, prefix string, vars map[string]struct{}) *apis.FieldError {
	return substitution.ValidateVariable(name, value, prefix, "(?:inputs|outputs).", "step", "taskspec.steps", vars)
}

func validateTaskNoArrayReferenced(name, value, prefix string, arrayNames map[string]struct{}) *apis.FieldError {
	return substitution.ValidateVariableProhibited(name, value, prefix, "(?:inputs|outputs).", "step", "taskspec.steps", arrayNames)
}

func validateTaskArraysIsolated(name, value, prefix string, arrayNames map[string]struct{}) *apis.FieldError {
	return substitution.ValidateVariableIsolated(name, value, prefix, "(?:inputs|outputs).", "step", "taskspec.steps", arrayNames)
}

func checkForDuplicates(resources []TaskResource, path string) *apis.FieldError {
	encountered := map[string]struct{}{}
	for _, r := range resources {
		if _, ok := encountered[strings.ToLower(r.Name)]; ok {
			return apis.ErrMultipleOneOf(path)
		}
		encountered[strings.ToLower(r.Name)] = struct{}{}
	}
	return nil
}

func validateResourceType(r TaskResource, path string) *apis.FieldError {
	for _, allowed := range AllResourceTypes {
		if r.Type == allowed {
			return nil
		}
	}
	return apis.ErrInvalidValue(string(r.Type), path)
}
