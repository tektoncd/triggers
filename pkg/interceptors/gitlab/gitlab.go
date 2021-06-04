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

package gitlab

import (
	"context"
	"crypto/subtle"
	"fmt"

	"github.com/tektoncd/triggers/pkg/interceptors"
	"google.golang.org/grpc/codes"
	corev1lister "k8s.io/client-go/listers/core/v1"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"

	"go.uber.org/zap"
)

var _ triggersv1.InterceptorInterface = (*Interceptor)(nil)

type Interceptor struct {
	SecretLister corev1lister.SecretLister
	Logger       *zap.SugaredLogger
}

func NewInterceptor(sl corev1lister.SecretLister, l *zap.SugaredLogger) *Interceptor {
	return &Interceptor{
		SecretLister: sl,
		Logger:       l,
	}
}

func (w *Interceptor) Process(ctx context.Context, r *triggersv1.InterceptorRequest) *triggersv1.InterceptorResponse {
	p := triggersv1.GitLabInterceptor{}
	if err := interceptors.UnmarshalParams(r.InterceptorParams, &p); err != nil {
		return interceptors.Failf(codes.InvalidArgument, "failed to parse interceptor params: %v", err)
	}

	headers := interceptors.Canonical(r.Header)

	// Check if the event type is in the allow-list
	if p.EventTypes != nil {
		actualEvent := headers.Get("X-GitLab-Event")
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

	// Next validate secrets
	if p.SecretRef != nil {
		// Check the secret to see if it is empty
		if p.SecretRef.SecretKey == "" {
			return interceptors.Fail(codes.FailedPrecondition, "gitlab interceptor secretRef.secretKey is empty")
		}
		header := headers.Get("X-GitLab-Token")
		if header == "" {
			return interceptors.Fail(codes.InvalidArgument, "no X-GitLab-Token header set")
		}

		ns, _ := triggersv1.ParseTriggerID(r.Context.TriggerID)
		secret, err := w.SecretLister.Secrets(ns).Get(p.SecretRef.SecretName)
		if err != nil {
			return interceptors.Fail(codes.Internal, fmt.Sprintf("error getting secret: %v", err))
		}
		secretToken := secret.Data[p.SecretRef.SecretKey]

		// Make sure to use a constant time comparison here.
		if subtle.ConstantTimeCompare([]byte(header), secretToken) == 0 {
			return interceptors.Fail(codes.InvalidArgument, "Invalid X-GitLab-Token")
		}
	}
	return &triggersv1.InterceptorResponse{
		Continue: true,
	}
}
