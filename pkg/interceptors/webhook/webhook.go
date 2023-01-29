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

package webhook

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/triggers/pkg/interceptors"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"

	"go.uber.org/zap"
)

const (
	// Timeout for outgoing requests to interceptor services
	interceptorTimeout = 5 * time.Second
	// the incoming request URL is passed through to the webhook in this header.
	webhookURLHeader = "EventListener-Request-URL"
)

type Interceptor struct {
	HTTPClient       *http.Client
	TriggerNamespace string
	Logger           *zap.SugaredLogger
	Webhook          *triggersv1.WebhookInterceptor
}

func NewInterceptor(wh *triggersv1.WebhookInterceptor, c *http.Client, ns string, l *zap.SugaredLogger) interceptors.Interceptor {
	timeoutClient := &http.Client{
		Transport: c.Transport,
		Timeout:   interceptorTimeout,
	}
	return &Interceptor{
		HTTPClient:       timeoutClient,
		TriggerNamespace: ns,
		Logger:           l,
		Webhook:          wh,
	}
}

func (w *Interceptor) ExecuteTrigger(request *http.Request) (*http.Response, error) {
	u, err := getURI(w.Webhook, w.TriggerNamespace) // TODO: Cache this result or do this on initialization
	if err != nil {
		return nil, err
	}
	request.Header.Set(webhookURLHeader, request.URL.String())
	request.URL = u
	request.Host = u.Host
	addInterceptorHeaders(request.Header, w.Webhook.Header)

	resp, err := w.HTTPClient.Do(request)
	if err != nil {
		return resp, err
	}
	if resp.StatusCode != http.StatusOK {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return resp, errors.New("failed to parse response body")
		}
		return resp, fmt.Errorf("request rejected; status: %s; message: %s", resp.Status, respBody)
	}
	return resp, err
}

// getURI retrieves the ObjectReference to URI.
func getURI(webhook *triggersv1.WebhookInterceptor, ns string) (*url.URL, error) {
	// TODO: This should work for any Addressable.
	// Use something like https://github.com/knative/eventing-contrib/blob/7c0fc5cfa8bd44da0767d9e7b250264ea6eb7d8d/pkg/controller/sinks/sinks.go#L32
	switch {
	case webhook.URL != nil:
		return webhook.URL.URL(), nil
	case webhook.ObjectRef.Kind == "Service" && webhook.ObjectRef.APIVersion == "v1":
		// TODO: Also assuming port 80 and http here. Use DNS/or the env vars?
		if webhook.ObjectRef.Namespace != "" {
			ns = webhook.ObjectRef.Namespace
		}
		return url.Parse(fmt.Sprintf("http://%s.%s.svc/", webhook.ObjectRef.Name, ns))
	default:
		return nil, errors.New("invalid objRef")
	}
}

func addInterceptorHeaders(header http.Header, headerParams []pipelinev1.Param) {
	// This clobbers any matching headers
	for _, param := range headerParams {
		if param.Value.Type == pipelinev1.ParamTypeString {
			header.Set(param.Name, param.Value.StringVal)
		} else {
			header.Del(param.Name)
			for _, v := range param.Value.ArrayVal {
				header.Add(param.Name, v)
			}
		}
	}
}
