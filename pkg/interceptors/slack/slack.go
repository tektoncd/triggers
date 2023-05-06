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
	"encoding/json"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"google.golang.org/grpc/codes"
)

var _ triggersv1.InterceptorInterface = (*InterceptorImpl)(nil)

type InterceptorImpl struct {
	SecretGetter interceptors.SecretGetter
}

// Interceptor parses all the requests fields from the slack form-data request
// and adds them to the extension
// revive:disable:unused-parameter
func (*InterceptorImpl) Process(ctx context.Context, r *triggersv1.InterceptorRequest) *triggersv1.InterceptorResponse {
	headers := interceptors.Canonical(r.Header)

	// validate slack headers
	if v := headers.Get("Content-Type"); v != "application/x-www-form-urlencoded" {
		return interceptors.Fail(codes.InvalidArgument, "missing header in payload: ContentType application/x-www-form-urlencoded")
	}

	if s := headers.Get("X-Slack-Signature"); s == "" {
		return interceptors.Fail(codes.InvalidArgument, "missing header in payload: ContentType application/x-www-form-urlencoded")
	}

	// get requests fields
	var payload map[string][]string
	if err := json.Unmarshal([]byte(r.Body), &payload); err != nil {
		return interceptors.Failf(codes.InvalidArgument, "failed to unmarshl slack payload: %v", err)
	}

	// get requests fields
	p := InterceptorParams{}
	if err := interceptors.UnmarshalParams(r.InterceptorParams, &p); err != nil {
		return interceptors.Failf(codes.InvalidArgument, "failed to parse interceptor params: %v", err)
	}

	// validate RequestedFields exists
	if p.RequestedFields == nil {
		return interceptors.Fail(codes.NotFound, "missing requested field definition")
	}

	// extract required fields values
	extensions := make(map[string]interface{})

	for _, field := range p.RequestedFields {
		if value, ok := payload[field]; ok {
			extensions[field] = value
		} else {
			return interceptors.Failf(codes.NotFound, "requested field does not exists in payload %v", payload)
		}
	}
	return &triggersv1.InterceptorResponse{
		Continue:   true,
		Extensions: extensions,
	}
}

// revive:enable:unused-parameter

func NewInterceptor(sg interceptors.SecretGetter) *InterceptorImpl {
	return &InterceptorImpl{
		SecretGetter: sg,
	}
}

type InterceptorParams struct {
	// the Requested fields to be extracted from data form

	// +listType=atomic
	RequestedFields []string `json:"requestedFields,omitempty"`
}
