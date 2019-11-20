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

package sink

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/logging"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	faketriggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned/fake"
	dynamicclientset "github.com/tektoncd/triggers/pkg/client/dynamic/clientset"
	"github.com/tektoncd/triggers/pkg/client/dynamic/clientset/tekton"
	"github.com/tektoncd/triggers/pkg/template"
	bldr "github.com/tektoncd/triggers/test/builder"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

const (
	resourceLabel = triggersv1.GroupName + triggersv1.EventListenerLabelKey
	triggerLabel  = triggersv1.GroupName + triggersv1.TriggerLabelKey
	eventIDLabel  = triggersv1.GroupName + triggersv1.EventIDLabelKey

	triggerName = "trigger"
	eventID     = "12345"
)

func init() {
	// Override UID generator for consistent test results.
	template.UID = func() string { return eventID }
}

func Test_findAPIResource_error(t *testing.T) {
	s := Sink{DiscoveryClient: fakekubeclientset.NewSimpleClientset().Discovery()}
	if _, err := s.findAPIResource("v1", "Pod"); err == nil {
		t.Error("findAPIResource() did not return error when expected")
	}
}

func addTektonResources(clientset *fakekubeclientset.Clientset) {
	nameKind := map[string]string{
		"triggertemplates":  "TriggerTemplate",
		"pipelineruns":      "PipelineRun",
		"taskruns":          "TaskRun",
		"pipelineresources": "PipelineResource",
	}
	resources := make([]metav1.APIResource, 0, len(nameKind))
	for name, kind := range nameKind {
		resources = append(resources, metav1.APIResource{
			Group:      "tekton.dev",
			Version:    "v1alpha1",
			Namespaced: true,
			Name:       name,
			Kind:       kind,
		})
	}

	clientset.Resources = append(clientset.Resources, &metav1.APIResourceList{
		GroupVersion: "tekton.dev/v1alpha1",
		APIResources: resources,
	})
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
	addTektonResources(kubeClient)
	s := Sink{DiscoveryClient: kubeClient.Discovery()}

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
			got, err := s.findAPIResource(tt.apiVersion, tt.kind)
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
	elNamespace := "bar"

	kubeClient := fakekubeclientset.NewSimpleClientset()
	addTektonResources(kubeClient)

	dynamicClient := fakedynamic.NewSimpleDynamicClient(runtime.NewScheme())
	dynamicSet := dynamicclientset.New(tekton.WithClient(dynamicClient))

	logger, _ := logging.NewLogger("", "")

	tests := []struct {
		name     string
		resource pipelinev1.PipelineResource
		want     pipelinev1.PipelineResource
	}{{
		name: "PipelineResource without namespace",
		resource: pipelinev1.PipelineResource{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "tekton.dev/v1alpha1",
				Kind:       "PipelineResource",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:   "my-pipelineresource",
				Labels: map[string]string{"woriginal-label-1": "label-1"},
			},
			Spec: pipelinev1.PipelineResourceSpec{},
		},
		want: pipelinev1.PipelineResource{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "tekton.dev/v1alpha1",
				Kind:       "PipelineResource",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-pipelineresource",
				Labels: map[string]string{
					"woriginal-label-1": "label-1",
					resourceLabel:       elName,
					triggerLabel:        triggerName,
					eventIDLabel:        eventID,
				},
			},
			Spec: pipelinev1.PipelineResourceSpec{},
		},
	}, {
		name: "PipelineResource with namespace",
		resource: pipelinev1.PipelineResource{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "tekton.dev/v1alpha1",
				Kind:       "PipelineResource",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "foo",
				Name:      "my-pipelineresource",
				Labels:    map[string]string{"woriginal-label-1": "label-1"},
			},
			Spec: pipelinev1.PipelineResourceSpec{},
		},
		want: pipelinev1.PipelineResource{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "tekton.dev/v1alpha1",
				Kind:       "PipelineResource",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "foo",
				Name:      "my-pipelineresource",
				Labels: map[string]string{
					"woriginal-label-1": "label-1",
					resourceLabel:       elName,
					triggerLabel:        triggerName,
					eventIDLabel:        eventID,
				},
			},
			Spec: pipelinev1.PipelineResourceSpec{},
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dynamicClient.ClearActions()

			r := Sink{
				DynamicClient:          dynamicSet,
				DiscoveryClient:        kubeClient.Discovery(),
				EventListenerNamespace: elNamespace,
				EventListenerName:      elName,
				Logger:                 logger,
			}

			b, err := json.Marshal(tt.resource)
			if err != nil {
				t.Fatalf("error marshalling resource: %v", tt.resource)
			}
			if err := r.createResource(b, triggerName, eventID); err != nil {
				t.Errorf("createResource() returned error: %s", err)
			}

			gvr := schema.GroupVersionResource{
				Group:    "tekton.dev",
				Version:  "v1alpha1",
				Resource: "pipelineresources",
			}
			namespace := tt.want.Namespace
			if namespace == "" {
				namespace = elNamespace
			}
			want := []ktesting.Action{ktesting.NewCreateAction(gvr, namespace, toUnstructured(t, tt.want))}
			if diff := cmp.Diff(want, dynamicClient.Actions()); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestHandleEvent(t *testing.T) {
	namespace := "foo"
	eventBody := json.RawMessage(`{"head_commit": {"id": "testrevision"}, "repository": {"url": "testurl"}}`)
	pipelineResource := pipelinev1.PipelineResource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tekton.dev/v1alpha1",
			Kind:       "PipelineResource",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-pipelineresource",
			Namespace: namespace,
			Labels:    map[string]string{"app": "$(params.appLabel)"},
		},
		Spec: pipelinev1.PipelineResourceSpec{
			Type: pipelinev1.PipelineResourceTypeGit,
			Params: []pipelinev1.ResourceParam{
				{Name: "url", Value: "$(params.url)"},
				{Name: "revision", Value: "$(params.revision)"},
				{Name: "contenttype", Value: "$(params.contenttype)"},
			},
		},
	}
	pipelineResourceBytes, err := json.Marshal(pipelineResource)
	if err != nil {
		t.Fatalf("Error unmarshalling pipelineResource: %s", err)
	}

	tt := bldr.TriggerTemplate("my-triggertemplate", namespace,
		bldr.TriggerTemplateSpec(
			bldr.TriggerTemplateParam("url", "", ""),
			bldr.TriggerTemplateParam("revision", "", ""),
			bldr.TriggerTemplateParam("appLabel", "", "foo"),
			bldr.TriggerTemplateParam("contenttype", "", ""),
			bldr.TriggerResourceTemplate(json.RawMessage(pipelineResourceBytes)),
		))
	tb := bldr.TriggerBinding("my-triggerbinding", namespace,
		bldr.TriggerBindingSpec(
			bldr.TriggerBindingParam("url", "$(body.repository.url)"),
			bldr.TriggerBindingParam("revision", "$(body.head_commit.id)"),
			bldr.TriggerBindingParam("contenttype", "$(header.Content-Type)"),
		))
	el := bldr.EventListener("my-eventlistener", namespace,
		bldr.EventListenerSpec(bldr.EventListenerTrigger("my-triggerbinding", "my-triggertemplate", "v1alpha1")))

	kubeClient := fakekubeclientset.NewSimpleClientset()
	addTektonResources(kubeClient)

	triggersClient := faketriggersclientset.NewSimpleClientset()
	if _, err := triggersClient.TektonV1alpha1().TriggerTemplates(namespace).Create(tt); err != nil {
		t.Fatalf("Error creating TriggerTemplate: %s", err)
	}
	if _, err := triggersClient.TektonV1alpha1().TriggerBindings(namespace).Create(tb); err != nil {
		t.Fatalf("Error creating TriggerBinding: %s", err)
	}
	el, err = triggersClient.TektonV1alpha1().EventListeners(namespace).Create(el)
	if err != nil {
		t.Fatalf("Error creating EventListener: %s", err)
	}

	logger, _ := logging.NewLogger("", "")

	dynamicClient := fakedynamic.NewSimpleDynamicClient(runtime.NewScheme())
	dynamicSet := dynamicclientset.New(tekton.WithClient(dynamicClient))

	r := Sink{
		EventListenerName:      el.Name,
		EventListenerNamespace: namespace,
		DynamicClient:          dynamicSet,
		DiscoveryClient:        kubeClient.Discovery(),
		TriggersClient:         triggersClient,
		Logger:                 logger,
	}
	ts := httptest.NewServer(http.HandlerFunc(r.HandleEvent))
	defer ts.Close()

	var wg sync.WaitGroup
	wg.Add(1)

	dynamicClient.PrependReactor("*", "*", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
		defer wg.Done()
		return false, nil, nil
	})

	resp, err := http.Post(ts.URL, "application/json", bytes.NewReader(eventBody))
	if err != nil {
		t.Fatalf("Error creating Post request: %s", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Response code doesn't match: %v", resp.Status)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Error reading response body: %s", err)
		return
	}
	if !strings.Contains(string(body), "event (EventID: ") {
		t.Errorf("Response body doesn't have eventid")
	}
	wantPayload := `EventListener: my-eventlistener in Namespace: foo handling event`
	if !strings.Contains(string(body), wantPayload) {
		t.Errorf("Diff response body: %s, should have: %s", string(body), wantPayload)
	}

	wg.Wait()
	wantResource := pipelinev1.PipelineResource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tekton.dev/v1alpha1",
			Kind:       "PipelineResource",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-pipelineresource",
			Namespace: namespace,
			Labels: map[string]string{
				"app":         "foo",
				resourceLabel: "my-eventlistener",
				triggerLabel:  el.Spec.Triggers[0].Name,
				eventIDLabel:  eventID,
			},
		},
		Spec: pipelinev1.PipelineResourceSpec{
			Type: pipelinev1.PipelineResourceTypeGit,
			Params: []pipelinev1.ResourceParam{
				{Name: "url", Value: "testurl"},
				{Name: "revision", Value: "testrevision"},
				{Name: "contenttype", Value: "application/json"},
			},
		},
	}
	gvr := schema.GroupVersionResource{
		Group:    "tekton.dev",
		Version:  "v1alpha1",
		Resource: "pipelineresources",
	}
	want := []ktesting.Action{ktesting.NewCreateAction(gvr, "foo", toUnstructured(t, wantResource))}
	if diff := cmp.Diff(want, dynamicClient.Actions()); diff != "" {
		t.Error(diff)
	}
}

