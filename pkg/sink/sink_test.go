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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	cloudeventstest "github.com/cloudevents/sdk-go/v2/client/test"
	"github.com/cloudevents/sdk-go/v2/protocol"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/gorilla/mux"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	triggersv1alpha1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	triggersv1beta1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	dynamicclientset "github.com/tektoncd/triggers/pkg/client/dynamic/clientset"
	"github.com/tektoncd/triggers/pkg/client/dynamic/clientset/tekton"
	clusterinterceptorinformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1alpha1/clusterinterceptor"
	interceptorinformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1alpha1/interceptor"
	clustertriggerbindinginformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1beta1/clustertriggerbinding"
	eventlistenerinformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1beta1/eventlistener"
	triggerinformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1beta1/trigger"
	triggerbindinginformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1beta1/triggerbinding"
	triggertemplateinformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1beta1/triggertemplate"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"github.com/tektoncd/triggers/pkg/interceptors/server"
	"github.com/tektoncd/triggers/pkg/template"
	"github.com/tektoncd/triggers/test"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	ktesting "k8s.io/client-go/testing"
	"knative.dev/pkg/apis"
	fakekubeclient "knative.dev/pkg/client/injection/kube/client/fake"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/ptr"
)

const (
	eventID   = "12345"
	namespace = "foo"
	elUID     = "el-uid"
)

func init() {
	// Override UID generator for consistent test results.
	template.UUID = func() string { return eventID }
}

var (
	github = &triggersv1alpha1.ClusterInterceptor{
		ObjectMeta: metav1.ObjectMeta{
			Name: "github",
		},
		Spec: triggersv1alpha1.ClusterInterceptorSpec{
			ClientConfig: triggersv1alpha1.ClientConfig{
				URL: &apis.URL{
					Scheme: "http",
					Host:   "tekton-triggers-core-interceptors",
					Path:   "/github",
				},
			},
		},
	}
	cel = &triggersv1alpha1.ClusterInterceptor{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cel",
		},
		Spec: triggersv1alpha1.ClusterInterceptorSpec{
			ClientConfig: triggersv1alpha1.ClientConfig{
				URL: &apis.URL{
					Scheme: "http",
					Host:   "tekton-triggers-core-interceptors",
					Path:   "/cel",
				},
			},
		},
	}
	bitbucket = &triggersv1alpha1.ClusterInterceptor{
		ObjectMeta: metav1.ObjectMeta{
			Name: "bitbucket",
		},
		Spec: triggersv1alpha1.ClusterInterceptorSpec{
			ClientConfig: triggersv1alpha1.ClientConfig{
				URL: &apis.URL{
					Scheme: "http",
					Host:   "tekton-triggers-core-interceptors",
					Path:   "/bitbucket",
				},
			},
		},
	}
	nsInterceptor = &triggersv1alpha1.Interceptor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "bitbucket",
			Namespace: namespace,
		},
		Spec: triggersv1alpha1.InterceptorSpec{
			ClientConfig: triggersv1alpha1.ClientConfig{
				URL: &apis.URL{
					Scheme: "http",
					Host:   "tekton-triggers-core-interceptors",
					Path:   "/bitbucket",
				},
			},
		},
	}
)

// getSinkAssets seeds test resources and returns a testable Sink and a dynamic client. The returned client is used to
// create the fake resources and can be used to check if the correct resources were created.
func getSinkAssets(t *testing.T, res test.Resources, elName string, webhookInterceptor http.Handler) (Sink, *fakedynamic.FakeDynamicClient) {
	t.Helper()
	ctx, _ := test.SetupFakeContext(t)
	clients := test.SeedResources(t, ctx, res)

	logger := zaptest.NewLogger(t)

	dynamicClient := fakedynamic.NewSimpleDynamicClient(runtime.NewScheme())
	dynamicSet := dynamicclientset.New(tekton.WithClient(dynamicClient))

	// Setup a handler for core interceptors using httptest
	httpClient := setupInterceptors(t, clients.Kube, logger.Sugar(), webhookInterceptor)

	ceClient, _ := cloudeventstest.NewMockSenderClient(t, 1)

	recorder, _ := NewRecorder()
	r := Sink{
		EventListenerName:           elName,
		EventListenerNamespace:      namespace,
		DynamicClient:               dynamicSet,
		DiscoveryClient:             clients.Kube.Discovery(),
		KubeClientSet:               clients.Kube,
		TriggersClient:              clients.Triggers,
		HTTPClient:                  httpClient,
		CEClient:                    ceClient,
		Logger:                      logger.Sugar(),
		Auth:                        DefaultAuthOverride{},
		WGProcessTriggers:           &sync.WaitGroup{},
		EventRecorder:               controller.GetEventRecorder(ctx),
		Recorder:                    recorder,
		EventListenerLister:         eventlistenerinformer.Get(ctx).Lister(),
		TriggerLister:               triggerinformer.Get(ctx).Lister(),
		TriggerBindingLister:        triggerbindinginformer.Get(ctx).Lister(),
		ClusterTriggerBindingLister: clustertriggerbindinginformer.Get(ctx).Lister(),
		TriggerTemplateLister:       triggertemplateinformer.Get(ctx).Lister(),
		ClusterInterceptorLister:    clusterinterceptorinformer.Get(ctx).Lister(),
		InterceptorLister:           interceptorinformer.Get(ctx).Lister(),
		PayloadValidation:           true,
	}
	return r, dynamicClient
}

// setupInterceptors creates a httptest server with all coreInterceptors and any passed in webhook interceptor
// It returns a http.Client that can be used to talk to these interceptors
func setupInterceptors(t *testing.T, k kubernetes.Interface, l *zap.SugaredLogger, webhookInterceptor http.Handler) *http.Client {
	t.Helper()
	// Setup a handler for core interceptors using httptest
	coreInterceptors, err := server.NewWithCoreInterceptors(interceptors.DefaultSecretGetter(k.CoreV1()), l)
	if err != nil {
		t.Fatalf("failed to initialize core interceptors: %v", err)
	}
	rtr := mux.NewRouter()
	// server core interceptors by matching on req host
	rtr.MatcherFunc(func(r *http.Request, _ *mux.RouteMatch) bool {
		return strings.Contains(r.Host, interceptors.CoreInterceptorsHost)
	}).Handler(coreInterceptors)

	if webhookInterceptor != nil {
		rtr.Handle("/", webhookInterceptor)
	}
	srv := httptest.NewServer(rtr)
	t.Cleanup(func() {
		srv.Close()
	})
	httpClient := srv.Client()
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("testServer() url parse err: %v", err)
	}
	httpClient.Transport = &http.Transport{
		Proxy: http.ProxyURL(u),
	}
	return httpClient
}

// toTaskRun returns the task run that were created from the given actions
func toTaskRun(t *testing.T, actions []ktesting.Action) []pipelinev1.TaskRun {
	t.Helper()
	trs := []pipelinev1.TaskRun{}
	for i := range actions {
		obj := actions[i].(ktesting.CreateAction).GetObject()
		// Since we use dynamic client, we cannot directly get the concrete type
		uns := obj.(*unstructured.Unstructured).Object
		tr := pipelinev1.TaskRun{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(uns, &tr); err != nil {
			t.Errorf("failed to get created pipeline resource: %v", err)
		}
		trs = append(trs, tr)
	}
	return trs
}

// checkSinkResponse checks that the sink response status code is 202 and that
// the body returns the EventListener, namespace, and eventID.
func checkSinkResponse(t *testing.T, resp *http.Response, elName string) {
	t.Helper()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected response code 202 but got: %v", resp.Status)
	}
	var gotBody Response
	if err := json.NewDecoder(resp.Body).Decode(&gotBody); err != nil {
		t.Fatalf("Error reading response body: %s", err)
	}
	wantBody := Response{
		EventListener:    elName,
		EventListenerUID: elUID,
		Namespace:        namespace,
		EventID:          eventID,
	}
	if diff := cmp.Diff(wantBody, gotBody); diff != "" {
		t.Errorf("did not get expected response back -want,+got: %s", diff)
	}
}

