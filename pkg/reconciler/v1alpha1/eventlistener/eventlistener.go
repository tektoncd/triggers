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
	"reflect"
	"strconv"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	listers "github.com/tektoncd/triggers/pkg/client/listers/triggers/v1alpha1"
	"go.uber.org/zap"
	"golang.org/x/xerrors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	appsv1lister "k8s.io/client-go/listers/apps/v1"
	corev1lister "k8s.io/client-go/listers/core/v1"
	"knative.dev/pkg/logging"
	pkgreconciler "knative.dev/pkg/reconciler"

	triggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	eventlistenerreconciler "github.com/tektoncd/triggers/pkg/client/injection/reconciler/triggers/v1alpha1/eventlistener"
	"knative.dev/pkg/ptr"
)

const (
	// ControllerName defines the name for EventListener Controller
	ControllerName = "EventListener"
	// eventListenerConfigMapName is for the automatically created ConfigMap
	eventListenerConfigMapName = "config-logging-triggers"
	// eventListenerServicePortName defines service port name for EventListener Service
	eventListenerServicePortName = "http-listener"
	// GeneratedResourcePrefix is the name prefix for resources generated in the
	// EventListener reconciler
	GeneratedResourcePrefix = "el"

	defaultConfig = `{"level": "info","development": false,"sampling": {"initial": 100,"thereafter": 100},"outputPaths": ["stdout"],"errorOutputPaths": ["stderr"],"encoding": "json","encoderConfig": {"timeKey": "","levelKey": "level","nameKey": "logger","callerKey": "caller","messageKey": "msg","stacktraceKey": "stacktrace","lineEnding": "","levelEncoder": "","timeEncoder": "","durationEncoder": "","callerEncoder": ""}}`
)

var (
	// The container that we use to run in the EventListener Pods
	elImage = flag.String("el-image", "override-with-el:latest",
		"The container image for the EventListener Pod.")
	// ElPort defines the port for the EventListener to listen on
	ElPort = flag.Int("el-port", 8080,
		"The container port for the EventListener to listen on.")
	// PeriodSeconds defines Period Seconds for the EventListener Liveness and Readiness Probes
	PeriodSeconds = flag.Int("period-seconds", 10,
		"The Period Seconds for the EventListener Liveness and Readiness Probes.")
	// FailureThreshold defines the Failure Threshold for the EventListener Liveness and Readiness Probes
	FailureThreshold = flag.Int("failure-threshold", 3,
		"The Failure Threshold for the EventListener Liveness and Readiness Probes.")
	// StaticResourceLabels is a map with all the labels that should be on
	// all resources generated by the EventListener
	StaticResourceLabels = map[string]string{
		"app.kubernetes.io/managed-by": "EventListener",
		"app.kubernetes.io/part-of":    "Triggers",
	}
)

// Reconciler implements controller.Reconciler for Configuration resources.
type Reconciler struct {

	// KubeClientSet allows us to talk to the k8s for core APIs
	KubeClientSet kubernetes.Interface

	// TriggersClientSet allows us to configure triggers objects
	TriggersClientSet triggersclientset.Interface

	// listers index properties about resources
	configmapLister     corev1lister.ConfigMapLister
	deploymentLister    appsv1lister.DeploymentLister
	eventListenerLister listers.EventListenerLister
	serviceLister       corev1lister.ServiceLister

	// systemNamespace is the namespace where the reconciler is deployed
	systemNamespace string
}

var (
	// Check that our Reconciler implements eventlistenerreconciler.Interface
	_ eventlistenerreconciler.Interface = (*Reconciler)(nil)
	// Check that our Reconciler implements eventlistenerreconciler.Finalizer
	_ eventlistenerreconciler.Finalizer = (*Reconciler)(nil)
)

// ReconcileKind compares the actual state with the desired, and attempts to
// converge the two.
func (r *Reconciler) ReconcileKind(ctx context.Context, el *v1alpha1.EventListener) pkgreconciler.Event {
	// Initial reconciliation
	el.Status.InitializeConditions()
	el.Status.Configuration.GeneratedResourceName = fmt.Sprintf("%s-%s", GeneratedResourcePrefix, el.Name)

	logger := logging.FromContext(ctx)

	// We may be reading a version of the object that was stored at an older version
	// and may not have had all of the assumed default specified.
	el.SetDefaults(v1alpha1.WithUpgradeViaDefaulting(ctx))

	serviceReconcileError := r.reconcileService(logger, el)
	deploymentReconcileError := r.reconcileDeployment(logger, el)

	return wrapError(serviceReconcileError, deploymentReconcileError)
}

