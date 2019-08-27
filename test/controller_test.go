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
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	rtesting "knative.dev/pkg/reconciler/testing"
)

func TestGetTestResourcesFromClients(t *testing.T) {
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
	eventListener1 := &v1alpha1.EventListener{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "foo",
			Name:      "my-eventlistener1",
		},
	}
	eventListener2 := &v1alpha1.EventListener{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "foo",
			Name:      "my-eventlistener2",
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
	sa1 := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "foo",
			Name:      "my-sa1",
		},
	}
	sa2 := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "foo",
			Name:      "my-sa2",
		},
	}
	role1 := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "foo",
			Name:      "my-role1",
		},
	}
	role2 := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "foo",
			Name:      "my-role2",
		},
	}
	roleBinding1 := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "foo",
			Name:      "my-rolebinding1",
		},
	}
	roleBinding2 := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "foo",
			Name:      "my-rolebinding2",
		},
	}

	tests := []struct {
		name          string
		testResources TestResources
	}{
		{
			name:          "empty",
			testResources: TestResources{},
		},
		{
			name: "one resource each",
			testResources: TestResources{
				Namespaces:      []*corev1.Namespace{nsFoo},
				EventListeners:  []*v1alpha1.EventListener{eventListener1},
				Deployments:     []*appsv1.Deployment{deployment1},
				Services:        []*corev1.Service{service1},
				ServiceAccounts: []*corev1.ServiceAccount{sa1},
				Roles:           []*rbacv1.Role{role1},
				RoleBindings:    []*rbacv1.RoleBinding{roleBinding1},
			},
		},
		{
			name: "two resources each",
			testResources: TestResources{
				Namespaces:      []*corev1.Namespace{nsFoo, nsTektonPipelines},
				EventListeners:  []*v1alpha1.EventListener{eventListener1, eventListener2},
				Deployments:     []*appsv1.Deployment{deployment1, deployment2},
				Services:        []*corev1.Service{service1, service2},
				ServiceAccounts: []*corev1.ServiceAccount{sa1, sa2},
				Roles:           []*rbacv1.Role{role1, role2},
				RoleBindings:    []*rbacv1.RoleBinding{roleBinding1, roleBinding2},
			},
		},
		{
			name: "only namespaces",
			testResources: TestResources{
				Namespaces: []*corev1.Namespace{nsFoo, nsTektonPipelines},
			},
		},
		{
			name: "only eventlisteners (and namespaces)",
			testResources: TestResources{
				Namespaces:     []*corev1.Namespace{nsFoo, nsTektonPipelines},
				EventListeners: []*v1alpha1.EventListener{eventListener1, eventListener2},
			},
		},
		{
			name: "only Deployments (and namespaces)",
			testResources: TestResources{
				Namespaces:  []*corev1.Namespace{nsFoo, nsTektonPipelines},
				Deployments: []*appsv1.Deployment{deployment1, deployment2},
			},
		},
		{
			name: "only Services (and namespaces)",
			testResources: TestResources{
				Namespaces: []*corev1.Namespace{nsFoo, nsTektonPipelines},
				Services:   []*corev1.Service{service1},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _ := rtesting.SetupFakeContext(t)
			clients := SeedTestResources(t, ctx, tt.testResources)
			actualTestResources, err := GetTestResourcesFromClients(clients)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(*actualTestResources, tt.testResources) {
				t.Errorf("GetTestResourcesFromClients = %v, expected %v", *actualTestResources, tt.testResources)
				t.Log(cmp.Diff(*actualTestResources, tt.testResources))
			}
		})
	}
}
