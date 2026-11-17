/*
Copyright 2026 The Tekton Authors

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

package cronrunner

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	cehttp "github.com/cloudevents/sdk-go/v2/protocol/http"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"go.uber.org/zap"
	"knative.dev/pkg/apis"
)

// cloudEventDeliveryTimeout bounds CloudEvent delivery so an unresponsive sink
// cannot keep the runner Job alive indefinitely. Overridable in tests.
var cloudEventDeliveryTimeout = 30 * time.Second

// cloudEventPayload is the JSON body sent to spec.cloudEventSink.
type cloudEventPayload struct {
	ScheduledTemplate string `json:"scheduledTemplate"`
	Namespace         string `json:"namespace"`
	ExecutionID       string `json:"executionID"`
	ResourceCount     int    `json:"resourceCount,omitempty"`
	Error             string `json:"error,omitempty"`
}

// newTimedHTTPClient returns an HTTP client that fails hung connects/headers
// within cloudEventDeliveryTimeout (covers sinks that accept TCP but never respond).
func newTimedHTTPClient() *http.Client {
	timeout := cloudEventDeliveryTimeout
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   timeout,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ResponseHeaderTimeout: timeout,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   timeout,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
}

// newTimedCloudEventClient builds an HTTP CloudEvents client with outbound timeouts.
func newTimedCloudEventClient() (cloudevents.Client, error) {
	return cloudevents.NewClientHTTP(cehttp.WithClient(*newTimedHTTPClient()))
}

// sendCloudEventToSink sends a CloudEvent to sink. Failures are logged only;
// they do not change the create outcome.
func sendCloudEventToSink(ctx context.Context, logger *zap.SugaredLogger, sink *apis.URL, st *v1alpha1.ScheduledTemplate, executionID, eventType string, payload cloudEventPayload, client cloudevents.Client) {
	if sink == nil {
		return
	}
	target := sink.String()
	if target == "" {
		return
	}

	if client == nil {
		c, err := newTimedCloudEventClient()
		if err != nil {
			logger.With(zap.Error(err)).Error("failed to create cloudevent client")
			return
		}
		client = c
	}

	event := cloudevents.NewEvent()
	event.SetID(executionID)
	event.SetType(eventType)
	event.SetSubject(fmt.Sprintf("%s/%s %s", st.Namespace, st.Name, executionID))
	event.SetSource(fmt.Sprintf("/apis/%s/%s/namespaces/%s/scheduledtemplates/%s",
		v1alpha1.SchemeGroupVersion.Group,
		v1alpha1.SchemeGroupVersion.Version,
		st.Namespace,
		st.Name,
	))

	if err := event.SetData(cloudevents.ApplicationJSON, payload); err != nil {
		logger.With(zap.Error(err)).Error("failed to set cloudevent data")
		return
	}

	// Bound wall time. Retries are omitted so a hung peer cannot multiply the budget.
	deliveryCtx, cancel := context.WithTimeout(ctx, cloudEventDeliveryTimeout)
	defer cancel()

	result := client.Send(cloudevents.ContextWithTarget(deliveryCtx, target), event)
	if !cloudevents.IsACK(result) {
		logger.With(zap.Error(result)).Errorf("failed to send cloudevent to %s", target)
	}
}
