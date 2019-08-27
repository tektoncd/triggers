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
	"context"
	"testing"

	// Link in the fakes so they get injected into injection.Fake
	fakepipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client/fake"
	faketriggersclient "github.com/tektoncd/triggers/pkg/client/injection/client/fake"
	fakeeventlistenerinformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1alpha1/eventlistener/fake"
	fakekubeclient "knative.dev/pkg/injection/clients/kubeclient/fake"

	fakepipelineclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	faketriggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned/fake"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"
	"knative.dev/pkg/controller"
)

// TestResources represents the desired state of the system (i.e. existing resources)
// to seed controllers with.
type TestResources struct {
	Namespaces      []*corev1.Namespace
	EventListeners  []*v1alpha1.EventListener
	Deployments     []*appsv1.Deployment
	Services        []*corev1.Service
	ServiceAccounts []*corev1.ServiceAccount
	Roles           []*rbacv1.Role
	RoleBindings    []*rbacv1.RoleBinding
}

// Clients holds references to clients which are useful for reconciler tests.
type Clients struct {
	Kube     *fakekubeclientset.Clientset
	Triggers *faketriggersclientset.Clientset
	Pipeline *fakepipelineclientset.Clientset
}

// TestAssets holds references to the controller and clients.
type TestAssets struct {
	Controller *controller.Impl
	Clients    Clients
}

// SeedTestResources returns Clients populated with the given TestResources
func SeedTestResources(t *testing.T, ctx context.Context, r TestResources) Clients {
	t.Helper()
	c := Clients{
		Kube:     fakekubeclient.Get(ctx),
		Triggers: faketriggersclient.Get(ctx),
		Pipeline: fakepipelineclient.Get(ctx),
	}

	// Setup fake informer for reconciler tests
	elInformer := fakeeventlistenerinformer.Get(ctx)

	// Create Namespaces
	for _, ns := range r.Namespaces {
		if _, err := c.Kube.CoreV1().Namespaces().Create(ns); err != nil {
			t.Fatal(err)
		}
	}
	// Create TestResources
	for _, el := range r.EventListeners {
		if err := elInformer.Informer().GetIndexer().Add(el); err != nil {
			t.Fatal(err)
		}
		if _, err := c.Triggers.TektonV1alpha1().EventListeners(el.Namespace).Create(el); err != nil {
			t.Fatal(err)
		}
	}
	for _, d := range r.Deployments {
		if _, err := c.Kube.AppsV1().Deployments(d.Namespace).Create(d); err != nil {
			t.Fatal(err)
		}
	}
	for _, svc := range r.Services {
		if _, err := c.Kube.CoreV1().Services(svc.Namespace).Create(svc); err != nil {
			t.Fatal(err)
		}
	}
	for _, sa := range r.ServiceAccounts {
		if _, err := c.Kube.CoreV1().ServiceAccounts(sa.Namespace).Create(sa); err != nil {
			t.Fatal(err)
		}
	}
	for _, role := range r.Roles {
		if _, err := c.Kube.RbacV1().Roles(role.Namespace).Create(role); err != nil {
			t.Fatal(err)
		}
	}
	for _, rb := range r.RoleBindings {
		if _, err := c.Kube.RbacV1().RoleBindings(rb.Namespace).Create(rb); err != nil {
			t.Fatal(err)
		}
	}
	c.Kube.ClearActions()
	c.Triggers.ClearActions()
	c.Pipeline.ClearActions()
	return c
}

// GetTestResourcesFromClients returns the TestResources in the Clients provided
// Precondition: all Namespaces used in TestResources must be listed in TestResources.Namespaces
func GetTestResourcesFromClients(c Clients) (*TestResources, error) {
	testResources := &TestResources{}
	nsList, err := c.Kube.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, ns := range nsList.Items {
		// Add Namespace
		testResources.Namespaces = append(testResources.Namespaces, ns.DeepCopy())
		// Add EventListeners
		elList, err := c.Triggers.TektonV1alpha1().EventListeners(ns.Name).List(metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, el := range elList.Items {
			testResources.EventListeners = append(testResources.EventListeners, el.DeepCopy())
		}
		// Add Deployments
		dList, err := c.Kube.AppsV1().Deployments(ns.Name).List(metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, d := range dList.Items {
			testResources.Deployments = append(testResources.Deployments, d.DeepCopy())
		}
		// Add Services
		svcList, err := c.Kube.CoreV1().Services(ns.Name).List(metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, svc := range svcList.Items {
			testResources.Services = append(testResources.Services, svc.DeepCopy())
		}
		// Add ServiceAccounts
		saList, err := c.Kube.CoreV1().ServiceAccounts(ns.Name).List(metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, sa := range saList.Items {
			testResources.ServiceAccounts = append(testResources.ServiceAccounts, sa.DeepCopy())
		}
		// Add Roles
		roleList, err := c.Kube.RbacV1().Roles(ns.Name).List(metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, role := range roleList.Items {
			testResources.Roles = append(testResources.Roles, role.DeepCopy())
		}
		// Add RoleBindings
		roleBindingList, err := c.Kube.RbacV1().RoleBindings(ns.Name).List(metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, roleBinding := range roleBindingList.Items {
			testResources.RoleBindings = append(testResources.RoleBindings, roleBinding.DeepCopy())
		}
	}
	return testResources, nil
}
