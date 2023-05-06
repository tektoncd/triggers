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

package interceptors_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"github.com/tektoncd/triggers/pkg/interceptors/cel"
	"github.com/tektoncd/triggers/pkg/interceptors/server"
	"github.com/tektoncd/triggers/test"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc/codes"
	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

func TestGetInterceptorParams(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   triggersv1.EventInterceptor
		want map[string]interface{}
	}{{
		name: "cel",
		in: triggersv1.EventInterceptor{
			Ref: triggersv1.InterceptorRef{Name: "cel"},
			Params: []triggersv1.InterceptorParams{{
				Name:  "filter",
				Value: test.ToV1JSON(t, `header.match("foo", "bar")`),
			}, {
				Name: "overlays",
				Value: test.ToV1JSON(t, []cel.Overlay{{
					Key:        "short_sha",
					Expression: "body.ref.truncate(7)",
				}}),
			}},
		},
		want: map[string]interface{}{
			"filter": test.ToV1JSON(t, `header.match("foo", "bar")`),
			"overlays": test.ToV1JSON(t, []cel.Overlay{{
				Key:        "short_sha",
				Expression: "body.ref.truncate(7)",
			}}),
		},
	}, {
		name: "gitlab",
		in: triggersv1.EventInterceptor{
			Ref: triggersv1.InterceptorRef{Name: "gitlab"},
			Params: []triggersv1.InterceptorParams{{
				Name: "secretRef",
				Value: test.ToV1JSON(t, &triggersv1.SecretRef{
					SecretKey:  "test-secret",
					SecretName: "token",
				}),
			}, {
				Name:  "eventTypes",
				Value: test.ToV1JSON(t, []string{"push"}),
			}},
		},
		want: map[string]interface{}{
			"eventTypes": test.ToV1JSON(t, []string{"push"}),
			"secretRef": test.ToV1JSON(t, &triggersv1.SecretRef{
				SecretKey:  "test-secret",
				SecretName: "token",
			}),
		},
	}, {
		name: "github",
		in: triggersv1.EventInterceptor{
			Ref: triggersv1.InterceptorRef{Name: "github"},
			Params: []triggersv1.InterceptorParams{{
				Name: "secretRef",
				Value: test.ToV1JSON(t, &triggersv1.SecretRef{
					SecretKey:  "test-secret",
					SecretName: "token",
				}),
			}, {
				Name:  "eventTypes",
				Value: test.ToV1JSON(t, []string{"push"}),
			}},
		},
		want: map[string]interface{}{
			"eventTypes": test.ToV1JSON(t, []string{"push"}),
			"secretRef": test.ToV1JSON(t, &triggersv1.SecretRef{
				SecretKey:  "test-secret",
				SecretName: "token",
			}),
		},
	}, {
		name: "bitbucket",
		in: triggersv1.EventInterceptor{
			Ref: triggersv1.InterceptorRef{Name: "bitbucket"},
			Params: []triggersv1.InterceptorParams{{
				Name: "secretRef",
				Value: test.ToV1JSON(t, &triggersv1.SecretRef{
					SecretKey:  "test-secret",
					SecretName: "token",
				}),
			}, {
				Name:  "eventTypes",
				Value: test.ToV1JSON(t, []string{"push"}),
			}},
		},
		want: map[string]interface{}{
			"eventTypes": test.ToV1JSON(t, []string{"push"}),
			"secretRef": test.ToV1JSON(t, &triggersv1.SecretRef{
				SecretKey:  "test-secret",
				SecretName: "token",
			}),
		},
	}, {
		name: "webhook",
		in: triggersv1.EventInterceptor{
			Webhook: &triggersv1.WebhookInterceptor{
				ObjectRef: &corev1.ObjectReference{
					Kind:       "Service",
					APIVersion: "v1",
					Namespace:  "default",
					Name:       "foo",
				},
				Header: []pipelinev1.Param{{
					Name: "p1",
					Value: pipelinev1.ParamValue{
						Type:     pipelinev1.ParamTypeArray,
						ArrayVal: []string{"v1", "v2"},
					},
				}},
			},
		},
		want: map[string]interface{}{
			"objectRef": &corev1.ObjectReference{
				Kind:       "Service",
				APIVersion: "v1",
				Namespace:  "default",
				Name:       "foo",
			},
			"header": []pipelinev1.Param{{
				Name: "p1",
				Value: pipelinev1.ParamValue{
					Type:     pipelinev1.ParamTypeArray,
					ArrayVal: []string{"v1", "v2"},
				},
			}},
		},
	}, {
		name: "interceptor using ref",
		in: triggersv1.EventInterceptor{
			Ref: triggersv1.InterceptorRef{
				Name: "gitlab",
			},
			Params: []triggersv1.InterceptorParams{{
				Name:  "eventTypes",
				Value: test.ToV1JSON(t, []string{"push"}),
			}, {
				Name: "secretRef",
				Value: test.ToV1JSON(t, triggersv1.SecretRef{
					SecretKey:  "test-secret",
					SecretName: "token",
				}),
			}},
		},
		want: map[string]interface{}{
			"eventTypes": test.ToV1JSON(t, []string{"push"}),
			"secretRef": test.ToV1JSON(t, triggersv1.SecretRef{
				SecretKey:  "test-secret",
				SecretName: "token",
			}),
		},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			got := interceptors.GetInterceptorParams(&tc.in)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("GetInterceptorParams() failed. Diff (-want/+got): %s", diff)
			}
		})
	}
}

