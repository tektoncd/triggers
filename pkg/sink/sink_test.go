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
	"testing"

	"github.com/google/go-cmp/cmp"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	faketriggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned/fake"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	fakerestclient "k8s.io/client-go/rest/fake"
)

func Test_createRequestURI(t *testing.T) {
	tests := []struct {
		apiVersion string
		namePlural string
		namespace  string
		namespaced bool
		want       string
	}{
		{
			apiVersion: "tekton.dev/v1alpha1",
			namePlural: "pipelineruns",
			namespace:  "foo",
			namespaced: true,
			want:       "apis/tekton.dev/v1alpha1/namespaces/foo/pipelineruns",
		},
		{
			apiVersion: "v1",
			namePlural: "secrets",
			namespace:  "foo",
			namespaced: true,
			want:       "api/v1/namespaces/foo/secrets",
		},
		{
			apiVersion: "v1",
			namePlural: "namespaces",
			namespace:  "",
			namespaced: false,
			want:       "api/v1/namespaces",
		},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := createRequestURI(tt.apiVersion, tt.namePlural, tt.namespace, tt.namespaced)
			if got != tt.want {
				t.Errorf("createRequestURI() = %s, want = %s", got, tt.want)
			}
		})
	}
}

func Test_findAPIResource(t *testing.T) {
	// Create fake kubeclient with list of resources
	kubeClient := fakekubeclientset.NewSimpleClientset()
	kubeClient.Resources = []*metav1.APIResourceList{
		&metav1.APIResourceList{
			GroupVersion: "v1",
			APIResources: []metav1.APIResource{
				metav1.APIResource{
					Name:       "pods",
					Namespaced: true,
					Kind:       "Pod",
				},
				metav1.APIResource{
					Name:       "namespaces",
					Namespaced: false,
					Kind:       "Namespace",
				},
			},
		},
		&metav1.APIResourceList{
			GroupVersion: "tekton.dev/v1alpha1",
			APIResources: []metav1.APIResource{
				metav1.APIResource{
					Name:       "triggertemplates",
					Namespaced: true,
					Kind:       "TriggerTemplate",
				},
				metav1.APIResource{
					Name:       "pipelineruns",
					Namespaced: true,
					Kind:       "PipelineRun",
				},
			},
		},
	}
	tests := []struct {
		apiVersion string
		kind       string
		want       *metav1.APIResource
	}{
		{
			apiVersion: "v1",
			kind:       "Pod",
			want: &metav1.APIResource{
				Name:       "pods",
				Namespaced: true,
				Kind:       "Pod",
			},
		},
		{
			apiVersion: "v1",
			kind:       "Namespace",
			want: &metav1.APIResource{
				Name:       "namespaces",
				Namespaced: false,
				Kind:       "Namespace",
			},
		},
		{
			apiVersion: "tekton.dev/v1alpha1",
			kind:       "TriggerTemplate",
			want: &metav1.APIResource{
				Name:       "triggertemplates",
				Namespaced: true,
				Kind:       "TriggerTemplate",
			},
		},
		{
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

func Test_findAPIResource_error(t *testing.T) {
	kubeClient := fakekubeclientset.NewSimpleClientset()
	_, err := findAPIResource(kubeClient.Discovery(), "v1", "Pod")
	if err == nil {
		t.Error("findAPIResource() did not return error when expected")
	}
}

func Test_createResource(t *testing.T) {
	pr1 := pipelinev1.PipelineResource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tekton.dev/v1alpha1",
			Kind:       "PipelineResource",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-pipelineresource",
		},
		Spec: pipelinev1.PipelineResourceSpec{},
	}
	pr2 := pr1
	pr2.Namespace = "foo"

	pr1Bytes, err := json.Marshal(pr1)
	if err != nil {
		t.Fatalf("Error marshalling PipelineResource: %s", err)
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
		},
	}
	namespaceBytes, err := json.Marshal(namespace)
	if err != nil {
		t.Fatalf("Error marshalling Namespace: %s", err)
	}
	kubeClient := fakekubeclientset.NewSimpleClientset()
	kubeClient.Resources = []*metav1.APIResourceList{
		&metav1.APIResourceList{
			GroupVersion: "tekton.dev/v1alpha1",
			APIResources: []metav1.APIResource{
				metav1.APIResource{
					Name:       "pipelineresources",
					Kind:       "PipelineResource",
					Namespaced: true,
				},
			},
		},
		&metav1.APIResourceList{
			GroupVersion: "v1",
			APIResources: []metav1.APIResource{
				metav1.APIResource{
					Name:       "namespaces",
					Kind:       "Namespace",
					Namespaced: false,
				},
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
	tests := []struct {
		name                   string
		resource               json.RawMessage
		eventListenerNamespace string
		wantURLPath            string
	}{
		{
			name:                   "PipelineResource without namespace",
			resource:               json.RawMessage(pr1Bytes),
			eventListenerNamespace: "bar",
			wantURLPath:            "/apis/tekton.dev/v1alpha1/namespaces/bar/pipelineresources",
		},
		{
			name:                   "PipelineResource with namespace",
			resource:               json.RawMessage(pr2Bytes),
			eventListenerNamespace: "bar",
			wantURLPath:            "/apis/tekton.dev/v1alpha1/namespaces/foo/pipelineresources",
		},
		{
			name:                   "Namespace",
			resource:               json.RawMessage(namespaceBytes),
			eventListenerNamespace: "",
			wantURLPath:            "/api/v1/namespaces",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup fake client
			numRequests := 0
			restClient.Client = fakerestclient.CreateHTTPClient(func(request *http.Request) (*http.Response, error) {
				numRequests += 1
				if diff := cmp.Diff(tt.wantURLPath, request.URL.Path); diff != "" {
					t.Errorf("Diff request uri: -want +got: %s", diff)
				}
				body, err := ioutil.ReadAll(request.Body)
				if err != nil {
					return nil, err
				}
				if diff := cmp.Diff(string(tt.resource), string(body)); diff != "" {
					t.Errorf("Diff request body: -want +got: %s", diff)
				}
				return &http.Response{StatusCode: http.StatusCreated, Body: ioutil.NopCloser(bytes.NewReader([]byte{}))}, nil
			})
			// Run test
			err := createResource(tt.resource, restClient, kubeClient.Discovery(), tt.eventListenerNamespace)
			if err != nil {
				t.Errorf("createResource() returned error: %s", err)
			}
			if numRequests != 1 {
				t.Errorf("Expected 1 request, got %d requests", numRequests)
			}
		})
	}
}

func Test_HandleEvent(t *testing.T) {
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
		},
		Spec: pipelinev1.PipelineResourceSpec{
			Type: pipelinev1.PipelineResourceTypeGit,
			Params: []pipelinev1.ResourceParam{
				pipelinev1.ResourceParam{Name: "url", Value: "$(params.url)"},
				pipelinev1.ResourceParam{Name: "revision", Value: "$(params.revision)"},
			},
		},
	}
	wantPipelineResource := pipelinev1.PipelineResource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tekton.dev/v1alpha1",
			Kind:       "PipelineResource",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-pipelineresource",
			Namespace: namespace,
		},
		Spec: pipelinev1.PipelineResourceSpec{
			Type: pipelinev1.PipelineResourceTypeGit,
			Params: []pipelinev1.ResourceParam{
				pipelinev1.ResourceParam{Name: "url", Value: "testurl"},
				pipelinev1.ResourceParam{Name: "revision", Value: "testrevision"},
			},
		},
	}
	pipelineResourceBytes, err := json.Marshal(pipelineResource)
	if err != nil {
		t.Fatalf("Error unmarshalling pipelineResource: %s", err)
	}
	wantPipelineResourceBytes, err := json.Marshal(wantPipelineResource)
	if err != nil {
		t.Fatalf("Error unmarshalling wantPipelineResource: %s", err)
	}
	wantPipelineResourceURLPath := "/apis/tekton.dev/v1alpha1/namespaces/foo/pipelineresources"
	tt := &triggersv1.TriggerTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-triggertemplate",
		},
		Spec: triggersv1.TriggerTemplateSpec{
			Params: []pipelinev1.ParamSpec{
				pipelinev1.ParamSpec{Name: "url"},
				pipelinev1.ParamSpec{Name: "revision"},
			},
			ResourceTemplates: []triggersv1.TriggerResourceTemplate{
				triggersv1.TriggerResourceTemplate{RawMessage: json.RawMessage(pipelineResourceBytes)},
			},
		},
	}
	tb := &triggersv1.TriggerBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-triggerbinding",
		},
		Spec: triggersv1.TriggerBindingSpec{
			Params: []pipelinev1.Param{
				pipelinev1.Param{
					Name:  "url",
					Value: pipelinev1.ArrayOrString{StringVal: "$(event.repository.url)", Type: pipelinev1.ParamTypeString},
				},
				pipelinev1.Param{
					Name:  "revision",
					Value: pipelinev1.ArrayOrString{StringVal: "$(event.head_commit.id)", Type: pipelinev1.ParamTypeString},
				},
			},
		},
	}
	el := &triggersv1.EventListener{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-eventlistener",
		},
		Spec: triggersv1.EventListenerSpec{
			Triggers: []triggersv1.Trigger{
				triggersv1.Trigger{
					TriggerBinding:  triggersv1.TriggerBindingRef{Name: "my-triggerbinding"},
					TriggerTemplate: triggersv1.TriggerTemplateRef{Name: "my-triggertemplate"},
				},
			},
		},
	}

	kubeClient := fakekubeclientset.NewSimpleClientset()
	kubeClient.Resources = []*metav1.APIResourceList{
		&metav1.APIResourceList{
			GroupVersion: "tekton.dev/v1alpha1",
			APIResources: []metav1.APIResource{
				metav1.APIResource{
					Name:       "pipelineresources",
					Kind:       "PipelineResource",
					Namespaced: true,
				},
			},
		},
	}
	triggersClient := faketriggersclientset.NewSimpleClientset()
	if _, err := triggersClient.TektonV1alpha1().TriggerTemplates(namespace).Create(tt); err != nil {
		t.Fatalf("Error creating TriggerTemplate: %s", err)
	}
	if _, err := triggersClient.TektonV1alpha1().TriggerBindings(namespace).Create(tb); err != nil {
		t.Fatalf("Error creating TriggerBinding: %s", err)
	}
	if _, err := triggersClient.TektonV1alpha1().EventListeners(namespace).Create(el); err != nil {
		t.Fatalf("Error creating EventListener: %s", err)
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
	r := Resource{
		EventListenerName:      el.Name,
		EventListenerNamespace: namespace,
		RESTClient:             restClient,
		DiscoveryClient:        kubeClient.Discovery(),
		TriggersClient:         triggersClient,
	}
	ts := httptest.NewServer(http.HandlerFunc(r.HandleEvent))
	defer ts.Close()

	numRequests := 0
	restClient.Client = fakerestclient.CreateHTTPClient(func(request *http.Request) (*http.Response, error) {
		numRequests += 1
		if diff := cmp.Diff(wantPipelineResourceURLPath, request.URL.Path); diff != "" {
			t.Errorf("Diff request uri: -want +got: %s", diff)
		}
		body, err := ioutil.ReadAll(request.Body)
		if err != nil {
			return nil, err
		}
		if diff := cmp.Diff(string(wantPipelineResourceBytes), string(body)); diff != "" {
			t.Errorf("Diff request body: -want +got: %s", diff)
		}
		return &http.Response{StatusCode: http.StatusCreated, Body: ioutil.NopCloser(bytes.NewReader([]byte{}))}, nil
	})

	_, err = http.Post(ts.URL, "application/json", bytes.NewReader(eventBody))
	if err != nil {
		t.Fatalf("Error creating Post request: %s", err)
	}

	if numRequests != 1 {
		t.Errorf("Expected 1 request, got %d requests", numRequests)
	}
}
