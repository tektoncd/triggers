/*
Copyright 2019 The Knative Authors.

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

	"github.com/knative/pkg/controller"
	listers "github.com/tektoncd/triggers/pkg/client/listers/triggers/v1alpha1"
	"github.com/tektoncd/triggers/pkg/reconciler"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

const (
	// eventListenerAgentName defines logging agent name for EventListener Controller
	eventListenerAgentName = "eventlistener-controller"
	// eventListenerControllerName defines name for EventListener Controller
	eventListenerControllerName = "EventListener"
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
	c.Logger.Info("event-listener-reconcile")
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
		c.Logger.Infof("event listener %q in work queue no longer exists", key)
		return nil
	} else if err != nil {
		c.Logger.Errorf("Error retreiving EventListener %q: %s", name, err)
		return err
	}

	// Don't modify the informer's copy
	el := original.DeepCopy()

	// Propagate labels from EventListener to Deployment
	labels := make(map[string]string, len(el.ObjectMeta.Labels)+1)
	for key, val := range el.ObjectMeta.Labels {
		labels[key] = val
	}
	labels["app"] = el.Name

	// Create the EventListener Deployment
	c.Logger.Infof("creating EventListener %s deployment in namespace %s", el.Name, el.Namespace)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			// Create the EventListener's Deployment in the same Namespace as where
			// the EventListener was created
			Namespace: el.Namespace,
			// Give the Deployment the same name as the EventListener
			Name: el.Name,
			// If our EventListener is deleted, then its Deployment should be as well
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
					Containers: []corev1.Container{
						{
							Name:  "event-listener",
							Image: *elImage,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: int32(8082),
								},
							},
						},
					},
				},
			},
		},
	}
	oldDeployment, err := c.KubeClientSet.AppsV1().Deployments(el.Namespace).Get(el.Name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			c.Logger.Errorf("Error getting Deployments: %s", err)
			return err
		}

		// Create the EventListener Deployment
		_, err = c.KubeClientSet.AppsV1().Deployments(el.Namespace).Create(deployment)
		if err != nil {
			c.Logger.Errorf("Error creating EventListener Deployment: %s", err)
			return err
		}
	} else if !reflect.DeepEqual(oldDeployment, deployment) {
		// Update the EventListener Deployment
		_, err = c.KubeClientSet.AppsV1().Deployments(el.Namespace).Update(deployment)
		if err != nil {
			c.Logger.Errorf("Error updating EventListener Deployment: %s", err)
			return err
		}
	}

	// TODO: This is an example Service that we will probably want to modify in the future
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			// Create the EventListener's Service in the same Namespace as where the
			// EventListener was created
			Namespace: el.Namespace,
			// Give the Service the same name as the EventListener
			Name: el.Name,
			// If our EventListener is deleted, then its Service should be as well
			OwnerReferences: []metav1.OwnerReference{*el.GetOwnerReference()},
			Labels:          labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": el.Name},
			Type:     corev1.ServiceTypeLoadBalancer,
			Ports: []corev1.ServicePort{
				{
					Protocol: corev1.ProtocolTCP,
					Port:     int32(8082),
				},
			},
		},
	}
	oldService, err := c.KubeClientSet.CoreV1().Services(el.Namespace).Get(el.Name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			c.Logger.Errorf("Error getting Services: %s", err)
			return err
		}

		// Create the EventListener Service
		_, err = c.KubeClientSet.CoreV1().Services(el.Namespace).Create(service)
		if err != nil {
			c.Logger.Errorf("Error creating EventListener Service: %s", err)
			return err
		}
	} else if !reflect.DeepEqual(oldService, service) {
		// Update the EventListener Service
		_, err = c.KubeClientSet.CoreV1().Services(el.Namespace).Update(service)
		if err != nil {
			c.Logger.Errorf("Error updating EventListener Service: %s", err)
			return err
		}
	}

	return nil
}