func TestCanonical(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   map[string][]string
		want map[string][]string
	}{{
		name: "all uppercase",
		in: map[string][]string{
			"X-ABC": {"foo"},
		},
		want: map[string][]string{
			"X-Abc": {"foo"},
		},
	}, {
		name: "all lowercase",
		in: map[string][]string{
			"x-abc": {"a", "v"},
		},
		want: map[string][]string{
			"X-Abc": {"a", "v"},
		},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			got := interceptors.Canonical(tc.in)
			want := http.Header(tc.want)
			if diff := cmp.Diff(want, got); diff != "" {
				t.Fatalf("Canonical() failed. Diff (-want/+got): %s", diff)
			}
		})
	}
}

func TestUnmarshalParam(t *testing.T) {
	in := map[string]interface{}{
		"secretKey":  "key",
		"secretName": "name",
	}

	got := triggersv1.SecretRef{}
	if err := interceptors.UnmarshalParams(in, &got); err != nil {
		t.Fatalf("UnmarshalParams() unexpected error: %v", err)
	}

	want := triggersv1.SecretRef{
		SecretKey:  "key",
		SecretName: "name",
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("UnmarshalParams() failed. Diff (-want/+got): %s", diff)
	}
}

func TestGetInterceptorParams_Error(t *testing.T) {
	for _, tc := range []struct {
		ip         map[string]interface{}
		p          interface{}
		wantErrMsg string
	}{{
		ip: map[string]interface{}{
			"secretKey": func() {},
		},
		p:          triggersv1.SecretRef{},
		wantErrMsg: "failed to marshal json",
	}} {
		t.Run(tc.wantErrMsg, func(t *testing.T) {
			err := interceptors.UnmarshalParams(tc.ip, &tc.p)
			if err == nil {
				t.Fatalf("UnmarshalParams() expected error but got nil")
			}

			if !strings.Contains(err.Error(), tc.wantErrMsg) {
				t.Fatalf("UnmarshalParams() expected err to contain %s but got %s", tc.wantErrMsg, err.Error())
			}
		})
	}
}

func TestResolveToURL(t *testing.T) {
	tests := []struct {
		name   string
		getter interceptors.InterceptorGetter
		itype  string
		want   string
	}{{
		name: "ClusterInterceptor has status.address.url",
		getter: func(n string) (*v1alpha1.ClusterInterceptor, error) {
			return &v1alpha1.ClusterInterceptor{
				Status: v1alpha1.ClusterInterceptorStatus{
					AddressStatus: duckv1.AddressStatus{
						Address: &duckv1.Addressable{
							URL: &apis.URL{
								Scheme: "http",
								Host:   "some-host",
								Path:   "cel",
							},
						},
					},
				},
			}, nil
		},
		itype: "cel",
		want:  "http://some-host/cel",
	}, {
		name: "ClusterInterceptor does not have a status",
		getter: func(n string) (*v1alpha1.ClusterInterceptor, error) {
			return &v1alpha1.ClusterInterceptor{
				Spec: v1alpha1.ClusterInterceptorSpec{
					ClientConfig: v1alpha1.ClientConfig{
						URL: &apis.URL{
							Scheme: "http",
							Host:   "some-host",
							Path:   n,
						},
					},
				},
			}, nil
		},
		itype: "cel",
		want:  "http://some-host/cel",
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := interceptors.ResolveToURL(tc.getter, tc.itype)
			if err != nil {
				t.Fatalf("ResolveToURL() error: %s", err)
			}

			if got.String() != tc.want {
				t.Fatalf("ResolveToURL() want: %s; got: %s", tc.want, got)
			}
		})
	}

	t.Run("interceptor has no URL", func(t *testing.T) {
		fakeGetter := func(name string) (*v1alpha1.ClusterInterceptor, error) {
			return &v1alpha1.ClusterInterceptor{
				Spec: v1alpha1.ClusterInterceptorSpec{
					ClientConfig: v1alpha1.ClientConfig{
						URL: nil,
					},
				},
			}, nil
		}
		_, err := interceptors.ResolveToURL(fakeGetter, "cel")
		if !errors.Is(err, v1alpha1.ErrNilURL) {
			t.Fatalf("ResolveToURL expected error to be %s but got %s", v1alpha1.ErrNilURL, err)
		}
	})
}

