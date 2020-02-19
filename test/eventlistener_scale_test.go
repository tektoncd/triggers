// +build e2e

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

package test

import (
	"fmt"
	"testing"

	bldr "github.com/tektoncd/triggers/test/builder"
	knativetest "knative.dev/pkg/test"
)

/*
 * Test creating an EventListener with a large number of Triggers.
 * This is a regression test for issue #356.
 */
func TestEventListenerScale(t *testing.T) {
	c, namespace := setup(t)
	t.Parallel()

	defer tearDown(t, c, namespace)
	knativetest.CleanupOnInterrupt(func() { tearDown(t, c, namespace) }, t.Logf)

	t.Log("Start EventListener Scale e2e test")

	// Create an EventListener with 1000 Triggers
	var err error
	el := bldr.EventListener("my-eventlistener", namespace)
	for i := 0; i < 1000; i++ {
		trigger := bldr.Trigger("my-triggertemplate", "v1alpha1",
			bldr.EventListenerTriggerBinding("my-triggerbinding", "", "v1alpha1"),
		)
		trigger.Name = fmt.Sprintf("%d", i)
		el.Spec.Triggers = append(el.Spec.Triggers, trigger)
	}
	el, err = c.TriggersClient.TektonV1alpha1().EventListeners(namespace).Create(el)
	if err != nil {
		t.Fatalf("Error creating EventListener: %s", err)
	}

	// Verify that the EventListener was created properly
	if err := WaitFor(eventListenerReady(t, c, namespace, el.Name)); err != nil {
		t.Fatalf("EventListener is not ready: %s", err)
	}
	t.Log("EventListener is ready")
}
