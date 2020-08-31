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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"knative.dev/pkg/apis"
)

// Validate EventListener.
func (e *EventListener) Validate(ctx context.Context) *apis.FieldError {
	return e.Spec.validate(ctx)
}

func (s *EventListenerSpec) validate(ctx context.Context) *apis.FieldError {
	if s.Replicas != nil {
		if *s.Replicas < 0 {
			return apis.ErrInvalidValue(*s.Replicas, "spec.replicas")
		}
	}
	if len(s.Triggers) == 0 {
		return apis.ErrMissingField("spec.triggers")
	}
	for i, trigger := range s.Triggers {
		if err := trigger.validate(ctx).ViaField(fmt.Sprintf("spec.triggers[%d]", i)); err != nil {
			return err
		}
	}
	if s.Resources.KubernetesResource != nil {
		return validateKubernetesObject(s.Resources.KubernetesResource)
	}
	return nil
}

func validateKubernetesObject(orig *KubernetesResource) *apis.FieldError {
	var errs *apis.FieldError
	if len(orig.Template.Spec.Containers) > 1 {
		errs = errs.Also(apis.ErrMultipleOneOf("containers").ViaField("spec.template.spec"))
	}
	errs = errs.Also(apis.CheckDisallowedFields(orig.Template.Spec,
		*podSpecMask(&orig.Template.Spec)).ViaField("spec.template.spec"))

	// bounded by condition because containers fields are optional so there is a chance that containers can be nil.
	if len(orig.Template.Spec.Containers) == 1 {
		errs = errs.Also(apis.CheckDisallowedFields(orig.Template.Spec.Containers[0],
			*containerFieldMask(&orig.Template.Spec.Containers[0])).ViaField("spec.template.spec.containers[0]"))
	}

	return errs
}

func containerFieldMask(in *corev1.Container) *corev1.Container {
	out := new(corev1.Container)

	// Disallowed fields
	// This list clarifies which all container attributes are not allowed.
	out.Name = ""
	out.Image = ""
	out.Args = nil
	out.Ports = nil
	out.LivenessProbe = nil
	out.ReadinessProbe = nil
	out.StartupProbe = nil
	out.Command = nil
	out.VolumeMounts = nil
	out.ImagePullPolicy = ""
	out.Lifecycle = nil
	out.SecurityContext = nil
	out.Stdin = false
	out.StdinOnce = false
	out.TerminationMessagePath = ""
	out.TerminationMessagePolicy = ""
	out.WorkingDir = ""
	out.TTY = false
	out.VolumeDevices = nil
	out.EnvFrom = nil
	out.Resources = corev1.ResourceRequirements{}
	out.Env = nil

	return out
}

// podSpecMask performs a _shallow_ copy of the Kubernetes PodSpec object to a new
// Kubernetes PodSpec object bringing over only the fields allowed in the Triggers EvenListener.
func podSpecMask(in *corev1.PodSpec) *corev1.PodSpec {
	out := new(corev1.PodSpec)

	// Allowed fields
	out.ServiceAccountName = in.ServiceAccountName
	out.Containers = in.Containers
	out.Tolerations = in.Tolerations
	out.NodeSelector = in.NodeSelector

	// Disallowed fields
	// This list clarifies which all podspec fields are not allowed.
	out.Volumes = nil
	out.ImagePullSecrets = nil
	out.EnableServiceLinks = nil
	out.ImagePullSecrets = nil
	out.InitContainers = nil
	out.RestartPolicy = ""
	out.TerminationGracePeriodSeconds = nil
	out.ActiveDeadlineSeconds = nil
	out.DNSPolicy = ""
	out.AutomountServiceAccountToken = nil
	out.NodeName = ""
	out.HostNetwork = false
	out.HostPID = false
	out.HostIPC = false
	out.ShareProcessNamespace = nil
	out.SecurityContext = nil
	out.Hostname = ""
	out.Subdomain = ""
	out.Affinity = nil
	out.SchedulerName = ""
	out.HostAliases = nil
	out.PriorityClassName = ""
	out.Priority = nil
	out.DNSConfig = nil
	out.ReadinessGates = nil
	out.RuntimeClassName = nil

	return out
}

func (t *EventListenerTrigger) validate(ctx context.Context) *apis.FieldError {
	if t.Template == nil && t.TriggerRef == "" {
		return apis.ErrMissingOneOf("template", "triggerRef")
	}

	// Validate optional Bindings
	if err := triggerSpecBindingArray(t.Bindings).validate(ctx); err != nil {
		return err
	}
	if t.Template != nil {
		// Validate required TriggerTemplate
		if err := t.Template.validate(ctx); err != nil {
			return err
		}
	}

	// Validate optional Interceptors
	for i, interceptor := range t.Interceptors {
		if err := interceptor.validate(ctx).ViaField(fmt.Sprintf("interceptors[%d]", i)); err != nil {
			return err
		}
	}

	// The trigger name is added as a label value for 'tekton.dev/trigger' so it must follow the k8s label guidelines:
	// https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set
	if errs := validation.IsValidLabelValue(t.Name); len(errs) > 0 {
		return apis.ErrInvalidValue(fmt.Sprintf("trigger name '%s' must be a valid label value", t.Name), "name")
	}

	return nil
}
