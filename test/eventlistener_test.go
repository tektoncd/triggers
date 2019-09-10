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
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	bldr "github.com/tektoncd/triggers/test/builder"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	knativetest "knative.dev/pkg/test"
)

func TestEventListenerCreate(t *testing.T) {
	c, namespace := setup(t)
	t.Parallel()

	defer tearDown(t, c, namespace)
	knativetest.CleanupOnInterrupt(func() { tearDown(t, c, namespace) }, t.Logf)

	t.Log("Start EventListener e2e test")

	// ResourceTemplates
	rtTriggerTemplate1 := bldr.TriggerTemplate("rt-triggertemplate1", namespace,
		bldr.TriggerTemplateMeta(
			bldr.Label("$(params.oneparam)", "$(params.oneparam)"),
			bldr.TypeMeta("TriggerTemplate", "tekton.dev/v1alpha1"),
		),
	)
	rtBytes1, err := json.Marshal(rtTriggerTemplate1)
	if err != nil {
		t.Fatalf("Error marshalling ResourceTemplate TriggerTemplate 1: %s", err)
	}
	// This is a templated resource, which does not have a namespace.
	// This is defaulted to the EventListener namespace.
	rtTriggerTemplate2 := bldr.TriggerTemplate("rt-triggertemplate2", "",
		bldr.TriggerTemplateMeta(
			bldr.Label("$(params.twoparamname)", "$(params.twoparamvalue)"),
			bldr.TypeMeta("TriggerTemplate", "tekton.dev/v1alpha1"),
		),
	)
	rtBytes2, err := json.Marshal(rtTriggerTemplate2)
	if err != nil {
		t.Fatalf("Error marshalling ResourceTemplate TriggerTemplate 2: %s", err)
	}

	// TriggerTemplate
	tt, err := c.TriggersClient.TektonV1alpha1().TriggerTemplates(namespace).Create(
		bldr.TriggerTemplate("my-triggertemplate", "",
			bldr.TriggerTemplateSpec(
				bldr.TriggerTemplateParam("oneparam", "", ""),
				bldr.TriggerTemplateParam("twoparamname", "", ""),
				bldr.TriggerTemplateParam("twoparamvalue", "", "defaultvalue"),
				bldr.TriggerResourceTemplate(rtBytes1),
				bldr.TriggerResourceTemplate(rtBytes2),
			),
		),
	)
	if err != nil {
		t.Fatalf("Error creating TriggerTemplate: %s", err)
	}

	// TriggerBinding
	tb, err := c.TriggersClient.TektonV1alpha1().TriggerBindings(namespace).Create(
		bldr.TriggerBinding("my-triggerbinding", "",
			bldr.TriggerBindingSpec(
				bldr.TriggerBindingParam("oneparam", "$(event.one)"),
				bldr.TriggerBindingParam("twoparamname", "$(event.two.name)"),
			),
		),
	)
	if err != nil {
		t.Fatalf("Error creating TriggerBinding: %s", err)
	}

	// Event body & Expected ResourceTemplates after instantiation
	eventBodyJSON := []byte(`{"one": "onevalue", "two": {"name": "foo", "value": "bar"}}`)
	wantRtTriggerTemplate1 := bldr.TriggerTemplate("rt-triggertemplate1", namespace,
		bldr.TriggerTemplateMeta(
			bldr.Label("onevalue", "onevalue"),
		),
	)
	wantRtTriggerTemplate2 := bldr.TriggerTemplate("rt-triggertemplate2", namespace,
		bldr.TriggerTemplateMeta(
			bldr.Label("foo", "defaultvalue"),
		),
	)

	// ServiceAccount + Role + RoleBinding to authorize the creation of our
	// templated resources
	sa, err := c.KubeClient.CoreV1().ServiceAccounts(namespace).Create(
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{Name: "my-serviceaccount"},
		},
	)
	if err != nil {
		t.Fatalf("Error creating ServiceAccount: %s", err)
	}
	_, err = c.KubeClient.RbacV1().Roles(namespace).Create(
		&rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{Name: "my-role"},
			Rules: []rbacv1.PolicyRule{
				rbacv1.PolicyRule{
					APIGroups: []string{"tekton.dev"},
					Resources: []string{"eventlisteners", "triggerbindings", "triggertemplates"},
					Verbs:     []string{"create", "get"},
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("Error creating Role: %s", err)
	}
	_, err = c.KubeClient.RbacV1().RoleBindings(namespace).Create(
		&rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: "my-rolebinding"},
			Subjects: []rbacv1.Subject{
				rbacv1.Subject{
					Kind:      "ServiceAccount",
					Name:      sa.Name,
					Namespace: namespace,
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Role",
				Name:     "my-role",
			},
		},
	)
	if err != nil {
		t.Fatalf("Error creating RoleBinding: %s", err)
	}

	// EventListener
	el, err := c.TriggersClient.TektonV1alpha1().EventListeners(namespace).Create(
		bldr.EventListener("my-eventlistener", "",
			bldr.EventListenerMeta(
				bldr.Label("triggers", "eventlistener"),
			),
			bldr.EventListenerSpec(
				bldr.EventListenerServiceAccount(sa.Name),
				bldr.EventListenerTrigger(tb.Name, tt.Name, ""),
			),
		),
	)
	if err != nil {
		t.Fatalf("Failed to create EventListener: %s", err)
	}

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

	// Wait for EventListener sink to be running
	sinkPods, err := c.KubeClient.CoreV1().Pods(namespace).List(metav1.ListOptions{LabelSelector: fmt.Sprintf("app=%s", el.Name)})
	if err != nil {
		t.Fatalf("Error listing EventListener sink pods: %s", err)
	}
	for _, pod := range sinkPods.Items {
		if err = WaitForPodRunning(c, namespace, pod.Name); err != nil {
			t.Fatalf("Error EventListener sink pod failed to enter the running phase: %s", err)
		}
	}
	t.Log("EventListener sink pod is running")

	// Port forward sink pod for http request
	cmd := exec.Command("kubectl", "port-forward", sinkPods.Items[0].Name, "-n", namespace, "8082:8082")
	err = cmd.Start()
	if err != nil {
		t.Fatalf("Error starting port-forward command: %s", err)
	}
	if cmd.Process == nil {
		t.Fatalf("Error starting command. Process is nil")
	}
	defer func() {
		if err = cmd.Process.Kill(); err != nil {
			t.Fatalf("Error killing port-forward process: %s", err)
		}
	}()
	// Wait for port forward to take effect
	time.Sleep(5 * time.Second)

	// Send POST request to EventListener sink
	req, err := http.NewRequest("POST", "http://127.0.0.1:8082", bytes.NewBuffer(eventBodyJSON))
	if err != nil {
		t.Fatalf("Error creating POST request: %s", err)
	}
	req.Header.Add("Content-Type", "application/json")
	_, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Error sending POST request: %s", err)
	}

	// Check ResourceTemplate TriggerTemplate 1
	if err = WaitForTriggerTemplateToExist(c, namespace, wantRtTriggerTemplate1.Name); err != nil {
		t.Fatalf("Failed to create ResourceTemplate TriggerTemplate 1: %s", err)
	}
	gotRtTriggerTemplate1, err := c.TriggersClient.TektonV1alpha1().TriggerTemplates(namespace).Get(wantRtTriggerTemplate1.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Error getting ResourceTemplate TriggerTemplate 1: %s: %s", wantRtTriggerTemplate1.Name, err)
	}
	if diff := cmp.Diff(wantRtTriggerTemplate1.Labels, gotRtTriggerTemplate1.Labels); diff != "" {
		t.Fatalf("Diff instantiated ResourceTemplate TriggerTemplate 1: -want +got: %s", diff)
	}
	// Check ResourceTemplate TriggerTemplate 2
	if err = WaitForTriggerTemplateToExist(c, namespace, wantRtTriggerTemplate2.Name); err != nil {
		t.Fatalf("Failed to create ResourceTemplate TriggerTemplate 2: %s", err)
	}
	gotRtTriggerTemplate2, err := c.TriggersClient.TektonV1alpha1().TriggerTemplates(namespace).Get(wantRtTriggerTemplate2.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Error getting ResourceTemplate TriggerTemplate 2: %s: %s", wantRtTriggerTemplate2.Name, err)
	}
	if diff := cmp.Diff(wantRtTriggerTemplate2.Labels, gotRtTriggerTemplate2.Labels); diff != "" {
		t.Fatalf("Diff instantiated ResourceTemplate TriggerTemplate 2: -want +got: %s", diff)
	}

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
