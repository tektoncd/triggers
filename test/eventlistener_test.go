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
	"testing"

	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	knativetest "knative.dev/pkg/test"
)

func TestEventListenerCreate(t *testing.T) {
	c, namespace := setup(t)
	t.Parallel()

	defer tearDown(t, c, namespace)
	knativetest.CleanupOnInterrupt(func() { tearDown(t, c, namespace) }, t.Logf)

	t.Log("Start EventListener e2e test")

	// Create EventListener
	el, err := c.TriggersClient.TektonV1alpha1().EventListeners(namespace).Create(
		&v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-eventlistener",
			},
			Spec: v1alpha1.EventListenerSpec{
				ServiceAccountName: "some-service-account",
				Triggers: []v1alpha1.Trigger{
					v1alpha1.Trigger{
						TriggerBinding: v1alpha1.TriggerBindingRef{
							Name: "some-trigger-binding",
						},
						TriggerTemplate: v1alpha1.TriggerTemplateRef{
							Name: "some-trigger-template",
						},
					},
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("Failed to create EventListener: %s", err)
	}
	t.Logf("Created EventListener %s in namespace %s", el.Name, el.Namespace)

	// Verify the EventListener's Deployment is created
	if err = WaitForDeploymentToExist(c, namespace, el.Name); err != nil {
		t.Fatalf("Failed to create EventListener Deployment: %s", err)
	}
	t.Log("Found EventListener's Deployment")

	// Verify the EventListener's Service is created
	if err = WaitForServiceToExist(c, namespace, el.Name); err != nil {
		t.Fatalf("Failed to create EventListener Service: %s", err)
	}
	t.Log("Found EventListener's Service")

	// Delete EventListener
	err = c.TriggersClient.TektonV1alpha1().EventListeners(namespace).Delete(el.Name, &metav1.DeleteOptions{})
	if err != nil {
		t.Fatalf("Failed to delete EventListener: %s", err)
	}
	t.Log("Deleted EventListener")

	// Verify the EventListener's Deployment is deleted
	if err = WaitForDeploymentToNotExist(c, namespace, el.Name); err != nil {
		t.Fatalf("Failed to delete EventListener Deployment: %s", err)
	}
	t.Log("EventListener's Deployment was deleted")

	// Verify the EventListener's Service is deleted
	if err = WaitForServiceToNotExist(c, namespace, el.Name); err != nil {
		t.Fatalf("Failed to delete EventListener Service: %s", err)
	}
	t.Log("EventListener's Service was deleted")
}