// trResourceTemplate returns a resourceTemplate for a git-clone taskRun
func trResourceTemplate(t testing.TB) runtime.RawExtension {
	return test.RawExtension(t, pipelinev1.TaskRun{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tekton.dev/v1beta1",
			Kind:       "TaskRun",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "$(tt.params.name)",
			Namespace: namespace,
			Labels: map[string]string{
				"app":  "$(tt.params.app)",
				"type": "$(tt.params.type)",
			},
		},
		Spec: pipelinev1.TaskRunSpec{
			Params: []pipelinev1.Param{{
				Name: "url",
				Value: pipelinev1.ArrayOrString{
					Type:      pipelinev1.ParamTypeString,
					StringVal: "$(tt.params.url)",
				},
			}, {
				Name: "git-revision",
				Value: pipelinev1.ArrayOrString{
					Type:      pipelinev1.ParamTypeString,
					StringVal: "$(tt.params.revision)",
				},
			}},
			TaskRef: &pipelinev1.TaskRef{
				Name: "git-clone",
			},
		},
	})
}

func TestHandleEvent(t *testing.T) {
	var (
		eventListenerName = "my-el"
		eventBody         = json.RawMessage(`{"head_commit": {"id": "testrevision"}, "repository": {"url": "testurl"}, "foo": "bar\t\r\nbaz昨"}`)
		gitCloneTT        = &triggersv1beta1.TriggerTemplate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "git-clone",
				Namespace: namespace,
			},
			Spec: *makeGitCloneTTSpec(t, "git-clone-test-run"),
		}
		gitCloneTBSpec = []*triggersv1beta1.TriggerSpecBinding{
			{Name: "url", Value: ptr.String("$(body.repository.url)")},
			{Name: "revision", Value: ptr.String("$(body.head_commit.id)")},
			{Name: "name", Value: ptr.String("git-clone-run")},
			{Name: "app", Value: ptr.String("$(body.foo)")},
			{Name: "type", Value: ptr.String("$(header.Content-Type)")},
		}

		gitCloneTB = &triggersv1beta1.TriggerBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "git-clone",
				Namespace: namespace,
			},
			Spec: triggersv1beta1.TriggerBindingSpec{
				Params: []triggersv1beta1.Param{
					{Name: "url", Value: "$(body.repository.url)"},
					{Name: "revision", Value: "$(body.head_commit.id)"},
					{Name: "name", Value: "git-clone-run"},
					{Name: "app", Value: "$(body.foo)"},
					{Name: "type", Value: "$(header.Content-Type)"},
				},
			},
		}

		// gitCloneTaskRun with values from gitCloneTBSpec
		gitCloneTaskRun = pipelinev1.TaskRun{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "tekton.dev/v1beta1",
				Kind:       "TaskRun",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "git-clone-run",
				Namespace: namespace,
				Labels: map[string]string{
					"app":                                  "bar\t\r\nbaz昨",
					"type":                                 "application/json",
					"triggers.tekton.dev/eventlistener":    eventListenerName,
					"triggers.tekton.dev/trigger":          "git-clone-trigger",
					"triggers.tekton.dev/triggers-eventid": "12345",
				},
			},
			Spec: pipelinev1.TaskRunSpec{
				Params: []pipelinev1.Param{{
					Name:  "url",
					Value: pipelinev1.ArrayOrString{Type: pipelinev1.ParamTypeString, StringVal: "testurl"},
				}, {
					Name:  "git-revision",
					Value: pipelinev1.ArrayOrString{Type: pipelinev1.ParamTypeString, StringVal: "testrevision"},
				}},
				TaskRef: &pipelinev1.TaskRef{Name: "git-clone"},
			},
		}
	)

	// tenGitCloneTriggers is a slice to ten triggers named git-clone-$i
	tenGitCloneTriggers := []*triggersv1beta1.Trigger{}
	for i := 0; i < 10; i++ {
		tenGitCloneTriggers = append(tenGitCloneTriggers, &triggersv1beta1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("git-clone-trigger-%d", i),
				Namespace: namespace,
			},
			Spec: triggersv1beta1.TriggerSpec{
				Bindings: []*triggersv1beta1.TriggerSpecBinding{
					{Name: "url", Value: ptr.String("$(body.repository.url)")},
					{Name: "revision", Value: ptr.String("$(body.head_commit.id)")},
					{Name: "name", Value: ptr.String(fmt.Sprintf("git-clone-run-%d", i))},
					{Name: "app", Value: ptr.String("$(body.foo)")},
					{Name: "type", Value: ptr.String("$(header.Content-Type)")},
				},
				Template: triggersv1beta1.TriggerSpecTemplate{Spec: makeGitCloneTTSpec(t, fmt.Sprintf("git-clone-run-%d", i))},
			},
		})
	}

	tenGitCloneTaskRuns := []pipelinev1.TaskRun{}
	for i := 0; i < 10; i++ {
		tr := gitCloneTaskRun.DeepCopy()
		tr.Name = fmt.Sprintf("git-clone-run-%d", i)
		tr.Labels["triggers.tekton.dev/trigger"] = fmt.Sprintf("git-clone-trigger-%d", i)
		tenGitCloneTaskRuns = append(tenGitCloneTaskRuns, *tr)
	}

	tests := []struct {
		name string
		// resources are the K8s objects to setup the test env.
		resources test.Resources
		// webhookInterceptor is a handler that implements any webhook interceptors referenced in the triggers.
		webhookInterceptor http.HandlerFunc
		eventBody          []byte
		headers            map[string][]string
		// want is the resulting TaskRun(s) created by the EventListener
		want []pipelinev1.TaskRun
	}{{
		name: "single trigger embedded within EventListener",
		resources: test.Resources{
			EventListeners: []*triggersv1beta1.EventListener{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      eventListenerName,
					Namespace: namespace,
					UID:       types.UID(elUID),
				},
				Spec: triggersv1beta1.EventListenerSpec{
					Triggers: []triggersv1beta1.EventListenerTrigger{{
						Name: "git-clone-trigger",
						Bindings: []*triggersv1beta1.EventListenerBinding{{
							Ref:  "git-clone",
							Kind: triggersv1beta1.NamespacedTriggerBindingKind,
						}},
						Template: &triggersv1beta1.EventListenerTemplate{
							Ref: ptr.String("git-clone"),
						},
					}},
				},
			}},
			TriggerBindings:  []*triggersv1beta1.TriggerBinding{gitCloneTB},
			TriggerTemplates: []*triggersv1beta1.TriggerTemplate{gitCloneTT},
		},
		eventBody: eventBody,
		want:      []pipelinev1.TaskRun{gitCloneTaskRun},
	}, {
		name: "namespace selector match names",
		resources: test.Resources{
			Namespaces: []*corev1.Namespace{{
				ObjectMeta: metav1.ObjectMeta{
					Name: "bar",
				},
			}},
			TriggerBindings: []*triggersv1beta1.TriggerBinding{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "git-clone",
					Namespace: "bar",
				},
				Spec: triggersv1beta1.TriggerBindingSpec{
					Params: []triggersv1beta1.Param{
						{Name: "url", Value: "$(body.repository.url)"},
						{Name: "revision", Value: "$(body.head_commit.id)"},
						{Name: "name", Value: "git-clone-run"},
						{Name: "app", Value: "$(body.foo)"},
						{Name: "type", Value: "$(header.Content-Type)"},
					},
				},
			}},
			TriggerTemplates: []*triggersv1beta1.TriggerTemplate{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "git-clone",
					Namespace: "bar",
				},
				Spec: *makeGitCloneTTSpec(t, "git-clone-test-run"),
			}},
			Triggers: []*triggersv1beta1.Trigger{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "git-clone-trigger",
					Namespace: "bar",
				},
				Spec: triggersv1beta1.TriggerSpec{
					Bindings: []*triggersv1beta1.TriggerSpecBinding{{Ref: "git-clone"}},
					Template: triggersv1beta1.TriggerSpecTemplate{Ref: ptr.String("git-clone")},
				},
			}},
			EventListeners: []*triggersv1beta1.EventListener{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      eventListenerName,
					Namespace: namespace,
					UID:       types.UID(elUID),
				},
				Spec: triggersv1beta1.EventListenerSpec{
					NamespaceSelector: triggersv1beta1.NamespaceSelector{
						MatchNames: []string{"bar"},
					},
				},
			}},
		},
		eventBody: eventBody,
		want:      []pipelinev1.TaskRun{gitCloneTaskRun},
	}, {
		name: "label selector match expressions",
		resources: test.Resources{
			Namespaces: []*corev1.Namespace{{
				ObjectMeta: metav1.ObjectMeta{
					Name: "bar",
				},
			}},
			TriggerBindings: []*triggersv1beta1.TriggerBinding{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "git-clone",
					Namespace: "bar",
				},
				Spec: triggersv1beta1.TriggerBindingSpec{
					Params: []triggersv1beta1.Param{
						{Name: "url", Value: "$(body.repository.url)"},
						{Name: "revision", Value: "$(body.head_commit.id)"},
						{Name: "name", Value: "git-clone-run"},
						{Name: "app", Value: "$(body.foo)"},
						{Name: "type", Value: "$(header.Content-Type)"},
					},
				},
			}},
			TriggerTemplates: []*triggersv1beta1.TriggerTemplate{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "git-clone",
					Namespace: "bar",
				},
				Spec: *makeGitCloneTTSpec(t, "git-clone-test-run"),
			}},
			Triggers: []*triggersv1beta1.Trigger{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "git-clone-trigger",
					Namespace: "bar",
					Labels: map[string]string{
						"foo": "bar",
					},
				},
				Spec: triggersv1beta1.TriggerSpec{
					Bindings: []*triggersv1beta1.TriggerSpecBinding{{Ref: "git-clone"}},
					Template: triggersv1beta1.TriggerSpecTemplate{Ref: ptr.String("git-clone")},
				},
			}, {
				ObjectMeta: metav1.ObjectMeta{
					Name:      "second-not-invoked-trigger",
					Namespace: "bar",
					Labels: map[string]string{
						"foo": "notbar",
					},
				},
				Spec: triggersv1beta1.TriggerSpec{
					Bindings: []*triggersv1beta1.TriggerSpecBinding{{Ref: "git-clone"}},
					Template: triggersv1beta1.TriggerSpecTemplate{Ref: ptr.String("git-clone")},
				},
			}},
			EventListeners: []*triggersv1beta1.EventListener{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      eventListenerName,
					Namespace: namespace,
					UID:       types.UID(elUID),
				},
				Spec: triggersv1beta1.EventListenerSpec{
					NamespaceSelector: triggersv1beta1.NamespaceSelector{
						MatchNames: []string{"bar"},
					},
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"foo": "bar"},
					},
				},
			}},
		},
		eventBody: eventBody,
		// only one of the tasks is invoked
		want: []pipelinev1.TaskRun{gitCloneTaskRun},
	}, {
		name: "label selector match expressions without namespace selector",
		resources: test.Resources{
			Namespaces: []*corev1.Namespace{{
				ObjectMeta: metav1.ObjectMeta{
					Name: "bar",
				},
			}},
			TriggerBindings: []*triggersv1beta1.TriggerBinding{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "git-clone",
					Namespace: namespace,
				},
				Spec: triggersv1beta1.TriggerBindingSpec{
					Params: []triggersv1beta1.Param{
						{Name: "url", Value: "$(body.repository.url)"},
						{Name: "revision", Value: "$(body.head_commit.id)"},
						{Name: "name", Value: "git-clone-run"},
						{Name: "app", Value: "$(body.foo)"},
						{Name: "type", Value: "$(header.Content-Type)"},
					},
				},
			}},
			TriggerTemplates: []*triggersv1beta1.TriggerTemplate{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "git-clone",
					Namespace: namespace,
				},
				Spec: *makeGitCloneTTSpec(t, "git-clone-test-run"),
			}},
			Triggers: []*triggersv1beta1.Trigger{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "git-clone-trigger",
					Namespace: namespace,
					Labels: map[string]string{
						"foo": "bar",
					},
				},
				Spec: triggersv1beta1.TriggerSpec{
					Bindings: []*triggersv1beta1.TriggerSpecBinding{{Ref: "git-clone"}},
					Template: triggersv1beta1.TriggerSpecTemplate{Ref: ptr.String("git-clone")},
				},
			}, {
				ObjectMeta: metav1.ObjectMeta{
					Name:      "second-not-invoked-trigger",
					Namespace: namespace,
					Labels: map[string]string{
						"foo": "notbar",
					},
				},
				Spec: triggersv1beta1.TriggerSpec{
					Bindings: []*triggersv1beta1.TriggerSpecBinding{{Ref: "git-clone"}},
					Template: triggersv1beta1.TriggerSpecTemplate{Ref: ptr.String("git-clone")},
				},
			}},
			EventListeners: []*triggersv1beta1.EventListener{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      eventListenerName,
					Namespace: namespace,
					UID:       types.UID(elUID),
				},
				Spec: triggersv1beta1.EventListenerSpec{
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"foo": "bar"},
					},
				},
			}},
		},
		eventBody: eventBody,
		// only one of the tasks is invoked
		want: []pipelinev1.TaskRun{gitCloneTaskRun},
	}, {
		name: "eventlistener with a trigger ref",
		resources: test.Resources{
			TriggerBindings:  []*triggersv1beta1.TriggerBinding{gitCloneTB},
			TriggerTemplates: []*triggersv1beta1.TriggerTemplate{gitCloneTT},
			Triggers: []*triggersv1beta1.Trigger{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "git-clone-trigger",
					Namespace: namespace,
				},
				Spec: triggersv1beta1.TriggerSpec{
					Bindings: []*triggersv1beta1.TriggerSpecBinding{{Ref: "git-clone"}},
					Template: triggersv1beta1.TriggerSpecTemplate{Ref: ptr.String("git-clone")},
				},
			}},
			EventListeners: []*triggersv1beta1.EventListener{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      eventListenerName,
					Namespace: namespace,
					UID:       types.UID(elUID),
				},
				Spec: triggersv1beta1.EventListenerSpec{
					Triggers: []triggersv1beta1.EventListenerTrigger{{
						TriggerRef: "git-clone-trigger",
					}},
				},
			}},
		},
		eventBody: eventBody,
		want:      []pipelinev1.TaskRun{gitCloneTaskRun},
	}, {
		name: "eventlistener with ref to trigger with embedded spec",
		resources: test.Resources{
			Triggers: []*triggersv1beta1.Trigger{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "git-clone-trigger",
					Namespace: namespace,
				},
				Spec: triggersv1beta1.TriggerSpec{
					Bindings: gitCloneTBSpec,
					Template: triggersv1beta1.TriggerSpecTemplate{Spec: makeGitCloneTTSpec(t, "git-clone-test-run")},
				},
			}},
			EventListeners: []*triggersv1beta1.EventListener{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      eventListenerName,
					Namespace: namespace,
					UID:       types.UID(elUID),
				},
				Spec: triggersv1beta1.EventListenerSpec{
					Triggers: []triggersv1beta1.EventListenerTrigger{{
						TriggerRef: "git-clone-trigger",
					}},
				},
			}},
		},
		eventBody: eventBody,
		want:      []pipelinev1.TaskRun{gitCloneTaskRun},
	}, {
		name: "with GitHub and CEL interceptors",
		resources: test.Resources{
			Secrets: []*corev1.Secret{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: namespace,
				},
				Data: map[string][]byte{
					"secretKey": []byte("secret"),
				},
			}},
			ClusterInterceptors: []*triggersv1alpha1.ClusterInterceptor{github, cel},
			Triggers: []*triggersv1beta1.Trigger{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "git-clone-trigger",
					Namespace: namespace,
				},
				Spec: triggersv1beta1.TriggerSpec{
					Interceptors: []*triggersv1beta1.EventInterceptor{{
						Ref: triggersv1beta1.InterceptorRef{Name: "github", Kind: triggersv1beta1.ClusterInterceptorKind},
						Params: []triggersv1beta1.InterceptorParams{{
							Name: "secretRef",
							Value: test.ToV1JSON(t, &triggersv1beta1.SecretRef{
								SecretKey:  "secretKey",
								SecretName: "secret",
							}),
						}, {
							Name:  "eventTypes",
							Value: test.ToV1JSON(t, []string{"pull_request"}),
						}},
					}},
					Bindings: gitCloneTBSpec,
					Template: triggersv1beta1.TriggerSpecTemplate{Spec: makeGitCloneTTSpec(t, "git-clone-test-run")},
				},
			}},
			EventListeners: []*triggersv1beta1.EventListener{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      eventListenerName,
					Namespace: namespace,
					UID:       types.UID(elUID),
				},
				Spec: triggersv1beta1.EventListenerSpec{
					Triggers: []triggersv1beta1.EventListenerTrigger{{
						TriggerRef: "git-clone-trigger",
					}},
				},
			}},
		},
		eventBody: eventBody,
		headers: map[string][]string{
			"X-GitHub-Event":  {"pull_request"},
			"X-Hub-Signature": {test.HMACHeader(t, "secret", eventBody, "sha1")},
		},
		want: []pipelinev1.TaskRun{gitCloneTaskRun},
	}, {
		name: "with BitBucket interceptor",
		resources: test.Resources{
			Secrets: []*corev1.Secret{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: namespace,
				},
				Data: map[string][]byte{
					"secretKey": []byte("secret"),
				},
			}},
			ClusterInterceptors: []*triggersv1alpha1.ClusterInterceptor{bitbucket},
			Triggers: []*triggersv1beta1.Trigger{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "git-clone-trigger",
					Namespace: namespace,
				},
				Spec: triggersv1beta1.TriggerSpec{
					Interceptors: []*triggersv1beta1.EventInterceptor{{
						Ref: triggersv1beta1.InterceptorRef{Name: "bitbucket", Kind: triggersv1beta1.ClusterInterceptorKind},
						Params: []triggersv1beta1.InterceptorParams{{
							Name: "secretRef",
							Value: test.ToV1JSON(t, &triggersv1beta1.SecretRef{
								SecretKey:  "secretKey",
								SecretName: "secret",
							}),
						}, {
							Name:  "eventTypes",
							Value: test.ToV1JSON(t, []string{"repo:refs_changed"}),
						}},
					}},
					Bindings: gitCloneTBSpec,
					Template: triggersv1beta1.TriggerSpecTemplate{Spec: makeGitCloneTTSpec(t, "git-clone-test-run")},
				},
			}},
			EventListeners: []*triggersv1beta1.EventListener{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      eventListenerName,
					Namespace: namespace,
					UID:       types.UID(elUID),
				},
				Spec: triggersv1beta1.EventListenerSpec{
					Triggers: []triggersv1beta1.EventListenerTrigger{{
						TriggerRef: "git-clone-trigger",
					}},
				},
			}},
		},
		eventBody: eventBody,
		headers: map[string][]string{
			"X-Event-Key":     {"repo:refs_changed"},
			"X-Hub-Signature": {test.HMACHeader(t, "secret", eventBody, "sha1")},
		},
		want: []pipelinev1.TaskRun{gitCloneTaskRun},
	}, {
		name: "with namespaced interceptor",
		resources: test.Resources{
			Secrets: []*corev1.Secret{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: namespace,
				},
				Data: map[string][]byte{
					"secretKey": []byte("secret"),
				},
			}},
			Interceptors: []*triggersv1alpha1.Interceptor{nsInterceptor},
			Triggers: []*triggersv1beta1.Trigger{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "git-clone-trigger",
					Namespace: namespace,
				},
				Spec: triggersv1beta1.TriggerSpec{
					Interceptors: []*triggersv1beta1.EventInterceptor{{
						Ref: triggersv1beta1.InterceptorRef{Name: "bitbucket", Kind: triggersv1beta1.NamespacedInterceptorKind},
						Params: []triggersv1beta1.InterceptorParams{{
							Name: "secretRef",
							Value: test.ToV1JSON(t, &triggersv1beta1.SecretRef{
								SecretKey:  "secretKey",
								SecretName: "secret",
							}),
						}, {
							Name:  "eventTypes",
							Value: test.ToV1JSON(t, []string{"repo:refs_changed"}),
						}},
					}},
					Bindings: gitCloneTBSpec,
					Template: triggersv1beta1.TriggerSpecTemplate{Spec: makeGitCloneTTSpec(t, "git-clone-test-run")},
				},
			}},
			EventListeners: []*triggersv1beta1.EventListener{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      eventListenerName,
					Namespace: namespace,
					UID:       types.UID(elUID),
				},
				Spec: triggersv1beta1.EventListenerSpec{
					Triggers: []triggersv1beta1.EventListenerTrigger{{
						TriggerRef: "git-clone-trigger",
					}},
				},
			}},
		},
		eventBody: eventBody,
		headers: map[string][]string{
			"X-Event-Key":     {"repo:refs_changed"},
			"X-Hub-Signature": {test.HMACHeader(t, "secret", eventBody, "sha1")},
		},
		want: []pipelinev1.TaskRun{gitCloneTaskRun},
	}, {
		name: "eventListener with multiple triggers",
		resources: test.Resources{
			EventListeners: []*triggersv1beta1.EventListener{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      eventListenerName,
					Namespace: namespace,
					UID:       types.UID(elUID),
				},
				Spec: triggersv1beta1.EventListenerSpec{
					NamespaceSelector: triggersv1beta1.NamespaceSelector{
						MatchNames: []string{namespace},
					},
				},
			}},
			Triggers: tenGitCloneTriggers,
		},
		eventBody: eventBody,
		want:      tenGitCloneTaskRuns,
	}, {
		name: "with webhook interceptors",
		resources: test.Resources{
			Triggers: []*triggersv1beta1.Trigger{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "git-clone-trigger",
					Namespace: namespace,
				},
				Spec: triggersv1beta1.TriggerSpec{
					Interceptors: []*triggersv1beta1.EventInterceptor{{
						Webhook: &triggersv1beta1.WebhookInterceptor{
							ObjectRef: &corev1.ObjectReference{
								APIVersion: "v1",
								Kind:       "Service",
								Name:       "foo",
							},
							Header: []pipelinev1.Param{{
								Name: "Name",
								Value: pipelinev1.ArrayOrString{
									Type:      pipelinev1.ParamTypeString,
									StringVal: "name-from-webhook",
								},
							}},
						},
					}},
					Bindings: []*triggersv1beta1.TriggerSpecBinding{
						{Name: "url", Value: ptr.String("https://github.com/tektoncd/triggers")},
						{Name: "revision", Value: ptr.String("master")},
						{Name: "name", Value: ptr.String("$(body.name)")}, // Header added by Webhook Interceptor
					},
					Template: triggersv1beta1.TriggerSpecTemplate{Spec: makeGitCloneTTSpec(t, "git-clone-test-run")},
				},
			}},
			EventListeners: []*triggersv1beta1.EventListener{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      eventListenerName,
					Namespace: namespace,
					UID:       types.UID(elUID),
				},
				Spec: triggersv1beta1.EventListenerSpec{
					Triggers: []triggersv1beta1.EventListenerTrigger{{
						TriggerRef: "git-clone-trigger",
					}},
				},
			}},
		},
		webhookInterceptor: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Copy over all headers
			for k := range r.Header {
				for _, v := range r.Header[k] {
					w.Header().Add(k, v)
				}
			}
			// Read the Name header
			var name string
			if nameValue, ok := r.Header["Name"]; ok {
				name = nameValue[0]
			}
			// Write the name to the body
			body := fmt.Sprintf(`{"name": "%s"}`, name)
			_, _ = w.Write([]byte(body))
		}),
		eventBody: eventBody,
		want: []pipelinev1.TaskRun{{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "tekton.dev/v1beta1",
				Kind:       "TaskRun",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name-from-webhook",
				Namespace: namespace,
				Labels: map[string]string{
					"app":                                  "triggers",
					"type":                                 "bar",
					"triggers.tekton.dev/eventlistener":    eventListenerName,
					"triggers.tekton.dev/trigger":          "git-clone-trigger",
					"triggers.tekton.dev/triggers-eventid": "12345",
				},
			},
			Spec: pipelinev1.TaskRunSpec{
				Params: []pipelinev1.Param{{
					Name:  "url",
					Value: pipelinev1.ArrayOrString{Type: pipelinev1.ParamTypeString, StringVal: "https://github.com/tektoncd/triggers"},
				}, {
					Name:  "git-revision",
					Value: pipelinev1.ArrayOrString{Type: pipelinev1.ParamTypeString, StringVal: "master"},
				}},
				TaskRef: &pipelinev1.TaskRef{Name: "git-clone"},
			},
		}},
	}, {
		name: "with interceptors overlays race",
		resources: test.Resources{
			ClusterInterceptors: []*triggersv1alpha1.ClusterInterceptor{cel},
			Triggers: []*triggersv1beta1.Trigger{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "git-clone-trigger",
						Namespace: namespace,
					},
					Spec: triggersv1beta1.TriggerSpec{
						Interceptors: []*triggersv1beta1.EventInterceptor{{
							Ref: triggersv1beta1.InterceptorRef{
								Name: "cel",
								Kind: triggersv1beta1.ClusterInterceptorKind,
							},
							Params: []triggersv1beta1.InterceptorParams{
								{Name: "filter", Value: test.ToV1JSON(t, "has(body.head_commit)")},
								{Name: "overlays", Value: test.ToV1JSON(t, []triggersv1beta1.CELOverlay{{
									Key:        "foo",
									Expression: "has(body.head_commit)",
								}})},
							},
						}},
						Bindings: []*triggersv1beta1.TriggerSpecBinding{
							{Name: "url", Value: ptr.String("https://github.com/tektoncd/triggers")},
							{Name: "revision", Value: ptr.String("master")},
							{Name: "name", Value: ptr.String("$(body.name)")}, // Header added by Webhook Interceptor
						},
						Template: triggersv1beta1.TriggerSpecTemplate{Spec: makeGitCloneTTSpec(t, "git-clone-trigger")},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "git-clone-trigger-2",
						Namespace: namespace,
					},
					Spec: triggersv1beta1.TriggerSpec{
						Interceptors: []*triggersv1beta1.EventInterceptor{{
							Ref: triggersv1beta1.InterceptorRef{
								Name: "cel",
								Kind: triggersv1beta1.ClusterInterceptorKind,
							},
							Params: []triggersv1beta1.InterceptorParams{
								{Name: "filter", Value: test.ToV1JSON(t, "has(body.head_commit)")},
								{Name: "overlays", Value: test.ToV1JSON(t, []triggersv1beta1.CELOverlay{{
									Key:        "foo",
									Expression: "has(body.head_commit)",
								}})},
							},
						}},
						Bindings: []*triggersv1beta1.TriggerSpecBinding{
							{Name: "url", Value: ptr.String("https://github.com/tektoncd/triggers")},
							{Name: "revision", Value: ptr.String("master")},
							{Name: "name", Value: ptr.String("$(body.name)")}, // Header added by Webhook Interceptor
						},
						Template: triggersv1beta1.TriggerSpecTemplate{Spec: makeGitCloneTTSpec(t, "git-clone-trigger-2")},
					},
				},
			},
			EventListeners: []*triggersv1beta1.EventListener{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      eventListenerName,
					Namespace: namespace,
					UID:       types.UID(elUID),
				},
				Spec: triggersv1beta1.EventListenerSpec{
					Triggers: []triggersv1beta1.EventListenerTrigger{
						{
							TriggerRef: "git-clone-trigger",
							Interceptors: []*triggersv1beta1.TriggerInterceptor{{
								Ref: triggersv1beta1.InterceptorRef{Name: "cel", Kind: triggersv1beta1.ClusterInterceptorKind},
								Params: []triggersv1beta1.InterceptorParams{{
									Name:  "filter",
									Value: test.ToV1JSON(t, "has(body.head_commit)"),
								}},
							}},
						},
						{
							TriggerRef: "git-clone-trigger-2",
							Interceptors: []*triggersv1beta1.TriggerInterceptor{{
								Ref: triggersv1beta1.InterceptorRef{Name: "cel", Kind: triggersv1beta1.ClusterInterceptorKind},
								Params: []triggersv1beta1.InterceptorParams{{
									Name:  "filter",
									Value: test.ToV1JSON(t, "has(body.head_commit)"),
								}},
							}},
						},
					},
				},
			}},
		},
		webhookInterceptor: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Copy over all headers
			for k := range r.Header {
				for _, v := range r.Header[k] {
					w.Header().Add(k, v)
				}
			}
			// Read the Name header
			var name string
			if nameValue, ok := r.Header["Name"]; ok {
				name = nameValue[0]
			}
			// Write the name to the body
			body := fmt.Sprintf(`{"name": "%s"}`, name)
			_, _ = w.Write([]byte(body))
		}),
		eventBody: eventBody,
		want: []pipelinev1.TaskRun{
			{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "tekton.dev/v1beta1",
					Kind:       "TaskRun",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "git-clone-trigger",
					Namespace: namespace,
					Labels: map[string]string{
						"app":                                  "triggers",
						"type":                                 "bar",
						"triggers.tekton.dev/eventlistener":    eventListenerName,
						"triggers.tekton.dev/trigger":          "git-clone-trigger",
						"triggers.tekton.dev/triggers-eventid": "12345",
					},
				},
				Spec: pipelinev1.TaskRunSpec{
					Params: []pipelinev1.Param{{
						Name:  "url",
						Value: pipelinev1.ArrayOrString{Type: pipelinev1.ParamTypeString, StringVal: "https://github.com/tektoncd/triggers"},
					}, {
						Name:  "git-revision",
						Value: pipelinev1.ArrayOrString{Type: pipelinev1.ParamTypeString, StringVal: "master"},
					}},
					TaskRef: &pipelinev1.TaskRef{Name: "git-clone"},
				},
			},
			{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "tekton.dev/v1beta1",
					Kind:       "TaskRun",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "git-clone-trigger-2",
					Namespace: namespace,
					Labels: map[string]string{
						"app":                                  "triggers",
						"type":                                 "bar",
						"triggers.tekton.dev/eventlistener":    eventListenerName,
						"triggers.tekton.dev/trigger":          "git-clone-trigger-2",
						"triggers.tekton.dev/triggers-eventid": "12345",
					},
				},
				Spec: pipelinev1.TaskRunSpec{
					Params: []pipelinev1.Param{{
						Name:  "url",
						Value: pipelinev1.ArrayOrString{Type: pipelinev1.ParamTypeString, StringVal: "https://github.com/tektoncd/triggers"},
					}, {
						Name:  "git-revision",
						Value: pipelinev1.ArrayOrString{Type: pipelinev1.ParamTypeString, StringVal: "master"},
					}},
					TaskRef: &pipelinev1.TaskRef{Name: "git-clone"},
				},
			},
		},
	}, {
		name: "single trigger within EventListener triggerGroup",
		resources: test.Resources{
			Triggers: []*triggersv1beta1.Trigger{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "git-clone-trigger",
					Namespace: namespace,
					Labels: map[string]string{
						"foo": "bar",
					},
				},
				Spec: triggersv1beta1.TriggerSpec{
					Bindings: []*triggersv1beta1.TriggerSpecBinding{{
						Ref:  "git-clone",
						Kind: triggersv1beta1.NamespacedTriggerBindingKind,
					}},
					Template: triggersv1beta1.TriggerSpecTemplate{
						Ref: ptr.String("git-clone"),
					},
				},
			}},
			EventListeners: []*triggersv1beta1.EventListener{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      eventListenerName,
					Namespace: namespace,
					UID:       types.UID(elUID),
				},
				Spec: triggersv1beta1.EventListenerSpec{
					TriggerGroups: []triggersv1beta1.EventListenerTriggerGroup{{
						Name: "filter-event",
						Interceptors: []*triggersv1beta1.TriggerInterceptor{{
							Ref: triggersv1beta1.InterceptorRef{Name: "cel", Kind: triggersv1beta1.ClusterInterceptorKind},
							Params: []triggersv1beta1.InterceptorParams{{
								Name:  "filter",
								Value: test.ToV1JSON(t, "has(body.head_commit)"),
							}},
						}},
						TriggerSelector: triggersv1beta1.EventListenerTriggerSelector{
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"foo": "bar",
								},
							},
						},
					}},
				},
			}},
			TriggerBindings:     []*triggersv1beta1.TriggerBinding{gitCloneTB},
			TriggerTemplates:    []*triggersv1beta1.TriggerTemplate{gitCloneTT},
			ClusterInterceptors: []*triggersv1alpha1.ClusterInterceptor{cel},
		},
		eventBody: eventBody,
		want:      []pipelinev1.TaskRun{gitCloneTaskRun},
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// TODO: Do we ever support multiple eventListeners? Maybe change test.Resources to only accept one?
			elName := tc.resources.EventListeners[0].Name
			sink, dynamicClient := getSinkAssets(t, tc.resources, elName, tc.webhookInterceptor)

			for _, j := range tc.resources.EventListeners {
				j.Status.SetCondition(&apis.Condition{
					Type:    apis.ConditionReady,
					Status:  corev1.ConditionTrue,
					Message: "EventListener is Ready",
				})
			}

			metricsRecorder := &MetricsHandler{Handler: http.HandlerFunc(sink.HandleEvent)}
			ts := httptest.NewServer(metricsRecorder.Intercept(sink.NewMetricsRecorderInterceptor()))
			defer ts.Close()
			req, err := http.NewRequest(http.MethodPost, ts.URL, bytes.NewReader(tc.eventBody))
			if err != nil {
				t.Fatalf("error creating request: %s", err)
			}
			if tc.headers != nil {
				req.Header = http.Header(tc.headers)
			}
			req.Header.Add("Content-Type", "application/json")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("error sending request: %s", err)
			}
			checkSinkResponse(t, resp, elName)
			sink.WGProcessTriggers.Wait()
			// Check right resources were created.
			got := toTaskRun(t, dynamicClient.Actions())

			// Sort TaskRuns when comparing (we do not know what order they were created in)
			compareTaskRuns := func(x, y pipelinev1.TaskRun) bool {
				return x.Name < y.Name
			}
			if diff := cmp.Diff(tc.want, got, cmpopts.SortSlices(compareTaskRuns)); diff != "" {
				t.Errorf("Created resources mismatch (-want + got): %s", diff)
			}
		})
	}
}

