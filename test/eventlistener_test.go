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
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	reconciler "github.com/tektoncd/triggers/pkg/reconciler/v1alpha1/eventlistener"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	knativetest "knative.dev/pkg/test"
	"testing"
)

func TestEventListener(t *testing.T) {
	c, namespace := setup(t)
	t.Parallel()

	defer tearDown(t, c, namespace)
	knativetest.CleanupOnInterrupt(func() { tearDown(t, c, namespace) }, t.Logf)
	t.Log("Start EventListener e2e test")
	// Create SA
	sa, err := c.KubeClient.CoreV1().ServiceAccounts(namespace).Create(
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name: "sa",
			},
		},
	)
	if err != nil {
		t.Fatalf("Failed to create ServiceAccount: %s", err)
	}
	t.Logf("Created ServiceAccount %s in namespace %s", sa.Name, sa.Namespace)

	// Create EventListener
	el, err := c.TriggersClient.TektonV1alpha1().EventListeners(namespace).Create(
		&v1alpha1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-eventlistener",
			},
			Spec: v1alpha1.EventListenerSpec{
				ServiceAccountName: sa.Name,
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

	// Verify the EventListener's Service is created
	if err = WaitForService(c, namespace, el.Name, true); err != nil {
		t.Fatalf("Failed to create EventListener Service: %s", err)
	}
	t.Log("Found EventListener's Service")
	// Verify the EventListener's Deployment is created
	if err = WaitForDeployment(c, namespace, el.Name, true); err != nil {
		t.Fatalf("Failed to create EventListener Deployment: %s", err)
	}
	t.Log("Found EventListener's Deployment")
	// Verify the EventListener's Role is created
	if err = WaitForRole(c, namespace, fmt.Sprintf("%s%s", el.Name, reconciler.RolePostfix), true); err != nil {
		t.Fatalf("Failed to create EventListener Role: %s", err)
	}
	t.Log("Found EventListener's Role")
	// Verify the EventListener's RoleBinding is created
	if err = WaitForRoleBinding(c, namespace, fmt.Sprintf("%s%s", el.Name, reconciler.RoleBindingPostfix), true); err != nil {
		t.Fatalf("Failed to create EventListener RoleBinding: %s", err)
	}
	t.Log("Found EventListener's RoleBinding")

	// Delete EventListener
	err = c.TriggersClient.TektonV1alpha1().EventListeners(namespace).Delete(el.Name, &metav1.DeleteOptions{})
	if err != nil {
		t.Fatalf("Failed to delete EventListener: %s", err)
	}
	t.Log("Deleted EventListener")

	// Verify the EventListener's Service is deleted
	if err = WaitForService(c, namespace, el.Name, false); err != nil {
		t.Fatalf("Failed to delete EventListener Service: %s", err)
	}
	t.Log("Deleted EventListener's Service")
	// Verify the EventListener's Deployment is deleted
	if err = WaitForDeployment(c, namespace, el.Name, false); err != nil {
		t.Fatalf("Failed to delete EventListener Deployment: %s", err)
	}
	t.Log("Deleted EventListener's Deployment")
	// Verify the EventListener's Role is deleted
	if err = WaitForRole(c, namespace, fmt.Sprintf("%s%s", el.Name, reconciler.RolePostfix), false); err != nil {
		t.Fatalf("Failed to delete EventListener Role: %s", err)
	}
	t.Log("Deleted EventListener's Role")
	// Verify the EventListener's RoleBinding is deleted
	if err = WaitForRoleBinding(c, namespace, fmt.Sprintf("%s%s", el.Name, reconciler.RoleBindingPostfix), false); err != nil {
		t.Fatalf("Failed to delete EventListener RoleBinding: %s", err)
	}
	t.Log("Deleted EventListener's RoleBinding")
}
