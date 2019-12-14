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
	bldr "github.com/tektoncd/triggers/test/builder"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	fakerestclient "k8s.io/client-go/rest/fake"
)

const (
	resourceLabel = triggersv1.GroupName + triggersv1.EventListenerLabelKey
	triggerLabel  = triggersv1.GroupName + triggersv1.TriggerLabelKey
	eventIDLabel  = triggersv1.GroupName + triggersv1.EventIDLabelKey
)

func Test_findAPIResource_error(t *testing.T) {
	kubeClient := fakekubeclientset.NewSimpleClientset()
	_, err := findAPIResource(kubeClient.Discovery(), "v1", "Pod")
	if err == nil {
		t.Error("findAPIResource() did not return error when expected")
	}
}

func Test_createRequestURI(t *testing.T) {
	tests := []struct {
		apiVersion string
		namePlural string
		namespace  string
		namespaced bool
		want       string
	}{{
		apiVersion: "tekton.dev/v1alpha1",
		namePlural: "pipelineruns",
		namespace:  "foo",
		namespaced: true,
		want:       "apis/tekton.dev/v1alpha1/namespaces/foo/pipelineruns",
	}, {
		apiVersion: "v1",
		namePlural: "secrets",
		namespace:  "foo",
		namespaced: true,
		want:       "api/v1/namespaces/foo/secrets",
	}, {
		apiVersion: "v1",
		namePlural: "namespaces",
		namespace:  "",
		namespaced: false,
		want:       "api/v1/namespaces",
	}}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := createRequestURI(tt.apiVersion, tt.namePlural, tt.namespace, tt.namespaced)
			if got != tt.want {
				t.Errorf("createRequestURI() = %s, want = %s", got, tt.want)
			}
		})
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
	}, {
		GroupVersion: "tekton.dev/v1alpha1",
		APIResources: []metav1.APIResource{{
			Name:       "triggertemplates",
			Namespaced: true,
			Kind:       "TriggerTemplate",
		}, {
			Name:       "pipelineruns",
			Namespaced: true,
			Kind:       "PipelineRun",
		}},
	},
	}
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
			Kind:       "Pod",
		},
	}, {
		apiVersion: "v1",
		kind:       "Namespace",
		want: &metav1.APIResource{
			Name:       "namespaces",
			Namespaced: false,
			Kind:       "Namespace",
		},
	}, {
		apiVersion: "tekton.dev/v1alpha1",
		kind:       "TriggerTemplate",
		want: &metav1.APIResource{
			Name:       "triggertemplates",
			Namespaced: true,
			Kind:       "TriggerTemplate",
		},
	}, {
		apiVersion: "tekton.dev/v1alpha1",
		kind:       "PipelineRun",
		want: &metav1.APIResource{
			Name:       "pipelineruns",
			Namespaced: true,
			Kind:       "PipelineRun",
		},
	},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", tt.apiVersion, tt.kind), func(t *testing.T) {
			got, err := findAPIResource(kubeClient.Discovery(), tt.apiVersion, tt.kind)
			if err != nil {
				t.Errorf("findAPIResource() returned error: %s", err)
			} else if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("findAPIResource() Diff: -want +got: %s", diff)
			}
		})
	}
}

func TestFindAPIResource_error(t *testing.T) {
	kubeClient := fakekubeclientset.NewSimpleClientset()
	_, err := findAPIResource(kubeClient.Discovery(), "v1", "Pod")
	if err == nil {
		t.Error("findAPIResource() did not return error when expected")
	}
}

