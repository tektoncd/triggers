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
	"flag"
	"reflect"

	"fmt"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	listers "github.com/tektoncd/triggers/pkg/client/listers/triggers/v1alpha1"
	"github.com/tektoncd/triggers/pkg/reconciler"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	"knative.dev/pkg/controller"
)

const (
	// eventListenerAgentName defines logging agent name for EventListener Controller
	eventListenerAgentName = "eventlistener-controller"
	// eventListenerControllerName defines name for EventListener Controller
	eventListenerControllerName = "EventListener"
	// RolePostfix is the postfix added to the EventListener name when creating Roles to review Trigger resources
	RolePostfix = "-trigger-role"
	// RolePostfix is the postfix added to the EventListener name when creating RoleBindings to review Trigger resources
	RoleBindingPostfix = "-trigger-rolebinding"
	// Port defines the port for the EventListener to listen on
	Port = 8082
)

var (
	// The container that we use to run in the EventListener Pods
	elImage = flag.String("el-image", "override-with-el:latest",
		"The container image for the EventListener Pod.")
)

// Reconciler implements controller.Reconciler for Configuration resources.
type Reconciler struct {
	*reconciler.Base

	// listers index properties about resources
	eventListenerLister listers.EventListenerLister
}

// Check that our Reconciler implements controller.Reconciler
var _ controller.Reconciler = (*Reconciler)(nil)

// Reconcile compares the actual state with the desired, and attempts to
// converge the two.
func (c *Reconciler) Reconcile(ctx context.Context, key string) error {
	c.Logger.Infof("event-listener-reconcile %s", key)
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		c.Logger.Errorf("invalid resource key: %s", key)
		return nil
	}

	// Get the Event Listener resource with this namespace/name
	original, err := c.eventListenerLister.EventListeners(namespace).Get(name)
	if errors.IsNotFound(err) {
		// The resource no longer exists, in which case we stop processing.
		c.Logger.Infof("EventListener %q in work queue no longer exists", key)
		return nil
	} else if err != nil {
		c.Logger.Errorf("Error retreiving EventListener %q: %s", name, err)
		return err
	}

	// Don't modify the informer's copy
	el := original.DeepCopy()

	// Propagate labels from EventListener to all generated resourced
	labels := make(map[string]string, len(el.ObjectMeta.Labels)+1)
	for key, val := range el.ObjectMeta.Labels {
		labels[key] = val
	}
	// Additional labels added to all generated resources
	// Potentially overrides
	labels["app"] = el.Name

	if err := reconcileAddressable(el, labels, c); err != nil {
		c.Logger.Errorf("Addressable reconciliation error: %s", err)
		return err
	}
	if err := reconcileRBAC(el, labels, c); err != nil {
		c.Logger.Errorf("RBAC reconciliation error: %s", err)
		return err
	}
	return nil
}

// reconcileAddressable reconciles the Service and Deployment for the specified EventListener.
// On create, the Service and Deployment are created.
// On update, the Deployment ServiceAccount is updated.
func reconcileAddressable(el *v1alpha1.EventListener, labels map[string]string, c *Reconciler) error {
	oldService, err := c.KubeClientSet.CoreV1().Services(el.Namespace).Get(el.Name, metav1.GetOptions{})
	switch {
	case err == nil:
		if !reflect.DeepEqual(oldService.Labels, labels) {
			oldService.Labels = labels
			oldService.Spec.Selector = labels
			// Update the EventListener Service
			_, err = c.KubeClientSet.CoreV1().Services(el.Namespace).Update(oldService)
			if err != nil {
				c.Logger.Errorf("Error updating EventListener Service: %s", err)
				return err
			}
			c.Logger.Info("Updated EventListener Service %s in Namespace %s", el.Name, el.Namespace)
		}
	case errors.IsNotFound(err):
		// TODO: This is an example Service that we will probably want to modify in the future
		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:       el.Namespace,
				Name:            el.Name,
				OwnerReferences: []metav1.OwnerReference{*el.GetOwnerReference()},
				Labels:          labels,
			},
			Spec: corev1.ServiceSpec{
				Selector: labels,
				Type:     corev1.ServiceTypeLoadBalancer,
				Ports: []corev1.ServicePort{
					{
						Protocol: corev1.ProtocolTCP,
						Port:     int32(Port),
					},
				},
			},
		}
		// Create the EventListener Service
		_, err = c.KubeClientSet.CoreV1().Services(el.Namespace).Create(service)
		if err != nil {
			c.Logger.Errorf("Error creating EventListener Service: %s", err)
			return err
		}
		c.Logger.Infof("Created EventListener Service %s in Namespace %s", el.Name, el.Namespace)
	default:
		c.Logger.Error(err)
		return err
	}

	oldDeployment, err := c.KubeClientSet.AppsV1().Deployments(el.Namespace).Get(el.Name, metav1.GetOptions{})
	switch {
	case err == nil:
		if oldDeployment.Spec.Template.Spec.ServiceAccountName != el.Spec.ServiceAccountName ||
			!reflect.DeepEqual(oldDeployment.Labels, labels) {
			oldDeployment.Spec.Template.Spec.ServiceAccountName = el.Spec.ServiceAccountName
			oldDeployment.Spec.Selector.MatchLabels = labels
			oldDeployment.Labels = labels
			oldDeployment.Spec.Template.Labels = labels
			oldDeployment.Spec.Template.Spec.ServiceAccountName = el.Spec.ServiceAccountName
			// Update the EventListener Deployment
			_, err = c.KubeClientSet.AppsV1().Deployments(el.Namespace).Update(oldDeployment)
			if err != nil {
				c.Logger.Errorf("Error updating EventListener Deployment: %s", err)
				return err
			}
			c.Logger.Info("Updated EventListener Deployment %s in Namespace %s", el.Name, el.Namespace)
		}
	case errors.IsNotFound(err):
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:       el.Namespace,
				Name:            el.Name,
				OwnerReferences: []metav1.OwnerReference{*el.GetOwnerReference()},
				Labels:          labels,
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: labels,
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: labels,
					},
					Spec: corev1.PodSpec{
						ServiceAccountName: el.Spec.ServiceAccountName,
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
									corev1.EnvVar{
										Name:  "LISTENER_NAME",
										Value: el.Name,
									},
									corev1.EnvVar{
										Name:  "LISTENER_NAMESPACE",
										Value: el.Namespace,
									},
								},
							},
						},
					},
				},
			},
		}
		// Create the EventListener Deployment
		_, err = c.KubeClientSet.AppsV1().Deployments(el.Namespace).Create(deployment)
		if err != nil {
			c.Logger.Errorf("Error creating EventListener Deployment: %s", err)
			return err
		}
		c.Logger.Infof("Created EventListener Deployment %s in Namespace %s", el.Name, el.Namespace)
	default:
		c.Logger.Error(err)
		return err
	}
	return nil
}

