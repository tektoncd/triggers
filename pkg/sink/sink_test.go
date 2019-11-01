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
	"net/url"
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

const resourceLabel = triggersv1.GroupName + triggersv1.EventListenerLabelKey
const eventIDLabel = triggersv1.GroupName + triggersv1.EventIDLabelKey

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
		{
			GroupVersion: "v1",
			APIResources: []metav1.APIResource{
				{
					Name:       "pods",
					Namespaced: true,
					Kind:       "Pod",
				},
				{
					Name:       "namespaces",
					Namespaced: false,
					Kind:       "Namespace",
				},
			},
		},
		{
			GroupVersion: "tekton.dev/v1alpha1",
			APIResources: []metav1.APIResource{
				{
					Name:       "triggertemplates",
					Namespaced: true,
					Kind:       "TriggerTemplate",
				},
				{
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
			Name:   "my-pipelineresource",
			Labels: map[string]string{"woriginal-label-1": "label-1", resourceLabel: "foo-el", eventIDLabel: "12345"},
		},
		Spec: pipelinev1.PipelineResourceSpec{},
	}

	pr2 := pr1
	pr2.Namespace = "foo"
	pr2.Labels = map[string]string{resourceLabel: "bar-el", eventIDLabel: "54321"}

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
			Name:   "tekton-pipelines",
			Labels: map[string]string{resourceLabel: "test-el", eventIDLabel: "12321"},
		},
	}
	namespaceBytes, err := json.Marshal(namespace)
	if err != nil {
		t.Fatalf("Error marshalling Namespace: %s", err)
	}
	kubeClient := fakekubeclientset.NewSimpleClientset()
	kubeClient.Resources = []*metav1.APIResourceList{
		{
			GroupVersion: "tekton.dev/v1alpha1",
			APIResources: []metav1.APIResource{
				{
					Name:       "pipelineresources",
					Kind:       "PipelineResource",
					Namespaced: true,
				},
			},
		},
		{
			GroupVersion: "v1",
			APIResources: []metav1.APIResource{
				{
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
		wantResource           json.RawMessage
		eventListenerNamespace string
		eventListenerName      string
		eventID                string
		wantURLPath            string
	}{
		{
			name:                   "PipelineResource without namespace",
			resource:               json.RawMessage(pr1Bytes),
			wantResource:           json.RawMessage(pr1WantBytes),
			eventListenerNamespace: "bar",
			eventListenerName:      "foo-el",
			eventID:                "12345",
			wantURLPath:            "/apis/tekton.dev/v1alpha1/namespaces/bar/pipelineresources",
		},
		{
			name:                   "PipelineResource with namespace",
			resource:               json.RawMessage(pr2Bytes),
			wantResource:           json.RawMessage(pr2Bytes),
			eventListenerNamespace: "bar",
			eventListenerName:      "bar-el",
			eventID:                "54321",
			wantURLPath:            "/apis/tekton.dev/v1alpha1/namespaces/foo/pipelineresources",
		},
		{
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

			// Run test
			err := createResource(tt.resource, restClient, kubeClient.Discovery(), tt.eventListenerNamespace, tt.eventListenerName, tt.eventID, logger)
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
	wantPipelineResource := pipelinev1.PipelineResource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tekton.dev/v1alpha1",
			Kind:       "PipelineResource",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-pipelineresource",
			Namespace: namespace,
			Labels:    map[string]string{"app": "foo", "tekton.dev/eventlistener": "my-eventlistener", eventIDLabel: "12345"},
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
	pipelineResourceBytes, err := json.Marshal(pipelineResource)
	if err != nil {
		t.Fatalf("Error unmarshalling pipelineResource: %s", err)
	}
	wantPipelineResourceURLPath := "/apis/tekton.dev/v1alpha1/namespaces/foo/pipelineresources"

	tt := bldr.TriggerTemplate("my-triggertemplate", namespace,
		bldr.TriggerTemplateSpec(
			bldr.TriggerTemplateParam("url", "", ""),
			bldr.TriggerTemplateParam("revision", "", ""),
			bldr.TriggerTemplateParam("appLabel", "", ""),
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
		bldr.EventListenerSpec(
			bldr.EventListenerTrigger("my-triggerbinding", "my-triggertemplate", "v1alpha1",
				bldr.EventListenerTriggerParam("appLabel", "foo")),
		))

	kubeClient := fakekubeclientset.NewSimpleClientset()
	kubeClient.Resources = []*metav1.APIResourceList{
		{
			GroupVersion: "tekton.dev/v1alpha1",
			APIResources: []metav1.APIResource{
				{
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
	logger, _ := logging.NewLogger("", "")
	r := Resource{
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

func TestResource_processEvent(t *testing.T) {
	logger, _ := logging.NewLogger("", "")
	r := Resource{
		HTTPClient:             http.DefaultClient,
		EventListenerName:      "foo-listener",
		EventListenerNamespace: "foo",
		Logger:                 logger,
	}

	payload, _ := json.Marshal(map[string]string{
		"eventType": "push",
		"foo":       "bar",
	})

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cmp.Diff(r.Header["Param-Header"], []string{"val"}) != "" {
			http.Error(w, "Expected header does not match", http.StatusBadRequest)
			return
		}
		_, _ = w.Write(payload)
	}))
	defer ts.Close()

	req, err := http.NewRequest(http.MethodPost, "http://some-url/", nil)
	if err != nil {
		t.Fatalf("Error trying to create request: %q", err)
	}

	interceptorURL, _ := url.Parse(ts.URL)
	params := []pipelinev1.Param{
		{
			Name: "Param-Header",
			Value: pipelinev1.ArrayOrString{
				Type:      pipelinev1.ParamTypeString,
				StringVal: "val",
			},
		},
	}
	requestHeader := make(http.Header, len(req.Header))
	for k, v := range req.Header {
		v2 := make([]string, len(v))
		copy(v2, v)
		requestHeader[k] = v2
	}
	resPayload, err := r.processEvent(interceptorURL, req, payload, params)

	if err != nil {
		t.Errorf("Unexpected error in process event: %q", err)
	}

	if diff := cmp.Diff(payload, resPayload); diff != "" {
		t.Errorf("Did not get expected payload back: %s", diff)
	}

	// Verify that the parameter header was not added to the request header
	if diff := cmp.Diff(req.Header, requestHeader); diff != "" {
		t.Errorf("processEvent() changed request header unexpectedly: %s", diff)
	}
}

func Test_addInterceptorHeaders(t *testing.T) {
	type args struct {
		header       http.Header
		headerParams []pipelinev1.Param
	}
	tests := []struct {
		name string
		args args
		want http.Header
	}{
		{
			name: "Empty params",
			args: args{
				header: map[string][]string{
					"header1": {"val"},
				},
				headerParams: []pipelinev1.Param{},
			},
			want: map[string][]string{
				"header1": {"val"},
			},
		},
		{
			name: "One string param",
			args: args{
				header: map[string][]string{
					"header1": {"val"},
				},
				headerParams: []pipelinev1.Param{
					{
						Name: "header2",
						Value: pipelinev1.ArrayOrString{
							Type:      pipelinev1.ParamTypeString,
							StringVal: "val",
						},
					},
				},
			},
			want: map[string][]string{
				"header1": {"val"},
				"header2": {"val"},
			},
		},
		{
			name: "One array param",
			args: args{
				header: map[string][]string{
					"header1": {"val"},
				},
				headerParams: []pipelinev1.Param{
					{
						Name: "header2",
						Value: pipelinev1.ArrayOrString{
							Type:     pipelinev1.ParamTypeArray,
							ArrayVal: []string{"val1", "val2"},
						},
					},
				},
			},
			want: map[string][]string{
				"header1": {"val"},
				"header2": {"val1", "val2"},
			},
		},
		{
			name: "Clobber param",
			args: args{
				header: map[string][]string{
					"header1": {"val"},
				},
				headerParams: []pipelinev1.Param{
					{
						Name: "header1",
						Value: pipelinev1.ArrayOrString{
							Type:     pipelinev1.ParamTypeArray,
							ArrayVal: []string{"new_val"},
						},
					},
				},
			},
			want: map[string][]string{
				"header1": {"new_val"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addInterceptorHeaders(tt.args.header, tt.args.headerParams)
			if diff := cmp.Diff(tt.want, tt.args.header); diff != "" {
				t.Errorf("addInterceptorHeaders() Diff: -want +got: %s", diff)
			}
		})
	}
}