// FinalizeKind cleans up associated logging config maps when an EventListener is deleted
func (r *Reconciler) FinalizeKind(ctx context.Context, el *v1alpha1.EventListener) pkgreconciler.Event {
	logger := logging.FromContext(ctx)
	cfgs, err := r.eventListenerLister.EventListeners(el.Namespace).List(labels.Everything())
	if err != nil {
		return err
	}
	if len(cfgs) != 1 {
		logger.Infof("Not deleting logging config map since more than one EventListener present in the namespace %s", el.Namespace)
		return nil
	}
	// only one EL left
	lastEL := cfgs[0]
	if lastEL.Namespace == r.systemNamespace {
		logger.Infof("Not deleting logging config map since EventListener is in the same namespace (%s) as the controller", el.Namespace)
		return nil
	}
	if err = r.KubeClientSet.CoreV1().ConfigMaps(el.Namespace).Delete(eventListenerConfigMapName, &metav1.DeleteOptions{}); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	logger.Infof("Deleted logging config map since last EventListener in the namespace %s was deleted", lastEL.Namespace)
	return nil
}

func reconcileObjectMeta(oldMeta *metav1.ObjectMeta, newMeta metav1.ObjectMeta) (updated bool) {
	if !reflect.DeepEqual(oldMeta.Labels, newMeta.Labels) {
		updated = true
		oldMeta.Labels = newMeta.Labels
	}
	if !reflect.DeepEqual(oldMeta.Annotations, newMeta.Annotations) {
		updated = true
		oldMeta.Annotations = newMeta.Annotations
	}
	if !reflect.DeepEqual(oldMeta.OwnerReferences, newMeta.OwnerReferences) {
		updated = true
		oldMeta.OwnerReferences = newMeta.OwnerReferences
	}
	return
}

func (r *Reconciler) reconcileService(logger *zap.SugaredLogger, el *v1alpha1.EventListener) error {
	service := &corev1.Service{
		ObjectMeta: generateObjectMeta(el),
		Spec: corev1.ServiceSpec{
			Selector: GenerateResourceLabels(el.Name),
			Type:     el.Spec.ServiceType,
			Ports: []corev1.ServicePort{{
				Name:     eventListenerServicePortName,
				Protocol: corev1.ProtocolTCP,
				Port:     int32(*ElPort),
				TargetPort: intstr.IntOrString{
					IntVal: int32(*ElPort),
				}},
			},
		},
	}
	existingService, err := r.serviceLister.Services(el.Namespace).Get(el.Status.Configuration.GeneratedResourceName)
	switch {
	case err == nil:
		// Determine if reconciliation has to occur
		updated := reconcileObjectMeta(&existingService.ObjectMeta, service.ObjectMeta)
		el.Status.SetExistsCondition(v1alpha1.ServiceExists, nil)
		el.Status.SetAddress(listenerHostname(service.Name, el.Namespace, *ElPort))
		if !reflect.DeepEqual(existingService.Spec.Selector, service.Spec.Selector) {
			existingService.Spec.Selector = service.Spec.Selector
			updated = true
		}
		if existingService.Spec.Type != service.Spec.Type {
			existingService.Spec.Type = service.Spec.Type
			// When transitioning from NodePort or LoadBalancer to ClusterIP
			// we need to remove NodePort from Ports
			existingService.Spec.Ports = service.Spec.Ports
			updated = true
		}
		if !cmp.Equal(existingService.Spec.Ports, service.Spec.Ports, cmpopts.IgnoreFields(corev1.ServicePort{}, "NodePort")) {
			existingService.Spec.Ports = service.Spec.Ports
			updated = true
		}
		if updated {
			if _, err := r.KubeClientSet.CoreV1().Services(el.Namespace).Update(existingService); err != nil {
				logger.Errorf("Error updating EventListener Service: %s", err)
				return err
			}
			logger.Infof("Updated EventListener Service %s in Namespace %s", existingService.Namespace, el.Namespace)
		}
	case errors.IsNotFound(err):
		// Create the EventListener Service
		_, err = r.KubeClientSet.CoreV1().Services(el.Namespace).Create(service)
		el.Status.SetExistsCondition(v1alpha1.ServiceExists, err)
		if err != nil {
			logger.Errorf("Error creating EventListener Service: %s", err)
			return err
		}
		el.Status.SetAddress(listenerHostname(service.Name, el.Namespace, *ElPort))
		logger.Infof("Created EventListener Service %s in Namespace %s", service.Name, el.Namespace)
	default:
		logger.Error(err)
		return err
	}
	return nil
}

