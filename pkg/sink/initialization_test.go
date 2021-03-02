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

package sink

import (
	"flag"
	"strconv"
	"testing"
	"time"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	eventlistenerinformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1alpha1/eventlistener"
	"github.com/tektoncd/triggers/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	rtesting "knative.dev/pkg/reconciler/testing"
)

var testBackoff = wait.Backoff{
	Duration: 50 * time.Millisecond,
	Factor:   1.0,
	Jitter:   0.1,
	Steps:    1,
	Cap:      100 * time.Millisecond,
}

func Test_GetArgs(t *testing.T) {
	if err := flag.Set(name, "elname"); err != nil {
		t.Errorf("Error setting flag el-name: %s", err)
	}
	if err := flag.Set(elNamespace, "elnamespace"); err != nil {
		t.Errorf("Error setting flag el-namespace: %s", err)
	}
	if err := flag.Set(port, "port"); err != nil {
		t.Errorf("Error setting flag port: %s", err)
	}
	if err := flag.Set(isMultiNS, "true"); err != nil {
		t.Errorf("Error setting flag isMultiNS: %s", err)
	}

	sinkArgs, err := GetArgs()
	if err != nil {
		t.Fatalf("GetArgs() returned unexpected error: %s", err)
	}
	if sinkArgs.ElName != "elname" {
		t.Errorf("Error el-name want elname, got %s", sinkArgs.ElName)
	}
	if sinkArgs.ElNamespace != "elnamespace" {
		t.Errorf("Error el-namespace want elnamespace, got %s", sinkArgs.ElNamespace)
	}
	if sinkArgs.Port != "port" {
		t.Errorf("Error port want port, got %s", sinkArgs.Port)
	}
	if sinkArgs.IsMultiNS != true {
		t.Errorf("Error EL Type want type, got %t", sinkArgs.IsMultiNS)
	}
}

func Test_GetArgs_error(t *testing.T) {
	tests := []struct {
		name        string
		elName      string
		elNamespace string
		port        string
		isMultiNS   bool
	}{{
		name:        "no eventlistener name",
		elName:      "",
		elNamespace: "elnamespace",
		port:        "port",
		isMultiNS:   false,
	}, {
		name:        "no eventlistener namespace",
		elName:      "elname",
		elNamespace: "",
		port:        "port",
		isMultiNS:   false,
	}, {
		name:        "no eventlistener namespace",
		elName:      "elname",
		elNamespace: "elnamespace",
		port:        "",
		isMultiNS:   false,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := flag.Set(name, tt.elName); err != nil {
				t.Errorf("Error setting flag %s: %s", name, err)
			}
			if err := flag.Set("el-namespace", tt.elNamespace); err != nil {
				t.Errorf("Error setting flag %s: %s", namespace, err)
			}
			if err := flag.Set("port", tt.port); err != nil {
				t.Errorf("Error setting flag %s: %s", port, err)
			}
			isMulNS := strconv.FormatBool(tt.isMultiNS)
			if err := flag.Set("is-multi-ns", isMulNS); err != nil {
				t.Errorf("Error setting flag %s: %s", isMultiNS, err)
			}
			if sinkArgs, err := GetArgs(); err == nil {
				t.Errorf("GetArgs() did not return error when expected; sinkArgs: %v", sinkArgs)
			}
		})
	}
}

func TestWaitForEventlistener(t *testing.T) {
	eventListenerName := "my-eventlistener"
	namespace := "my-namespace"
	eventListener := &triggersv1.EventListener{
		ObjectMeta: metav1.ObjectMeta{
			Name:      eventListenerName,
			Namespace: namespace,
		},
		Spec: triggersv1.EventListenerSpec{},
	}
	ctx, _ := rtesting.SetupFakeContext(t)
	test.SeedResources(t, ctx, test.Resources{EventListeners: []*triggersv1.EventListener{eventListener}})
	r := Sink{
		EventListenerName:      eventListenerName,
		EventListenerNamespace: namespace,
		EventListenerLister:    eventlistenerinformer.Get(ctx).Lister(),
	}

	err := r.WaitForEventListener(testBackoff)
	if err != nil {
		t.Fatalf("Expected no error, received %s", err)
	}
}

func TestWaitForEventlistener_Fatal(t *testing.T) {
	eventListenerName := "my-eventlistener"
	namespace := "my-namespace"
	ctx, _ := rtesting.SetupFakeContext(t)
	test.SeedResources(t, ctx, test.Resources{EventListeners: []*triggersv1.EventListener{}})
	r := Sink{
		EventListenerName:      eventListenerName,
		EventListenerNamespace: namespace,
		EventListenerLister:    eventlistenerinformer.Get(ctx).Lister(),
	}
	// will fail
	err := r.WaitForEventListener(testBackoff)
	if err == nil {
		t.Fatalf("expected eventlistener wait to fail, instead succeeded")
	}
	if err.Error() != "Unable to retrieve EventListener my-eventlistener in Namespace my-namespace: timed out waiting for the condition" {
		t.Fatalf("got incorrect error message, received: %s", err)
	}
}
