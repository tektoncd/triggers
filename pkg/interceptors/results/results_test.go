/*
Copyright 2021 The Tekton Authors

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

package results

import (
	"fmt"
	"github.com/tektoncd/results/pkg/api/server/test"
	"google.golang.org/grpc"
	"net"
	"net/http"
	"testing"

	server "github.com/tektoncd/results/pkg/api/server/v1alpha2"
	v1alpha2pb "github.com/tektoncd/results/proto/v1alpha2/results_go_proto"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"go.uber.org/zap/zaptest"
	fakekubeclient "knative.dev/pkg/client/injection/kube/client/fake"
	rtesting "knative.dev/pkg/reconciler/testing"
)

const (
	port = ":0"
)

func TestInterceptor_ExecuteTrigger_ShouldContinue(t *testing.T) {
	tests := []struct {
		name              string
		interceptorParams *triggersv1.ResultsInterceptor
		payload           []byte
	}{{
		name:              "no parameter",
		interceptorParams: &triggersv1.ResultsInterceptor{},
	}, {
		name: "valid parameters",
		interceptorParams: &triggersv1.ResultsInterceptor{
			APIAddr: "localhost:50051",
			Token:   "123",
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _ := rtesting.SetupFakeContext(t)
			logger := zaptest.NewLogger(t)
			kubeClient := fakekubeclient.Get(ctx)
			req := &triggersv1.InterceptorRequest{
				Body: string(tt.payload),
				Header: http.Header{
					"Content-Type": []string{"application/json"},
				},
				InterceptorParams: map[string]interface{}{
					"apiAddr": tt.interceptorParams.APIAddr,
					"token":   tt.interceptorParams.Token,
				},
				Context: &triggersv1.TriggerContext{
					EventURL:  "https://testing.example.com",
					EventID:   "abcde",
					TriggerID: "namespaces/default/triggers/example-trigger",
				},
			}
			w := &Interceptor{
				KubeClientSet: kubeClient,
				Logger:        logger.Sugar(),
				ResultClient:  fakeResultClient(t),
			}
			res := w.Process(ctx, req)
			if !res.Continue {
				t.Fatalf("Interceptor.Process() expected res.Continue to be : true but got %t. \nStatus.Err(): %v", res.Continue, res.Status.Err())
			}
		})
	}
}

func TestInterceptor_Process_InvalidParams(t *testing.T) {
	ctx, _ := rtesting.SetupFakeContext(t)
	logger := zaptest.NewLogger(t)
	kubeClient := fakekubeclient.Get(ctx)

	w := &Interceptor{
		KubeClientSet: kubeClient,
		Logger:        logger.Sugar(),
		ResultClient:  fakeResultClient(t),
	}

	req := &triggersv1.InterceptorRequest{
		Body: `{}`,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		InterceptorParams: map[string]interface{}{
			"blah": func() {},
		},
		Context: &triggersv1.TriggerContext{
			EventURL:  "https://testing.example.com",
			EventID:   "abcde",
			TriggerID: "namespaces/default/triggers/example-trigger",
		},
	}

	res := w.Process(ctx, req)
	if res.Continue {
		t.Fatalf("Interceptor.Process() expected res.Continue to be false but got %t. \nStatus.Err(): %v", res.Continue, res.Status.Err())
	}
}

func fakeResultClient(t *testing.T) v1alpha2pb.ResultsClient {
	t.Helper()
	srv, err := server.New(test.NewDB(t))

	if err != nil {
		t.Fatalf("Failed to create fake server: %v", err)
	}
	s := grpc.NewServer()
	v1alpha2pb.RegisterResultsServer(s, srv) // local test server
	lis, err := net.Listen("tcp", port)
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	go func() {
		if err := s.Serve(lis); err != nil {
			fmt.Printf("error starting result server: %v\n", err)
		}
	}()
	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		t.Fatalf("did not connect: %v", err)
	}
	t.Cleanup(func() {
		s.Stop()
		lis.Close()
		conn.Close()
	})
	return v1alpha2pb.NewResultsClient(conn)
}
