//go:build e2e
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
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/tektoncd/triggers/pkg/apis/triggers"
	"knative.dev/pkg/ptr"

	"k8s.io/client-go/kubernetes"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	eventReconciler "github.com/tektoncd/triggers/pkg/reconciler/eventlistener"
	"github.com/tektoncd/triggers/pkg/reconciler/eventlistener/resources"
	"github.com/tektoncd/triggers/pkg/sink"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	knativetest "knative.dev/pkg/test"
)

const (
	resourceLabel = triggers.GroupName + triggers.EventListenerLabelKey
	triggerLabel  = triggers.GroupName + triggers.TriggerLabelKey
	eventIDLabel  = triggers.GroupName + triggers.EventIDLabelKey

	examplePRJsonFilename = "pr.json"
)

var (
	// ignoreSATaskRunSpec ignores the service account in the TaskRunSpec as it may differ across platforms
	ignoreSATaskRunSpec = cmpopts.IgnoreFields(pipelinev1.TaskRunSpec{}, "ServiceAccountName")
)

func loadExamplePREventBytes(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("testdata", examplePRJsonFilename)
	bytes, err := os.ReadFile(path)
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
	defer cleanup(t, c, namespace, "my-eventlistener")
	knativetest.CleanupOnInterrupt(func() { cleanup(t, c, namespace, "my-eventlistener") }, t.Logf)

	t.Log("Start EventListener e2e test")

	// TemplatedPipelineResources
	tr1 := pipelinev1.TaskRun{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tekton.dev/v1",
			Kind:       "TaskRun",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tr1",
			Namespace: namespace,
			Labels: map[string]string{
				"$(tt.params.oneparam)": "$(tt.params.oneparam)",
			},
		},
		Spec: pipelinev1.TaskRunSpec{
			Params: []pipelinev1.Param{{
				Name: "url",
				Value: pipelinev1.ParamValue{
					Type:      pipelinev1.ParamTypeString,
					StringVal: "$(tt.params.url)",
				}}},
			TaskRef: &pipelinev1.TaskRef{
				Name: "git-clone",
			},
		},
	}

	tr1Bytes, err := json.Marshal(tr1)
	if err != nil {
		t.Fatalf("Error marshalling TaskRun 1: %s", err)
	}
	defaultValueStr := "defaultvalue"

	// TriggerTemplate
	tt, err := c.TriggersClient.TriggersV1alpha1().TriggerTemplates(namespace).Create(context.Background(),
		&triggersv1.TriggerTemplate{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-triggertemplate",
				Annotations: map[string]string{
					"triggers.tekton.dev/old-escape-quotes": "true",
				},
			},
			Spec: triggersv1.TriggerTemplateSpec{
				Params: []triggersv1.ParamSpec{
					{
						Name: "oneparam",
					},
					{
						Name: "twoparamname",
					},
					{
						Name:    "url",
						Default: &defaultValueStr,
					},
					{
						Name: "license",
					},
					{
						Name: "header",
					},
					{
						Name: "prmessage",
					},
				},
				ResourceTemplates: []triggersv1.TriggerResourceTemplate{
					{
						RawExtension: runtime.RawExtension{Raw: tr1Bytes},
					},
				},
			},
		}, metav1.CreateOptions{},
	)
	if err != nil {
		t.Fatalf("Error creating TriggerTemplate: %s", err)
	}

	// TriggerBinding
	tb, err := c.TriggersClient.TriggersV1alpha1().TriggerBindings(namespace).Create(context.Background(),
		&triggersv1.TriggerBinding{
			ObjectMeta: metav1.ObjectMeta{Name: "my-triggerbinding"},
			Spec: triggersv1.TriggerBindingSpec{
				Params: []triggersv1.Param{
					{
						Name:  "oneparam",
						Value: "$(body.action)",
					},
					{
						Name:  "twoparamname",
						Value: "$(body.pull_request.state)",
					},
					{
						Name:  "prmessage",
						Value: "$(body.pull_request.body)",
					},
				},
			},
		}, metav1.CreateOptions{},
	)
	if err != nil {
		t.Fatalf("Error creating TriggerBinding: %s", err)
	}

	// ClusterTriggerBinding
	ctb, err := c.TriggersClient.TriggersV1alpha1().ClusterTriggerBindings().Create(context.Background(),
		&triggersv1.ClusterTriggerBinding{
			ObjectMeta: metav1.ObjectMeta{Name: "my-clustertriggerbinding"},
			Spec: triggersv1.TriggerBindingSpec{
				Params: []triggersv1.Param{
					{
						Name:  "license",
						Value: "$(body.repository.license)",
					},
					{
						Name:  "header",
						Value: "$(header)",
					},
				},
			},
		}, metav1.CreateOptions{},
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
				APIGroups: []string{triggers.GroupName},
				Resources: []string{"clustertriggerbindings", "eventlisteners", "clusterinterceptors", "interceptors", "triggerbindings", "triggertemplates", "triggers"},
				Verbs:     []string{"get", "list", "watch"},
			}, {
				APIGroups: []string{"tekton.dev"},
				Resources: []string{"taskruns"},
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
					Ref: triggersv1.InterceptorRef{Name: "cel"},
					Params: []triggersv1.InterceptorParams{{
						Name:  "filter",
						Value: ToV1JSON(t, `body.action == "edited"`),
					}},
				}},
			}},
			Resources: triggersv1.Resources{
				KubernetesResource: &triggersv1.KubernetesResource{
					Replicas: ptr.Int32(3),
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

	timeout := metav1.Duration{Duration: time.Hour}
	// Event body & Expected ResourceTemplates after instantiation
	wantTr1 := pipelinev1.TaskRun{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tekton.dev/v1",
			Kind:       "TaskRun",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tr1",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "tekton-pipelines",
				resourceLabel:                  "my-eventlistener",
				triggerLabel:                   el.Spec.Triggers[0].Name,
				"edited":                       "edited",
			},
		},
		Spec: pipelinev1.TaskRunSpec{
			Params: []pipelinev1.Param{{
				Name: "url",
				Value: pipelinev1.ParamValue{
					Type:      pipelinev1.ParamTypeString,
					StringVal: defaultValueStr,
				}}},
			TaskRef: &pipelinev1.TaskRef{
				Name: "git-clone",
				Kind: "Task",
			},
			Timeout: &timeout,
		},
	}

	labelSelector := fields.SelectorFromSet(resources.GenerateLabels(el.Name, resources.DefaultStaticResourceLabels)).String()
	// Grab EventListener sink pods
	sinkPods, err := c.KubeClient.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		t.Fatalf("Error listing EventListener sink pods: %s", err)
	}

	// ElPort forward sink pod for http request
	portString := strconv.Itoa(8080)
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
			// revive:disable:empty-block
			for range readyChan {
			}
			// revive:enable:empty-block
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
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://127.0.0.1:%s", portString), bytes.NewBuffer(eventBodyJSON))
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
		EventListener:    "my-eventlistener",
		Namespace:        namespace,
		EventListenerUID: string(el.GetUID()),
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

	for _, wantTr := range []pipelinev1.TaskRun{wantTr1} {
		if err = WaitFor(taskrunExist(t, c, namespace, wantTr.Name)); err != nil {
			t.Fatalf("Failed to create TaskRun %s: %s", wantTr.Name, err)
		}
		gotTr, err := c.PipelineClient.TektonV1().TaskRuns(namespace).Get(context.Background(), wantTr.Name, metav1.GetOptions{})
		if err != nil {
			t.Errorf("Error getting TaskRun: %s: %s", wantTr.Name, err)
		}
		if gotTr.Labels[eventIDLabel] == "" {
			t.Errorf("Instantiated TaskRun missing EventId")
		} else {
			delete(gotTr.Labels, eventIDLabel)
		}
		if diff := cmp.Diff(wantTr.Labels, gotTr.Labels); diff != "" {
			t.Errorf("Diff instantiated TaskRun labels %s: -want +got: %s", wantTr.Name, diff)
		}
		if diff := cmp.Diff(wantTr.Spec, gotTr.Spec, ignoreSATaskRunSpec); diff != "" {
			t.Errorf("Diff instantiated TaskRun spec %s: -want +got: %s", wantTr.Name, diff)
		}
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
	req, err = http.NewRequest(http.MethodPost, fmt.Sprintf("http://127.0.0.1:%s", portString), bytes.NewBuffer(eventBodyJSON))
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
