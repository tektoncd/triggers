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

package resources

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/triggers/pkg/apis/config"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/kmeta"
)

var (
	eventListenerName     = "my-eventlistener"
	generatedResourceName = fmt.Sprintf("el-%s", eventListenerName)

	namespace = "test-pipelines"

	// Standard labels added by EL reconciler to the underlying el-deployments/services
	generatedLabels = map[string]string{
		"app.kubernetes.io/managed-by": "EventListener",
		"app.kubernetes.io/part-of":    "Triggers",
		"eventlistener":                eventListenerName,
	}
)

func TestGenerateLabels(t *testing.T) {
	staticResourceLabels := map[string]string{
		"app.kubernetes.io/managed-by": "EventListener",
		"app.kubernetes.io/part-of":    "Triggers",
	}

	expectedLabels := kmeta.UnionMaps(staticResourceLabels, map[string]string{"eventlistener": eventListenerName})
	actualLabels := GenerateLabels(eventListenerName, staticResourceLabels)
	if diff := cmp.Diff(expectedLabels, actualLabels); diff != "" {
		t.Errorf("mergeLabels() did not return expected. -want, +got: %s", diff)
	}
}

func TestObjectMeta(t *testing.T) {
	blockOwnerDeletion := true
	isController := true
	tests := []struct {
		name               string
		el                 *v1beta1.EventListener
		expectedObjectMeta metav1.ObjectMeta
	}{{
		name: "Empty EventListener",
		el: &v1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      eventListenerName,
				Namespace: "",
			},
		},
		expectedObjectMeta: metav1.ObjectMeta{
			Namespace: "",
			Name:      "",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion:         "triggers.tekton.dev/v1beta1",
				Kind:               "EventListener",
				Name:               eventListenerName,
				UID:                "",
				Controller:         &isController,
				BlockOwnerDeletion: &blockOwnerDeletion,
			}},
			Labels: generatedLabels,
		},
	}, {
		name: "EventListener with Configuration",
		el: &v1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      eventListenerName,
				Namespace: "",
			},
			Status: v1beta1.EventListenerStatus{
				Configuration: v1beta1.EventListenerConfig{
					GeneratedResourceName: "generatedName",
				},
			},
		},

		expectedObjectMeta: metav1.ObjectMeta{
			Namespace: "",
			Name:      "generatedName",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion:         "triggers.tekton.dev/v1beta1",
				Kind:               "EventListener",
				Name:               eventListenerName,
				UID:                "",
				Controller:         &isController,
				BlockOwnerDeletion: &blockOwnerDeletion,
			}},
			Labels: generatedLabels,
		},
	}, {
		name: "EventListener with Labels",
		el: &v1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      eventListenerName,
				Namespace: "",
				Labels: map[string]string{
					"k": "v",
				},
			},
		},
		expectedObjectMeta: metav1.ObjectMeta{
			Namespace: "",
			Name:      "",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion:         "triggers.tekton.dev/v1beta1",
				Kind:               "EventListener",
				Name:               eventListenerName,
				UID:                "",
				Controller:         &isController,
				BlockOwnerDeletion: &blockOwnerDeletion,
			}},
			Labels: kmeta.UnionMaps(map[string]string{"k": "v"}, generatedLabels),
		},
	}, {
		name: "EventListener with Annotation",
		el: &v1beta1.EventListener{
			ObjectMeta: metav1.ObjectMeta{
				Name:      eventListenerName,
				Namespace: "",
				Annotations: map[string]string{
					"k": "v",
				},
			},
		},
		expectedObjectMeta: metav1.ObjectMeta{
			Namespace: "",
			Name:      "",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion:         "triggers.tekton.dev/v1beta1",
				Kind:               "EventListener",
				Name:               eventListenerName,
				UID:                "",
				Controller:         &isController,
				BlockOwnerDeletion: &blockOwnerDeletion,
			}},
			Labels:      generatedLabels,
			Annotations: map[string]string{"k": "v"},
		},
	}}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			actualObjectMeta := ObjectMeta(tests[i].el, FilterLabels(context.Background(), tests[i].el.Labels), DefaultStaticResourceLabels)
			if diff := cmp.Diff(tests[i].expectedObjectMeta, actualObjectMeta); diff != "" {
				t.Errorf("generateObjectMeta() did not return expected. -want, +got: %s", diff)
			}
		})
	}
}

func TestFilterLabels(t *testing.T) {

	tests := []struct {
		name           string
		ctx            context.Context
		elLabels       map[string]string
		expectedLabels map[string]string
	}{{
		name: "exclusion pattern not defined",
		ctx: config.ToContext(context.Background(), &config.Config{
			FeatureFlags: &config.FeatureFlags{
				LabelsExclusionPattern: "",
			}}),
		elLabels: map[string]string{
			"tekton.dev/abc": "xyz",
			"tekton.dev/foo": "bar",
			"abc.com/key":    "value",
		},
		expectedLabels: map[string]string{
			"tekton.dev/abc": "xyz",
			"tekton.dev/foo": "bar",
			"abc.com/key":    "value",
		},
	}, {
		name: "exclusion pattern defined",
		ctx: config.ToContext(context.Background(), &config.Config{
			FeatureFlags: &config.FeatureFlags{
				LabelsExclusionPattern: "^tekton.*",
			}}),
		elLabels: map[string]string{
			"tekton.dev/abc": "xyz",
			"tekton.dev/foo": "bar",
			"abc.com/key":    "value",
		},
		expectedLabels: map[string]string{
			"abc.com/key": "value",
		},
	}}

	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			actualLabels := FilterLabels(tests[i].ctx, tests[i].elLabels)
			if diff := cmp.Diff(tests[i].expectedLabels, actualLabels); diff != "" {
				t.Errorf("generateObjectMeta() did not return expected. -want, +got: %s", diff)
			}
		})
	}
}
