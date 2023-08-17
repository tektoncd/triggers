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

package resources

import (
	"encoding/json"
	"fmt"
	"testing"

	"go.uber.org/zap/zaptest"

	"github.com/google/go-cmp/cmp"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/triggers/pkg/apis/triggers"
	dynamicclientset "github.com/tektoncd/triggers/pkg/client/dynamic/clientset"
	"github.com/tektoncd/triggers/pkg/client/dynamic/clientset/tekton"
	"github.com/tektoncd/triggers/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

const (
	resourceLabel = triggers.GroupName + triggers.EventListenerLabelKey
	triggerLabel  = triggers.GroupName + triggers.TriggerLabelKey
	eventIDLabel  = triggers.GroupName + triggers.EventIDLabelKey
	triggerName   = "trigger"
	eventID       = "12345"
)

func Test_FindAPIResource_error(t *testing.T) {
	dc := fakekubeclientset.NewSimpleClientset().Discovery()
	if _, err := findAPIResource("v1", "Pod", dc); err == nil {
		t.Error("findAPIResource() did not return error when expected")
	}
}

func TestFindAPIResource(t *testing.T) {
	// Create fake kubeclient with list of resources
	kubeClient := fakekubeclientset.NewSimpleClientset()
	kubeClient.Resources = []*metav1.APIResourceList{{
		GroupVersion: "v1",
		APIResources: []metav1.APIResource{{
			Name:       "pods",
			Namespaced: true,
			Kind:       "Pod",
		}, {
			Name:       "namespaces",
			Namespaced: false,
			Kind:       "Namespace",
		}},
	}}
	test.AddTektonResources(kubeClient)
	dc := kubeClient.Discovery()

	tests := []struct {
		apiVersion string
		kind       string
		want       *metav1.APIResource
	}{{
		apiVersion: "v1",
		kind:       "Pod",
		want: &metav1.APIResource{
			Name:       "pods",
			Namespaced: true,
			Version:    "v1",
			Kind:       "Pod",
		},
	}, {
		apiVersion: "v1",
		kind:       "Namespace",
		want: &metav1.APIResource{
			Name:       "namespaces",
			Namespaced: false,
			Version:    "v1",
			Kind:       "Namespace",
		},
	}, {
		apiVersion: "tekton.dev/v1alpha1",
		kind:       "TriggerTemplate",
		want: &metav1.APIResource{
			Group:      "tekton.dev",
			Version:    "v1alpha1",
			Name:       "triggertemplates",
			Namespaced: true,
			Kind:       "TriggerTemplate",
		},
	}, {
		apiVersion: "tekton.dev/v1alpha1",
		kind:       "PipelineRun",
		want: &metav1.APIResource{
			Group:      "tekton.dev",
			Version:    "v1alpha1",
			Name:       "pipelineruns",
			Namespaced: true,
			Kind:       "PipelineRun",
		},
	},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", tt.apiVersion, tt.kind), func(t *testing.T) {
			got, err := findAPIResource(tt.apiVersion, tt.kind, dc)
			if err != nil {
				t.Errorf("findAPIResource() returned error: %s", err)
			} else if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("findAPIResource() Diff: -want +got: %s", diff)
			}
		})
	}
}