func TestHandleEvent_Error(t *testing.T) {
	var eventBody = json.RawMessage(`{"head_commit": {"id": "testrevision"}, "repository": {"url": "testurl"}}`)
	const defaultELName = "test-el"
	for _, tc := range []struct {
		name           string
		testResources  test.Resources
		condition      *apis.Condition
		eventBody      []byte
		wantStatusCode int
		wantErrLogMsg  string
	}{{
		name: "missing eventListener",
		testResources: test.Resources{
			EventListeners: []*triggersv1beta1.EventListener{},
		},
		eventBody:      eventBody,
		wantStatusCode: http.StatusInternalServerError,
		wantErrLogMsg:  "Error getting EventListener test-el in Namespace foo: eventlistener.triggers.tekton.dev \"test-el\" not found",
	}, {
		name: "eventlistener with unknown triggers",
		testResources: test.Resources{
			EventListeners: []*triggersv1beta1.EventListener{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultELName,
					Namespace: namespace,
				},
				Spec: triggersv1beta1.EventListenerSpec{
					Triggers: []triggersv1beta1.EventListenerTrigger{{
						TriggerRef: "unknown",
					}},
				},
			}},
		},
		condition: &apis.Condition{
			Type:    apis.ConditionReady,
			Status:  corev1.ConditionTrue,
			Message: "EventListener is ready",
		},
		eventBody:      eventBody,
		wantStatusCode: http.StatusAccepted,
		wantErrLogMsg:  "Error getting Trigger unknown in Namespace foo: trigger.triggers.tekton.dev \"unknown\" not found",
	}} {
		t.Run(tc.name, func(t *testing.T) {
			elName := defaultELName
			if len(tc.testResources.EventListeners) > 0 {
				elName = tc.testResources.EventListeners[0].Name
			}
			sink, _ := getSinkAssets(t, tc.testResources, elName, nil)

			// Setup a logger to capture logs to compare later
			core, logs := observer.New(zapcore.DebugLevel)
			logger := zaptest.NewLogger(t, zaptest.WrapOptions(zap.WrapCore(func(zapcore.Core) zapcore.Core { return core }))).Sugar()
			sink.Logger = logger

			for _, el := range tc.testResources.EventListeners {
				el.Status.SetCondition(tc.condition)
			}

			ts := httptest.NewServer(http.HandlerFunc(sink.HandleEvent))
			defer ts.Close()

			resp, err := http.Post(ts.URL, "application/json", bytes.NewReader(tc.eventBody))
			if err != nil {
				t.Fatalf("error making request to eventListener: %s", err)
			}
			if resp.StatusCode != tc.wantStatusCode {
				t.Fatalf("Status code mismatch: got %d, want %d", resp.StatusCode, http.StatusInternalServerError)
			}
			if tc.wantErrLogMsg != "" {
				matches := logs.FilterMessage(tc.wantErrLogMsg)
				if matches == nil || matches.Len() == 0 {
					t.Fatalf("did not find log entry: %s.\n Logs are: %v", tc.wantErrLogMsg, logs.All())
				}
			}
		})
	}
}

