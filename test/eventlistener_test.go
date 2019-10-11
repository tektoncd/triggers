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

	"strconv"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	eventReconciler "github.com/tektoncd/triggers/pkg/reconciler/v1alpha1/eventlistener"
	bldr "github.com/tektoncd/triggers/test/builder"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	knativetest "knative.dev/pkg/test"
)

const resourceLabel = triggersv1.GroupName + triggersv1.EventListenerLabelKey
const eventIdLabel = triggersv1.GroupName + triggersv1.EventIDLabelKey

func TestEventListenerCreate(t *testing.T) {
	c, namespace := setup(t)
	t.Parallel()

	defer tearDown(t, c, namespace)
	knativetest.CleanupOnInterrupt(func() { tearDown(t, c, namespace) }, t.Logf)

	t.Log("Start EventListener e2e test")

	// TemplatedPipelineResources
	pr1 := v1alpha1.PipelineResource{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PipelineResource",
			APIVersion: "tekton.dev/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pr1",
			Namespace: namespace,
			Labels: map[string]string{
				"$(params.oneparam)": "$(params.oneparam)",
			},
		},
		Spec: v1alpha1.PipelineResourceSpec{
			Type: "git",
		},
	}
	pr1Bytes, err := json.Marshal(pr1)
	if err != nil {
		t.Fatalf("Error marshalling PipelineResource 1: %s", err)
	}

	// This is a templated resource, which does not have a namespace.
	// This is defaulted to the EventListener namespace.
	pr2 := v1alpha1.PipelineResource{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PipelineResource",
			APIVersion: "tekton.dev/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "pr2",
			Labels: map[string]string{
				"$(params.twoparamname)": "$(params.twoparamvalue)",
				"threeparam":             "$(params.threeparam)",
			},
		},
		Spec: v1alpha1.PipelineResourceSpec{
			Type: "git",
			Params: []v1alpha1.ResourceParam{
				{Name: "body", Value: "$(params.body)"},
				{Name: "header", Value: "$(params.header)"},
			},
		},
	}

	pr2Bytes, err := json.Marshal(pr2)
	if err != nil {
		t.Fatalf("Error marshalling ResourceTemplate PipelineResource 2: %s", err)
	}

	// TriggerTemplate
	tt, err := c.TriggersClient.TektonV1alpha1().TriggerTemplates(namespace).Create(
		bldr.TriggerTemplate("my-triggertemplate", "",
			bldr.TriggerTemplateSpec(
				bldr.TriggerTemplateParam("oneparam", "", ""),
				bldr.TriggerTemplateParam("twoparamname", "", ""),
				bldr.TriggerTemplateParam("twoparamvalue", "", "defaultvalue"),
				bldr.TriggerTemplateParam("threeparam", "", ""),
				bldr.TriggerTemplateParam("body", "", ""),
				bldr.TriggerTemplateParam("header", "", ""),
				bldr.TriggerResourceTemplate(pr1Bytes),
				bldr.TriggerResourceTemplate(pr2Bytes),
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
				bldr.TriggerBindingParam("oneparam", "$(body.one)"),
				bldr.TriggerBindingParam("twoparamname", "$(body.two.name)"),
				bldr.TriggerBindingParam("body", "$(body)"),
				bldr.TriggerBindingParam("header", "$(header)"),
			),
		),
	)
	if err != nil {
		t.Fatalf("Error creating TriggerBinding: %s", err)
	}

	// Event body & Expected ResourceTemplates after instantiation
	eventBodyJSON := []byte(`{"one": "zonevalue", "two": {"name": "zfoo", "value": "bar"}}`)
	wantPr1 := v1alpha1.PipelineResource{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PipelineResource",
			APIVersion: "tekton.dev/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pr1",
			Namespace: namespace,
			Labels: map[string]string{
				resourceLabel: "my-eventlistener",
				"zonevalue":   "zonevalue",
			},
		},
		Spec: v1alpha1.PipelineResourceSpec{
			Type: "git",
		},
	}

	wantPr2 := v1alpha1.PipelineResource{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PipelineResource",
			APIVersion: "tekton.dev/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pr2",
			Namespace: namespace,
			Labels: map[string]string{
				resourceLabel: "my-eventlistener",
				"zfoo":        "defaultvalue",
				"threeparam":  "threevalue",
			},
		},
		Spec: v1alpha1.PipelineResourceSpec{
			Type: "git",
			Params: []v1alpha1.ResourceParam{
				{Name: "body", Value: `{"one": "zonevalue", "two": {"name": "zfoo", "value": "bar"}}`},
				{Name: "header", Value: `{"Accept-Encoding":["gzip"],"Content-Length":["61"],"Content-Type":["application/json"],"User-Agent":["Go-http-client/1.1"]}`},
			},
		},
	}

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
			Rules: []rbacv1.PolicyRule{{
				APIGroups: []string{"tekton.dev"},
				Resources: []string{"eventlisteners", "triggerbindings", "triggertemplates", "pipelineresources"},
				Verbs:     []string{"create", "get", "watch", "list"},
			}, {
				APIGroups: []string{""},
				Resources: []string{"events"},
				Verbs:     []string{"create"},
			}}},
	)
	if err != nil {
		t.Fatalf("Error creating Role: %s", err)
	}
	_, err = c.KubeClient.RbacV1().RoleBindings(namespace).Create(
		&rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: "my-rolebinding"},
			Subjects: []rbacv1.Subject{{
				Kind:      "ServiceAccount",
				Name:      sa.Name,
				Namespace: namespace,
			}},
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
		bldr.EventListener("my-eventlistener", namespace,
			bldr.EventListenerMeta(
				bldr.Label("triggers", "eventlistener"),
			),
			bldr.EventListenerSpec(
				bldr.EventListenerServiceAccount(sa.Name),
				bldr.EventListenerTrigger(tb.Name, tt.Name, "",
					bldr.EventListenerTriggerParam("threeparam", "threevalue")),
			),
		),
	)
	if err != nil {
		t.Fatalf("Failed to create EventListener: %s", err)
	}

	// Verify the EventListener to be ready
	if err := WaitFor(eventListenerReady(t, c, namespace, el.Name)); err != nil {
		t.Fatalf("EventListener not ready: %s", err)
	}
	t.Log("EventListener is ready")

	labelSelector := fields.SelectorFromSet(eventReconciler.GenerateResourceLabels(el.Name)).String()
	// Grab EventListener sink pods
	sinkPods, err := c.KubeClient.CoreV1().Pods(namespace).List(metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		t.Fatalf("Error listing EventListener sink pods: %s", err)
	}

	// Port forward sink pod for http request
	portString := strconv.Itoa(eventReconciler.Port)
	cmd := exec.Command("kubectl", "port-forward", sinkPods.Items[0].Name, "-n", namespace, fmt.Sprintf("%s:%s", portString, portString))
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
	req, err := http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:%s", portString), bytes.NewBuffer(eventBodyJSON))
	if err != nil {
		t.Fatalf("Error creating POST request: %s", err)
	}
	req.Header.Add("Content-Type", "application/json")
	_, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Error sending POST request: %s", err)
	}

	for _, wantPr := range []v1alpha1.PipelineResource{wantPr1, wantPr2} {
		if err = WaitFor(pipelineResourceExist(t, c, namespace, wantPr.Name)); err != nil {
			t.Fatalf("Failed to create ResourceTemplate %s: %s", wantPr.Name, err)
		}
		gotPr, err := c.PipelineClient.TektonV1alpha1().PipelineResources(namespace).Get(wantPr.Name, metav1.GetOptions{})
		if err != nil {
			t.Fatalf("Error getting ResourceTemplate: %s: %s", wantPr.Name, err)
		}
		if gotPr.Labels[eventIdLabel] == "" {
			t.Fatalf("Instantiated ResourceTemplate missing EventId")
		} else {
			delete(gotPr.Labels, eventIdLabel)
		}
		if diff := cmp.Diff(wantPr.Labels, gotPr.Labels); diff != "" {
			t.Fatalf("Diff instantiated ResourceTemplate labels %s: -want +got: %s", wantPr.Name, diff)
		}
		if diff := cmp.Diff(wantPr.Spec, gotPr.Spec); diff != "" {
			t.Fatalf("Diff instantiated ResourceTemplate spec %s: -want +got: %s", wantPr.Name, diff)
		}
	}

	// Delete TriggerBinding
	err = c.TriggersClient.TektonV1alpha1().TriggerBindings(namespace).Delete(tb.Name, &metav1.DeleteOptions{})
	if err != nil {
		t.Fatalf("Failed to delete TriggerBinding: %s", err)
	}
	t.Log("Deleted TriggerBinding")
	resourceLabels := make(map[string]string, 2)
	resourceLabels["involvedObject.kind"] = "EventListener"
	resourceLabels["involvedObject.name"] = el.Name
	fieldSelector := fields.SelectorFromSet(resourceLabels).String()
	_, err = c.KubeClient.CoreV1().Events(namespace).List(metav1.ListOptions{FieldSelector: fieldSelector})
	if err != nil {
		t.Fatalf("Failed to get the event trigger by TriggerBinding delete: %s", err)
	}

	// Delete EventListener
	err = c.TriggersClient.TektonV1alpha1().EventListeners(namespace).Delete(el.Name, &metav1.DeleteOptions{})
	if err != nil {
		t.Fatalf("Failed to delete EventListener: %s", err)
	}
	t.Log("Deleted EventListener")

	// Verify the EventListener's Deployment is deleted
	if err = WaitFor(deploymentNotExist(t, c, namespace, fmt.Sprintf("%s-%s", eventReconciler.GeneratedResourcePrefix, el.Name))); err != nil {
		t.Fatalf("Failed to delete EventListener Deployment: %s", err)
	}
	t.Log("EventListener's Deployment was deleted")

	// Verify the EventListener's Service is deleted
	if err = WaitFor(serviceNotExist(t, c, namespace, fmt.Sprintf("%s-%s", eventReconciler.GeneratedResourcePrefix, el.Name))); err != nil {
		t.Fatalf("Failed to delete EventListener Service: %s", err)
	}
	t.Log("EventListener's Service was deleted")
}
