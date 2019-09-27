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
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"
	"reflect"
	"strconv"

	"github.com/tektoncd/pipeline/pkg/names"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	listers "github.com/tektoncd/triggers/pkg/client/listers/triggers/v1alpha1"
	"github.com/tektoncd/triggers/pkg/reconciler"
	"golang.org/x/xerrors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/cache"

	"knative.dev/pkg/controller"
)

const (
	// eventListenerAgentName defines logging agent name for EventListener Controller
	eventListenerAgentName = "eventlistener-controller"
	// eventListenerControllerName defines name for EventListener Controller
	eventListenerControllerName = "EventListener"
	// Port defines the port for the EventListener to listen on
	Port = 8082
)

var (
	// The container that we use to run in the EventListener Pods
	elImage = flag.String("el-image", "override-with-el:latest",
		"The container image for the EventListener Pod.")
	// GeneratedResourceLabels is a map with all the labels that should be on
	// all resources generated by the EventListener
	GeneratedResourceLabels = map[string]string{
		"app.kubernetes.io/managed-by": "EventListener",
		"app.kubernetes.io/part-of":    "Triggers",
	}
	servicePort = corev1.ServicePort{
		Protocol: corev1.ProtocolTCP,
		Port:     int32(Port),
		TargetPort: intstr.IntOrString{
			IntVal: int32(Port),
		},
	}
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
func (c *Reconciler) Reconcile(ctx context.Context, key string) (returnError error) {
	c.Logger.Infof("event-listener-reconcile %s", key)
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		c.Logger.Errorf("invalid resource key: %s", key)
		return nil
	}

	// Get the EventListener resource
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
	// Initial reconciliation
	if equality.Semantic.DeepEqual(el.Status, v1alpha1.EventListenerStatus{}) {
		el.Status.InitializeConditions()
		el.Status.Configuration.GeneratedResourceName = names.SimpleNameGenerator.RestrictLengthWithRandomSuffix(el.Name)
	}

	// Reconcile this copy of the EventListener and then write back any status
	// updates
	reconcileErr := c.reconcile(ctx, el)
	if !equality.Semantic.DeepEqual(original.Status, el.Status) {
		_, err = c.TriggersClientSet.TektonV1alpha1().EventListeners(namespace).UpdateStatus(el)
		if err != nil {
			c.Logger.Warn("Failed to update EventListener status", err.Error())
			return err
		}
	}
	return reconcileErr
}

func (c *Reconciler) reconcile(ctx context.Context, el *v1alpha1.EventListener) error {
	// TODO(dibyom): Once #70 is merged, we should validate triggerTemplate here
	// and update the StatusCondition

	// TODO(vtereso): Create the resources within the reconciler, but restrict
	// updates within an admission webhook instead. The reconciler is resolving
	// behavior after it has been approved, which is from the wrong point of the
	// lifecycle and presents inherent problems.
	serviceReconcileError := c.reconcileService(el)
	deploymentReconcileError := c.reconcileDeployment(el)
	return wrapError(serviceReconcileError, deploymentReconcileError)
}

func reconcileObjectMeta(oldMeta *metav1.ObjectMeta, newMeta metav1.ObjectMeta) (updated bool) {
	if !reflect.DeepEqual(oldMeta.Labels, newMeta.Labels) {
		updated = true
		oldMeta.Labels = newMeta.Labels
	}
	if !reflect.DeepEqual(oldMeta.OwnerReferences, newMeta.OwnerReferences) {
		updated = true
		oldMeta.OwnerReferences = newMeta.OwnerReferences
	}
	return
}

func (c *Reconciler) reconcileService(el *v1alpha1.EventListener) error {
	service := &corev1.Service{
		ObjectMeta: GeneratedObjectMeta(el),
		Spec: corev1.ServiceSpec{
			Selector: mergeLabels(el.Labels, GeneratedResourceLabels),
			// Cannot be changed
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				servicePort,
			},
		},
	}
	existingService, err := c.KubeClientSet.CoreV1().Services(el.Namespace).Get(el.Status.Configuration.GeneratedResourceName, metav1.GetOptions{})
	switch {
	case err == nil:
		// Determine if reconciliation has to occur
		updated := reconcileObjectMeta(&existingService.ObjectMeta, service.ObjectMeta)
		if !reflect.DeepEqual(existingService.Spec.Selector, service.Spec.Selector) {
			existingService.Spec.Selector = service.Spec.Selector
			updated = true
		}
		if !reflect.DeepEqual(existingService.Spec.Ports, service.Spec.Ports) {
			existingService.Spec.Ports = service.Spec.Ports
			updated = true
		}
		if updated {
			_, err = c.KubeClientSet.CoreV1().Services(el.Namespace).Update(existingService)
			if err != nil {
				c.Logger.Errorf("Error updating EventListener Service: %s", err)
				return err
			} else {
				c.Logger.Infof("Updated EventListener Service %s in Namespace %s", existingService.Namespace, el.Namespace)
			}
		}
	case errors.IsNotFound(err):
		// Create the EventListener Service
		_, err = c.KubeClientSet.CoreV1().Services(el.Namespace).Create(service)
		el.Status.SetExistsCondition(v1alpha1.ServiceExists, err)
		if err != nil {
			c.Logger.Errorf("Error creating EventListener Service: %s", err)
			return err
		} else {
			el.Status.SetAddress(listenerHostname(service.Name, el.Namespace))
			c.Logger.Infof("Created EventListener Service %s in Namespace %s", service.Name, el.Namespace)
		}
	default:
		c.Logger.Error(err)
		return err
	}
	return nil
}

