/*
Copyright 2022 The Tekton Authors

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

package cloudevent

import (
	"context"
	"fmt"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"go.uber.org/zap"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
)

// CEClient matches the `Client` interface from github.com/cloudevents/sdk-go/v2/cloudevents
type CEClient cloudevents.Client

// Resource structure defines parameters needed to send cloudEvents
type Resource struct {
	EventID   string
	EventType string
	TargetURI string
	Client    CEClient
	Logger    *zap.SugaredLogger
	Data      []byte
	EL        triggersv1.EventListener
}

// SendCloudEvents is used by the EventListener to send cloud events status of
// trigger processing
func (r Resource) SendCloudEvents() {
	logger := r.Logger.With(zap.String("trigger", r.EL.Name))

	event := cloudevents.NewEvent()
	event.SetID(r.EventID)
	event.SetSubject(r.EL.Name + " processing " + r.EventID)
	gvk := r.EL.GetObjectKind().GroupVersionKind()
	source := fmt.Sprintf("/apis/%s/%s/namespaces/%s/%s/%s",
		gvk.Group,
		gvk.Version,
		r.EL.Namespace,
		gvk.Kind,
		r.EL.Name)
	event.SetSource(source)
	event.SetType(r.EventType)

	if err := event.SetData(cloudevents.ApplicationJSON, r.Data); err != nil {
		logger.With(zap.Error(err)).Error("failed to set cloudevent data.")
		return
	}

	// Send the event.
	result := r.Client.Send(cloudevents.ContextWithTarget(cloudevents.ContextWithRetriesExponentialBackoff(context.Background(), 10*time.Millisecond, 10), r.TargetURI), event)

	if !cloudevents.IsACK(result) {
		logger.With(zap.Error(result)).Error("failed to send cloudevent.")
		return
	}
}
