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

package eventlistener

import (
	"context"
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/pipeline/pkg/system"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"github.com/tektoncd/triggers/test"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/configmap"
	rtesting "knative.dev/pkg/reconciler/testing"
	"testing"
)

// getEventListenerTestAssets returns TestAssets that have been seeded with the
// given TestResources r where r represents the state of the system
func getEventListenerTestAssets(t *testing.T, r test.TestResources) (test.TestAssets, context.CancelFunc) {
	t.Helper()
	ctx, _ := rtesting.SetupFakeContext(t)
	ctx, cancel := context.WithCancel(ctx)
	clients := test.SeedTestResources(t, ctx, r)
	cmw := configmap.NewInformedWatcher(clients.Kube, system.GetNamespace())
	return test.TestAssets{
		Controller: NewController(ctx, cmw),
		Clients:    clients,
	}, cancel
}

func TestReconcile(t *testing.T) {
	eventListenerName := "my-eventlistener"
	eventListenerCreateLabels := map[string]string{"test": "create"}
	eventListenerUpdateLabels := map[string]string{"test": "update", "updated": "true"}
	overrideLabels := map[string]string{"app": eventListenerName}
	createLabels := test.MergeLabels(eventListenerCreateLabels, overrideLabels)
	updateLabels := test.MergeLabels(eventListenerUpdateLabels, overrideLabels)
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "tekton-pipelines",
		},
	}
	sa1 := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-serviceaccount",
			Namespace: namespace.Name,
		},
	}
	sa2 := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-serviceaccount2",
			Namespace: namespace.Name,
		},
	}

	eventListener := &v1alpha1.EventListener{
		ObjectMeta: metav1.ObjectMeta{
			Name:      eventListenerName,
			Namespace: namespace.Name,
			Labels:    eventListenerCreateLabels,
		},
		Spec: v1alpha1.EventListenerSpec{
			ServiceAccountName: sa1.Name,
			Triggers: []v1alpha1.Trigger{
				v1alpha1.Trigger{
					TriggerBinding: v1alpha1.TriggerBindingRef{
						Name: "my-triggerbinding",
					},
					TriggerTemplate: v1alpha1.TriggerTemplateRef{
						Name: "my-triggertemplate",
					},
				},
			},
		},
	}
	updatedEventListener := eventListener.DeepCopy()
	updatedEventListener.Labels = eventListenerUpdateLabels
	updatedEventListener.Spec.ServiceAccountName = sa2.Name

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:            eventListenerName,
			Namespace:       eventListener.Namespace,
			OwnerReferences: []metav1.OwnerReference{*eventListener.GetOwnerReference()},
			Labels:          createLabels,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: createLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: createLabels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: sa1.Name,
					Containers: []corev1.Container{
						{
							Name:  "event-listener",
							Image: *elImage,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: int32(Port),
								},
							},
							Env: []corev1.EnvVar{
								{Name: "LISTENER_NAME", Value: eventListener.Name},
								{Name: "LISTENER_NAMESPACE", Value: eventListener.Namespace},
							},
						},
					},
				},
			},
		},
	}
	updatedDeployment := deployment.DeepCopy()
	updatedDeployment.Spec.Selector.MatchLabels = updateLabels
	updatedDeployment.Labels = updateLabels
	updatedDeployment.Spec.Template.Labels = updateLabels
	updatedDeployment.Spec.Template.Spec.ServiceAccountName = updatedEventListener.Spec.ServiceAccountName

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            eventListenerName,
			Namespace:       eventListener.Namespace,
			OwnerReferences: []metav1.OwnerReference{*eventListener.GetOwnerReference()},
			Labels:          createLabels,
		},
		Spec: corev1.ServiceSpec{
			Selector: createLabels,
			Type:     corev1.ServiceTypeLoadBalancer,
			Ports: []corev1.ServicePort{
				{
					Protocol: corev1.ProtocolTCP,
					Port:     int32(Port),
				},
			},
		},
	}
	updatedService := service.DeepCopy()
	updatedService.Labels = updateLabels
	updatedService.Spec.Selector = updateLabels

	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:            fmt.Sprintf("%s%s", eventListenerName, RolePostfix),
			Namespace:       eventListener.Namespace,
			OwnerReferences: []metav1.OwnerReference{*eventListener.GetOwnerReference()},
			Labels:          createLabels,
		},
		Rules: []rbacv1.PolicyRule{
			rbacv1.PolicyRule{
				APIGroups: []string{
					"tekton.dev",
				},
				Resources: []string{
					"eventlisteners",
					"triggerbindings",
					"triggertemplates",
				},
				Verbs: []string{
					"get",
				},
			},
		},
	}
	updatedRole := role.DeepCopy()
	updatedRole.Labels = updateLabels

	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:            fmt.Sprintf("%s%s", eventListenerName, RoleBindingPostfix),
			Namespace:       eventListener.Namespace,
			OwnerReferences: []metav1.OwnerReference{*eventListener.GetOwnerReference()},
			Labels:          createLabels,
		},
		Subjects: []rbacv1.Subject{
			rbacv1.Subject{
				Kind:      "ServiceAccount",
				Name:      eventListener.Spec.ServiceAccountName,
				Namespace: eventListener.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     role.Name,
		},
	}
	updatedRoleBinding := roleBinding.DeepCopy()
	updatedRoleBinding.Labels = updateLabels
	updatedRoleBinding.Subjects[0].Name = updatedEventListener.Spec.ServiceAccountName

	tests := []struct {
		name               string
		key                string
		testResourcesStart test.TestResources
		testResourcesEnd   test.TestResources
		wantErr            bool
	}{
		{
			name: "create-eventlistener",
			key:  "tekton-pipelines/my-eventlistener",
			testResourcesStart: test.TestResources{
				Namespaces:      []*corev1.Namespace{namespace},
				ServiceAccounts: []*corev1.ServiceAccount{sa1, sa2},
				EventListeners:  []*v1alpha1.EventListener{eventListener},
			},
			testResourcesEnd: test.TestResources{
				Namespaces:      []*corev1.Namespace{namespace},
				ServiceAccounts: []*corev1.ServiceAccount{sa1, sa2},
				EventListeners:  []*v1alpha1.EventListener{eventListener},
				Deployments:     []*appsv1.Deployment{deployment},
				Services:        []*corev1.Service{service},
				Roles:           []*rbacv1.Role{role},
				RoleBindings:    []*rbacv1.RoleBinding{roleBinding},
			},
			wantErr: false,
		},
		{
			name: "update-eventlistener",
			key:  "tekton-pipelines/my-eventlistener",
			testResourcesStart: test.TestResources{
				Namespaces:      []*corev1.Namespace{namespace},
				ServiceAccounts: []*corev1.ServiceAccount{sa1, sa2},
				EventListeners:  []*v1alpha1.EventListener{updatedEventListener},
				Deployments:     []*appsv1.Deployment{deployment},
				Services:        []*corev1.Service{service},
				Roles:           []*rbacv1.Role{role},
				RoleBindings:    []*rbacv1.RoleBinding{roleBinding},
			},
			testResourcesEnd: test.TestResources{
				Namespaces:      []*corev1.Namespace{namespace},
				ServiceAccounts: []*corev1.ServiceAccount{sa1, sa2},
				EventListeners:  []*v1alpha1.EventListener{updatedEventListener},
				Deployments:     []*appsv1.Deployment{updatedDeployment},
				Services:        []*corev1.Service{updatedService},
				Roles:           []*rbacv1.Role{updatedRole},
				RoleBindings:    []*rbacv1.RoleBinding{updatedRoleBinding},
			},
			wantErr: false,
		},
		{
			name:               "delete-eventlistener",
			key:                "tekton-pipelines/my-eventlistener",
			testResourcesStart: test.TestResources{},
			testResourcesEnd:   test.TestResources{},
			wantErr:            false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup with testResourcesStart
			testAssets, cancel := getEventListenerTestAssets(t, tt.testResourcesStart)
			defer cancel()

			// Run Reconcile
			err := testAssets.Controller.Reconciler.Reconcile(context.Background(), tt.key)

			// Check error matches wantErr
			if (tt.wantErr && (err == nil)) || (!tt.wantErr && (err != nil)) {
				t.Errorf("eventlistener.Reconcile() error = %v, wantErr = %v", err, tt.wantErr)
			}

			// Check current resources match endTestResources
			actualTestResourcesEnd, err := test.GetTestResourcesFromClients(testAssets.Clients)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(*actualTestResourcesEnd, tt.testResourcesEnd); diff != "" {
				t.Errorf("eventlistener.Reconcile() diff testResourcesEnd actual vs expected: %s", diff)
			}
		})
	}
}
