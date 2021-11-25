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

package test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

func TestGetResourcesFromClients(t *testing.T) {
	nsFoo := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo",
		},
	}
	nsTektonPipelines := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "tekton-pipelines",
		},
	}
	clusterTriggerBinding1 := &v1beta1.ClusterTriggerBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-clusterTriggerBinding1",
		},
	}
	clusterTriggerBinding2 := &v1beta1.ClusterTriggerBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-clusterTriggerBinding2",
		},
	}
	eventListener1 := &v1beta1.EventListener{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "foo",
			Name:      "my-eventlistener1",
		},
	}
	eventListener2 := &v1beta1.EventListener{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "foo",
			Name:      "my-eventlistener2",
		},
	}
	triggerBinding1 := &v1beta1.TriggerBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "foo",
			Name:      "my-triggerBinding1",
		},
	}
	triggerBinding2 := &v1beta1.TriggerBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "foo",
			Name:      "my-triggerBinding2",
		},
	}
	triggerTemplate1 := &v1beta1.TriggerTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "foo",
			Name:      "my-triggerTemplate1",
		},
	}
	triggerTemplate2 := &v1beta1.TriggerTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "foo",
			Name:      "my-triggerTemplate2",
		},
	}
	trigger1 := &v1beta1.Trigger{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "foo",
			Name:      "my-trigger1",
		},
	}
	trigger2 := &v1beta1.Trigger{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "foo",
			Name:      "my-trigger2",
		},
	}
	deployment1 := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "foo",
			Name:      "my-deployment1",
		},
	}
	deployment2 := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "foo",
			Name:      "my-deployment2",
		},
	}
	service1 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "foo",
			Name:      "my-service1",
		},
	}
	service2 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "foo",
			Name:      "my-service2",
		},
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "foo",
			Name:      "such-secret",
		},
	}
	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "foo",
			Name:      "my-pod-1",
		},
	}
	pod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "foo",
			Name:      "my-pod-2",
		},
	}

	cData := &duckv1.WithPod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "serving.knative.dev/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "knativeservice",
			Namespace: "foo",
		},
	}
	cData1 := &duckv1.WithPod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "serving.knative.dev/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "knativeservice1",
			Namespace: "foo",
		},
	}

	tests := []struct {
		name      string
		Resources Resources
	}{
		{
			name:      "empty",
			Resources: Resources{},
		},
		{
			name: "one resource each",
			Resources: Resources{
				Namespaces:             []*corev1.Namespace{nsFoo},
				ClusterTriggerBindings: []*v1beta1.ClusterTriggerBinding{clusterTriggerBinding1},
				EventListeners:         []*v1beta1.EventListener{eventListener1},
				TriggerBindings:        []*v1beta1.TriggerBinding{triggerBinding1},
				TriggerTemplates:       []*v1beta1.TriggerTemplate{triggerTemplate1},
				Triggers:               []*v1beta1.Trigger{trigger1},
				Deployments:            []*appsv1.Deployment{deployment1},
				Services:               []*corev1.Service{service1},
				Pods:                   []*corev1.Pod{pod1},
				WithPod:                []*duckv1.WithPod{cData},
			},
		},
		{
			name: "two resources each",
			Resources: Resources{
				Namespaces:             []*corev1.Namespace{nsFoo, nsTektonPipelines},
				ClusterTriggerBindings: []*v1beta1.ClusterTriggerBinding{clusterTriggerBinding1, clusterTriggerBinding2},
				EventListeners:         []*v1beta1.EventListener{eventListener1, eventListener2},
				TriggerBindings:        []*v1beta1.TriggerBinding{triggerBinding1, triggerBinding2},
				TriggerTemplates:       []*v1beta1.TriggerTemplate{triggerTemplate1, triggerTemplate2},
				Triggers:               []*v1beta1.Trigger{trigger1, trigger2},
				Deployments:            []*appsv1.Deployment{deployment1, deployment2},
				Services:               []*corev1.Service{service1, service2},
				Pods:                   []*corev1.Pod{pod1, pod2},
				WithPod:                []*duckv1.WithPod{cData, cData1},
			},
		},
		{
			name: "only namespaces",
			Resources: Resources{
				Namespaces: []*corev1.Namespace{nsFoo, nsTektonPipelines},
			},
		},
		{
			name: "only clustertriggerbindings",
			Resources: Resources{
				ClusterTriggerBindings: []*v1beta1.ClusterTriggerBinding{clusterTriggerBinding1, clusterTriggerBinding2},
			},
		},
		{
			name: "only eventlisteners (and namespaces)",
			Resources: Resources{
				Namespaces:     []*corev1.Namespace{nsFoo, nsTektonPipelines},
				EventListeners: []*v1beta1.EventListener{eventListener1, eventListener2},
			},
		},
		{
			name: "only triggerBindings (and namespaces)",
			Resources: Resources{
				Namespaces:      []*corev1.Namespace{nsFoo, nsTektonPipelines},
				TriggerBindings: []*v1beta1.TriggerBinding{triggerBinding1, triggerBinding2},
			},
		},
		{
			name: "only triggerTemplates (and namespaces)",
			Resources: Resources{
				Namespaces:       []*corev1.Namespace{nsFoo, nsTektonPipelines},
				TriggerTemplates: []*v1beta1.TriggerTemplate{triggerTemplate1, triggerTemplate2},
			},
		},
		{
			name: "only Deployments (and namespaces)",
			Resources: Resources{
				Namespaces:  []*corev1.Namespace{nsFoo, nsTektonPipelines},
				Deployments: []*appsv1.Deployment{deployment1, deployment2},
			},
		},
		{
			name: "only Services (and namespaces)",
			Resources: Resources{
				Namespaces: []*corev1.Namespace{nsFoo, nsTektonPipelines},
				Services:   []*corev1.Service{service1},
			},
		}, {
			name: "only secrets (and namespaces)",
			Resources: Resources{
				Namespaces: []*corev1.Namespace{nsFoo},
				Secrets:    []*corev1.Secret{secret},
			},
		}, {
			name: "only pods (and namespaces)",
			Resources: Resources{
				Namespaces: []*corev1.Namespace{nsFoo},
				Pods:       []*corev1.Pod{pod1},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _ := SetupFakeContext(t)
			clients := SeedResources(t, ctx, tt.Resources)
			actualResources, err := GetResourcesFromClients(clients)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tt.Resources, *actualResources); diff != "" {
				t.Errorf("Diff request body: -want +got: %s", cmp.Diff(tt.Resources, *actualResources))
			}
		})
	}
}
