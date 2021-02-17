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
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"knative.dev/pkg/ptr"

	"k8s.io/client-go/kubernetes"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	eventReconciler "github.com/tektoncd/triggers/pkg/reconciler/v1alpha1/eventlistener"
	"github.com/tektoncd/triggers/pkg/sink"
	bldr "github.com/tektoncd/triggers/test/builder"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	knativetest "knative.dev/pkg/test"
)

const (
	resourceLabel = triggersv1.GroupName + triggersv1.EventListenerLabelKey
	triggerLabel  = triggersv1.GroupName + triggersv1.TriggerLabelKey
	eventIDLabel  = triggersv1.GroupName + triggersv1.EventIDLabelKey

	examplePRJsonFilename = "pr.json"
)

func loadExamplePREventBytes(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("testdata", examplePRJsonFilename)
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("Couldn't load test data example PullREquest event data: %v", err)
	}
	return bytes
}

func impersonateRBAC(t *testing.T, sa, namespace string, kubeClient kubernetes.Interface) {
	impersonateName := fmt.Sprintf("impersonate-%s-defaultSA", namespace)
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{Name: impersonateName, Namespace: namespace},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:           []string{"impersonate"},
				APIGroups:       []string{""},
				Resources:       []string{"serviceaccounts"},
				ResourceNames:   nil,
				NonResourceURLs: nil,
			},
		},
	}
	_, err := kubeClient.RbacV1().Roles(namespace).Get(context.Background(), impersonateName, metav1.GetOptions{})
	if err == nil || errors.IsNotFound(err) {
		_, err := kubeClient.RbacV1().Roles(namespace).Create(context.Background(), role, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("impersonate role creation failed namespace %q: %s", namespace, err)
		}
	} else {
		t.Fatalf("Pre-check for impersonate failed namespace %q: %s", namespace, err)
	}
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: impersonateName, Namespace: namespace},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      sa,
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     impersonateName,
		},
	}
	_, err = kubeClient.RbacV1().RoleBindings(namespace).Get(context.Background(), impersonateName, metav1.GetOptions{})
	if err == nil || errors.IsNotFound(err) {
		_, err := kubeClient.RbacV1().RoleBindings(namespace).Create(context.Background(), roleBinding, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("View rolebinding creation failed namespace %q: %s", namespace, err)
		}
	} else {
		t.Fatalf("Pre-check for view rolebinding failed namespace %q: %s", namespace, err)
	}
}

