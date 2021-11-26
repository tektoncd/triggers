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
	"net/http"

	"google.golang.org/grpc/codes"
	corev1lister "k8s.io/client-go/listers/core/v1"

	gh "github.com/google/go-github/v31/github"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"github.com/tektoncd/triggers/pkg/interceptors"
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
	p := triggersv1.BitbucketInterceptor{}
	if err := interceptors.UnmarshalParams(r.InterceptorParams, &p); err != nil {
		return interceptors.Failf(codes.InvalidArgument, "failed to parse interceptor params: %v", err)
	}

	headers := interceptors.Canonical(r.Header)

	// Check if the event type is in the allow-list
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

	// Next validate secrets if set
	if p.SecretRef != nil {
		// Check the secret to see if it is empty
		if p.SecretRef.SecretKey == "" {
			return interceptors.Fail(codes.FailedPrecondition, "bitbucket interceptor secretRef.secretKey is empty")
		}
		header := headers.Get("X-Hub-Signature")
		if header == "" {
			return interceptors.Fail(codes.InvalidArgument, "no X-Hub-Signature header set")
		}
		ns, _ := triggersv1.ParseTriggerID(r.Context.TriggerID)
		secretToken, err := interceptors.GetSecretToken(nil, w.SecretLister, p.SecretRef, ns)
		if err != nil {
			return interceptors.Failf(codes.FailedPrecondition, "error getting secret: %v", err)
		}

		if err := gh.ValidateSignature(header, []byte(r.Body), secretToken); err != nil {
			return interceptors.Failf(codes.FailedPrecondition, err.Error())
		}
	}

	return &triggersv1.InterceptorResponse{
		Continue: true,
	}
}
