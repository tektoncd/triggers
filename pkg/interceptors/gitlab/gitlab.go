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
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"google.golang.org/grpc/codes"
)

const (
	signingTokenPrefix = "whsec_"
	maxTimestampSkew   = 5 * time.Minute
)

// now is overridable in tests.
var now = time.Now

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
	// SigningSecretRef is the GitLab 19+ Standard Webhooks signing token (whsec_…).
	SigningSecretRef *triggersv1.SecretRef `json:"signingSecretRef,omitempty"`
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
		actualEvent := headers.Get("X-Gitlab-Event")
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

	if resp := w.verifyWebhookAuth(ctx, r, headers, p); resp != nil {
		return resp
	}
	return &triggersv1.InterceptorResponse{
		Continue: true,
	}
}

// verifyWebhookAuth succeeds if any configured check passes. When both a secret
// token and a signing token are configured, either a valid X-Gitlab-Token or a
// valid webhook-signature is enough (GitLab's zero-downtime migration).
func (w *InterceptorImpl) verifyWebhookAuth(ctx context.Context, r *triggersv1.InterceptorRequest, headers http.Header, p InterceptorParams) *triggersv1.InterceptorResponse {
	if p.SigningSecretRef == nil && p.SecretRef == nil {
		return nil
	}

	var signErr, tokenErr *triggersv1.InterceptorResponse
	if p.SigningSecretRef != nil {
		signErr = w.verifySigningToken(ctx, r, headers, p.SigningSecretRef)
		if signErr == nil {
			return nil
		}
	}
	if p.SecretRef != nil {
		tokenErr = w.verifySecretToken(ctx, r, headers, p.SecretRef)
		if tokenErr == nil {
			return nil
		}
	}

	if signErr != nil && tokenErr != nil {
		return interceptors.Failf(codes.InvalidArgument, "gitlab webhook authentication failed: %s; %s", signErr.Status.Message, tokenErr.Status.Message)
	}
	if signErr != nil {
		return signErr
	}
	return tokenErr
}

func (w *InterceptorImpl) verifySecretToken(ctx context.Context, r *triggersv1.InterceptorRequest, headers http.Header, sr *triggersv1.SecretRef) *triggersv1.InterceptorResponse {
	if sr.SecretKey == "" {
		return interceptors.Fail(codes.FailedPrecondition, "gitlab interceptor secretRef.secretKey is empty")
	}
	header := headers.Get("X-Gitlab-Token")
	if header == "" {
		return interceptors.Fail(codes.InvalidArgument, "no X-Gitlab-Token header set")
	}

	if r.Context == nil {
		return interceptors.Failf(codes.InvalidArgument, "no request context passed")
	}

	ns, _ := triggersv1.ParseTriggerID(r.Context.TriggerID)
	secretToken, err := w.SecretGetter.Get(ctx, ns, sr)
	if err != nil {
		return interceptors.Failf(codes.FailedPrecondition, "error getting secret: %v", err)
	}

	// Make sure to use a constant time comparison here.
	if subtle.ConstantTimeCompare([]byte(header), secretToken) == 0 {
		return interceptors.Fail(codes.InvalidArgument, "Invalid X-GitLab-Token")
	}
	return nil
}

func (w *InterceptorImpl) verifySigningToken(ctx context.Context, r *triggersv1.InterceptorRequest, headers http.Header, sr *triggersv1.SecretRef) *triggersv1.InterceptorResponse {
	if sr.SecretKey == "" {
		return interceptors.Fail(codes.FailedPrecondition, "gitlab interceptor signingSecretRef.secretKey is empty")
	}

	messageID := headers.Get("Webhook-Id")
	if messageID == "" {
		return interceptors.Fail(codes.InvalidArgument, "no webhook-id header set")
	}
	timestamp := headers.Get("Webhook-Timestamp")
	if timestamp == "" {
		return interceptors.Fail(codes.InvalidArgument, "no webhook-timestamp header set")
	}
	signatures := headers.Get("Webhook-Signature")
	if signatures == "" {
		return interceptors.Fail(codes.InvalidArgument, "no webhook-signature header set")
	}

	if err := validateWebhookTimestamp(timestamp); err != nil {
		return interceptors.Fail(codes.InvalidArgument, err.Error())
	}

	if r.Context == nil {
		return interceptors.Failf(codes.InvalidArgument, "no request context passed")
	}

	ns, _ := triggersv1.ParseTriggerID(r.Context.TriggerID)
	signingToken, err := w.SecretGetter.Get(ctx, ns, sr)
	if err != nil {
		return interceptors.Failf(codes.FailedPrecondition, "error getting secret: %v", err)
	}

	key, err := decodeSigningKey(signingToken)
	if err != nil {
		return interceptors.Failf(codes.FailedPrecondition, "error decoding signing token: %v", err)
	}

	expected := computeSignature(key, messageID, timestamp, r.Body)
	if !matchSignature(expected, signatures) {
		return interceptors.Fail(codes.InvalidArgument, "Invalid webhook-signature")
	}
	return nil
}

func validateWebhookTimestamp(timestamp string) error {
	sec, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid webhook-timestamp: %w", err)
	}
	skew := now().Sub(time.Unix(sec, 0))
	if skew < 0 {
		skew = -skew
	}
	if skew > maxTimestampSkew {
		return errors.New("webhook-timestamp is outside the allowed tolerance")
	}
	return nil
}

func decodeSigningKey(token []byte) ([]byte, error) {
	s := strings.TrimSpace(string(token))
	s = strings.TrimPrefix(s, signingTokenPrefix)
	return base64.StdEncoding.DecodeString(s)
}

func computeSignature(key []byte, messageID, timestamp, body string) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(messageID + "." + timestamp + "." + body))
	return "v1," + base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func matchSignature(expected, header string) bool {
	matched := false
	for _, sig := range strings.Split(header, " ") {
		if !strings.HasPrefix(sig, "v1,") {
			continue
		}
		if hmac.Equal([]byte(expected), []byte(sig)) {
			matched = true
		}
	}
	return matched
}
