/*
Copyright 2020 The Tekton Authors

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

package bitbucket

import (
	"context"
	"fmt"
	"net/http"

	"google.golang.org/grpc/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	gh "github.com/google/go-github/v31/github"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
)

var _ triggersv1.InterceptorInterface = (*Interceptor)(nil)

type Interceptor struct {
	KubeClientSet kubernetes.Interface
	Logger        *zap.SugaredLogger
}

func NewInterceptor(k kubernetes.Interface, l *zap.SugaredLogger) interceptors.Interceptor {
	return &Interceptor{
		Logger:        l,
		KubeClientSet: k,
	}
}

func (w *Interceptor) ExecuteTrigger(request *http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("executeTrigger() is deprecated. Call Process() instead")
}

func (w *Interceptor) Process(ctx context.Context, r *triggersv1.InterceptorRequest) *triggersv1.InterceptorResponse {
	p := triggersv1.BitbucketInterceptor{}
	if err := interceptors.UnmarshalParams(r.InterceptorParams, &p); err != nil {
		return interceptors.Failf(codes.InvalidArgument, "failed to parse interceptor params: %v", err)
	}

	headers := interceptors.Canonical(r.Header)
	// Validate secrets first before anything else, if set
	if p.SecretRef != nil {
		header := headers.Get("X-Hub-Signature")
		if header == "" {
			return interceptors.Fail(codes.InvalidArgument, "no X-Hub-Signature header set")
		}
		ns, _ := triggersv1.ParseTriggerID(r.Context.TriggerID)
		secret, err := w.KubeClientSet.CoreV1().Secrets(ns).Get(ctx, p.SecretRef.SecretName, metav1.GetOptions{})
		if err != nil {
			return interceptors.Failf(codes.Internal, fmt.Sprintf("error getting secret: %v", err))
		}
		secretToken := secret.Data[p.SecretRef.SecretKey]

		if err := gh.ValidateSignature(header, r.Body, secretToken); err != nil {
			return interceptors.Failf(codes.FailedPrecondition, err.Error())
		}
	}

	// Next see if the event type is in the allow-list
	if p.EventTypes != nil {
		actualEvent := http.Header(r.Header).Get("X-Event-Key")
		isAllowed := false
		for _, allowedEvent := range p.EventTypes {
			if actualEvent == allowedEvent {
				isAllowed = true
				break
			}
		}
		if !isAllowed {
			return interceptors.Failf(codes.FailedPrecondition, "event type %s is not allowed", actualEvent)
		}
	}
	return &triggersv1.InterceptorResponse{
		Continue: true,
	}
}