// sequentialInterceptor is a HTTP server that will return sequential responses.
// It expects a request of the form `{"i": n}`.
// The response body will always return with the next value set, whereas the
// headers will append new values in the sequence.
type sequentialInterceptor struct {
	called bool
}

func (f *sequentialInterceptor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.called = true
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var data map[string]int
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	defer r.Body.Close()
	data["i"]++

	// Copy over all old headers, then set new value.
	key := "Foo"
	for _, v := range r.Header[key] {
		w.Header().Add(key, v)
	}
	w.Header().Add(key, strconv.Itoa(data["i"]))
	if err := json.NewEncoder(w).Encode(data); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
	}
}

// TestExecuteInterceptor_Sequential tests that two interceptors can be called
// sequentially. It uses a HTTP server that returns a sequential response
// and two webhook interceptors pointing at the test server, validating
// that the last response is as expected.
func TestExecuteInterceptor_Sequential(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx, _ := test.SetupFakeContext(t)
	httpClient := setupInterceptors(t, fakekubeclient.Get(ctx), logger.Sugar(), &sequentialInterceptor{})

	r := Sink{
		HTTPClient: httpClient,
		Logger:     logger.Sugar(),
	}

	a := &triggersv1beta1.EventInterceptor{
		Webhook: &triggersv1beta1.WebhookInterceptor{
			ObjectRef: &corev1.ObjectReference{
				APIVersion: "v1",
				Kind:       "Service",
				Name:       "foo",
			},
		},
	}
	trigger := triggersv1beta1.Trigger{
		Spec: triggersv1beta1.TriggerSpec{
			Interceptors: []*triggersv1beta1.EventInterceptor{a, a}},
	}

	for _, method := range []string{
		http.MethodGet,
		http.MethodHead,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodConnect,
		http.MethodOptions,
		http.MethodTrace,
	} {
		t.Run(method, func(t *testing.T) {
			req, err := http.NewRequest(method, "/", nil)
			if err != nil {
				t.Fatalf("http.NewRequest: %v", err)
			}
			resp, header, _, err := r.ExecuteTriggerInterceptors(trigger, req, []byte(`{}`), logger.Sugar(), eventID, map[string]interface{}{})
			if err != nil {
				t.Fatalf("executeInterceptors: %v", err)
			}

			var got map[string]int
			if err := json.Unmarshal(resp, &got); err != nil {
				t.Fatalf("json.Unmarshal: %v", err)
			}
			want := map[string]int{"i": 2}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("Body: -want +got: %s", diff)
			}
			if diff := cmp.Diff([]string{"1", "2"}, header["Foo"]); diff != "" {
				t.Errorf("Header: -want +got: %s", diff)
			}
		})
	}
}

