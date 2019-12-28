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
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/tektoncd/triggers/pkg/interceptors"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"

	"go.uber.org/zap"
	"golang.org/x/xerrors"
)

// Timeout for outgoing requests to interceptor services
const interceptorTimeout = 5 * time.Second

type Interceptor struct {
	HTTPClient             *http.Client
	EventListenerNamespace string
	Logger                 *zap.SugaredLogger
	Webhook                *triggersv1.WebhookInterceptor
}

func NewInterceptor(wh *triggersv1.WebhookInterceptor, c *http.Client, ns string, l *zap.SugaredLogger) interceptors.Interceptor {
	return &Interceptor{
		HTTPClient:             c,
		EventListenerNamespace: ns,
		Logger:                 l,
		Webhook:                wh,
	}
}

func (w *Interceptor) ExecuteTrigger(payload []byte, request *http.Request, trigger *triggersv1.EventListenerTrigger, eventID string) ([]byte, http.Header, error) {
	responseHeader := make(http.Header)
	interceptorURL, err := GetURI(w.Webhook.ObjectRef, w.EventListenerNamespace) // TODO: Cache this result or do this on initialization
	if err != nil {
		return nil, responseHeader, err
	}

	modifiedPayload, responseHeader, err := w.processEvent(interceptorURL, request, payload, w.Webhook.Header, interceptorTimeout)
	if err != nil {
		return nil, responseHeader, err
	}
	return modifiedPayload, responseHeader, nil
}

func (w *Interceptor) processEvent(interceptorURL *url.URL, request *http.Request, payload []byte, headerParams []pipelinev1.Param, timeout time.Duration) ([]byte, http.Header, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	outgoing := createOutgoingRequest(ctx, request, interceptorURL, payload, headerParams)
	addInterceptorHeaders(outgoing.Header, headerParams)
	respPayload, respHeader, err := makeRequest(w.HTTPClient, outgoing)
	if err != nil {
		return nil, respHeader, xerrors.Errorf("Not OK response from Event Processor: %w", err)
	}
	return respPayload, respHeader, nil
}