func TestEventListenerCreate(t *testing.T) {
	c, namespace := setup(t)
	t.Parallel()
	defer cleanup(t, c, namespace, "my-eventlistener")
	knativetest.CleanupOnInterrupt(func() { cleanup(t, c, namespace, "my-eventlistener") }, t.Logf)

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
				"$(tt.params.oneparam)": "$(tt.params.oneparam)",
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
				"$(tt.params.twoparamname)": "$(tt.params.twoparamvalue)",
			},
		},
		Spec: v1alpha1.PipelineResourceSpec{
			Type: "git",
			Params: []v1alpha1.ResourceParam{
				{Name: "license", Value: "$(tt.params.license)"},
				{Name: "header", Value: "$(tt.params.header)"},
				{Name: "prmessage", Value: "$(tt.params.prmessage)"},
			},
		},
	}
	pr2Bytes, err := json.Marshal(pr2)
	if err != nil {
		t.Fatalf("Error marshalling ResourceTemplate PipelineResource 2: %s", err)
	}

	// TriggerTemplate
	tt, err := c.TriggersClient.TriggersV1alpha1().TriggerTemplates(namespace).Create(context.Background(),
		bldr.TriggerTemplate("my-triggertemplate", "",
			bldr.TriggerTemplateMeta(
				bldr.Annotation("triggers.tekton.dev/old-escape-quotes", "true"),
			),
			bldr.TriggerTemplateSpec(
				bldr.TriggerTemplateParam("oneparam", "", ""),
				bldr.TriggerTemplateParam("twoparamname", "", ""),
				bldr.TriggerTemplateParam("twoparamvalue", "", "defaultvalue"),
				bldr.TriggerTemplateParam("license", "", ""),
				bldr.TriggerTemplateParam("header", "", ""),
				bldr.TriggerTemplateParam("prmessage", "", ""),
				bldr.TriggerResourceTemplate(runtime.RawExtension{Raw: pr1Bytes}),
				bldr.TriggerResourceTemplate(runtime.RawExtension{Raw: pr2Bytes}),
			),
		), metav1.CreateOptions{},
	)
	if err != nil {
		t.Fatalf("Error creating TriggerTemplate: %s", err)
	}

	// TriggerBinding
	tb, err := c.TriggersClient.TriggersV1alpha1().TriggerBindings(namespace).Create(context.Background(),
		bldr.TriggerBinding("my-triggerbinding", "",
			bldr.TriggerBindingSpec(
				bldr.TriggerBindingParam("oneparam", "$(body.action)"),
				bldr.TriggerBindingParam("twoparamname", "$(body.pull_request.state)"),
				bldr.TriggerBindingParam("prmessage", "$(body.pull_request.body)"),
			),
		), metav1.CreateOptions{},
	)
	if err != nil {
		t.Fatalf("Error creating TriggerBinding: %s", err)
	}

	// ClusterTriggerBinding
	ctb, err := c.TriggersClient.TriggersV1alpha1().ClusterTriggerBindings().Create(context.Background(),
		bldr.ClusterTriggerBinding("my-clustertriggerbinding",
			bldr.ClusterTriggerBindingSpec(
				bldr.TriggerBindingParam("license", "$(body.repository.license)"),
				bldr.TriggerBindingParam("header", "$(header)"),
			),
		), metav1.CreateOptions{},
	)
	if err != nil {
		t.Fatalf("Error creating ClusterTriggerBinding: %s", err)
	}

	// ServiceAccount + Role + RoleBinding to authorize the creation of our
	// templated resources
	sa, err := c.KubeClient.CoreV1().ServiceAccounts(namespace).Create(context.Background(),
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{Name: "my-serviceaccount"},
		}, metav1.CreateOptions{},
	)
	if err != nil {
		t.Fatalf("Error creating ServiceAccount: %s", err)
	}
	_, err = c.KubeClient.RbacV1().ClusterRoles().Create(context.Background(),
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{Name: "my-role"},
			Rules: []rbacv1.PolicyRule{{
				APIGroups: []string{triggersv1.GroupName},
				Resources: []string{"clustertriggerbindings", "eventlisteners", "clusterinterceptors", "triggerbindings", "triggertemplates", "triggers"},
				Verbs:     []string{"get", "list", "watch"},
			}, {
				APIGroups: []string{"tekton.dev"},
				Resources: []string{"pipelineresources"},
				Verbs:     []string{"create"},
			}, {
				APIGroups: []string{""},
				Resources: []string{"configmaps", "serviceaccounts", "secrets"},
				Verbs:     []string{"get", "list", "watch"},
			},
			},
		}, metav1.CreateOptions{},
	)
	if err != nil {
		t.Fatalf("Error creating Role: %s", err)
	}
	_, err = c.KubeClient.RbacV1().ClusterRoleBindings().Create(context.Background(),
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: "my-rolebinding"},
			Subjects: []rbacv1.Subject{{
				Kind:      "ServiceAccount",
				Name:      sa.Name,
				Namespace: namespace,
			}},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "my-role",
			},
		}, metav1.CreateOptions{},
	)
	if err != nil {
		t.Fatalf("Error creating RoleBinding: %s", err)
	}
	impersonateRBAC(t, sa.Name, namespace, c.KubeClient)

	el, err := c.TriggersClient.TriggersV1alpha1().EventListeners(namespace).Create(context.Background(), &triggersv1.EventListener{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-eventlistener",
			Namespace: namespace,
		},
		Spec: triggersv1.EventListenerSpec{
			Triggers: []triggersv1.EventListenerTrigger{{
				Bindings: []*triggersv1.EventListenerBinding{{
					Ref:  tb.Name,
					Kind: triggersv1.NamespacedTriggerBindingKind,
				}, {
					Ref:  ctb.Name,
					Kind: triggersv1.ClusterTriggerBindingKind,
				}},
				Template: &triggersv1.EventListenerTemplate{
					Ref: ptr.String(tt.Name),
				},
				Interceptors: []*triggersv1.EventInterceptor{{
					CEL: &triggersv1.CELInterceptor{
						Filter: `body.action == "edited"`,
					},
				}},
			}},
			Replicas: ptr.Int32(3),
			Resources: triggersv1.Resources{
				KubernetesResource: &triggersv1.KubernetesResource{
					WithPodSpec: duckv1.WithPodSpec{
						Template: duckv1.PodSpecable{
							Spec: corev1.PodSpec{
								ServiceAccountName: sa.Name,
								NodeSelector:       map[string]string{"beta.kubernetes.io/os": "linux"},
								Tolerations: []corev1.Toleration{{
									Key:      "key",
									Operator: "Equal",
									Value:    "value",
									Effect:   "NoSchedule",
								}},
							},
						},
					},
				},
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create EventListener: %s", err)
	}
	// Verify the EventListener to be ready
	if err := WaitFor(eventListenerReady(t, c, namespace, el.Name)); err != nil {
		t.Fatalf("EventListener not ready: %s", err)
	}
	t.Log("EventListener is ready")

	// Load the example pull request event data
	eventBodyJSON := loadExamplePREventBytes(t)

	// Event body & Expected ResourceTemplates after instantiation
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
				triggerLabel:  el.Spec.Triggers[0].Name,
				"edited":      "edited",
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
				triggerLabel:  el.Spec.Triggers[0].Name,
				"open":        "defaultvalue",
			},
		},
		Spec: v1alpha1.PipelineResourceSpec{
			Type: "git",
			Params: []v1alpha1.ResourceParam{
				{Name: "license", Value: `{"key":"apache-2.0","name":"Apache License 2.0","spdx_id":"Apache-2.0","url":"https://api.github.com/licenses/apache-2.0","node_id":"MDc6TGljZW5zZTI="}`},
				{Name: "header", Value: `{"Accept-Encoding":"gzip","Content-Length":"2154","Content-Type":"application/json","User-Agent":"Go-http-client/1.1"}`},
				{Name: "prmessage", Value: "Git admission control\r\n\r\nNow with new lines!\r\n\r\n# :sunglasses: \r\n\r\naw yis"},
			},
		},
	}

	labelSelector := fields.SelectorFromSet(eventReconciler.GenerateResourceLabels(el.Name, eventReconciler.DefaultStaticResourceLabels)).String()
	// Grab EventListener sink pods
	sinkPods, err := c.KubeClient.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		t.Fatalf("Error listing EventListener sink pods: %s", err)
	}

	// ElPort forward sink pod for http request
	portString := strconv.Itoa(8000)
	podName := sinkPods.Items[0].Name
	stopChan, errChan := make(chan struct{}, 1), make(chan error, 1)

	defer func() {
		close(stopChan)
	}()
	go func(stopChan chan struct{}, errChan chan error) {
		config, err := clientcmd.BuildConfigFromFlags("", knativetest.Flags.Kubeconfig)
		if err != nil {
			errChan <- err
			return
		}
		roundTripper, upgrader, err := spdy.RoundTripperFor(config)
		if err != nil {
			errChan <- err
			return
		}

		path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", namespace, podName)
		hostIP := strings.TrimPrefix(config.Host, "https://")
		serverURL := url.URL{Scheme: "https", Path: path, Host: hostIP}
		dialer := spdy.NewDialer(upgrader, &http.Client{Transport: roundTripper}, http.MethodPost, &serverURL)
		out, errOut := new(Buffer), new(Buffer)
		readyChan := make(chan struct{}, 1)
		forwarder, err := portforward.New(dialer, []string{portString}, stopChan, readyChan, out, errOut)
		if err != nil {
			errChan <- err
			return
		}
		go func() {
			for range readyChan {
			}
			if len(errOut.String()) != 0 {
				errChan <- fmt.Errorf("%s", errOut)
			}
			close(errChan)
		}()
		if err = forwarder.ForwardPorts(); err != nil { // This locks until stopChan is closed.
			errChan <- err
			return
		}
	}(stopChan, errChan)

	if err := <-errChan; err != nil {
		t.Fatalf("Forwarding stream of data failed:: %v", err)
	}
	// Send POST request to EventListener sink
	req, err := http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:%s", portString), bytes.NewBuffer(eventBodyJSON))
	if err != nil {
		t.Fatalf("Error creating POST request: %s", err)
	}
	req.Header.Add("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Error sending POST request: %v", err)
	}

	if resp.StatusCode > http.StatusAccepted {
		t.Errorf("sink did not return 2xx response. Got status code: %d", resp.StatusCode)
	}
	wantBody := sink.Response{
		EventListener: "my-eventlistener",
		Namespace:     namespace,
	}
	var gotBody sink.Response
	if err := json.NewDecoder(resp.Body).Decode(&gotBody); err != nil {
		t.Fatalf("failed to read/decode sink response: %v", err)
	}
	if diff := cmp.Diff(wantBody, gotBody, cmpopts.IgnoreFields(sink.Response{}, "EventID")); diff != "" {
		t.Errorf("unexpected sink response -want/+got: %s", diff)
	}
	if gotBody.EventID == "" {
		t.Errorf("sink response no eventID")
	}

	for _, wantPr := range []v1alpha1.PipelineResource{wantPr1, wantPr2} {
		if err = WaitFor(pipelineResourceExist(t, c, namespace, wantPr.Name)); err != nil {
			t.Fatalf("Failed to create ResourceTemplate %s: %s", wantPr.Name, err)
		}
		gotPr, err := c.ResourceClient.TektonV1alpha1().PipelineResources(namespace).Get(context.Background(), wantPr.Name, metav1.GetOptions{})
		if err != nil {
			t.Errorf("Error getting ResourceTemplate: %s: %s", wantPr.Name, err)
		}
		if gotPr.Labels[eventIDLabel] == "" {
			t.Errorf("Instantiated ResourceTemplate missing EventId")
		} else {
			delete(gotPr.Labels, eventIDLabel)
		}
		if diff := cmp.Diff(wantPr.Labels, gotPr.Labels); diff != "" {
			t.Errorf("Diff instantiated ResourceTemplate labels %s: -want +got: %s", wantPr.Name, diff)
		}
		if diff := cmp.Diff(wantPr.Spec, gotPr.Spec, cmp.Comparer(compareParamsWithLicenseJSON)); diff != "" {
			t.Errorf("Diff instantiated ResourceTemplate spec %s: -want +got: %s", wantPr.Name, diff)
		}
	}

	// Now let's override auth at the trigger level and make sure we get a permission problem

	// create SA/secret with insufficient permissions to set at trigger level
	userWithoutPermissions := "user-with-no-permissions"
	triggerSA := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      userWithoutPermissions,
			Namespace: namespace,
			UID:       types.UID(userWithoutPermissions),
		},
	}

	_, err = c.KubeClient.CoreV1().ServiceAccounts(namespace).Create(context.Background(), triggerSA, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating trigger SA: %s", err.Error())
	}

	if err := WaitFor(func() (bool, error) {
		el, err := c.TriggersClient.TriggersV1alpha1().EventListeners(namespace).Get(context.Background(), el.Name, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}
		for i, trigger := range el.Spec.Triggers {
			trigger.ServiceAccountName = userWithoutPermissions
			el.Spec.Triggers[i] = trigger
		}
		_, err = c.TriggersClient.TriggersV1alpha1().EventListeners(namespace).Update(context.Background(), el, metav1.UpdateOptions{})
		if err != nil {
			return false, nil
		}
		return true, nil
	}); err != nil {
		t.Fatalf("Failed to update EventListener for trigger auth test: %s", err)
	}

	// Verify the EventListener is ready with the new update
	if err := WaitFor(eventListenerReady(t, c, namespace, el.Name)); err != nil {
		t.Fatalf("EventListener not ready after trigger auth update: %s", err)
	}
	// Send POST request to EventListener sink
	req, err = http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:%s", portString), bytes.NewBuffer(eventBodyJSON))
	if err != nil {
		t.Fatalf("Error creating POST request for trigger auth: %s", err)
	}
	req.Header.Add("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Error sending POST request for trigger auth: %s", err)
	}

	if resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusForbidden {
		t.Errorf("sink did not return 401/403 response. Got status code: %d", resp.StatusCode)
	}

	// now set the trigger SA to the original one, should not get a 401/403
	if err := WaitFor(func() (bool, error) {
		el, err := c.TriggersClient.TriggersV1alpha1().EventListeners(namespace).Get(context.Background(), el.Name, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}
		for i, trigger := range el.Spec.Triggers {
			trigger.ServiceAccountName = sa.Name
			el.Spec.Triggers[i] = trigger
		}
		_, err = c.TriggersClient.TriggersV1alpha1().EventListeners(namespace).Update(context.Background(), el, metav1.UpdateOptions{})
		if err != nil {
			return false, nil
		}
		return true, nil
	}); err != nil {
		t.Fatalf("Failed to update EventListener for trigger auth test: %s", err)
	}

	// Verify the EventListener is ready with the new update
	if err := WaitFor(eventListenerReady(t, c, namespace, el.Name)); err != nil {
		t.Fatalf("EventListener not ready after trigger auth update: %s", err)
	}
	// Send POST request to EventListener sink
	req, err = http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:%s", portString), bytes.NewBuffer(eventBodyJSON))
	if err != nil {
		t.Fatalf("Error creating POST request for trigger auth: %s", err)
	}
	req.Header.Add("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Error sending POST request for trigger auth: %s", err)
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		t.Errorf("sink returned 401/403 response: %d", resp.StatusCode)
	}
}