// testServer creates a httptest server with the passed in handler and returns a http.Client that
// can be used to talk to these interceptors
func testServer(t testing.TB, handler http.Handler, d ...*http.Transport) *http.Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(func() {
		srv.Close()
	})
	httpClient := srv.Client()
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("testServer() url parse err: %v", err)
	}
	if d != nil {
		httpClient.Transport = d[0] // as we are passing only one data so referring 0th index
	} else {
		httpClient.Transport = &http.Transport{
			Proxy: http.ProxyURL(u),
		}
	}
	return httpClient
}

func TestExecute(t *testing.T) {
	defaultHeader := http.Header(map[string][]string{
		"Content-Type": {"application/json"},
	})
	defaultTriggerContext := &triggersv1.TriggerContext{
		EventURL:  "http://someurl.com",
		EventID:   "abcde",
		TriggerID: "namespaces/default/triggers/test-trigger",
	}
	for _, tc := range []struct {
		name string
		req  *triggersv1.InterceptorRequest
		url  string
		want *triggersv1.InterceptorResponse
	}{{
		name: "cel filter pass",
		req: &triggersv1.InterceptorRequest{
			Header: defaultHeader,
			InterceptorParams: map[string]interface{}{
				"filter": `header.match("Content-Type", "application/json")`,
			},
			Context: defaultTriggerContext,
		},
		url: "http://tekton-triggers-core-interceptors.knative-test.svc/cel",
		want: &triggersv1.InterceptorResponse{
			Continue: true,
		},
	}, {
		name: "cel filter fail",
		req: &triggersv1.InterceptorRequest{
			Header: defaultHeader,
			InterceptorParams: map[string]interface{}{
				"filter": `header.match("Content-Type", "application/xml")`,
			},
			Context: defaultTriggerContext,
		},
		url: "http://tekton-triggers-core-interceptors.knative-test.svc/cel",
		want: &triggersv1.InterceptorResponse{
			Continue: false,
			Status: triggersv1.Status{
				Code:    codes.FailedPrecondition,
				Message: `expression header.match("Content-Type", "application/xml") did not return true`,
			},
		},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			coreInterceptors, err := server.NewWithCoreInterceptors(nil, zaptest.NewLogger(t).Sugar())
			if err != nil {
				t.Fatalf("failed to initialize core interceptors: %v", err)
			}
			httpClient := testServer(t, coreInterceptors)
			got, err := interceptors.Execute(context.Background(), httpClient, tc.req, tc.url)
			if err != nil {
				t.Fatalf("Execute() unexpected error: %s", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("Execute() diff -want/+got: %s", diff)
			}
		})
	}
}

func TestExecute_Error(t *testing.T) {
	defaultReq := &triggersv1.InterceptorRequest{
		Body: `{}`,
		Header: http.Header(map[string][]string{
			"Content-Type": {"application/json"},
		}),
		Context: &triggersv1.TriggerContext{
			EventURL:  "http://someurl.com",
			EventID:   "abcde",
			TriggerID: "namespaces/default/triggers/test-trigger",
		},
		InterceptorParams: map[string]interface{}{
			"filter": `header.match("Content-Type", "application/json")`,
		},
	}
	coreInterceptors, err := server.NewWithCoreInterceptors(nil, zaptest.NewLogger(t).Sugar())
	if err != nil {
		t.Fatalf("failed to initialize core interceptors: %v", err)
	}
	for _, tc := range []struct {
		name string
		req  *triggersv1.InterceptorRequest
		url  string
		svr  http.Handler
	}{{
		name: "bad URL",
		req:  defaultReq,
		url:  "not_a_url",
		svr:  coreInterceptors,
	}, {
		name: "non 200 response",
		req:  defaultReq,
		url:  "http://tekton-triggers-core-interceptors.knative-test.svc/cel",
		svr: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		}),
	}, {
		name: "incorrect response format",
		req:  defaultReq,
		url:  "http://tekton-triggers-core-interceptors.knative-test.svc/cel",
		svr: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`not_json`))
		}),
	}, {
		name: "HTTPS URL",
		req:  defaultReq,
		url:  "https://interceptorurl.com",
		svr:  coreInterceptors,
	}} {
		t.Run(tc.name, func(t *testing.T) {
			var (
				got *triggersv1.InterceptorResponse
				err error
			)
			if strings.Contains(tc.url, "https") {
				client := testServer(t, tc.svr, &http.Transport{
					TLSClientConfig: &tls.Config{
						RootCAs:    x509.NewCertPool(),
						MinVersion: tls.VersionTLS13, // Added MinVersion to avoid  G402: TLS MinVersion too low. (gosec)
					},
				})
				got, err = interceptors.Execute(context.Background(), client, tc.req, tc.url)
			} else {
				client := testServer(t, tc.svr)
				got, err = interceptors.Execute(context.Background(), client, tc.req, tc.url)
			}
			if err == nil {
				t.Fatalf("Execute() did not get expected error. Response was %+v", got)
			}
		})
	}
}