func (r *Reconciler) reconcileLoggingConfig(logger *zap.SugaredLogger, el *v1alpha1.EventListener) error {
	if _, err := r.configmapLister.ConfigMaps(el.Namespace).Get(eventListenerConfigMapName); errors.IsNotFound(err) {
		// create default config-logging ConfigMap
		if _, err := r.KubeClientSet.CoreV1().ConfigMaps(el.Namespace).Create(defaultLoggingConfigMap()); err != nil {
			logger.Errorf("Failed to create logging config: %s.  EventListener won't start.", err)
			return err
		}
	} else if err != nil {
		logger.Errorf("Error retrieving ConfigMap %q: %s", eventListenerConfigMapName, err)
		return err
	}
	return nil
}

func (r *Reconciler) reconcileDeployment(logger *zap.SugaredLogger, el *v1alpha1.EventListener) error {
	// check logging config, create if it doesn't exist
	if err := r.reconcileLoggingConfig(logger, el); err != nil {
		logger.Error(err)
		return err
	}

	labels := mergeLabels(el.Labels, GenerateResourceLabels(el.Name))
	var replicas = ptr.Int32(1)
	if el.Spec.Replicas != nil {
		replicas = el.Spec.Replicas
	}
	container := corev1.Container{
		Name:  "event-listener",
		Image: *elImage,
		Ports: []corev1.ContainerPort{{
			ContainerPort: int32(*ElPort),
			Protocol:      corev1.ProtocolTCP,
		}},
		LivenessProbe: &corev1.Probe{
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/live",
					Scheme: corev1.URISchemeHTTP,
					Port:   intstr.FromInt((*ElPort)),
				},
			},
			PeriodSeconds:    int32(*PeriodSeconds),
			FailureThreshold: int32(*FailureThreshold),
		},
		ReadinessProbe: &corev1.Probe{
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/live",
					Scheme: corev1.URISchemeHTTP,
					Port:   intstr.FromInt((*ElPort)),
				},
			},
			PeriodSeconds:    int32(*PeriodSeconds),
			FailureThreshold: int32(*FailureThreshold),
		},
		Args: []string{
			"-el-name", el.Name,
			"-el-namespace", el.Namespace,
			"-port", strconv.Itoa(*ElPort),
		},
		VolumeMounts: []corev1.VolumeMount{{
			Name:      "config-logging",
			MountPath: "/etc/config-logging",
		}},
		Env: []corev1.EnvVar{{
			Name: "SYSTEM_NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		}},
	}
	deployment := &appsv1.Deployment{
		ObjectMeta: generateObjectMeta(el),
		Spec: appsv1.DeploymentSpec{
			Replicas: replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: GenerateResourceLabels(el.Name),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Tolerations:        el.Spec.PodTemplate.Tolerations,
					NodeSelector:       el.Spec.PodTemplate.NodeSelector,
					ServiceAccountName: el.Spec.ServiceAccountName,
					Containers:         []corev1.Container{container},

					Volumes: []corev1.Volume{{
						Name: "config-logging",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: eventListenerConfigMapName,
								},
							},
						},
					}},
				},
			},
		},
	}
	existingDeployment, err := r.deploymentLister.Deployments(el.Namespace).Get(el.Status.Configuration.GeneratedResourceName)
	switch {
	case err == nil:
		el.Status.SetDeploymentConditions(existingDeployment.Status.Conditions)
		el.Status.SetExistsCondition(v1alpha1.DeploymentExists, nil)

		// Determine if reconciliation has to occur
		updated := reconcileObjectMeta(&existingDeployment.ObjectMeta, deployment.ObjectMeta)
		if *existingDeployment.Spec.Replicas != *deployment.Spec.Replicas {
			if el.Spec.Replicas != nil {
				existingDeployment.Spec.Replicas = replicas
				updated = true
			}
			// if no replicas found as part of el.Spec then replicas from existingDeployment will be considered
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
		if !reflect.DeepEqual(existingDeployment.Spec.Template.Spec.Tolerations, deployment.Spec.Template.Spec.Tolerations) {
			existingDeployment.Spec.Template.Spec.Tolerations = deployment.Spec.Template.Spec.Tolerations
			updated = true
		}
		if !reflect.DeepEqual(existingDeployment.Spec.Template.Spec.NodeSelector, deployment.Spec.Template.Spec.NodeSelector) {
			existingDeployment.Spec.Template.Spec.NodeSelector = deployment.Spec.Template.Spec.NodeSelector
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
			if len(existingDeployment.Spec.Template.Spec.Containers[0].VolumeMounts) == 0 {
				existingDeployment.Spec.Template.Spec.Containers[0].VolumeMounts = container.VolumeMounts
				updated = true
			}
			if !reflect.DeepEqual(existingDeployment.Spec.Template.Spec.Volumes, deployment.Spec.Template.Spec.Volumes) {
				existingDeployment.Spec.Template.Spec.Volumes = deployment.Spec.Template.Spec.Volumes
				updated = true
			}
		}
		if updated {
			if _, err := r.KubeClientSet.AppsV1().Deployments(el.Namespace).Update(existingDeployment); err != nil {
				logger.Errorf("Error updating EventListener Deployment: %s", err)
				return err
			}
			logger.Infof("Updated EventListener Deployment %s in Namespace %s", existingDeployment.Name, el.Namespace)
		}
	case errors.IsNotFound(err):
		// Create the EventListener Deployment
		deployment, err = r.KubeClientSet.AppsV1().Deployments(el.Namespace).Create(deployment)
		el.Status.SetExistsCondition(v1alpha1.DeploymentExists, err)
		if err != nil {
			logger.Errorf("Error creating EventListener Deployment: %s", err)
			return err
		}
		el.Status.SetDeploymentConditions(deployment.Status.Conditions)
		logger.Infof("Created EventListener Deployment %s in Namespace %s", deployment.Name, el.Namespace)
	default:
		logger.Error(err)
		return err
	}
	return nil
}

