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
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/gorilla/mux"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	dynamicclientset "github.com/tektoncd/triggers/pkg/client/dynamic/clientset"
	"github.com/tektoncd/triggers/pkg/client/dynamic/clientset/tekton"
	interceptorinformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1alpha1/clusterinterceptor"
	clustertriggerbindinginformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1alpha1/clustertriggerbinding"
	eventlistenerinformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1alpha1/eventlistener"
	triggerinformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1alpha1/trigger"
	triggerbindinginformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1alpha1/triggerbinding"
	triggertemplateinformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1alpha1/triggertemplate"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"github.com/tektoncd/triggers/pkg/interceptors/server"
	"github.com/tektoncd/triggers/pkg/template"
	"github.com/tektoncd/triggers/test"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	discoveryclient "k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	ktesting "k8s.io/client-go/testing"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/ptr"
	rtesting "knative.dev/pkg/reconciler/testing"
)

const (
	eventID   = "12345"
	namespace = "foo"
)

func init() {
	// Override UID generator for consistent test results.
	template.UUID = func() string { return eventID }
}

var (
	github = &triggersv1.ClusterInterceptor{
		ObjectMeta: metav1.ObjectMeta{
			Name: "github",
		},
		Spec: triggersv1.ClusterInterceptorSpec{
			ClientConfig: triggersv1.ClientConfig{
				URL: &apis.URL{
					Scheme: "http",
					Host:   "tekton-triggers-core-interceptors",
					Path:   "/github",
				},
			},
		},
	}
	cel = &triggersv1.ClusterInterceptor{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cel",
		},
		Spec: triggersv1.ClusterInterceptorSpec{
			ClientConfig: triggersv1.ClientConfig{
				URL: &apis.URL{
					Scheme: "http",
					Host:   "tekton-triggers-core-interceptors",
					Path:   "/cel",
				},
			},
		},
	}
	bitbucket = &triggersv1.ClusterInterceptor{
		ObjectMeta: metav1.ObjectMeta{
			Name: "bitbucket",
		},
		Spec: triggersv1.ClusterInterceptorSpec{
			ClientConfig: triggersv1.ClientConfig{
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
func getSinkAssets(t *testing.T, resources test.Resources, elName string, webhookInterceptor http.Handler) (Sink, *fakedynamic.FakeDynamicClient) {
	t.Helper()
	ctx, _ := rtesting.SetupFakeContext(t)
	clients := test.SeedResources(t, ctx, resources)

	logger := zaptest.NewLogger(t)

	dynamicClient := fakedynamic.NewSimpleDynamicClient(runtime.NewScheme())
	dynamicSet := dynamicclientset.New(tekton.WithClient(dynamicClient))

	// Setup a handler for core interceptors using httptest
	httpClient := setupInterceptors(t, clients.Kube, logger.Sugar(), webhookInterceptor)

	r := Sink{
		EventListenerName:           elName,
		EventListenerNamespace:      namespace,
		DynamicClient:               dynamicSet,
		DiscoveryClient:             clients.Kube.Discovery(),
		KubeClientSet:               clients.Kube,
		TriggersClient:              clients.Triggers,
		HTTPClient:                  httpClient,
		Logger:                      logger.Sugar(),
		Auth:                        DefaultAuthOverride{},
		EventListenerLister:         eventlistenerinformer.Get(ctx).Lister(),
		TriggerLister:               triggerinformer.Get(ctx).Lister(),
		TriggerBindingLister:        triggerbindinginformer.Get(ctx).Lister(),
		ClusterTriggerBindingLister: clustertriggerbindinginformer.Get(ctx).Lister(),
		TriggerTemplateLister:       triggertemplateinformer.Get(ctx).Lister(),
		ClusterInterceptorLister:    interceptorinformer.Get(ctx).Lister(),
	}
	return r, dynamicClient
}

// setupInterceptors creates a httptest server with all coreInterceptors and any passed in webhook interceptor
// It returns a http.Client that can be used to talk to these interceptors
func setupInterceptors(t *testing.T, k kubernetes.Interface, l *zap.SugaredLogger, webhookInterceptor http.Handler) *http.Client {
	t.Helper()
	// Setup a handler for core interceptors using httptest
	coreInterceptors, err := server.NewWithCoreInterceptors(k, l)
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

// checkSinkResponse checks that the sink response status code is 201 and that
// the body returns the EventListener, namespace, and eventID.
func checkSinkResponse(t *testing.T, resp *http.Response, elName string) {
	t.Helper()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected response code 201 but got: %v", resp.Status)
	}
	var gotBody Response
	if err := json.NewDecoder(resp.Body).Decode(&gotBody); err != nil {
		t.Fatalf("Error reading response body: %s", err)
	}
	wantBody := Response{
		EventListener: elName,
		Namespace:     namespace,
		EventID:       eventID,
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
		gitCloneTTSpec    = triggersv1.TriggerTemplateSpec{
			Params: []triggersv1.ParamSpec{
				{Name: "url"},
				{Name: "revision"},
				{Name: "name", Default: ptr.String("git-clone-test-run")},
				{Name: "app", Default: ptr.String("triggers")},
				{Name: "type", Default: ptr.String("bar")},
			},
			ResourceTemplates: []triggersv1.TriggerResourceTemplate{{
				RawExtension: trResourceTemplate(t),
			}},
		}
		gitCloneTT = &triggersv1.TriggerTemplate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "git-clone",
				Namespace: namespace,
			},
			Spec: gitCloneTTSpec,
		}
		gitCloneTBSpec = []*triggersv1.TriggerSpecBinding{
			{Name: "url", Value: ptr.String("$(body.repository.url)")},
			{Name: "revision", Value: ptr.String("$(body.head_commit.id)")},
			{Name: "name", Value: ptr.String("git-clone-run")},
			{Name: "app", Value: ptr.String("$(body.foo)")},
			{Name: "type", Value: ptr.String("$(header.Content-Type)")},
		}

		gitCloneTB = &triggersv1.TriggerBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "git-clone",
				Namespace: namespace,
			},
			Spec: triggersv1.TriggerBindingSpec{
				Params: []triggersv1.Param{
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
	tenGitCloneTriggers := []*triggersv1.Trigger{}
	for i := 0; i < 10; i++ {
		tenGitCloneTriggers = append(tenGitCloneTriggers, &triggersv1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("git-clone-trigger-%d", i),
				Namespace: namespace,
			},
			Spec: triggersv1.TriggerSpec{
				Bindings: []*triggersv1.TriggerSpecBinding{
					{Name: "url", Value: ptr.String("$(body.repository.url)")},
					{Name: "revision", Value: ptr.String("$(body.head_commit.id)")},
					{Name: "name", Value: ptr.String(fmt.Sprintf("git-clone-run-%d", i))},
					{Name: "app", Value: ptr.String("$(body.foo)")},
					{Name: "type", Value: ptr.String("$(header.Content-Type)")},
				},
				Template: triggersv1.TriggerSpecTemplate{Spec: &gitCloneTTSpec},
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
			EventListeners: []*triggersv1.EventListener{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      eventListenerName,
					Namespace: namespace,
				},
				Spec: triggersv1.EventListenerSpec{
					Triggers: []triggersv1.EventListenerTrigger{{
						Name: "git-clone-trigger",
						Bindings: []*triggersv1.EventListenerBinding{{
							Ref:  "git-clone",
							Kind: triggersv1.NamespacedTriggerBindingKind,
						}},
						Template: &triggersv1.EventListenerTemplate{
							Ref: ptr.String("git-clone"),
						},
					}},
				},
			}},
			TriggerBindings:  []*triggersv1.TriggerBinding{gitCloneTB},
			TriggerTemplates: []*triggersv1.TriggerTemplate{gitCloneTT},
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
			TriggerBindings: []*triggersv1.TriggerBinding{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "git-clone",
					Namespace: "bar",
				},
				Spec: triggersv1.TriggerBindingSpec{
					Params: []triggersv1.Param{
						{Name: "url", Value: "$(body.repository.url)"},
						{Name: "revision", Value: "$(body.head_commit.id)"},
						{Name: "name", Value: "git-clone-run"},
						{Name: "app", Value: "$(body.foo)"},
						{Name: "type", Value: "$(header.Content-Type)"},
					},
				},
			}},
			TriggerTemplates: []*triggersv1.TriggerTemplate{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "git-clone",
					Namespace: "bar",
				},
				Spec: gitCloneTTSpec,
			}},
			Triggers: []*triggersv1.Trigger{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "git-clone-trigger",
					Namespace: "bar",
				},
				Spec: triggersv1.TriggerSpec{
					Bindings: []*triggersv1.TriggerSpecBinding{{Ref: "git-clone"}},
					Template: triggersv1.TriggerSpecTemplate{Ref: ptr.String("git-clone")},
				},
			}},
			EventListeners: []*triggersv1.EventListener{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      eventListenerName,
					Namespace: namespace,
				},
				Spec: triggersv1.EventListenerSpec{
					NamespaceSelector: triggersv1.NamespaceSelector{
						MatchNames: []string{"bar"},
					},
				},
			}},
		},
		eventBody: eventBody,
		want:      []pipelinev1.TaskRun{gitCloneTaskRun},
	}, {
		name: "eventlistener with a trigger ref",
		resources: test.Resources{
			TriggerBindings:  []*triggersv1.TriggerBinding{gitCloneTB},
			TriggerTemplates: []*triggersv1.TriggerTemplate{gitCloneTT},
			Triggers: []*triggersv1.Trigger{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "git-clone-trigger",
					Namespace: namespace,
				},
				Spec: triggersv1.TriggerSpec{
					Bindings: []*triggersv1.TriggerSpecBinding{{Ref: "git-clone"}},
					Template: triggersv1.TriggerSpecTemplate{Ref: ptr.String("git-clone")},
				},
			}},
			EventListeners: []*triggersv1.EventListener{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      eventListenerName,
					Namespace: namespace,
				},
				Spec: triggersv1.EventListenerSpec{
					Triggers: []triggersv1.EventListenerTrigger{{
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
			Triggers: []*triggersv1.Trigger{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "git-clone-trigger",
					Namespace: namespace,
				},
				Spec: triggersv1.TriggerSpec{
					Bindings: gitCloneTBSpec,
					Template: triggersv1.TriggerSpecTemplate{Spec: &gitCloneTTSpec},
				},
			}},
			EventListeners: []*triggersv1.EventListener{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      eventListenerName,
					Namespace: namespace,
				},
				Spec: triggersv1.EventListenerSpec{
					Triggers: []triggersv1.EventListenerTrigger{{
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
			ClusterInterceptors: []*triggersv1.ClusterInterceptor{github, cel},
			Triggers: []*triggersv1.Trigger{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "git-clone-trigger",
					Namespace: namespace,
				},
				Spec: triggersv1.TriggerSpec{
					Interceptors: []*triggersv1.EventInterceptor{{
						GitHub: &triggersv1.GitHubInterceptor{
							SecretRef: &triggersv1.SecretRef{
								SecretKey:  "secretKey",
								SecretName: "secret",
							},
							EventTypes: []string{"pull_request"},
						},
					}, {
						CEL: &triggersv1.CELInterceptor{
							Overlays: []triggersv1.CELOverlay{
								// FIXME: We aren't really testing that this value can be used in a binding tho
								{Key: "new", Expression: "body.repository.url"},
							},
						},
					}},
					Bindings: gitCloneTBSpec,
					Template: triggersv1.TriggerSpecTemplate{Spec: &gitCloneTTSpec},
				},
			}},
			EventListeners: []*triggersv1.EventListener{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      eventListenerName,
					Namespace: namespace,
				},
				Spec: triggersv1.EventListenerSpec{
					Triggers: []triggersv1.EventListenerTrigger{{
						TriggerRef: "git-clone-trigger",
					}},
				},
			}},
		},
		eventBody: eventBody,
		headers: map[string][]string{
			"X-GitHub-Event":  {"pull_request"},
			"X-Hub-Signature": {test.HMACHeader(t, "secret", eventBody)},
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
			ClusterInterceptors: []*triggersv1.ClusterInterceptor{bitbucket},
			Triggers: []*triggersv1.Trigger{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "git-clone-trigger",
					Namespace: namespace,
				},
				Spec: triggersv1.TriggerSpec{
					Interceptors: []*triggersv1.EventInterceptor{{
						Bitbucket: &triggersv1.BitbucketInterceptor{
							SecretRef: &triggersv1.SecretRef{
								SecretKey:  "secretKey",
								SecretName: "secret",
							},
							EventTypes: []string{"repo:refs_changed"},
						},
					}},
					Bindings: gitCloneTBSpec,
					Template: triggersv1.TriggerSpecTemplate{Spec: &gitCloneTTSpec},
				},
			}},
			EventListeners: []*triggersv1.EventListener{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      eventListenerName,
					Namespace: namespace,
				},
				Spec: triggersv1.EventListenerSpec{
					Triggers: []triggersv1.EventListenerTrigger{{
						TriggerRef: "git-clone-trigger",
					}},
				},
			}},
		},
		eventBody: eventBody,
		headers: map[string][]string{
			"X-Event-Key":     {"repo:refs_changed"},
			"X-Hub-Signature": {test.HMACHeader(t, "secret", eventBody)},
		},
		want: []pipelinev1.TaskRun{gitCloneTaskRun},
	}, {
		name: "eventListener with multiple triggers",
		resources: test.Resources{
			EventListeners: []*triggersv1.EventListener{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      eventListenerName,
					Namespace: namespace,
				},
				Spec: triggersv1.EventListenerSpec{
					NamespaceSelector: triggersv1.NamespaceSelector{
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
			Triggers: []*triggersv1.Trigger{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "git-clone-trigger",
					Namespace: namespace,
				},
				Spec: triggersv1.TriggerSpec{
					Interceptors: []*triggersv1.EventInterceptor{{
						Webhook: &triggersv1.WebhookInterceptor{
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
					Bindings: []*triggersv1.TriggerSpecBinding{
						{Name: "url", Value: ptr.String("https://github.com/tektoncd/triggers")},
						{Name: "revision", Value: ptr.String("master")},
						{Name: "name", Value: ptr.String("$(body.name)")}, // Header added by Webhook Interceptor
					},
					Template: triggersv1.TriggerSpecTemplate{Spec: &gitCloneTTSpec},
				},
			}},
			EventListeners: []*triggersv1.EventListener{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      eventListenerName,
					Namespace: namespace,
				},
				Spec: triggersv1.EventListenerSpec{
					Triggers: []triggersv1.EventListenerTrigger{{
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
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// TODO: Do we ever support multiple eventListeners? Maybe change test.Resources to only accept one?
			elName := tc.resources.EventListeners[0].Name
			sink, dynamicClient := getSinkAssets(t, tc.resources, elName, tc.webhookInterceptor)
			ts := httptest.NewServer(http.HandlerFunc(sink.HandleEvent))
			defer ts.Close()
			req, err := http.NewRequest("POST", ts.URL, bytes.NewReader(tc.eventBody))
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

// Setup for TestHandleEvent_AuthOverride
const userWithPermissions = "user-with-permissions"
const userWithoutPermissions = "user-with-no-permissions"
const userWithForbiddenAccess = "user-forbidden"

var triggerAuthWG sync.WaitGroup

type fakeAuth struct {
}

func (r fakeAuth) OverrideAuthentication(sa string, _ string, _ *zap.SugaredLogger, defaultDiscoverClient discoveryclient.ServerResourcesInterface,
	defaultDynamicClient dynamic.Interface) (discoveryclient.ServerResourcesInterface, dynamic.Interface, error) {
	dynamicClient := fakedynamic.NewSimpleDynamicClient(runtime.NewScheme())
	dynamicSet := dynamicclientset.New(tekton.WithClient(dynamicClient))
	switch sa {
	case userWithoutPermissions:
		dynamicClient.PrependReactor("*", "*", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
			defer triggerAuthWG.Done()
			return true, nil, kerrors.NewUnauthorized(sa + " unauthorized")
		})
	case userWithForbiddenAccess:
		dynamicClient.PrependReactor("*", "*", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
			defer triggerAuthWG.Done()
			return true, nil, kerrors.NewForbidden(schema.GroupResource{}, sa, errors.New("action not Allowed"))
		})
	}
	return defaultDiscoverClient, dynamicSet, nil
}

func TestHandleEvent_AuthOverride(t *testing.T) {
	for _, testCase := range []struct {
		userVal    string
		statusCode int
	}{{
		userVal:    userWithoutPermissions,
		statusCode: http.StatusUnauthorized,
	}, {
		userVal:    userWithPermissions,
		statusCode: http.StatusCreated,
	}, {
		userVal:    userWithForbiddenAccess,
		statusCode: http.StatusForbidden,
	},
	} {
		t.Run(testCase.userVal, func(t *testing.T) {
			eventBody := json.RawMessage(`{"head_commit": {"id": "testrevision"}, "repository": {"url": "testurl"}}`)
			trigger := &triggersv1.Trigger{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "git-clone-trigger",
					Namespace: namespace,
				},
				Spec: triggersv1.TriggerSpec{
					ServiceAccountName: testCase.userVal,
					Interceptors: []*triggersv1.EventInterceptor{{
						GitHub: &triggersv1.GitHubInterceptor{
							SecretRef: &triggersv1.SecretRef{
								SecretKey:  "secretKey",
								SecretName: "secret",
							},
							EventTypes: []string{"pull_request"},
						},
					}},
					Bindings: []*triggersv1.TriggerSpecBinding{{
						Name:  "url",
						Value: ptr.String("$(body.repository.url)"),
					}},
					Template: triggersv1.TriggerSpecTemplate{
						Spec: &triggersv1.TriggerTemplateSpec{
							Params: []triggersv1.ParamSpec{
								{Name: "url"},
								{Name: "revision", Default: ptr.String("master")},
								{Name: "name", Default: ptr.String("git-clone-test-run")},
								{Name: "app", Default: ptr.String("triggers")},
								{Name: "type", Default: ptr.String("bar")},
							},
							ResourceTemplates: []triggersv1.TriggerResourceTemplate{{
								RawExtension: trResourceTemplate(t),
							}},
						},
					},
				},
			}
			authSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testCase.userVal,
					Namespace: testCase.userVal,
					Annotations: map[string]string{
						corev1.ServiceAccountNameKey: testCase.userVal,
						corev1.ServiceAccountUIDKey:  testCase.userVal,
					},
				},
				Type: corev1.SecretTypeServiceAccountToken,
				Data: map[string][]byte{
					corev1.ServiceAccountTokenKey: []byte(testCase.userVal),
				},
			}
			authSA := &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testCase.userVal,
					Namespace: testCase.userVal,
				},
				Secrets: []corev1.ObjectReference{{
					Name:      testCase.userVal,
					Namespace: testCase.userVal,
				}},
			}

			el := &triggersv1.EventListener{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "el",
					Namespace: namespace,
				},
				Spec: triggersv1.EventListenerSpec{
					Triggers: []triggersv1.EventListenerTrigger{{TriggerRef: "git-clone-trigger"}},
				},
			}
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: namespace,
				},
				Data: map[string][]byte{
					"secretKey": []byte("secret"),
				},
			}
			resources := test.Resources{
				ClusterInterceptors: []*triggersv1.ClusterInterceptor{github},
				Triggers:            []*triggersv1.Trigger{trigger},
				EventListeners:      []*triggersv1.EventListener{el},
				Secrets:             []*corev1.Secret{secret, authSecret},
				ServiceAccounts:     []*corev1.ServiceAccount{authSA},
			}
			sink, dynamicClient := getSinkAssets(t, resources, el.Name, nil)
			sink.Auth = fakeAuth{}
			ts := httptest.NewServer(http.HandlerFunc(sink.HandleEvent))
			defer ts.Close()

			triggerAuthWG.Add(1)
			dynamicClient.PrependReactor("*", "*", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
				defer triggerAuthWG.Done()
				return false, nil, nil
			})

			req, err := http.NewRequest("POST", ts.URL, bytes.NewReader(eventBody))
			if err != nil {
				t.Fatalf("Error creating Post request: %s", err)
			}
			req.Header.Add("Content-Type", "application/json")
			req.Header.Add("X-Github-Event", "pull_request")
			req.Header.Add("X-Hub-Signature", test.HMACHeader(t, "secret", eventBody))

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("Error sending Post request: %v", err)
			}

			if resp.StatusCode != testCase.statusCode {
				t.Fatalf("response code doesn't match: expected %d vs. actual %d, entire status %v", testCase.statusCode, resp.StatusCode, resp.Status)
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
		eventBody      []byte
		wantStatusCode int
		wantErrLogMsg  string
	}{{
		name: "missing eventListener",
		testResources: test.Resources{
			EventListeners: []*triggersv1.EventListener{},
		},
		eventBody:      eventBody,
		wantStatusCode: http.StatusInternalServerError,
		wantErrLogMsg:  "Error getting EventListener test-el in Namespace foo: eventlistener.triggers.tekton.dev \"test-el\" not found",
	}, {
		name: "eventlistener with unknown triggers",
		testResources: test.Resources{
			EventListeners: []*triggersv1.EventListener{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultELName,
					Namespace: namespace,
				},
				Spec: triggersv1.EventListenerSpec{
					Triggers: []triggersv1.EventListenerTrigger{{
						TriggerRef: "unknown",
					}},
				},
			}},
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
	w.Header().Add(key, strconv.Itoa(int(data["i"])))
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
	httpClient := setupInterceptors(t, nil, logger.Sugar(), &sequentialInterceptor{})

	r := Sink{
		HTTPClient: httpClient,
		Logger:     logger.Sugar(),
	}

	a := &triggersv1.EventInterceptor{
		Webhook: &triggersv1.WebhookInterceptor{
			ObjectRef: &corev1.ObjectReference{
				APIVersion: "v1",
				Kind:       "Service",
				Name:       "foo",
			},
		},
	}
	trigger := triggersv1.Trigger{
		Spec: triggersv1.TriggerSpec{
			Interceptors: []*triggersv1.EventInterceptor{a, a}},
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
			resp, header, _, err := r.ExecuteInterceptors(trigger, req, []byte(`{}`), logger.Sugar(), eventID)
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
	httpClient := setupInterceptors(t, nil, logger.Sugar(), r)

	s := Sink{
		HTTPClient: httpClient,
		Logger:     logger.Sugar(),
	}

	trigger := triggersv1.Trigger{
		Spec: triggersv1.TriggerSpec{
			Interceptors: []*triggersv1.EventInterceptor{{
				// Error interceptor needs to come first.
				Webhook: &triggersv1.WebhookInterceptor{
					ObjectRef: &corev1.ObjectReference{
						APIVersion: "v1",
						Kind:       "Service",
						Name:       errHost,
					},
				},
			}, {
				Webhook: &triggersv1.WebhookInterceptor{
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
	if resp, _, _, err := s.ExecuteInterceptors(trigger, req, nil, logger.Sugar(), eventID); err == nil {
		t.Errorf("expected error, got: %+v, %v", string(resp), err)
	}

	if si.called {
		t.Error("expected sequential interceptor to not be called")
	}
}

func TestExecuteInterceptor_NotContinue(t *testing.T) {
	resources := test.Resources{
		ClusterInterceptors: []*triggersv1.ClusterInterceptor{cel},
	}
	s, _ := getSinkAssets(t, resources, "el-name", nil)
	trigger := triggersv1.Trigger{
		Spec: triggersv1.TriggerSpec{
			Interceptors: []*triggersv1.EventInterceptor{{
				CEL: &triggersv1.CELInterceptor{
					Filter: `body.head == "abcde"`,
				},
			}}},
	}
	url, _ := url.Parse("http://example.com")
	_, _, resp, err := s.ExecuteInterceptors(trigger, &http.Request{URL: url}, json.RawMessage(`{"head": "blah"}`), s.Logger, "eventID")
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
		ClusterInterceptors: []*triggersv1.ClusterInterceptor{cel},
	}
	s, _ := getSinkAssets(t, resources, "", echoServer)

	sha := "abcdefghi" // Fake "sha" to send via body
	// trigger has a chain of 3 interceptors CEL(overlay) -> webhook -> CEL(filter)
	trigger := triggersv1.Trigger{
		Spec: triggersv1.TriggerSpec{
			Interceptors: []*triggersv1.EventInterceptor{{
				CEL: &triggersv1.CELInterceptor{
					Overlays: []triggersv1.CELOverlay{{
						Key:        "truncated_sha",
						Expression: "body.sha.truncate(5)",
					}},
				},
			}, {
				Webhook: &triggersv1.WebhookInterceptor{
					ObjectRef: &corev1.ObjectReference{
						APIVersion: "v1",
						Kind:       "Service",
						Name:       webhookInterceptorName,
					},
				},
			}, {
				CEL: &triggersv1.CELInterceptor{
					Filter: "body.extensions.truncated_sha == \"abcde\" && extensions.truncated_sha == \"abcde\"",
				},
			}},
		},
	}

	req, err := http.NewRequest("POST", "/", nil)
	if err != nil {
		t.Fatalf("http.NewRequest: %v", err)
	}
	body := fmt.Sprintf(`{"sha": "%s"}`, sha)
	resp, _, iresp, err := s.ExecuteInterceptors(trigger, req, []byte(body), s.Logger, eventID)
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
