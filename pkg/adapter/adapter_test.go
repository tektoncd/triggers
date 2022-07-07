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

package adapter

import (
	"log"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	faketriggersclient "github.com/tektoncd/triggers/pkg/client/injection/client/fake"
	fakeClusterInterceptorinformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1alpha1/clusterinterceptor/fake"
	"github.com/tektoncd/triggers/pkg/sink"
	pkgtesting "github.com/tektoncd/triggers/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/logging"
)

func TestGetHTTPClientEmptyCaBundle(t *testing.T) {
	recorder, err := sink.NewRecorder()
	if err != nil {
		log.Fatal(err.Error())
	}
	ctx, _ := pkgtesting.SetupFakeContext(t)
	s := sinker{
		Logger:    logging.FromContext(ctx),
		Namespace: "",
		Args:      sink.Args{},
		Clients: sink.Clients{
			TriggersClient: faketriggersclient.Get(ctx),
		},
		Recorder: recorder,
		injCtx:   ctx,
	}

	c, err := s.getHTTPClient()
	if err != nil && !strings.Contains(err.Error(), "empty caBundle in clusterInterceptor spec") {
		t.Fatal(err)
	}
	if err == nil {
		t.Fatalf("test should fail as clusterinterceptor spec cabundle is empty")
	}
	if diff := cmp.Diff(c, &http.Client{}); diff != "" {
		t.Errorf("Diff: -want +got: %s", cmp.Diff(c, &http.Client{}))
	}
}

func TestGetHTTPClient(t *testing.T) {
	recorder, err := sink.NewRecorder()
	if err != nil {
		log.Fatal(err.Error())
	}
	ctx, _ := pkgtesting.SetupFakeContext(t)
	s := sinker{
		Logger:    logging.FromContext(ctx),
		Namespace: "",
		Args:      sink.Args{},
		Clients: sink.Clients{
			TriggersClient: faketriggersclient.Get(ctx),
		},
		Recorder: recorder,
		injCtx:   ctx,
	}

	icInformer := fakeClusterInterceptorinformer.Get(ctx)
	triggerClient := faketriggersclient.Get(ctx)

	ci := &v1alpha1.ClusterInterceptor{
		ObjectMeta: metav1.ObjectMeta{
			Name: "github",
			Labels: map[string]string{
				"server/type": "https",
			},
		},
		Spec: v1alpha1.ClusterInterceptorSpec{ClientConfig: v1alpha1.ClientConfig{
			CaBundle: []byte("LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUM5ekNDQXAyZ0F3SUJBZ0lSQUtQL1liSlF4Q2M5Y3JhVlBPMkhrK0V3Q2dZSUtvWkl6ajBFQXdJd1Z6RVUKTUJJR0ExVUVDaE1MYTI1aGRHbDJaUzVrWlhZeFB6QTlCZ05WQkFNVE5uUmxhM1J2YmkxMGNtbG5aMlZ5Y3kxagpiM0psTFdsdWRHVnlZMlZ3ZEc5eWN5NTBaV3QwYjI0dGNHbHdaV3hwYm1WekxuTjJZekFnRncweU1qQTJNakl4Ck5qRXhNVFZhR0E4eU1USXlNRFV5T1RFMk1URXhOVm93VnpFVU1CSUdBMVVFQ2hNTGEyNWhkR2wyWlM1a1pYWXgKUHpBOUJnTlZCQU1UTm5SbGEzUnZiaTEwY21sbloyVnljeTFqYjNKbExXbHVkR1Z5WTJWd2RHOXljeTUwWld0MApiMjR0Y0dsd1pXeHBibVZ6TG5OMll6QlpNQk1HQnlxR1NNNDlBZ0VHQ0NxR1NNNDlBd0VIQTBJQUJEdVVwTStEClVuZUozY1FieDc0cEpHTGgyTWkxREc1ZU5hNmRtSG5MeHhhQnZXTUxOOXp3Y2dLd2J2NHVJV0hsK2Rqb2laVWgKNFFRaElGUE8vS0NtUnJhamdnRkdNSUlCUWpBT0JnTlZIUThCQWY4RUJBTUNBb1F3SFFZRFZSMGxCQll3RkFZSQpLd1lCQlFVSEF3RUdDQ3NHQVFVRkJ3TUNNQThHQTFVZEV3RUIvd1FGTUFNQkFmOHdIUVlEVlIwT0JCWUVGSlcrCjdROVJiZlo3UDFBY0lXdEI2ckNTOWtneE1JSGdCZ05WSFJFRWdkZ3dnZFdDSVhSbGEzUnZiaTEwY21sbloyVnkKY3kxamIzSmxMV2x1ZEdWeVkyVndkRzl5YzRJeWRHVnJkRzl1TFhSeWFXZG5aWEp6TFdOdmNtVXRhVzUwWlhKagpaWEIwYjNKekxuUmxhM1J2Ymkxd2FYQmxiR2x1WlhPQ05uUmxhM1J2YmkxMGNtbG5aMlZ5Y3kxamIzSmxMV2x1CmRHVnlZMlZ3ZEc5eWN5NTBaV3QwYjI0dGNHbHdaV3hwYm1WekxuTjJZNEpFZEdWcmRHOXVMWFJ5YVdkblpYSnoKTFdOdmNtVXRhVzUwWlhKalpYQjBiM0p6TG5SbGEzUnZiaTF3YVhCbGJHbHVaWE11YzNaakxtTnNkWE4wWlhJdQpiRzlqWVd3d0NnWUlLb1pJemowRUF3SURTQUF3UlFJaEFOOE5SMTBJS0h4YUtXa0o0cFV1d3ljNFpmZG4rNTd6CnplN3RnS050b3hWREFpQWRhQlYvMlRDeStnV2tjUFR4cHo3aE91MHZ6bGZmeDhzV0Z3Wk5XdlVYUEE9PQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg=="),
		}},
	}
	if err := icInformer.Informer().GetIndexer().Add(ci); err != nil {
		t.Fatal(err)
	}

	if _, err := triggerClient.TriggersV1alpha1().ClusterInterceptors().Create(ctx, ci, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}

	c, err := s.getHTTPClient()
	if err != nil && !strings.Contains(err.Error(), "unable to parse cert from") {
		t.Fatal(err)
	}
	if err == nil {
		t.Fatalf("test should fail as its failed to parse cert")
	}
	if diff := cmp.Diff(c, &http.Client{}); diff != "" {
		t.Errorf("Diff: -want +got: %s", cmp.Diff(c, http.Client{}))
	}
}
