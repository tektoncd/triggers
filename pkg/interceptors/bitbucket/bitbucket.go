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

	gh "github.com/google/go-github/v31/github"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"google.golang.org/grpc/codes"
)

var _ triggersv1.InterceptorInterface = (*InterceptorImpl)(nil)

type InterceptorImpl struct {
	SecretGetter interceptors.SecretGetter
}

func NewInterceptor(sg interceptors.SecretGetter) *InterceptorImpl {
	return &InterceptorImpl{
		SecretGetter: sg,
	}
}

// InterceptorParams provides a webhook to intercept and pre-process events
type InterceptorParams struct {
	SecretRef *triggersv1.SecretRef `json:"secretRef,omitempty"`
	// +listType=atomic
	EventTypes []string `json:"eventTypes,omitempty"`
}

func (w *InterceptorImpl) Process(ctx context.Context, r *triggersv1.InterceptorRequest) *triggersv1.InterceptorResponse {
	p := InterceptorParams{}
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

		if r.Context == nil {
			return interceptors.Failf(codes.InvalidArgument, "no request context passed")
		}

		ns, _ := triggersv1.ParseTriggerID(r.Context.TriggerID)
		secretToken, err := w.SecretGetter.Get(ctx, ns, p.SecretRef)
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
