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

package slack

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"google.golang.org/grpc/codes"
)

var _ triggersv1.InterceptorInterface = (*InterceptorImpl)(nil)

const (
	signingVersion = "v0"
	maxRequestAge  = 5 * time.Minute
)

type InterceptorImpl struct {
	SecretGetter interceptors.SecretGetter
}

// Interceptor parses request fields from the Slack form-data request
// and adds them to the extensions.
func (w *InterceptorImpl) Process(ctx context.Context, r *triggersv1.InterceptorRequest) *triggersv1.InterceptorResponse {
	headers := interceptors.Canonical(r.Header)

	// Validate the Content-Type header.
	if v := headers.Get("Content-Type"); v != "application/x-www-form-urlencoded" {
		return interceptors.Fail(codes.InvalidArgument, "missing header in payload: ContentType application/x-www-form-urlencoded")
	}

	p := InterceptorParams{}
	if err := interceptors.UnmarshalParams(r.InterceptorParams, &p); err != nil {
		return interceptors.Failf(codes.InvalidArgument, "failed to parse interceptor params: %v", err)
	}

	if p.SecretRef == nil {
		return interceptors.Fail(codes.FailedPrecondition, "slack interceptor requires secretRef")
	}
	if p.SecretRef.SecretName == "" {
		return interceptors.Fail(codes.FailedPrecondition, "slack interceptor secretRef.secretName is empty")
	}
	if p.SecretRef.SecretKey == "" {
		return interceptors.Fail(codes.FailedPrecondition, "slack interceptor secretRef.secretKey is empty")
	}

	signature := headers.Get("X-Slack-Signature")
	if signature == "" {
		return interceptors.Fail(codes.FailedPrecondition, "no X-Slack-Signature header set")
	}
	timestamp := headers.Get("X-Slack-Request-Timestamp")
	if timestamp == "" {
		return interceptors.Fail(codes.FailedPrecondition, "no X-Slack-Request-Timestamp header set")
	}
	timestampSeconds, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return interceptors.Failf(codes.FailedPrecondition, "invalid X-Slack-Request-Timestamp header: %v", err)
	}
	requestTime := time.Unix(timestampSeconds, 0)
	now := time.Now()
	if now.Sub(requestTime) > maxRequestAge || requestTime.Sub(now) > maxRequestAge {
		return interceptors.Fail(codes.FailedPrecondition, "X-Slack-Request-Timestamp is outside the allowed five minute window")
	}

	if r.RawBody == "" {
		return interceptors.Fail(codes.InvalidArgument, "RawBody is required for Slack signature verification")
	}

	if !strings.HasPrefix(signature, signingVersion+"=") {
		return interceptors.Failf(codes.FailedPrecondition, "unrecognized Slack signature version in %q, expected %s", signature, signingVersion)
	}

	if r.Context == nil {
		return interceptors.Fail(codes.InvalidArgument, "no request context passed")
	}
	namespace, _ := triggersv1.ParseTriggerID(r.Context.TriggerID)
	signingSecret, err := w.SecretGetter.Get(ctx, namespace, p.SecretRef)
	if err != nil {
		return interceptors.Failf(codes.FailedPrecondition, "error getting secret: %v", err)
	}

	baseString := fmt.Sprintf("%s:%s:%s", signingVersion, timestamp, r.RawBody)
	mac := hmac.New(sha256.New, signingSecret)
	_, _ = mac.Write([]byte(baseString))
	expectedSignature := fmt.Sprintf("%s=%x", signingVersion, mac.Sum(nil))
	if !hmac.Equal([]byte(expectedSignature), []byte(signature)) {
		return interceptors.Fail(codes.FailedPrecondition, "invalid Slack request signature")
	}

	// Parse the URL-encoded request body.
	payload, err := url.ParseQuery(r.RawBody)
	if err != nil {
		return interceptors.Failf(codes.InvalidArgument, "failed to unmarshal slack payload: %v", err)
	}

	// Validate that RequestedFields is configured.
	if len(p.RequestedFields) == 0 {
		return interceptors.Fail(codes.InvalidArgument, "missing requested field definition")
	}

	// Extract required field values.
	extensions := make(map[string]interface{})

	for _, field := range p.RequestedFields {
		if value, ok := payload[field]; ok {
			extensions[field] = value
		} else {
			return interceptors.Failf(codes.NotFound, "requested field does not exist in payload %v", payload)
		}
	}
	return &triggersv1.InterceptorResponse{
		Continue:   true,
		Extensions: extensions,
	}
}

func NewInterceptor(sg interceptors.SecretGetter) *InterceptorImpl {
	return &InterceptorImpl{
		SecretGetter: sg,
	}
}

type InterceptorParams struct {
	SecretRef *triggersv1.SecretRef `json:"secretRef,omitempty"`

	// the Requested fields to be extracted from data form

	// +listType=atomic
	RequestedFields []string `json:"requestedFields,omitempty"`
}
