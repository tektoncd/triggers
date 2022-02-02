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
	"testing"

	"github.com/cloudevents/sdk-go/v2/client/test"
	"github.com/cloudevents/sdk-go/v2/event"
	testCE "github.com/cloudevents/sdk-go/v2/test"
	"github.com/cloudevents/sdk-go/v2/types"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"go.uber.org/zap/zaptest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSendCloudEvents(t *testing.T) {
	client, eventCh := test.NewMockSenderClient(t, 3)
	logger := zaptest.NewLogger(t).Sugar()

	payload := []byte("hello")
	el := triggersv1.EventListener{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "el-name",
			Namespace: "el-ns",
		},
	}

	eventID := "1234567"
	eventType := "started"
	targetURI := "http://localhost"
	ctType := "application/json"
	subject := el.Name + " processing " + eventID
	e := event.Event{
		Context: event.EventContextV1{
			Type:            eventType,
			Source:          *types.ParseURIRef("/apis///namespaces/el-ns//el-name"),
			ID:              eventID,
			DataContentType: &ctType,
			Subject:         &subject,
		}.AsV1(),
		DataEncoded: payload,
	}

	resource := Resource{
		EventID:   eventID,
		EventType: eventType,
		TargetURI: targetURI,
		Client:    client,
		Logger:    logger,
		Data:      payload,
		EL:        el,
	}

	resource.SendCloudEvents()
	received := <-eventCh
	testCE.AssertEventEquals(t, e, received)
}
