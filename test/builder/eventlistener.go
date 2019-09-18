package builder

import (
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EventListenerOp is an operation which modifies the EventListener.
type EventListenerOp func(*v1alpha1.EventListener)

// EventListenerSpecOp is an operation which modifies the EventListenerSpec.
type EventListenerSpecOp func(*v1alpha1.EventListenerSpec)

// EventListenerTriggerOp is an operation which modifies the Trigger.
type EventListenerTriggerOp func(*v1alpha1.EventListenerTrigger)

// EventListener creates an EventListener with default values.
// Any number of EventListenerOp modifiers can be passed to transform it.
func EventListener(name, namespace string, ops ...EventListenerOp) *v1alpha1.EventListener {
	e := &v1alpha1.EventListener{
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

// EventListenerMeta sets the Meta structs of the EventListener.
// Any number of MetaOp modifiers can be passed.
func EventListenerMeta(ops ...MetaOp) EventListenerOp {
	return func(e *v1alpha1.EventListener) {
		for _, op := range ops {
			switch o := op.(type) {
			case ObjectMetaOp:
				o(&e.ObjectMeta)
			case TypeMetaOp:
				o(&e.TypeMeta)
			}
		}
	}
}

// EventListenerSpec sets the specified spec of the EventListener.
// Any number of EventListenerSpecOp modifiers can be passed to create/modify it.
func EventListenerSpec(ops ...EventListenerSpecOp) EventListenerOp {
	return func(e *v1alpha1.EventListener) {
		for _, op := range ops {
			op(&e.Spec)
		}
	}
}

// EventListenerServiceAccount sets the specified ServiceAccount of the EventListener.
func EventListenerServiceAccount(saName string) EventListenerSpecOp {
	return func(spec *v1alpha1.EventListenerSpec) {
		spec.ServiceAccountName = saName
	}
}

// EventListenerTrigger adds an EventListenerTrigger to the EventListenerSpec Triggers.
// Any number of EventListenerTriggerOp modifiers can be passed to create/modify it.
func EventListenerTrigger(tbName, ttName, apiVersion string, ops ...EventListenerTriggerOp) EventListenerSpecOp {
	return func(spec *v1alpha1.EventListenerSpec) {
		spec.Triggers = append(spec.Triggers, Trigger(tbName, ttName, apiVersion, ops...))
	}
}

// EventListenerTriggerParam adds a param to the EventListenerTrigger
func EventListenerTriggerParam(name, value string) EventListenerTriggerOp {
	return func(trigger *v1alpha1.EventListenerTrigger) {
		trigger.Binding.Params = append(trigger.Binding.Params,
			pipelinev1.Param{
				Name: name,
				Value: pipelinev1.ArrayOrString{
					StringVal: value,
					Type:      pipelinev1.ParamTypeString,
				},
			})
	}
}

// Trigger creates an EventListenerTrigger. Any number of EventListenerTriggerOp
// modifiers can be passed to create/modify it.
func Trigger(tbName, ttName, apiVersion string, ops ...EventListenerTriggerOp) v1alpha1.EventListenerTrigger {
	t := v1alpha1.EventListenerTrigger{
		Binding: v1alpha1.EventListenerBinding{
			Name:       tbName,
			APIVersion: apiVersion,
		},
		Template: v1alpha1.EventListenerTemplate{
			Name:       ttName,
			APIVersion: apiVersion,
		},
	}

	for _, op := range ops {
		op(&t)
	}

	return t
}
