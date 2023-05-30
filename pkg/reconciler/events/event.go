/*
Copyright 2021 The Tekton Authors

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
package events

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
)

const (
	// TriggerProcessingStartedV1 is sent for Sink Triggers when a trigger is started
	TriggerProcessingStartedV1 = "dev.tekton.event.triggers.started.v1"
	// TriggerProcessingSuccessfulV1 is sent for Sink Triggers when a trigger is successful
	TriggerProcessingSuccessfulV1 = "dev.tekton.event.triggers.successful.v1"
	// TriggerProcessingFailedEventV1 is sent for Sink Triggers when we fail to process trigger
	TriggerProcessingFailedV1 = "dev.tekton.event.triggers.failed.v1"
	// TriggerProcessingDoneV1 is sent for Sink Triggers when we are done
	// with eventlistener handler
	TriggerProcessingDoneV1 = "dev.tekton.event.triggers.done.v1"
	// EventAccepted is sent as response for CloudEvent compliant providers
	EventAccepted = "dev.tekton.event.triggers.accepted.v1"
)

// Emit emits events for object
// Supported events are k8s events.
func Emit(recorder record.EventRecorder, eventType string, object runtime.Object, err error) {
	sendKubernetesEvents(recorder, eventType, object, err)
}

func sendKubernetesEvents(c record.EventRecorder, eventType string, object runtime.Object, err error) {
	switch err {
	case nil:
		if eventType == TriggerProcessingFailedV1 {
			c.Event(object, corev1.EventTypeWarning, eventType, "")
		} else {
			c.Event(object, corev1.EventTypeNormal, eventType, "")
		}
	default:
		c.Event(object, corev1.EventTypeWarning, eventType, err.Error())
	}
}