// errorInterceptor is a HTTP server that will always return an error response.
type errorInterceptor struct{}

func (e *errorInterceptor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
}

func TestExecuteInterceptor_error(t *testing.T) {
	logger := zaptest.NewLogger(t)
	// Route requests to either the error interceptor or sequential interceptor based on the host.
	errHost := "error"
	match := func(r *http.Request, _ *mux.RouteMatch) bool {
		return strings.Contains(r.Host, errHost)
	}
	r := mux.NewRouter()
	r.MatcherFunc(match).Handler(&errorInterceptor{})
	si := &sequentialInterceptor{}
	r.Handle("/", si)
	ctx, _ := test.SetupFakeContext(t)
	httpClient := setupInterceptors(t, fakekubeclient.Get(ctx), logger.Sugar(), r)

	s := Sink{
		HTTPClient: httpClient,
		Logger:     logger.Sugar(),
	}

	trigger := triggersv1beta1.Trigger{
		Spec: triggersv1beta1.TriggerSpec{
			Interceptors: []*triggersv1beta1.EventInterceptor{{
				// Error interceptor needs to come first.
				Webhook: &triggersv1beta1.WebhookInterceptor{
					ObjectRef: &corev1.ObjectReference{
						APIVersion: "v1",
						Kind:       "Service",
						Name:       errHost,
					},
				},
			}, {
				Webhook: &triggersv1beta1.WebhookInterceptor{
					ObjectRef: &corev1.ObjectReference{
						APIVersion: "v1",
						Kind:       "Service",
						Name:       "foo",
					},
				},
			}},
		},
	}
	req, err := http.NewRequest(http.MethodPost, "/", nil)
	if err != nil {
		t.Fatalf("http.NewRequest: %v", err)
	}
	if resp, _, _, err := s.ExecuteTriggerInterceptors(trigger, req, nil, logger.Sugar(), eventID, map[string]interface{}{}); err == nil {
		t.Errorf("expected error, got: %+v, %v", string(resp), err)
	}

	if si.called {
		t.Error("expected sequential interceptor to not be called")
	}
}

