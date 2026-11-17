/*
Copyright 2026 The Tekton Authors

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

package cronrunner_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/triggers/pkg/apis/triggers"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	faketriggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned/fake"
	"github.com/tektoncd/triggers/pkg/cronrunner"
	"github.com/tektoncd/triggers/pkg/reconciler/events"
	"github.com/tektoncd/triggers/pkg/template"
	"github.com/tektoncd/triggers/test"
	"go.uber.org/zap/zaptest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/ptr"
)

func TestRunner_Run_Inline(t *testing.T) {
	origUUID := template.UUID
	template.UUID = func() string { return "exec-1" }
	t.Cleanup(func() { template.UUID = origUUID })

	st := &v1alpha1.ScheduledTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "daily",
			Namespace: "default",
		},
		Spec: v1alpha1.ScheduledTemplateSpec{
			Schedule: "* * * * *",
			TriggerTemplateSpec: v1alpha1.TriggerTemplateSpec{
				Params: []v1alpha1.ParamSpec{{
					Name:    "message",
					Default: ptr.String("hello"),
				}},
				ResourceTemplates: []v1alpha1.TriggerResourceTemplate{{
					RawExtension: test.RawExtension(t, pipelinev1.TaskRun{
						TypeMeta: metav1.TypeMeta{
							APIVersion: "tekton.dev/v1",
							Kind:       "TaskRun",
						},
						ObjectMeta: metav1.ObjectMeta{
							GenerateName: "tr-",
						},
						Spec: pipelinev1.TaskRunSpec{
							Params: []pipelinev1.Param{{
								Name: "msg",
								Value: pipelinev1.ParamValue{
									Type:      pipelinev1.ParamTypeString,
									StringVal: "$(tt.params.message)",
								},
							}},
						},
					}),
				}},
			},
		},
	}

	triggersClient := faketriggersclientset.NewSimpleClientset(st)
	kubeClient := fakekubeclientset.NewSimpleClientset()
	test.AddTektonResources(kubeClient)
	dynamicClient := fakedynamic.NewSimpleDynamicClient(runtime.NewScheme())

	r := &cronrunner.Runner{
		TriggersClient:  triggersClient,
		DynamicClient:   dynamicClient,
		DiscoveryClient: kubeClient.Discovery(),
		Logger:          zaptest.NewLogger(t).Sugar(),
	}

	if err := r.Run(context.Background(), "default", "daily"); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	assertCreatedWithLabels(t, dynamicClient, schema.GroupVersionResource{
		Group: "tekton.dev", Version: "v1", Resource: "taskruns",
	}, "default", map[string]string{
		triggers.GroupName + triggers.ScheduledTemplateLabelKey: "daily",
		triggers.GroupName + triggers.EventIDLabelKey:           "exec-1",
	})
}

func TestRunner_Run_Ref(t *testing.T) {
	origUUID := template.UUID
	template.UUID = func() string { return "exec-2" }
	t.Cleanup(func() { template.UUID = origUUID })

	tt := &v1alpha1.TriggerTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-tt",
			Namespace: "default",
		},
		Spec: v1alpha1.TriggerTemplateSpec{
			ResourceTemplates: []v1alpha1.TriggerResourceTemplate{{
				RawExtension: test.RawExtension(t, pipelinev1.TaskRun{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "tekton.dev/v1",
						Kind:       "TaskRun",
					},
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "tr-",
					},
				}),
			}},
		},
	}
	st := &v1alpha1.ScheduledTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hourly",
			Namespace: "default",
		},
		Spec: v1alpha1.ScheduledTemplateSpec{
			Schedule: "0 * * * *",
			Ref:      ptr.String("my-tt"),
		},
	}

	triggersClient := faketriggersclientset.NewSimpleClientset(st, tt)
	kubeClient := fakekubeclientset.NewSimpleClientset()
	test.AddTektonResources(kubeClient)
	dynamicClient := fakedynamic.NewSimpleDynamicClient(runtime.NewScheme())

	r := &cronrunner.Runner{
		TriggersClient:  triggersClient,
		DynamicClient:   dynamicClient,
		DiscoveryClient: kubeClient.Discovery(),
		Logger:          zaptest.NewLogger(t).Sugar(),
	}

	if err := r.Run(context.Background(), "default", "hourly"); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	assertCreatedWithLabels(t, dynamicClient, schema.GroupVersionResource{
		Group: "tekton.dev", Version: "v1", Resource: "taskruns",
	}, "default", map[string]string{
		triggers.GroupName + triggers.ScheduledTemplateLabelKey: "hourly",
		triggers.GroupName + triggers.EventIDLabelKey:           "exec-2",
	})
}

func TestRunner_Run_MissingName(t *testing.T) {
	r := &cronrunner.Runner{Logger: zaptest.NewLogger(t).Sugar()}
	if err := r.Run(context.Background(), "", ""); err == nil {
		t.Fatal("expected error for empty namespace/name")
	}
}

func TestRunner_Run_MissingRefEmitsCloudEventWithID(t *testing.T) {
	origUUID := template.UUID
	template.UUID = func() string { return "exec-missing-ref" }
	t.Cleanup(func() { template.UUID = origUUID })

	var (
		mu      sync.Mutex
		gotBody []byte
		gotType string
		gotID   string
	)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		gotBody = body
		gotType = r.Header.Get("Ce-Type")
		gotID = r.Header.Get("Ce-Id")
		mu.Unlock()
		w.WriteHeader(http.StatusAccepted)
	}))
	defer ts.Close()

	sink, err := apis.ParseURL(ts.URL)
	if err != nil {
		t.Fatalf("ParseURL: %v", err)
	}

	st := &v1alpha1.ScheduledTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hourly",
			Namespace: "default",
		},
		Spec: v1alpha1.ScheduledTemplateSpec{
			Schedule:       "0 * * * *",
			Ref:            ptr.String("missing-tt"),
			CloudEventSink: sink,
		},
	}
	triggersClient := faketriggersclientset.NewSimpleClientset(st)
	ceClient, err := cloudevents.NewClientHTTP()
	if err != nil {
		t.Fatalf("NewClientHTTP: %v", err)
	}

	r := &cronrunner.Runner{
		TriggersClient:   triggersClient,
		Logger:           zaptest.NewLogger(t).Sugar(),
		CloudEventClient: ceClient,
	}
	if err := r.Run(context.Background(), "default", "hourly"); err == nil {
		t.Fatal("expected error for missing TriggerTemplate ref")
	}

	mu.Lock()
	defer mu.Unlock()
	if gotID != "exec-missing-ref" {
		t.Errorf("Ce-Id = %q, want exec-missing-ref", gotID)
	}
	if gotType != events.ScheduledTemplateFailedV1 {
		t.Errorf("Ce-Type = %q, want %q", gotType, events.ScheduledTemplateFailedV1)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(gotBody, &payload); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if payload["executionID"] != "exec-missing-ref" {
		t.Errorf("payload executionID = %v, want exec-missing-ref", payload["executionID"])
	}
}

func assertCreatedWithLabels(t *testing.T, dynamicClient *fakedynamic.FakeDynamicClient, gvr schema.GroupVersionResource, namespace string, wantLabels map[string]string) {
	t.Helper()
	var created *unstructured.Unstructured
	for _, action := range dynamicClient.Actions() {
		ca, ok := action.(ktesting.CreateAction)
		if !ok {
			continue
		}
		if ca.GetResource() != gvr || ca.GetNamespace() != namespace {
			continue
		}
		obj, ok := ca.GetObject().(*unstructured.Unstructured)
		if !ok {
			t.Fatalf("create action object type %T, want *unstructured.Unstructured", ca.GetObject())
		}
		created = obj
		break
	}
	if created == nil {
		t.Fatalf("no create action for %v in namespace %s", gvr, namespace)
	}
	labels := created.GetLabels()
	for k, v := range wantLabels {
		if labels[k] != v {
			t.Errorf("label %s = %q, want %q", k, labels[k], v)
		}
	}
}