// GenerateResourceLabels generates the labels to be used on all generated resources.
func GenerateResourceLabels(eventListenerName string) map[string]string {
	resourceLabels := make(map[string]string, len(StaticResourceLabels)+1)
	for k, v := range StaticResourceLabels {
		resourceLabels[k] = v
	}
	resourceLabels["eventlistener"] = eventListenerName
	return resourceLabels
}

// generateObjectMeta generates the object meta that should be used by all
// resources generated by the EventListener reconciler
func generateObjectMeta(el *v1alpha1.EventListener) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace:       el.Namespace,
		Name:            el.Status.Configuration.GeneratedResourceName,
		OwnerReferences: []metav1.OwnerReference{*el.GetOwnerReference()},
		Labels:          mergeLabels(el.Labels, GenerateResourceLabels(el.Name)),
		Annotations:     el.Annotations,
	}
}

// mergeLabels merges the values in the passed maps into a new map.
// Values within m2 potentially clobber m1 values.
func mergeLabels(m1, m2 map[string]string) map[string]string {
	merged := make(map[string]string, len(m1)+len(m2))
	for k, v := range m1 {
		merged[k] = v
	}
	for k, v := range m2 {
		merged[k] = v
	}
	return merged
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
func listenerHostname(name, namespace string, port int) string {
	return fmt.Sprintf("%s.%s.svc.cluster.local:%d", name, namespace, port)
}

func defaultLoggingConfigMap() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: eventListenerConfigMapName},
		Data: map[string]string{
			"loglevel.eventlistener": "info",
			"zap-logger-config":      defaultConfig,
		},
	}
}
