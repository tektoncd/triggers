//go:build e2e
// +build e2e

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
	"fmt"
	"testing"

	"github.com/tektoncd/triggers/pkg/apis/triggers"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/ptr"
	knativetest "knative.dev/pkg/test"
)

/*
 * Test creating an EventListener with a large number of Triggers.
 * This is a regression test for issue #356.
 */
func TestEventListenerScale(t *testing.T) {
	c, namespace := setup(t)

	defer cleanupResources(t, c, namespace)
	knativetest.CleanupOnInterrupt(func() { cleanupResources(t, c, namespace) }, t.Logf)

	t.Log("Start EventListener Scale e2e test")

	saName := "scale-sa"
	createServiceAccount(t, c, namespace, saName)
	// Create an EventListener with 1000 Triggers
	var err error

	el := &triggersv1.EventListener{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-eventlistener",
			Namespace: namespace,
		},
		Spec: triggersv1.EventListenerSpec{
			ServiceAccountName: saName,
			Resources: triggersv1.Resources{
				KubernetesResource: &triggersv1.KubernetesResource{
					Replicas: ptr.Int32(3),
					WithPodSpec: duckv1.WithPodSpec{
						Template: duckv1.PodSpecable{
							Spec: corev1.PodSpec{
								NodeSelector: map[string]string{"beta.kubernetes.io/os": "linux"},
								Tolerations: []corev1.Toleration{{
									Key:      "key",
									Operator: "Equal",
									Value:    "value",
									Effect:   "NoSchedule",
								}},
							},
						},
					},
				},
			},
		},
	}
	for i := 0; i < 1000; i++ {
		trigger := triggersv1.EventListenerTrigger{
			Bindings: []*triggersv1.EventListenerBinding{{
				Kind:       triggersv1.NamespacedTriggerBindingKind,
				Ref:        "tb1",
				APIVersion: "v1alpha1",
			}},
			Template: &triggersv1.EventListenerTemplate{
				Ref:        ptr.String("my-triggertemplate"),
				APIVersion: "v1alpha1",
			},
		}
		trigger.Name = fmt.Sprintf("%d", i)
		el.Spec.Triggers = append(el.Spec.Triggers, trigger)
	}
	el, err = c.TriggersClient.TriggersV1alpha1().EventListeners(namespace).Create(context.Background(), el, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating EventListener: %s", err)
	}

	// Verify that the EventListener was created properly
	if err := WaitFor(eventListenerReady(t, c, namespace, el.Name)); err != nil {
		t.Fatalf("EventListener is not ready: %s", err)
	}
	t.Log("EventListener is ready")
}

func createServiceAccount(t *testing.T, c *clients, namespace, name string) {
	t.Helper()
	sa, err := c.KubeClient.CoreV1().ServiceAccounts(namespace).Create(context.Background(),
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{Name: name},
		}, metav1.CreateOptions{},
	)
	if err != nil {
		t.Fatalf("Error creating ServiceAccount: %s", err)
	}
	_, err = c.KubeClient.RbacV1().Roles(namespace).Create(context.Background(),
		&rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{Name: "sa-role"},
			Rules: []rbacv1.PolicyRule{{
				APIGroups: []string{triggers.GroupName},
				Resources: []string{"eventlisteners", "interceptors", "triggerbindings", "triggertemplates", "triggers"},
				Verbs:     []string{"get", "list", "watch"},
			}, {
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
				Verbs:     []string{"get", "list", "watch"},
			}},
		}, metav1.CreateOptions{},
	)
	if err != nil {
		t.Fatalf("Error creating Role: %s", err)
	}
	_, err = c.KubeClient.RbacV1().RoleBindings(namespace).Create(context.Background(),
		&rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: "sa-rolebinding"},
			Subjects: []rbacv1.Subject{{
				Kind:      "ServiceAccount",
				Name:      sa.Name,
				Namespace: namespace,
			}},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Role",
				Name:     "sa-role",
			},
		}, metav1.CreateOptions{},
	)
	if err != nil {
		t.Fatalf("Error creating RoleBinding: %s", err)
	}

	_, err = c.KubeClient.RbacV1().ClusterRoles().Create(context.Background(),
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{Name: "sa-clusterrole"},
			Rules: []rbacv1.PolicyRule{{
				APIGroups: []string{triggers.GroupName},
				Resources: []string{"clustertriggerbindings", "clusterinterceptors"},
				Verbs:     []string{"get", "list", "watch"},
			}, {
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"get", "list", "watch"},
			}},
		}, metav1.CreateOptions{},
	)
	if err != nil {
		t.Fatalf("Error creating ClusterRole: %s", err)
	}
	_, err = c.KubeClient.RbacV1().ClusterRoleBindings().Create(context.Background(),
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: "sa-clusterrolebinding"},
			Subjects: []rbacv1.Subject{{
				Kind:      "ServiceAccount",
				Name:      sa.Name,
				Namespace: namespace,
			}},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "sa-clusterrole",
			},
		}, metav1.CreateOptions{},
	)
	if err != nil {
		t.Fatalf("Error creating ClusterRoleBinding: %s", err)
	}
}

func cleanupResources(t *testing.T, c *clients, namespace string) {
	t.Helper()
	tearDown(t, c, namespace)

	// Cleanup cluster-scoped resources
	t.Logf("Deleting cluster-scoped resources")
	if err := c.KubeClient.RbacV1().ClusterRoles().Delete(context.Background(), "sa-clusterrole", metav1.DeleteOptions{}); err != nil {
		t.Errorf("Failed to delete clusterrole sa-clusterrole: %s", err)
	}
	if err := c.KubeClient.RbacV1().ClusterRoleBindings().Delete(context.Background(), "sa-clusterrolebinding", metav1.DeleteOptions{}); err != nil {
		t.Errorf("Failed to delete clusterrolebinding sa-clusterrolebinding: %s", err)
	}
}