// tests sink handles request with http form data
func TestExecuteInterceptor_Form(t *testing.T) {
	logger := zaptest.NewLogger(t)

	resources := test.Resources{
		ClusterInterceptors: []*triggersv1alpha1.ClusterInterceptor{cel},
		Interceptors:        []*triggersv1alpha1.Interceptor{nsInterceptor},
	}
	s, _ := getSinkAssets(t, resources, "el-name", nil)

	trigger := triggersv1beta1.Trigger{
		Spec: triggersv1beta1.TriggerSpec{
			Interceptors: []*triggersv1beta1.EventInterceptor{{
				Ref: triggersv1beta1.InterceptorRef{Name: "cel", Kind: triggersv1beta1.ClusterInterceptorKind},
			}}},
	}

	req, err := http.NewRequest(http.MethodPost, "/", nil)
	if err != nil {
		t.Fatalf("http.NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// form data
	data := url.Values{}
	data.Set("name", "Alice")
	data.Add("hobby", "reading")
	req.Body = io.NopCloser(bytes.NewBuffer(json.RawMessage(`{"head": "blah"}`)))

	if resp, _, _, err := s.ExecuteTriggerInterceptors(trigger, req, nil, logger.Sugar(), eventID, map[string]interface{}{}); err != nil {
		t.Errorf("got the following error: %+v, %v", string(resp), err)
	}

}

func TestExecuteInterceptor_Form_no_headers(t *testing.T) {
	logger := zaptest.NewLogger(t)

	resources := test.Resources{
		ClusterInterceptors: []*triggersv1alpha1.ClusterInterceptor{cel},
		Interceptors:        []*triggersv1alpha1.Interceptor{nsInterceptor},
	}
	s, _ := getSinkAssets(t, resources, "el-name", nil)

	trigger := triggersv1beta1.Trigger{
		Spec: triggersv1beta1.TriggerSpec{
			Interceptors: []*triggersv1beta1.EventInterceptor{{
				Ref: triggersv1beta1.InterceptorRef{Name: "cel", Kind: triggersv1beta1.ClusterInterceptorKind},
			}}},
	}

	req, err := http.NewRequest(http.MethodPost, "/", nil)
	if err != nil {
		t.Fatalf("http.NewRequest: %v", err)
	}
	// form data
	data := url.Values{}
	data.Set("name", "Alice")
	data.Add("hobby", "reading")
	req.Form = data
	req.Body = io.NopCloser(bytes.NewBuffer(json.RawMessage(`{"head": "blah"}`)))

	if resp, _, _, err := s.ExecuteTriggerInterceptors(trigger, req, nil, logger.Sugar(), eventID, map[string]interface{}{}); err != nil {
		t.Errorf("got the following error: %+v, %v", string(resp), err)
	}

}

func TestExecuteInterceptor_NotContinue(t *testing.T) {
	resources := test.Resources{
		ClusterInterceptors: []*triggersv1alpha1.ClusterInterceptor{cel},
	}
	s, _ := getSinkAssets(t, resources, "el-name", nil)
	trigger := triggersv1beta1.Trigger{
		Spec: triggersv1beta1.TriggerSpec{
			Interceptors: []*triggersv1beta1.EventInterceptor{{
				Ref: triggersv1beta1.InterceptorRef{Name: "cel", Kind: triggersv1beta1.ClusterInterceptorKind},
				Params: []triggersv1beta1.InterceptorParams{{
					Name:  "filter",
					Value: test.ToV1JSON(t, `body.head == "abcde"`),
				}},
			}}},
	}
	url, _ := url.Parse("http://example.com")
	_, _, resp, err := s.ExecuteTriggerInterceptors(trigger, &http.Request{URL: url}, json.RawMessage(`{"head": "blah"}`), s.Logger, "eventID", map[string]interface{}{})
	if err != nil {
		t.Fatalf("ExecuteInterceptor() unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatalf("ExecuteInterceptor() interceptor response was nil")
	}
	if resp.Continue {
		t.Fatalf("ExecuteInterceptor(). Expected response.conitnue to be false but got true. Response: %v", resp)
	}
}

// echoInterceptor stores and returns the body back
type echoInterceptor struct {
	body map[string]interface{}
}

func (f *echoInterceptor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	defer r.Body.Close()
	f.body = data

	if err := json.NewEncoder(w).Encode(data); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
	}
}

func TestExecuteInterceptor_ExtensionChaining(t *testing.T) {
	webhookInterceptorName := "foo"
	echoServer := &echoInterceptor{}
	resources := test.Resources{
		ClusterInterceptors: []*triggersv1alpha1.ClusterInterceptor{cel},
	}
	s, _ := getSinkAssets(t, resources, "", echoServer)

	sha := "abcdefghi" // Fake "sha" to send via body
	// trigger has a chain of 3 interceptors CEL(overlay) -> webhook -> CEL(filter)
	trigger := triggersv1beta1.Trigger{
		Spec: triggersv1beta1.TriggerSpec{
			Interceptors: []*triggersv1beta1.EventInterceptor{{
				Ref: triggersv1beta1.InterceptorRef{Name: "cel", Kind: triggersv1beta1.ClusterInterceptorKind},
				Params: []triggersv1beta1.InterceptorParams{{
					Name: "overlays",
					Value: test.ToV1JSON(t, []triggersv1beta1.CELOverlay{{
						Key:        "truncated_sha",
						Expression: "body.sha.truncate(5)",
					}}),
				}},
			}, {
				Webhook: &triggersv1beta1.WebhookInterceptor{
					ObjectRef: &corev1.ObjectReference{
						APIVersion: "v1",
						Kind:       "Service",
						Name:       webhookInterceptorName,
					},
				},
			}, {
				Ref: triggersv1beta1.InterceptorRef{Name: "cel", Kind: triggersv1beta1.ClusterInterceptorKind},
				Params: []triggersv1beta1.InterceptorParams{{
					Name:  "filter",
					Value: test.ToV1JSON(t, "body.extensions.truncated_sha == \"abcde\" && extensions.truncated_sha == \"abcde\""),
				}},
			}},
		},
	}

	req, err := http.NewRequest(http.MethodPost, "/", nil)
	if err != nil {
		t.Fatalf("http.NewRequest: %v", err)
	}
	body := fmt.Sprintf(`{"sha": "%s"}`, sha)
	resp, _, iresp, err := s.ExecuteTriggerInterceptors(trigger, req, []byte(body), s.Logger, eventID, map[string]interface{}{})
	if err != nil {
		t.Fatalf("executeInterceptors: %v", err)
	}

	wantBody := map[string]interface{}{
		"sha": sha,
		"extensions": map[string]interface{}{
			"truncated_sha": "abcde",
		},
	}
	var gotBody map[string]interface{}
	if err := json.Unmarshal(resp, &gotBody); err != nil {
		t.Fatalf("json.Unmarshal response body : %v\n Response is: %+v. \n", err, resp)
	}

	if diff := cmp.Diff(wantBody, gotBody); diff != "" {
		t.Errorf("Body: -want +got: %s", diff)
	}

	// Check Interceptor got the body with extensions
	if diff := cmp.Diff(wantBody, echoServer.body); diff != "" {
		t.Errorf("Echo Interceptor did not get correct body: -want +got: %s", diff)
	}

	// Check that we forward the extension correctly to the last interceptor
	if !iresp.Continue {
		t.Errorf("Response.continue expected true but got false. Response: %v", iresp)
	}

	// Check we maintain the extensions outside the body as well
	wantExtensions := map[string]interface{}{
		"truncated_sha": "abcde",
	}

	if diff := cmp.Diff(iresp.Extensions, wantExtensions); diff != "" {
		t.Errorf("Extensions: -want +got: %s", diff)
	}
}

func TestExtendBodyWithExtensions(t *testing.T) {
	tests := []struct {
		name       string
		body       []byte
		extensions map[string]interface{}
		want       map[string]interface{}
	}{{
		name: "merges all extensions to an extension field",
		body: json.RawMessage(`{"sha": "abcdef"}`),
		extensions: map[string]interface{}{
			"added_field": "val1",
			"nested": map[string]interface{}{
				"field": "nestedVal",
			},
		},
		want: map[string]interface{}{
			"sha": "abcdef",
			"extensions": map[string]interface{}{
				"added_field": "val1",
				"nested": map[string]interface{}{
					"field": "nestedVal",
				},
			},
		},
	}, {
		name: "body contains an extension already",
		body: json.RawMessage(`{"sha": "abcdef", "extensions": {"foo": "bar"}}`),
		extensions: map[string]interface{}{
			"added_field": "val1",
		},
		want: map[string]interface{}{
			"sha": "abcdef",
			"extensions": map[string]interface{}{
				"foo":         "bar",
				"added_field": "val1",
			},
		},
	}, {
		name:       "no extensions",
		body:       json.RawMessage(`{"sha": "abcdef"}`),
		extensions: map[string]interface{}{},
		want: map[string]interface{}{
			"sha": "abcdef",
		},
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := extendBodyWithExtensions(tc.body, tc.extensions)
			if err != nil {
				t.Fatalf("extendBodyWithExtensions() unexpected error: %v", err)
			}
			gotMap := map[string]interface{}{}
			if err := json.Unmarshal(got, &gotMap); err != nil {
				t.Fatalf("extendBodyWithExtensions() failed to unmarshal result: %v", err)
			}
			if diff := cmp.Diff(tc.want, gotMap); diff != "" {
				t.Fatalf("extendBodyWithExtensions() diff -want/+got: %s", diff)
			}
		})
	}
}

func TestCloudEventHandling(t *testing.T) {
	elName := "test-el"
	gitCloneTT := &triggersv1beta1.TriggerTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "git-clone",
			Namespace: namespace,
		},
		Spec: *makeGitCloneTTSpec(t, "git-clone-test-run"),
	}
	gitCloneTB := &triggersv1beta1.TriggerBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "git-clone",
			Namespace: namespace,
		},
		Spec: triggersv1beta1.TriggerBindingSpec{
			Params: []triggersv1beta1.Param{
				{Name: "url", Value: "$(body.repository.url)"},
				{Name: "revision", Value: "$(body.head_commit.id)"},
				{Name: "name", Value: "git-clone-run"},
				{Name: "app", Value: "$(body.foo)"},
				{Name: "type", Value: "$(header.Content-Type)"},
			},
		},
	}

	resources := test.Resources{
		EventListeners: []*triggersv1beta1.EventListener{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      elName,
				Namespace: namespace,
				UID:       types.UID(elUID),
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{{
					Name: "git-clone-trigger",
					Bindings: []*triggersv1beta1.EventListenerBinding{{
						Ref:  "git-clone",
						Kind: triggersv1beta1.NamespacedTriggerBindingKind,
					}},
					Template: &triggersv1beta1.EventListenerTemplate{
						Ref: ptr.String("git-clone"),
					},
				}},
			},
		}},
		TriggerBindings:  []*triggersv1beta1.TriggerBinding{gitCloneTB},
		TriggerTemplates: []*triggersv1beta1.TriggerTemplate{gitCloneTT},
	}

	sink, dynamicClient := getSinkAssets(t, resources, elName, nil)

	for _, j := range resources.EventListeners {
		j.Status.SetCondition(&apis.Condition{
			Type:    apis.ConditionReady,
			Status:  corev1.ConditionTrue,
			Message: "EventListener is Ready",
		})
	}

	ts := httptest.NewServer(http.HandlerFunc(sink.HandleEvent))
	t.Cleanup(func() {
		ts.Close()
	})

	c, err := cloudevents.NewClientHTTP()
	if err != nil {
		log.Fatalf("failed to create client, %v", err)
	}

	event := cloudevents.NewEvent()
	event.SetType("testing.cloudevent")
	event.SetSource("testing")
	event.SetData(cloudevents.ApplicationJSON, map[string]interface{}{
		"head_commit": map[string]interface{}{
			"id": "testrevision",
		},
		"repository": map[string]interface{}{
			"url": "testurl",
		},
		"foo": "bar\t\r\nbaz昨",
	})

	ctx := cloudevents.ContextWithTarget(context.Background(), ts.URL)

	evt, res := c.Request(ctx, event)
	if !protocol.IsACK(res) {
		t.Fatalf("failed to make request: %+v", err)
	}

	var decoded map[string]interface{}
	if err := evt.DataAs(&decoded); err != nil {
		t.Fatalf("failed to decode data in cloud event response: %s", err)
	}
	wantEvent := map[string]interface{}{
		"eventID":          "12345",
		"eventListener":    "test-el",
		"eventListenerUID": "el-uid",
		"namespace":        "foo",
	}
	if diff := cmp.Diff(wantEvent, decoded); diff != "" {
		t.Errorf("CloudEvent: -want +got: %s", diff)
	}

	gitCloneTaskRun := pipelinev1.TaskRun{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tekton.dev/v1beta1",
			Kind:       "TaskRun",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "git-clone-run",
			Namespace: namespace,
			Labels: map[string]string{
				"app":                                  "bar\t\r\nbaz昨",
				"type":                                 "application/json",
				"triggers.tekton.dev/eventlistener":    elName,
				"triggers.tekton.dev/trigger":          "git-clone-trigger",
				"triggers.tekton.dev/triggers-eventid": "12345",
			},
		},
		Spec: pipelinev1.TaskRunSpec{
			Params: []pipelinev1.Param{{
				Name:  "url",
				Value: pipelinev1.ArrayOrString{Type: pipelinev1.ParamTypeString, StringVal: "testurl"},
			}, {
				Name:  "git-revision",
				Value: pipelinev1.ArrayOrString{Type: pipelinev1.ParamTypeString, StringVal: "testrevision"},
			}},
			TaskRef: &pipelinev1.TaskRef{Name: "git-clone"},
		},
	}
	sink.WGProcessTriggers.Wait()
	wantTaskRuns := []pipelinev1.TaskRun{gitCloneTaskRun}
	got := toTaskRun(t, dynamicClient.Actions())
	if diff := cmp.Diff(wantTaskRuns, got); diff != "" {
		t.Errorf("Created resources mismatch (-want +got): %s", diff)
	}
}

func makeGitCloneTTSpec(t *testing.T, name string) *triggersv1beta1.TriggerTemplateSpec {
	return &triggersv1beta1.TriggerTemplateSpec{
		Params: []triggersv1beta1.ParamSpec{
			{Name: "url"},
			{Name: "revision"},
			{Name: "name", Default: ptr.String(name)},
			{Name: "app", Default: ptr.String("triggers")},
			{Name: "type", Default: ptr.String("bar")},
		},
		ResourceTemplates: []triggersv1beta1.TriggerResourceTemplate{{
			RawExtension: trResourceTemplate(t),
		}},
	}
}