// reconcileRBAC reconciles the Role and RoleBinding for the specified EventListener.
// On create, the Role and RoleBinding are created.
// On update, the RoleBinding ServiceAccount subject is updated.
func reconcileRBAC(el *v1alpha1.EventListener, labels map[string]string, c *Reconciler) error {
	roleName := fmt.Sprintf("%s%s", el.Name, RolePostfix)
	oldRole, err := c.KubeClientSet.RbacV1().Roles(el.Namespace).Get(roleName, metav1.GetOptions{})
	switch {
	case err == nil:
		if !reflect.DeepEqual(oldRole.Labels, labels) {
			oldRole.Labels = labels
			// Update the EventListener Role
			_, err = c.KubeClientSet.RbacV1().Roles(el.Namespace).Update(oldRole)
			if err != nil {
				c.Logger.Errorf("Error updating EventListener Role: %s", err)
				return err
			}
			c.Logger.Info("Updated EventListener Role %s in Namespace %s", el.Name, el.Namespace)
		}
	case errors.IsNotFound(err):
		role := &rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:            roleName,
				Namespace:       el.Namespace,
				OwnerReferences: []metav1.OwnerReference{*el.GetOwnerReference()},
				Labels:          labels,
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
		// Create the EventListener Role
		_, err = c.KubeClientSet.RbacV1().Roles(el.Namespace).Create(role)
		if err != nil {
			c.Logger.Errorf("Error creating EventListener Role: %s", err)
			return err
		}
		c.Logger.Infof("Created EventListener Role %s in Namespace %s", el.Name, el.Namespace)
	default:
		c.Logger.Error(err)
		return err
	}

	roleBindingName := fmt.Sprintf("%s%s", el.Name, RoleBindingPostfix)
	oldRoleBinding, err := c.KubeClientSet.RbacV1().RoleBindings(el.Namespace).Get(roleBindingName, metav1.GetOptions{})
	switch {
	case err == nil:
		if len(oldRoleBinding.Subjects) == 0 ||
			oldRoleBinding.Subjects[0].Name != el.Spec.ServiceAccountName ||
			!reflect.DeepEqual(oldRoleBinding, labels) {
			oldRoleBinding.Subjects = []rbacv1.Subject{
				rbacv1.Subject{
					Kind:      "ServiceAccount",
					Name:      el.Spec.ServiceAccountName,
					Namespace: el.Namespace,
				},
			}
			oldRoleBinding.Labels = labels
			// Update the EventListener RoleBinding
			_, err = c.KubeClientSet.RbacV1().RoleBindings(el.Namespace).Update(oldRoleBinding)
			if err != nil {
				c.Logger.Errorf("Error updating EventListener RoleBinding: %s", err)
				return err
			}
			c.Logger.Info("Updated EventListener RoleBinding %s in Namespace %s", el.Name, el.Namespace)
		}
	case errors.IsNotFound(err):
		roleBinding := &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:            roleBindingName,
				Namespace:       el.Namespace,
				OwnerReferences: []metav1.OwnerReference{*el.GetOwnerReference()},
				Labels:          labels,
			},
			Subjects: []rbacv1.Subject{
				rbacv1.Subject{
					Kind:      "ServiceAccount",
					Name:      el.Spec.ServiceAccountName,
					Namespace: el.Namespace,
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Role",
				Name:     roleName,
			},
		}
		// Create the EventListener RoleBinding
		_, err = c.KubeClientSet.RbacV1().RoleBindings(el.Namespace).Create(roleBinding)
		if err != nil {
			c.Logger.Errorf("Error creating EventListener RoleBinding: %s", err)
			return err
		}
		c.Logger.Infof("Created EventListener RoleBinding %s in Namespace %s", el.Name, el.Namespace)
	default:
		c.Logger.Error(err)
		return err
	}
	return nil
}
