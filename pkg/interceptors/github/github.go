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

package github

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	gh "github.com/google/go-github/v31/github"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
)

var _ triggersv1.InterceptorInterface = (*Interceptor)(nil)

// ErrInvalidContentType is returned when the content-type is not a JSON body.
var ErrInvalidContentType = errors.New("form parameter encoding not supported, please change the hook to send JSON payloads")

type Interceptor struct {
	KubeClientSet kubernetes.Interface
	Logger        *zap.SugaredLogger
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

	p := triggersv1.GitHubInterceptor{}
	if err := interceptors.UnmarshalParams(r.InterceptorParams, &p); err != nil {
		return interceptors.Failf(codes.InvalidArgument, "failed to parse interceptor params: %v", err)
	}
	// Validate secrets first before anything else, if set
	if p.SecretRef != nil {
		header := headers.Get("X-Hub-Signature")
		if header == "" {
			return interceptors.Fail(codes.FailedPrecondition, "no X-Hub-Signature header set")
		}

		ns, _ := triggersv1.ParseTriggerID(r.Context.TriggerID)
		secret, err := w.KubeClientSet.CoreV1().Secrets(ns).Get(ctx, p.SecretRef.SecretName, metav1.GetOptions{})
		if err != nil {
			return interceptors.Failf(codes.FailedPrecondition, "error getting secret: %v", err)
		}
		secretToken := secret.Data[p.SecretRef.SecretKey]

		if err := gh.ValidateSignature(header, []byte(r.Body), secretToken); err != nil {
			return interceptors.Fail(codes.FailedPrecondition, err.Error())
		}
	}

	// Next see if the event type is in the allow-list
	if p.EventTypes != nil {
		actualEvent := headers.Get("X-GitHub-Event")
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
