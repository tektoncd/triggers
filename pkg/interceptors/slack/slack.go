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
	"net/url"
	"strings"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"google.golang.org/grpc/codes"
)

var _ triggersv1.InterceptorInterface = (*Interceptor)(nil)

type Interceptor struct {
	SecretGetter interceptors.SecretGetter
}

// Interceptor parses all the requests fields from the slack form-data request
// and adds them to the extension
func (*Interceptor) Process(ctx context.Context, r *triggersv1.InterceptorRequest) *triggersv1.InterceptorResponse {

	//get requests fields
	p := triggersv1.SlackInterceptor{}

	if err := interceptors.UnmarshalParams(r.InterceptorParams, &p); err != nil {
		return interceptors.Failf(codes.InvalidArgument, "failed to parse interceptor params: %v", err)
	}
	// validate slack headers
	contentType := r.Header["Content-Type"]
	signature := r.Header["X-Slack-Signature"]

	if strings.Contains(contentType[0], "application/x-www-form-urlencoded") && signature[0] != "" {

		parsedBody, err := url.ParseQuery(r.Body)
		if err != nil {
			return interceptors.Failf(codes.FailedPrecondition, "unable to parse request body")
		}

		// decode form
		formData := decodeFormData(parsedBody)

		// extract required fields values
		extensions := make(map[string]interface{})

		for _, field := range p.RequestedFields {
			if value, ok := formData[field]; ok {
				extensions[field] = value
			}
		}
		return &triggersv1.InterceptorResponse{
			Continue:   true,
			Extensions: extensions,
		}

	} else {
		return interceptors.Fail(codes.FailedPrecondition, "Could find slack headers")

	}

}

func decodeFormData(parsedBody url.Values) map[string]string {
	form := make(map[string]string)
	for key, value := range parsedBody {
		if len(value) > 0 {
			form[key] = value[0]
		}
	}
	return form
}

func NewInterceptor(sg interceptors.SecretGetter) *Interceptor {
	return &Interceptor{
		SecretGetter: sg,
	}
}