func toUnstructured(t *testing.T, in interface{}) *unstructured.Unstructured {
	t.Helper()

	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("error encoding to JSON: %v", err)
	}

	out := new(unstructured.Unstructured)
	if err := out.UnmarshalJSON(b); err != nil {
		t.Fatalf("error encoding to unstructured: %v", err)
	}
	return out
}

func Test_addLabels(t *testing.T) {
	b, err := json.Marshal(map[string]interface{}{
		"metadata": map[string]interface{}{
			"labels": map[string]interface{}{
				// should be overwritten
				"tekton.dev/a": "0",
				// should be preserved.
				"tekton.dev/z":    "0",
				"best-palindrome": "tacocat",
			},
		},
	})
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	raw, err := addLabels(json.RawMessage(b), map[string]string{
		"a":   "1",
		"/b":  "2",
		"//c": "3",
	})
	if err != nil {
		t.Fatalf("addLabels: %v", err)
	}

	got := make(map[string]interface{})
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	want := map[string]interface{}{
		"metadata": map[string]interface{}{
			"labels": map[string]interface{}{
				"tekton.dev/a":    "1",
				"tekton.dev/b":    "2",
				"tekton.dev/c":    "3",
				"tekton.dev/z":    "0",
				"best-palindrome": "tacocat",
			},
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Error(diff)
	}
}
