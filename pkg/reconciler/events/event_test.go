/*
Copyright 2021 The Tekton Authors

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

package events

import (
	"errors"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"k8s.io/client-go/tools/record"
)

func TestEmit(t *testing.T) {
	testcases := []struct {
		name      string
		eventType string
		error     error
		reason    string
		wantEvent string
	}{{
		name:      "Trigger processing is success",
		eventType: TriggerProcessingSuccessfulV1,
		error:     nil,
		wantEvent: "Normal dev.tekton.event.triggers.successful.v1",
	}, {
		name:      "Trigger processing is started",
		eventType: TriggerProcessingStartedV1,
		error:     nil,
		wantEvent: "Normal dev.tekton.event.triggers.started.v1",
	}, {
		name:      "EventListener is Ready but some operation failed while serving HTTP req",
		eventType: TriggerProcessingFailedV1,
		error:     errors.New("failed to read request body"),
		wantEvent: "Warning dev.tekton.event.triggers.failed.v1 failed to read request body",
	}}

	for _, ts := range testcases {
		fr := record.NewFakeRecorder(1)
		tr := &v1beta1.EventListener{}
		sendKubernetesEvents(fr, ts.eventType, tr, ts.error)

		err := checkEvents(t, fr, ts.name, ts.wantEvent)
		if err != nil {
			t.Errorf(err.Error())
		}
	}
}

func eventFromChannel(c chan string, testName string, wantEvent string) error {
	timer := time.NewTimer(10 * time.Millisecond)
	select {
	case event := <-c:
		if wantEvent == "" {
			return fmt.Errorf("received event \"%s\" for %s but none expected", event, testName)
		}
		matching, err := regexp.MatchString(wantEvent, event)
		if err == nil {
			if !matching {
				return fmt.Errorf("expected event \"%s\" but got \"%s\" instead for %s", wantEvent, event, testName)
			}
		}
	case <-timer.C:
		if wantEvent != "" {
			return fmt.Errorf("received no events for %s but %s expected", testName, wantEvent)
		}
	}
	return nil
}

func checkEvents(t *testing.T, fr *record.FakeRecorder, testName string, wantEvent string) error {
	t.Helper()
	return eventFromChannel(fr.Events, testName, wantEvent)
}