func TestCreateResource(t *testing.T) {
	pr1 := pipelinev1.PipelineResource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tekton.dev/v1alpha1",
			Kind:       "PipelineResource",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   "my-pipelineresource",
			Labels: map[string]string{"woriginal-label-1": "label-1"},
		},
		Spec: pipelinev1.PipelineResourceSpec{},
	}
	Pr1Want := pipelinev1.PipelineResource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tekton.dev/v1alpha1",
			Kind:       "PipelineResource",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-pipelineresource",
			Labels: map[string]string{
				"woriginal-label-1": "label-1",
				resourceLabel:       "foo-el",
				triggerLabel:        "trigger",
				eventIDLabel:        "12345",
			},
		},
		Spec: pipelinev1.PipelineResourceSpec{},
	}

	pr2 := pr1
	pr2.Namespace = "foo"
	pr2.Labels = map[string]string{
		resourceLabel: "bar-el",
		triggerLabel:  "trigger",
		eventIDLabel:  "54321",
	}

	pr1Bytes, err := json.Marshal(pr1)
	if err != nil {
		t.Fatalf("Error marshalling PipelineResource: %s", err)
	}

	pr1WantBytes, err := json.Marshal(Pr1Want)
	if err != nil {
		t.Fatalf("Error marshalling wanted PipelineResource: %s", err)
	}

	pr2Bytes, err := json.Marshal(pr2)
	if err != nil {
		t.Fatalf("Error marshalling namespaced PipelineResource: %s", err)
	}

	namespace := corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "tekton-pipelines",
			Labels: map[string]string{
				resourceLabel: "test-el",
				triggerLabel:  "trigger",
				eventIDLabel:  "12321",
			},
		},
	}
	namespaceBytes, err := json.Marshal(namespace)
	if err != nil {
		t.Fatalf("Error marshalling Namespace: %s", err)
	}
	kubeClient := fakekubeclientset.NewSimpleClientset()
	kubeClient.Resources = []*metav1.APIResourceList{{
		GroupVersion: "tekton.dev/v1alpha1",
		APIResources: []metav1.APIResource{{
			Name:       "pipelineresources",
			Kind:       "PipelineResource",
			Namespaced: true,
		}},
	}, {
		GroupVersion: "v1",
		APIResources: []metav1.APIResource{
			{
				Name:       "namespaces",
				Kind:       "Namespace",
				Namespaced: false,
			},
		},
	}}
	restClient, err := restclient.RESTClientFor(&restclient.Config{
		APIPath: "/fooapipath",
		ContentConfig: restclient.ContentConfig{
			NegotiatedSerializer: scheme.Codecs,
			GroupVersion:         &schema.GroupVersion{Group: "foogroup", Version: "fooversion"},
		},
	})
	if err != nil {
		t.Fatalf("Error creating RESTClient: %s", err)
	}
	tests := []struct {
		name                   string
		resource               json.RawMessage
		wantResource           json.RawMessage
		eventListenerNamespace string
		eventListenerName      string
		eventID                string
		wantURLPath            string
	}{{
		name:                   "PipelineResource without namespace",
		resource:               json.RawMessage(pr1Bytes),
		wantResource:           json.RawMessage(pr1WantBytes),
		eventListenerNamespace: "bar",
		eventListenerName:      "foo-el",
		eventID:                "12345",
		wantURLPath:            "/apis/tekton.dev/v1alpha1/namespaces/bar/pipelineresources",
	}, {
		name:                   "PipelineResource with namespace",
		resource:               json.RawMessage(pr2Bytes),
		wantResource:           json.RawMessage(pr2Bytes),
		eventListenerNamespace: "bar",
		eventListenerName:      "bar-el",
		eventID:                "54321",
		wantURLPath:            "/apis/tekton.dev/v1alpha1/namespaces/foo/pipelineresources",
	}, {
		name:                   "Namespace",
		resource:               json.RawMessage(namespaceBytes),
		wantResource:           json.RawMessage(namespaceBytes),
		eventListenerNamespace: "",
		eventListenerName:      "test-el",
		eventID:                "12321",
		wantURLPath:            "/api/v1/namespaces",
	},
	}
	logger, _ := logging.NewLogger("", "")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup fake client
			numRequests := 0
			restClient.Client = fakerestclient.CreateHTTPClient(func(request *http.Request) (*http.Response, error) {
				numRequests++
				if diff := cmp.Diff(tt.wantURLPath, request.URL.Path); diff != "" {
					t.Errorf("Diff request uri: -want +got: %s", diff)
				}
				body, err := ioutil.ReadAll(request.Body)
				if err != nil {
					return nil, err
				}

				var actualResource, expectedResource interface{}
				if err := json.Unmarshal(body, &actualResource); err != nil {
					t.Error(err)
				}
				if err := json.Unmarshal(tt.wantResource, &expectedResource); err != nil {
					t.Error(err)
				}
				if diff := cmp.Diff(expectedResource, actualResource); diff != "" {
					t.Errorf("Diff request body: -want +got: %s", diff)
				}
				return &http.Response{StatusCode: http.StatusCreated, Body: ioutil.NopCloser(bytes.NewReader([]byte{}))}, nil
			})
			r := Sink{
				RESTClient:             restClient,
				DiscoveryClient:        kubeClient.Discovery(),
				EventListenerNamespace: tt.eventListenerNamespace,
				EventListenerName:      tt.eventListenerName,
				Logger:                 logger,
			}

			// Run test
			err := r.createResource(tt.resource, "trigger", tt.eventID)
			if err != nil {
				t.Errorf("createResource() returned error: %s", err)
			}
			if numRequests != 1 {
				t.Errorf("Expected 1 request, got %d requests", numRequests)
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
	wantPipelineResourceURLPath := "/apis/tekton.dev/v1alpha1/namespaces/foo/pipelineresources"

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
	kubeClient.Resources = []*metav1.APIResourceList{{
		GroupVersion: "tekton.dev/v1alpha1",
		APIResources: []metav1.APIResource{{
			Name:       "pipelineresources",
			Kind:       "PipelineResource",
			Namespaced: true,
		}},
	},
	}
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

	// Some resource data is derived from the created resources, so
	// don't populate the desired resource until everything is created.
	wantPipelineResource := pipelinev1.PipelineResource{
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
				eventIDLabel:  "12345",
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

	restClient, err := restclient.RESTClientFor(&restclient.Config{
		APIPath: "/fooapipath",
		ContentConfig: restclient.ContentConfig{
			NegotiatedSerializer: scheme.Codecs,
			GroupVersion:         &schema.GroupVersion{Group: "foogroup", Version: "fooversion"},
		},
	})
	if err != nil {
		t.Fatalf("Error creating RESTClient: %s", err)
	}
	logger, _ := logging.NewLogger("", "")
	r := Sink{
		EventListenerName:      el.Name,
		EventListenerNamespace: namespace,
		RESTClient:             restClient,
		DiscoveryClient:        kubeClient.Discovery(),
		TriggersClient:         triggersClient,
		Logger:                 logger,
	}
	ts := httptest.NewServer(http.HandlerFunc(r.HandleEvent))
	defer ts.Close()

	numRequests := 0
	var wg sync.WaitGroup
	wg.Add(1)

	restClient.Client = fakerestclient.CreateHTTPClient(func(request *http.Request) (*http.Response, error) {
		defer wg.Done()
		numRequests++
		if diff := cmp.Diff(wantPipelineResourceURLPath, request.URL.Path); diff != "" {
			t.Errorf("Diff request uri: -want +got: %s", diff)
		}
		body, err := ioutil.ReadAll(request.Body)
		if err != nil {
			return nil, err
		}
		gotPipelineResource := pipelinev1.PipelineResource{}
		if err = json.Unmarshal(body, &gotPipelineResource); err != nil {
			t.Errorf("Error unmarshalling body as pipelineResource: %s \n%s", string(body), err)
		}
		if gotPipelineResource.ObjectMeta.Labels["tekton.dev/triggers-eventid"] == "" {
			t.Errorf("EventId is missing")
		}
		gotPipelineResource.ObjectMeta.Labels["tekton.dev/triggers-eventid"] = "12345"

		if diff := cmp.Diff(wantPipelineResource, gotPipelineResource); diff != "" {
			t.Errorf("Diff request body: -want +got: %s", diff)
		}
		return &http.Response{StatusCode: http.StatusCreated, Body: ioutil.NopCloser(bytes.NewReader([]byte{}))}, nil
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
	wantPayload := `EventListener: my-eventlistener in Namespace: foo handling event`

	if !strings.Contains(string(body), "event (EventID: ") {
		t.Errorf("Response body doesn't have eventid")
	}

	if !strings.Contains(string(body), wantPayload) {
		t.Errorf("Diff response body: %s, should have: %s", string(body), wantPayload)
	}

	wg.Wait()
	if numRequests != 1 {
		t.Errorf("Expected 1 request, got %d requests", numRequests)
	}
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
