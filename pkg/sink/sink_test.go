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
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	fakepipelineclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	pipelinetest "github.com/tektoncd/pipeline/test"
	pipelinetb "github.com/tektoncd/pipeline/test/builder"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	faketriggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned/fake"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	fakerestclient "k8s.io/client-go/rest/fake"
	k8stest "k8s.io/client-go/testing"
	apis "knative.dev/pkg/apis"
	rtesting "knative.dev/pkg/reconciler/testing"
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
				numRequests++
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
	pipelineResourceBytes, err := json.Marshal(pipelineResource)
	if err != nil {
		t.Fatalf("Error unmarshalling pipelineResource: %s", err)
	}
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

	restClient.Client = fakerestclient.CreateHTTPClient(func(request *http.Request) (*http.Response, error) {
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
	wantPayload := `EventListener: my-eventlistener in Namespace: foo handling event with payload: {"head_commit": {"id": "testrevision"}, "repository": {"url": "testurl"}}`

	if !strings.Contains(string(body), wantPayload) {
		t.Errorf("Diff response body: %s, should have: %s", string(body), wantPayload)
	}
}

func TestResource_validateEvent(t *testing.T) {
	r := Resource{
		EventListenerName:      "foo-listener",
		EventListenerNamespace: "foo",
	}

	triggerValidate := &triggersv1.TriggerValidate{
		TaskRef: pipelinev1.TaskRef{
			Name: "bar",
		},
		ServiceAccountName: "foo",
	}
	task := pipelinetb.Task("bar", "foo", pipelinetb.TaskSpec(
		pipelinetb.TaskInputs(
			pipelinetb.InputsParamSpec("param", v1alpha1.ParamTypeString, pipelinetb.ParamSpecDescription("mydesc"), pipelinetb.ParamSpecDefault("default")),
			pipelinetb.InputsParamSpec("array-param", v1alpha1.ParamTypeString, pipelinetb.ParamSpecDescription("desc"), pipelinetb.ParamSpecDefault("array", "values")))))

	h := http.Header{}
	h.Set("X-Hub-Signature", "1234567")
	payload := []byte("test payload")

	tests := []struct {
		name    string
		pClient *fakepipelineclientset.Clientset
		wantErr bool
	}{
		{
			name: "Test_SecureEndpoint",
			pClient: func() *fakepipelineclientset.Clientset {
				// We put the succeeded taskrun in the pipeline
				tr := pipelinetb.TaskRun("bar-random", "foo", pipelinetb.TaskRunStatus(pipelinetb.StatusCondition(
					apis.Condition{
						Type:   apis.ConditionSucceeded,
						Status: corev1.ConditionTrue,
					},
				)))
				ctx, _ := rtesting.SetupFakeContext(t)
				clients, _ := pipelinetest.SeedTestData(t, ctx, pipelinetest.Data{
					Tasks:    []*pipelinev1.Task{task},
					TaskRuns: []*pipelinev1.TaskRun{tr},
				})
				pClient := clients.Pipeline
				// We add a prependreactor which just return the value and doesn't insert
				pClient.PrependReactor("create", "taskruns",
					func(action k8stest.Action) (bool, runtime.Object, error) {
						create := action.(k8stest.CreateActionImpl)
						obj := create.GetObject().(*v1alpha1.TaskRun)
						obj.Name = "bar-random"
						return true, obj, nil
					})
				return pClient
			}(),
		},
		{
			name: "Test_SecureEndpoint_ValidationFailure",
			pClient: func() *fakepipelineclientset.Clientset {
				// We put the succeeded taskrun in the pipeline
				tr := pipelinetb.TaskRun("bar-random", "foo", pipelinetb.TaskRunStatus(pipelinetb.StatusCondition(
					apis.Condition{
						Type:   apis.ConditionSucceeded,
						Status: corev1.ConditionFalse,
					},
				)))
				ctx, _ := rtesting.SetupFakeContext(t)
				clients, _ := pipelinetest.SeedTestData(t, ctx, pipelinetest.Data{
					Tasks:    []*pipelinev1.Task{task},
					TaskRuns: []*pipelinev1.TaskRun{tr},
				})
				pClient := clients.Pipeline
				// We add a prependreactor which just return the value and doesn't insert
				pClient.PrependReactor("create", "taskruns",
					func(action k8stest.Action) (bool, runtime.Object, error) {
						create := action.(k8stest.CreateActionImpl)
						obj := create.GetObject().(*v1alpha1.TaskRun)
						obj.Name = "bar-random"
						return true, obj, nil
					})
				return pClient
			}(),
			wantErr: true,
		},
		{
			name: "Test_SecureEndpoint_TaskrunCreateFailure",
			pClient: func() *fakepipelineclientset.Clientset {
				ctx, _ := rtesting.SetupFakeContext(t)
				clients, _ := pipelinetest.SeedTestData(t, ctx, pipelinetest.Data{
					Tasks: []*pipelinev1.Task{task},
				})
				pClient := clients.Pipeline
				pClient.PrependReactor("create", "taskruns",
					func(action k8stest.Action) (bool, runtime.Object, error) {
						return true, nil, errors.New("mock create taskrun error")
					})
				return pClient
			}(),
			wantErr: true,
		},
		{
			name: "Test_SecureEndpoint_TaskrunGetFailure",
			pClient: func() *fakepipelineclientset.Clientset {
				ctx, _ := rtesting.SetupFakeContext(t)
				clients, _ := pipelinetest.SeedTestData(t, ctx, pipelinetest.Data{
					Tasks: []*pipelinev1.Task{task},
				})
				pClient := clients.Pipeline
				pClient.PrependReactor("get", "taskruns",
					func(action k8stest.Action) (bool, runtime.Object, error) {
						return true, nil, errors.New("mock create taskrun error")
					})
				return pClient
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r.PipelineClient = tt.pClient
			if err := r.validateEvent(triggerValidate, h, payload); (err != nil) != tt.wantErr {
				t.Errorf("Resource.secureEndpoint() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestResource_createValidateTask(t *testing.T) {
	r := Resource{
		EventListenerName:      "foo-listener",
		EventListenerNamespace: "foo",
	}

	h := http.Header{}
	h.Set("X-Hub-Signature", "1234567")
	h["X-Hub-Array"] = []string{"1234567", "abcde"}
	hEncode, err := json.Marshal(h)
	if err != nil {
		t.Errorf("Resource.createValidateTask() Unexpected failure to marshal header: %+v", h)
	}

	payload := []byte("test payload")
	triggerValidate := &triggersv1.TriggerValidate{
		TaskRef: pipelinev1.TaskRef{
			Name: "bar",
		},
		ServiceAccountName: "foo",
		Params: []pipelinev1.Param{{
			Name: "Secret",
			Value: pipelinev1.ArrayOrString{
				Type:      pipelinev1.ParamTypeString,
				StringVal: "github-secret",
			},
		}},
	}

	tests := []struct {
		name    string
		pClient *fakepipelineclientset.Clientset
		want    *pipelinev1.TaskRun
		wantErr bool
	}{
		{
			name: "Test_createValidateTask",
			pClient: func() *fakepipelineclientset.Clientset {
				ctx, _ := rtesting.SetupFakeContext(t)
				clients, _ := pipelinetest.SeedTestData(t, ctx, pipelinetest.Data{
					Tasks: []*pipelinev1.Task{pipelinetb.Task("bar", "foo", pipelinetb.TaskSpec(
						pipelinetb.TaskInputs(
							pipelinetb.InputsParamSpec("Secret", v1alpha1.ParamTypeString, pipelinetb.ParamSpecDescription("mydesc")),
							pipelinetb.InputsParamSpec("EventBody", v1alpha1.ParamTypeString, pipelinetb.ParamSpecDescription("mydesc")),
							pipelinetb.InputsParamSpec("EventHeaders", v1alpha1.ParamTypeArray, pipelinetb.ParamSpecDescription("desc")))))},
				})
				return clients.Pipeline
			}(),
			want: &pipelinev1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "bar",
					Namespace:    "foo",
					Labels: map[string]string{
						"tekton.dev/eventlistener": "foo-listener"},
				},
				Spec: pipelinev1.TaskRunSpec{
					ServiceAccount: "foo",
					TaskRef:        &triggerValidate.TaskRef,
					Inputs: pipelinev1.TaskRunInputs{
						Params: []pipelinev1.Param{
							{
								Name: "Secret",
								Value: pipelinev1.ArrayOrString{
									Type:      v1alpha1.ParamTypeString,
									StringVal: "github-secret"}},
							{
								Name: "EventBody",
								Value: pipelinev1.ArrayOrString{
									Type:      v1alpha1.ParamTypeString,
									StringVal: string(payload)},
							},
							{
								Name: "EventHeaders",
								Value: pipelinev1.ArrayOrString{
									Type:      v1alpha1.ParamTypeString,
									StringVal: string(hEncode)},
							},
						}}},
			},
			wantErr: false,
		},
		{
			name:    "Test_createValidateTaskNotFound",
			pClient: fakepipelineclientset.NewSimpleClientset(),
			want:    nil,
			wantErr: true,
		},
		{
			name: "Test_createValidateTaskInputsNotFound",
			pClient: func() *fakepipelineclientset.Clientset {
				ctx, _ := rtesting.SetupFakeContext(t)
				clients, _ := pipelinetest.SeedTestData(t, ctx, pipelinetest.Data{
					Tasks: []*pipelinev1.Task{
						pipelinetb.Task("bar", "foo",
							pipelinetb.TaskSpec())},
				})
				return clients.Pipeline
			}(),
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r.PipelineClient = tt.pClient
			got, err := r.createValidateTask(triggerValidate, h, payload)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("Resource.createValidateTask() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("Resource.createValidateTask() = %v, want %v", got, tt.want)
			}
		})
	}
}
