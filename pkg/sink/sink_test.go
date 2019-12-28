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
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/logging"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	faketriggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned/fake"
	dynamicclientset "github.com/tektoncd/triggers/pkg/client/dynamic/clientset"
	"github.com/tektoncd/triggers/pkg/client/dynamic/clientset/tekton"
	"github.com/tektoncd/triggers/pkg/template"
	"github.com/tektoncd/triggers/test"
	bldr "github.com/tektoncd/triggers/test/builder"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	eventID = "12345"
)

func init() {
	// Override UID generator for consistent test results.
	template.UID = func() string { return eventID }
}

func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return false
	case <-time.After(timeout):
		return true
	}
}

func TestHandleEvent(t *testing.T) {
	namespace := "foo"
	eventBody := json.RawMessage(`{"head_commit": {"id": "testrevision"}, "repository": {"url": "testurl"}, "foo": "bar\t\r\nbaz昨"}`)

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
				{Name: "foo", Value: "$(params.foo)"},
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
			bldr.TriggerTemplateParam("foo", "", ""),
			bldr.TriggerResourceTemplate(json.RawMessage(pipelineResourceBytes)),
		))
	tb := bldr.TriggerBinding("my-triggerbinding", namespace,
		bldr.TriggerBindingSpec(
			bldr.TriggerBindingParam("url", "$(body.repository.url)"),
			bldr.TriggerBindingParam("revision", "$(body.head_commit.id)"),
			bldr.TriggerBindingParam("contenttype", "$(header.Content-Type)"),
			bldr.TriggerBindingParam("foo", "$(body.foo)"),
		))
	el := bldr.EventListener("my-eventlistener", namespace,
		bldr.EventListenerSpec(bldr.EventListenerTrigger("my-triggerbinding", "my-triggertemplate", "v1alpha1")))

	kubeClient := fakekubeclientset.NewSimpleClientset()
	test.AddTektonResources(kubeClient)

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
	}

	wantBody := Response{
		EventListener: el.Name,
		Namespace:     el.Namespace,
		EventID:       eventID,
	}

	got := Response{}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("Error unmarshalling response body: %s", err)
	}
	if diff := cmp.Diff(wantBody, got); diff != "" {
		t.Errorf("did not get expected response back -want,+got: %s", diff)
	}

	// We expect that the EventListener will be able to immediately handle the event so we
	// can use a very short timeout
	if waitTimeout(&wg, time.Second) {
		t.Fatalf("timed out waiting for reactor to fire")
	}
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
				{Name: "foo", Value: "bar\t\r\nbaz昨"},
			},
		},
	}
	gvr := schema.GroupVersionResource{
		Group:    "tekton.dev",
		Version:  "v1alpha1",
		Resource: "pipelineresources",
	}
	want := []ktesting.Action{ktesting.NewCreateAction(gvr, "foo", test.ToUnstructured(t, wantResource))}
	if diff := cmp.Diff(want, dynamicClient.Actions()); diff != "" {
		t.Error(diff)
	}
}