// The structure of this field corresponds to values for the `license` key in
// testdata/pr.json, and can be used to unmarshal the dat.
type license struct {
	Key    string `json:"key"`
	Name   string `json:"name"`
	SpdxID string `json:"spdx_id"`
	URL    string `json:"url"`
	NodeID string `json:"node_id"`
}

// compareParamsWithLicenseJSON will compare the passed in ResourceParams by further checking
// when the values aren't equal if they can be unmarshalled into the license object and if they are
// then equal. This is because the order of values in a dictionary is not deterministic and dictionary
// values passed through an event listener may change order.
func compareParamsWithLicenseJSON(x, y v1alpha1.ResourceParam) bool {
	xData := license{}
	yData := license{}
	if x.Name == y.Name {
		if x.Value != y.Value {
			// In order to compare these values, we are first unmarshalling them into the expected
			// structures because differences in the dictionary order of keys can cause
			// a string comparison to fail.
			if err := json.Unmarshal([]byte(x.Value), &xData); err != nil {
				return false
			}
			if err := json.Unmarshal([]byte(y.Value), &yData); err != nil {
				return false
			}
			if diff := cmp.Diff(xData, yData); diff != "" {
				return false
			}
		}
		return true
	}
	return false
}

func cleanup(t *testing.T, c *clients, namespace, elName string) {
	t.Helper()
	tearDown(t, c, namespace)
	// Delete EventListener
	err := c.TriggersClient.TriggersV1alpha1().EventListeners(namespace).Delete(context.Background(), elName, metav1.DeleteOptions{})
	if err != nil {
		t.Fatalf("Failed to delete EventListener: %s", err)
	}
	t.Log("Deleted EventListener")

	// Verify the EventListener's Deployment is deleted
	if err = WaitFor(deploymentNotExist(t, c, namespace, fmt.Sprintf("%s-%s", eventReconciler.GeneratedResourcePrefix, elName))); err != nil {
		t.Fatalf("Failed to delete EventListener Deployment: %s", err)
	}
	t.Log("EventListener's Deployment was deleted")

	// Verify the EventListener's Service is deleted
	if err = WaitFor(serviceNotExist(t, c, namespace, fmt.Sprintf("%s-%s", eventReconciler.GeneratedResourcePrefix, elName))); err != nil {
		t.Fatalf("Failed to delete EventListener Service: %s", err)
	}
	t.Log("EventListener's Service was deleted")

	// Cleanup cluster-scoped resources
	t.Logf("Deleting cluster-scoped resources")
	if err := c.KubeClient.RbacV1().ClusterRoles().Delete(context.Background(), "my-role", metav1.DeleteOptions{}); err != nil {
		t.Errorf("Failed to delete clusterrole my-role: %s", err)
	}
	if err := c.KubeClient.RbacV1().ClusterRoleBindings().Delete(context.Background(), "my-rolebinding", metav1.DeleteOptions{}); err != nil {
		t.Errorf("Failed to delete clusterrolebinding my-rolebinding: %s", err)
	}
	if err := c.TriggersClient.TriggersV1alpha1().ClusterTriggerBindings().Delete(context.Background(), "my-clustertriggerbinding", metav1.DeleteOptions{}); err != nil {
		t.Errorf("Failed to delete clustertriggerbinding my-clustertriggerbinding: %s", err)
	}
}
