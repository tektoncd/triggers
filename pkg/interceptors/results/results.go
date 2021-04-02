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
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"google.golang.org/genproto/googleapis/api/httpbody"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
	"google.golang.org/protobuf/types/known/anypb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/transport"

	v1alpha2pb "github.com/tektoncd/results/proto/v1alpha2/results_go_proto"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"github.com/tektoncd/triggers/pkg/interceptors"
)

const (
	// Service Account token path. See https://kubernetes.io/docs/tasks/access-application-cluster/access-cluster/#accessing-the-api-from-a-pod
	podTokenPath    = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	localAPIAddr    = "tekton-results-api-service.tekton-pipelines.svc.cluster.local:50051"
	tlsSecretName   = "tekton-results-tls"
	tektonNamespace = "tekton-pipelines"
)

var _ triggersv1.InterceptorInterface = (*Interceptor)(nil)

// ErrInvalidContentType is returned when the content-type is not a JSON body.
var ErrInvalidContentType = errors.New("form parameter encoding not supported, please change the hook to send JSON payloads")

type Interceptor struct {
	KubeClientSet kubernetes.Interface
	Logger        *zap.SugaredLogger
	ResultClient  v1alpha2pb.ResultsClient
}

func NewInterceptor(k kubernetes.Interface, l *zap.SugaredLogger) *Interceptor {
	return &Interceptor{
		Logger:        l,
		KubeClientSet: k,
	}
}

func (w *Interceptor) Process(ctx context.Context, r *triggersv1.InterceptorRequest) *triggersv1.InterceptorResponse {
	headers := interceptors.Canonical(r.Header)
	if v := headers.Get("Content-Type"); v == "application/x-www-form-urlencoded" {
		return interceptors.Fail(codes.InvalidArgument, ErrInvalidContentType.Error())
	}

	p := triggersv1.ResultsInterceptor{}
	if err := interceptors.UnmarshalParams(r.InterceptorParams, &p); err != nil {
		return interceptors.Failf(codes.InvalidArgument, "failed to parse interceptor params: %v", err)
	}
	if p.APIAddr == "" {
		p.APIAddr = localAPIAddr
	}
	if w.ResultClient == nil {
		conn, err := connectToAPIServer(ctx, w.KubeClientSet, p.APIAddr, p.Token)
		if err != nil {
			return interceptors.Failf(codes.InvalidArgument, "did not connect: %v", err)
		}
		defer conn.Close()
		w.ResultClient = v1alpha2pb.NewResultsClient(conn)
	}
	ns, _ := triggersv1.ParseTriggerID(r.Context.TriggerID)
	res, err := w.ResultClient.CreateResult(ctx, &v1alpha2pb.CreateResultRequest{
		Parent: ns,
		Result: &v1alpha2pb.Result{Name: fmt.Sprintf("%s/results/%s", ns, r.Context.EventID)},
	})
	if err != nil {
		return interceptors.Failf(codes.InvalidArgument, "failed to create result: %v", err)
	}
	data, err := json.Marshal(r)
	if err != nil {
		return interceptors.Failf(codes.InvalidArgument, "failed to marshal request: %v", err)
	}
	a, err := anypb.New(&httpbody.HttpBody{Data: data})
	if err != nil {
		return interceptors.Failf(codes.InvalidArgument, "error wrapping Any proto: %v", err)
	}
	_, err = w.ResultClient.CreateRecord(ctx, &v1alpha2pb.CreateRecordRequest{
		Parent: res.GetName(),
		Record: &v1alpha2pb.Record{
			Name: fmt.Sprintf("%s/records/%d", res.GetName(), 0),
			Data: a,
		},
	})

	return &triggersv1.InterceptorResponse{
		Continue: true,
	}
}

func connectToAPIServer(ctx context.Context, kubeClient kubernetes.Interface, apiAddr string, token string) (*grpc.ClientConn, error) {
	// Load TLS certs
	certs, err := loadCerts(ctx, kubeClient)
	if err != nil {
		log.Fatalf("error loading cert pool: %v", err)
	}
	cred := credentials.NewClientTLSFromCert(certs, "")
	var ts oauth2.TokenSource
	if token != "" {
		ts = oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	} else {
		ts = transport.NewCachedFileTokenSource(podTokenPath)
	}
	opts := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithReturnConnectionError(),
		grpc.WithDefaultCallOptions(grpc.PerRPCCredentials(oauth.TokenSource{TokenSource: ts})),
		grpc.WithTransportCredentials(cred),
	}

	log.Printf("dialing %s...\n", apiAddr)
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return grpc.DialContext(ctx, apiAddr, opts...)
}

func loadCerts(ctx context.Context, kubeClient kubernetes.Interface) (*x509.CertPool, error) {
	// Setup TLS certs to the server.
	secret, err := kubeClient.CoreV1().Secrets(tektonNamespace).Get(ctx, tlsSecretName, metav1.GetOptions{})
	if err != nil {
		log.Println("no local cluster cert found. Has results service started? Defaulting to system pool...")
		return x509.SystemCertPool()
	}

	certs, err := x509.SystemCertPool()
	if err != nil {
		return nil, fmt.Errorf("error loading cert pool: %v", err)
	}
	if ok := certs.AppendCertsFromPEM(secret.Data["tls.crt"]); !ok {
		return nil, fmt.Errorf("unable to add cert to pool")
	}
	return certs, nil
}
