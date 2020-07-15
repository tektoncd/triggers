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

package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httputil"
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
	pipelinev1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	triggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	"github.com/tektoncd/triggers/test"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	rtesting "knative.dev/pkg/reconciler/testing"
)

func TestReadTrigger(t *testing.T) {
	tri, err := readTrigger("../testdata/trigger.yaml")
	if err != nil {
		t.Fatalf("failed to read trigger:%+v", err)
	}

	want := []*v1alpha1.Trigger{{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tekton.dev/v1alpha1",
			Kind:       "Trigger",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "trigger-run",
		},
		Spec: v1alpha1.TriggerSpec{
			Bindings: []*v1alpha1.TriggerSpecBinding{
				{Name: "triggerSpecBinding1"},
				{Name: "triggerSpecBinding2"},
			},
			Template: v1alpha1.TriggerSpecTemplate{
				Name: "triggerSpecTemplate",
			},
		},
	}}

	if diff := cmp.Diff(want, tri); diff != "" {
		t.Errorf("-want +got: %s", diff)
	}

}

func TestReadHTTP(t *testing.T) {
	req, err := readHTTP("../testdata/http.txt")
	if err != nil {
		t.Fatalf("failed to read HTTP: %v", err)
	}

	out, err := httputil.DumpRequest(req, true)
	if err != nil {
		t.Fatalf("failed to read HTTP: %v", err)
	}
	outStr := string(out)
	re := regexp.MustCompile(`\r?\n`)
	outStr = re.ReplaceAllString(outStr, "\n")

	expect := `POST /foo HTTP/1.1
Content-Length: 16
Content-Type: application/json
X-Header: testheader

{"test": "body"}`

	if diff := cmp.Diff(expect, outStr); diff != "" {
		t.Errorf("-want +got: %s", diff)
	}
}

func Test_processTriggerSpec(t *testing.T) {
	type args struct {
		t         *triggersv1.Trigger
		request   *http.Request
		event     []byte
		resources test.Resources
	}
	eventBody := json.RawMessage(`{"repository": {"links": {"clone": [{"href": "testurl", "name": "ssh"}, {"href": "testurl", "name": "http"}]}}, "changes": [{"ref": {"displayId": "test-branch"}}]}`)
	r, err := http.NewRequest("POST", "URL", bytes.NewReader(eventBody))
	if err != nil {
		t.Errorf("Cannot create a new request:%s", err)
	}
	taskRunTemplate := pipelinev1alpha1.TaskRun{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tekton.dev/v1alpha1",
			Kind:       "TaskRun",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-taskrun",
			Namespace: "default",
			Labels: map[string]string{
				"someLabel": "$(params.foo)",
			},
		},
		Spec: pipelinev1alpha1.TaskRunSpec{
			TaskRef: &v1beta1.TaskRef{
				Name: "my-task", // non-existent task; just for testing
			},
		},
	}
	trBytes, err := json.Marshal(taskRunTemplate)
	if err != nil {
		t.Fatalf("failed to marshall taskrun to json: %v", err)
	}

	triggerTemplate := triggersv1.TriggerTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "triggerSpecTemplate",
			Namespace: "default",
		},
		Spec: triggersv1.TriggerTemplateSpec{
			Params: []triggersv1.ParamSpec{{
				Name: "foo",
			}},
			ResourceTemplates: []triggersv1.TriggerResourceTemplate{{
				RawExtension: runtime.RawExtension{Raw: trBytes},
			}},
		},
	}

	triggerBinding := v1alpha1.TriggerBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "triggerSpecBinding1",
			Namespace: "default",
		},
		Spec: triggersv1.TriggerBindingSpec{
			Params: []triggersv1.Param{{
				Name:  "foo",
				Value: "bar",
			}},
		},
	}

	wantTaskRun := pipelinev1alpha1.TaskRun{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tekton.dev/v1alpha1",
			Kind:       "TaskRun",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-taskrun",
			Namespace: "default",
			Labels: map[string]string{
				"someLabel": "bar", // replaced with the value of foo from bar
			},
		},
		Spec: pipelinev1alpha1.TaskRunSpec{
			TaskRef: taskRunTemplate.Spec.TaskRef, // non-existent task; just for testing
		},
	}
	wantTrBytes, err := json.Marshal(wantTaskRun)
	if err != nil {
		t.Fatalf("failed to marshal wantTaskrun: %v", err)
	}

	tests := []struct {
		name    string
		args    args
		want    []json.RawMessage
		wantErr bool
	}{
		{
			name: "testing-name",
			args: args{
				t: &v1alpha1.Trigger{
					TypeMeta: metav1.TypeMeta{
						Kind:       "TriggerRun",
						APIVersion: "tekton.dev/v1alpha1"},
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-triggerRun",
					},
					Spec: v1alpha1.TriggerSpec{
						Bindings: []*v1alpha1.TriggerSpecBinding{
							{Name: "triggerSpecBinding1"}, // These should be references to TriggerBindings defined below
						},
						Template: v1alpha1.TriggerSpecTemplate{
							Name: "triggerSpecTemplate", // This should be a reference to a TriggerTemplate defined below
						},
					},
				},

				request: r,
				event:   eventBody,
				resources: test.Resources{
					// Add any resources that we need to create with a fake client
					TriggerBindings:  []*v1alpha1.TriggerBinding{&triggerBinding},
					TriggerTemplates: []*triggersv1.TriggerTemplate{&triggerTemplate},
				},
			},
			want: []json.RawMessage{wantTrBytes},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventID := "some-id"
			logger, _ := zap.NewProduction()
			eventLog := logger.Sugar()
			client := getFakeTriggersClient(t, tt.args.resources)
			got, err := processTriggerSpec(client, tt.args.t, tt.args.request, tt.args.event, eventID, eventLog)
			if (err != nil) != tt.wantErr {
				t.Errorf("processTriggerSpec() error = %v. wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("did not get expected response back -want,+got: %s", diff)
			}
		})
	}
}

func getFakeTriggersClient(t *testing.T, resources test.Resources) triggersclientset.Interface {
	t.Helper()
	ctx, _ := rtesting.SetupFakeContext(t)
	clients := test.SeedResources(t, ctx, resources)
	return clients.Triggers
}