func TestCreateResource(t *testing.T) {
	elName := "foo-el"
	elNamespace := "foo"

	kubeClient := fakekubeclientset.NewSimpleClientset()
	test.AddTektonResources(kubeClient)

	dynamicClient := fakedynamic.NewSimpleDynamicClient(runtime.NewScheme())
	dynamicSet := dynamicclientset.New(tekton.WithClient(dynamicClient))

	logger := zaptest.NewLogger(t)

	tests := []struct {
		name string
		json []byte
		want pipelinev1.TaskRun
	}{{
		name: "TaskRun without namespace",
		json: json.RawMessage(`{"kind":"TaskRun","apiVersion":"tekton.dev/v1beta1","metadata":{"name":"my-taskrun","creationTimestamp":null,"labels":{"someLabel":"bar"}},"spec":{"serviceAccountName":"","taskRef":{"name":"my-task"}},"status":{"podName": ""}}`),
		want: pipelinev1.TaskRun{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "tekton.dev/v1beta1",
				Kind:       "TaskRun",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-taskrun",
				Labels: map[string]string{
					"someLabel":   "bar", // replaced with the value of foo from bar
					resourceLabel: "foo-el",
					triggerLabel:  triggerName,
					eventIDLabel:  eventID,
				},
			},
			Spec: pipelinev1.TaskRunSpec{
				TaskRef: &pipelinev1.TaskRef{
					Name: "my-task", // non-existent task; just for testing
				},
			},
			Status: pipelinev1.TaskRunStatus{},
		},
	},

		{
			name: "TaskRun with namespace",
			json: json.RawMessage(`{"kind":"TaskRun","apiVersion":"tekton.dev/v1beta1","metadata":{"name":"my-taskrun","namespace":"bar","creationTimestamp":null,"labels":{"someLabel":"bar"}},"spec":{"serviceAccountName":"","taskRef":{"name":"my-task"}},"status":{"podName":""}}`),
			want: pipelinev1.TaskRun{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "tekton.dev/v1beta1",
					Kind:       "TaskRun",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-taskrun",
					Namespace: "bar",
					Labels: map[string]string{
						"someLabel":   "bar", // replaced with the value of foo from bar
						resourceLabel: "foo-el",
						triggerLabel:  triggerName,
						eventIDLabel:  eventID,
					},
				},
				Spec: pipelinev1.TaskRunSpec{
					TaskRef: &pipelinev1.TaskRef{
						Name: "my-task", // non-existent task; just for testing
					},
				},
				Status: pipelinev1.TaskRunStatus{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dynamicClient.ClearActions()
			if err := Create(logger.Sugar(), tt.json, triggerName, eventID, elName, elNamespace, kubeClient.Discovery(), dynamicSet); err != nil {
				t.Errorf("createTaskRun() returned error: %s", err)
			}

			gvr := schema.GroupVersionResource{
				Group:    "tekton.dev",
				Version:  "v1beta1",
				Resource: "taskruns",
			}
			namespace := tt.want.Namespace
			if namespace == "" {
				namespace = elNamespace
			}
			want := []ktesting.Action{ktesting.NewCreateAction(gvr, namespace, test.ToUnstructured(t, tt.want))}
			if diff := cmp.Diff(want, dynamicClient.Actions()); diff != "" {
				fmt.Println("diff", diff)
				t.Error(diff)
			}
		})
	}
}

func Test_AddLabels(t *testing.T) {
	tests := []struct {
		name        string
		us          *unstructured.Unstructured
		labelsToAdd map[string]string
		want        *unstructured.Unstructured
	}{
		{
			name: "add to empty labels",
			us: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{},
				}},
			labelsToAdd: map[string]string{"foo": "bar"},
			want: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"labels": map[string]interface{}{
							"triggers.tekton.dev/foo": "bar",
						},
					},
				},
			},
		},
		{
			name: "overwrite label",
			us: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"labels": map[string]interface{}{
							"triggers.tekton.dev/foo": "bar",
						},
					},
				}},
			labelsToAdd: map[string]string{"foo": "foo"},
			want: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"labels": map[string]interface{}{
							"triggers.tekton.dev/foo": "foo",
						},
					},
				},
			},
		},
		{
			name: "add and overwrite labels",
			us: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"labels": map[string]interface{}{
							// should be overwritten
							"triggers.tekton.dev/foo":   "bar",
							"triggers.tekton.dev/hello": "world",
							// should be preserved
							"triggers.tekton.dev/z": "0",
							"best-palindrome":       "tacocat",
						},
					},
				}},
			labelsToAdd: map[string]string{
				"foo":   "foo",
				"hello": "there",
				"a":     "a",
				"b":     "b",
			},
			want: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"labels": map[string]interface{}{
							"triggers.tekton.dev/foo":   "foo",
							"triggers.tekton.dev/hello": "there",
							"triggers.tekton.dev/z":     "0",
							"best-palindrome":           "tacocat",
							"triggers.tekton.dev/a":     "a",
							"triggers.tekton.dev/b":     "b",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := addLabels(tt.us, tt.labelsToAdd)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("addLabels(): -want +got: %s", diff)
			}
		})
	}

	t.Run("non-string label", func(t *testing.T) {
		in := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"metadata": map[string]interface{}{
					"labels": map[string]interface{}{
						"foo": 0,
					},
				},
			},
		}
		if got, err := addLabels(in, map[string]string{"a": "b"}); err == nil {
			t.Errorf("expected error, got: %v", got)
		}
	})
}