func (c *Reconciler) reconcileDeployment(el *v1alpha1.EventListener) error {
	labels := mergeLabels(el.Labels, GeneratedResourceLabels)
	var replicas int32 = 1
	container := corev1.Container{
		Name:  "event-listener",
		Image: *elImage,
		Ports: []corev1.ContainerPort{
			{
				ContainerPort: int32(Port),
				Protocol:      corev1.ProtocolTCP,
			},
		},
		Args: []string{
			"-el-name", el.Name,
			"-el-namespace", el.Namespace,
			"-port", strconv.Itoa(Port),
		},
	}
	deployment := &appsv1.Deployment{
		ObjectMeta: GeneratedObjectMeta(el),
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: el.Spec.ServiceAccountName,
					Containers:         []corev1.Container{container},
				},
			},
		},
	}
	existingDeployment, err := c.KubeClientSet.AppsV1().Deployments(el.Namespace).Get(el.Status.Configuration.GeneratedResourceName, metav1.GetOptions{})
	switch {
	case err == nil:
		el.Status.SetDeploymentConditions(existingDeployment.Status.Conditions)
		// Determine if reconciliation has to occur
		updated := reconcileObjectMeta(&existingDeployment.ObjectMeta, deployment.ObjectMeta)
		if existingDeployment.Spec.Replicas == nil || *existingDeployment.Spec.Replicas == 0 {
			existingDeployment.Spec.Replicas = &replicas
			updated = true
		}
		if existingDeployment.Spec.Selector != deployment.Spec.Selector {
			existingDeployment.Spec.Selector = deployment.Spec.Selector
			updated = true
		}
		if !reflect.DeepEqual(existingDeployment.Spec.Template.Labels, deployment.Spec.Template.Labels) {
			existingDeployment.Spec.Template.Labels = deployment.Spec.Template.Labels
			updated = true
		}
		if existingDeployment.Spec.Template.Spec.ServiceAccountName != deployment.Spec.Template.Spec.ServiceAccountName {
			existingDeployment.Spec.Template.Spec.ServiceAccountName = deployment.Spec.Template.Spec.ServiceAccountName
			updated = true
		}
		if len(existingDeployment.Spec.Template.Spec.Containers) == 0 ||
			len(existingDeployment.Spec.Template.Spec.Containers) > 1 {
			existingDeployment.Spec.Template.Spec.Containers = []corev1.Container{container}
			updated = true
		} else {
			if existingDeployment.Spec.Template.Spec.Containers[0].Name != container.Name {
				existingDeployment.Spec.Template.Spec.Containers[0].Name = container.Name
				updated = true
			}
			if existingDeployment.Spec.Template.Spec.Containers[0].Image != container.Image {
				existingDeployment.Spec.Template.Spec.Containers[0].Image = container.Image
				updated = true
			}
			if !reflect.DeepEqual(existingDeployment.Spec.Template.Spec.Containers[0].Ports, container.Ports) {
				existingDeployment.Spec.Template.Spec.Containers[0].Ports = container.Ports
				updated = true
			}
			if !reflect.DeepEqual(existingDeployment.Spec.Template.Spec.Containers[0].Args, container.Args) {
				existingDeployment.Spec.Template.Spec.Containers[0].Args = container.Args
				updated = true
			}
			if existingDeployment.Spec.Template.Spec.Containers[0].Command != nil {
				existingDeployment.Spec.Template.Spec.Containers[0].Command = nil
				updated = true
			}
		}
		if updated {
			_, err = c.KubeClientSet.AppsV1().Deployments(el.Namespace).Update(existingDeployment)
			if err != nil {
				c.Logger.Errorf("Error updating EventListener Deployment: %s", err)
				return err
			} else {
				c.Logger.Infof("Updated EventListener Deployment %s in Namespace %s", existingDeployment.Name, el.Namespace)
			}
		}
	case errors.IsNotFound(err):
		// Create the EventListener Deployment
		deployment, err = c.KubeClientSet.AppsV1().Deployments(el.Namespace).Create(deployment)
		el.Status.SetExistsCondition(v1alpha1.DeploymentExists, err)
		if err != nil {
			c.Logger.Errorf("Error creating EventListener Deployment: %s", err)
			return err
		} else {
			el.Status.SetDeploymentConditions(deployment.Status.Conditions)
			c.Logger.Infof("Created EventListener Deployment %s in Namespace %s", deployment.Name, el.Namespace)
		}
	default:
		c.Logger.Error(err)
		return err
	}
	return nil
}

// GenerateObjectMeta generates the object meta that should be used by all
// resources generated by the EventListener reconciler
func GeneratedObjectMeta(el *v1alpha1.EventListener) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace:       el.Namespace,
		Name:            el.Status.Configuration.GeneratedResourceName,
		OwnerReferences: []metav1.OwnerReference{*el.GetOwnerReference()},
		Labels:          mergeLabels(el.Labels, GeneratedResourceLabels),
	}
}

// mergeLabels merges values in map m2 into m1, which potentially clobbers values.
func mergeLabels(m1, m2 map[string]string) map[string]string {
	if m1 == nil {
		return m2
	}
	for k, v := range m2 {
		m1[k] = v
	}
	return m1
}

// wrapError wraps errors together. If one of the errors is nil, the other is
// returned.
func wrapError(err1, err2 error) error {
	if err1 == nil {
		return err2
	}
	if err2 == nil {
		return err1
	}
	return xerrors.Errorf("%s : %s", err1.Error(), err2.Error())
}

// listenerHostname returns the intended hostname for the EventListener service.
func listenerHostname(name, namespace string) string {
	return fmt.Sprintf("%s.%s.svc.cluster.local", name, namespace)
}
