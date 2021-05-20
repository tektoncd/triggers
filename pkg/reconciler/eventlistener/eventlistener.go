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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/tektoncd/triggers/pkg/apis/triggers/contexts"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	triggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	eventlistenerreconciler "github.com/tektoncd/triggers/pkg/client/injection/reconciler/triggers/v1alpha1/eventlistener"
	listers "github.com/tektoncd/triggers/pkg/client/listers/triggers/v1alpha1"
	dynamicduck "github.com/tektoncd/triggers/pkg/dynamic"
	"go.uber.org/zap"
	"golang.org/x/xerrors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	appsv1lister "k8s.io/client-go/listers/apps/v1"
	corev1lister "k8s.io/client-go/listers/core/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/metrics"
	"knative.dev/pkg/ptr"
	pkgreconciler "knative.dev/pkg/reconciler"
)

const (
	// ControllerName defines the name for EventListener Controller
	ControllerName = "EventListener"
	// eventListenerConfigMapName is for the automatically created ConfigMap
	eventListenerConfigMapName = "config-logging-triggers"
	// eventListenerServicePortName defines service port name for EventListener Service
	eventListenerServicePortName = "http-listener"
	// eventListenerServiceTLSPortName defines service TLS port name for EventListener Service
	eventListenerServiceTLSPortName = "https-listener"
	// eventListenerContainerPort defines the port exposed by the EventListener Container
	eventListenerContainerPort = 8000
	// GeneratedResourcePrefix is the name prefix for resources generated in the
	// EventListener reconciler
	GeneratedResourcePrefix = "el"

	defaultConfig = `{"level": "info","development": false,"sampling": {"initial": 100,"thereafter": 100},"outputPaths": ["stdout"],"errorOutputPaths": ["stderr"],"encoding": "json","encoderConfig": {"timeKey": "ts","levelKey": "level","nameKey": "logger","callerKey": "caller","messageKey": "msg","stacktraceKey": "stacktrace","lineEnding": "","levelEncoder": "","timeEncoder": "iso8601","durationEncoder": "","callerEncoder": ""}}`

	triggersMetricsDomain = "tekton.dev/triggers"
)

// Reconciler implements controller.Reconciler for Configuration resources.
type Reconciler struct {
	DynamicClientSet dynamic.Interface

	// KubeClientSet allows us to talk to the k8s for core APIs
	KubeClientSet kubernetes.Interface

	// TriggersClientSet allows us to configure triggers objects
	TriggersClientSet triggersclientset.Interface

	// listers index properties about resources
	configmapLister     corev1lister.ConfigMapLister
	deploymentLister    appsv1lister.DeploymentLister
	eventListenerLister listers.EventListenerLister
	serviceLister       corev1lister.ServiceLister

	// config is the configuration options that the Reconciler accepts.
	config             Config
	podspecableTracker dynamicduck.ListableTracker
	onlyOnce           sync.Once
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
	el.SetDefaults(contexts.WithUpgradeViaDefaulting(ctx))

	if el.Spec.Resources.CustomResource != nil {
		kError := r.reconcileCustomObject(ctx, logger, el)
		return wrapError(kError, nil)
	}
	deploymentReconcileError := r.reconcileDeployment(ctx, logger, el)
	serviceReconcileError := r.reconcileService(ctx, logger, el)
	if el.Spec.Resources.CustomResource == nil {
		el.Status.SetReadyCondition()
	}
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
	if lastEL.Namespace == r.config.SystemNamespace {
		logger.Infof("Not deleting logging config map since EventListener is in the same namespace (%s) as the controller", el.Namespace)
		return nil
	}
	if err = r.KubeClientSet.CoreV1().ConfigMaps(el.Namespace).Delete(ctx, eventListenerConfigMapName, metav1.DeleteOptions{}); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	logger.Infof("Deleted logging config map since last EventListener in the namespace %s was deleted", lastEL.Namespace)
	return nil
}

func reconcileObjectMeta(existing *metav1.ObjectMeta, desired metav1.ObjectMeta) (updated bool) {
	if !reflect.DeepEqual(existing.Labels, desired.Labels) {
		updated = true
		existing.Labels = desired.Labels
	}

	// TODO(dibyom): We should exclude propagation of some annotations such as `kubernetes.io/last-applied-revision`
	if !reflect.DeepEqual(existing.Annotations, mergeMaps(existing.Annotations, desired.Annotations)) {
		updated = true
		existing.Annotations = desired.Annotations
	}

	if !reflect.DeepEqual(existing.OwnerReferences, desired.OwnerReferences) {
		updated = true
		existing.OwnerReferences = desired.OwnerReferences
	}
	return
}

func (r *Reconciler) reconcileService(ctx context.Context, logger *zap.SugaredLogger, el *v1alpha1.EventListener) error {
	// for backward compatibility with original behavior
	var serviceType corev1.ServiceType
	if el.Spec.Resources.KubernetesResource != nil && el.Spec.Resources.KubernetesResource.ServiceType != "" {
		serviceType = el.Spec.Resources.KubernetesResource.ServiceType
	}

	servicePort := getServicePort(el, r.config)

	service := &corev1.Service{
		ObjectMeta: generateObjectMeta(el, r.config.StaticResourceLabels),
		Spec: corev1.ServiceSpec{
			Selector: GenerateResourceLabels(el.Name, r.config.StaticResourceLabels),
			Type:     serviceType,
			Ports:    []corev1.ServicePort{servicePort}},
	}
	existingService, err := r.serviceLister.Services(el.Namespace).Get(el.Status.Configuration.GeneratedResourceName)
	switch {
	case err == nil:
		// Determine if reconciliation has to occur
		updated := reconcileObjectMeta(&existingService.ObjectMeta, service.ObjectMeta)
		el.Status.SetExistsCondition(v1alpha1.ServiceExists, nil)
		el.Status.SetAddress(listenerHostname(service.Name, el.Namespace, int(servicePort.Port)))
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
			if _, err := r.KubeClientSet.CoreV1().Services(el.Namespace).Update(ctx, existingService, metav1.UpdateOptions{}); err != nil {
				logger.Errorf("Error updating EventListener Service: %s", err)
				return err
			}
			logger.Infof("Updated EventListener Service %s in Namespace %s", existingService.Namespace, el.Namespace)
		}
	case errors.IsNotFound(err):
		// Create the EventListener Service
		_, err = r.KubeClientSet.CoreV1().Services(el.Namespace).Create(ctx, service, metav1.CreateOptions{})
		el.Status.SetExistsCondition(v1alpha1.ServiceExists, err)
		if err != nil {
			logger.Errorf("Error creating EventListener Service: %s", err)
			return err
		}
		el.Status.SetAddress(listenerHostname(service.Name, el.Namespace, int(servicePort.Port)))
		logger.Infof("Created EventListener Service %s in Namespace %s", service.Name, el.Namespace)
	default:
		logger.Error(err)
		return err
	}
	return nil
}

func (r *Reconciler) reconcileLoggingConfig(ctx context.Context, logger *zap.SugaredLogger, el *v1alpha1.EventListener) error {
	if _, err := r.configmapLister.ConfigMaps(el.Namespace).Get(eventListenerConfigMapName); errors.IsNotFound(err) {
		// create default config-logging ConfigMap
		if _, err := r.KubeClientSet.CoreV1().ConfigMaps(el.Namespace).Create(ctx, defaultLoggingConfigMap(), metav1.CreateOptions{}); err != nil {
			logger.Errorf("Failed to create logging config: %s.  EventListener won't start.", err)
			return err
		}
	} else if err != nil {
		logger.Errorf("Error retrieving ConfigMap %q: %s", eventListenerConfigMapName, err)
		return err
	}
	return nil
}

func (r *Reconciler) reconcileObservabilityConfig(ctx context.Context, logger *zap.SugaredLogger, el *v1alpha1.EventListener) error {
	if _, err := r.configmapLister.ConfigMaps(el.Namespace).Get(metrics.ConfigMapName()); errors.IsNotFound(err) {
		if _, err := r.KubeClientSet.CoreV1().ConfigMaps(el.Namespace).Create(ctx, defaultObservabilityConfigMap(), metav1.CreateOptions{}); err != nil {
			logger.Errorf("Failed to create observability config: %s.  EventListener won't start.", err)
			return err
		}
	} else if err != nil {
		logger.Errorf("Error retrieving ConfigMap %q: %s", metrics.ConfigMapName(), err)
		return err
	}
	return nil
}

func (r *Reconciler) reconcileDeployment(ctx context.Context, logger *zap.SugaredLogger, el *v1alpha1.EventListener) error {
	// check logging config, create if it doesn't exist
	if err := r.reconcileLoggingConfig(ctx, logger, el); err != nil {
		logger.Error(err)
		return err
	}
	if err := r.reconcileObservabilityConfig(ctx, logger, el); err != nil {
		logger.Error(err)
		return err
	}

	// METRICS_PROMETHEUS_PORT defines the port exposed by the EventListener metrics endpoint
	// env METRICS_PROMETHEUS_PORT set by controller
	metricsPort, err := strconv.ParseInt(os.Getenv("METRICS_PROMETHEUS_PORT"), 10, 64)
	if err != nil {
		logger.Error(err)
		return err
	}
	container := getContainer(el, r.config, nil)
	container.Ports = append(container.Ports, corev1.ContainerPort{
		ContainerPort: int32(metricsPort),
		Protocol:      corev1.ProtocolTCP,
	})
	container.VolumeMounts = []corev1.VolumeMount{{
		Name:      "config-logging",
		MountPath: "/etc/config-logging",
	}}
	container.Env = append(container.Env, corev1.EnvVar{
		Name: "SYSTEM_NAMESPACE",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.namespace",
			}},
	})
	container.Env = append(container.Env, corev1.EnvVar{
		Name:  "CONFIG_OBSERVABILITY_NAME",
		Value: metrics.ConfigMapName(),
	})
	container.Env = append(container.Env, corev1.EnvVar{
		Name:  "METRICS_DOMAIN",
		Value: triggersMetricsDomain,
	})
	// METRICS_PROMETHEUS_PORT defines the port exposed by the EventListener metrics endpoint
	// env METRICS_PROMETHEUS_PORT set by controller
	container.Env = append(container.Env, corev1.EnvVar{
		Name:  "METRICS_PROMETHEUS_PORT",
		Value: os.Getenv("METRICS_PROMETHEUS_PORT"),
	})
	container = addCertsForSecureConnection(container, r.config)

	deployment := getDeployment(el, container, r.config)

	existingDeployment, err := r.deploymentLister.Deployments(el.Namespace).Get(el.Status.Configuration.GeneratedResourceName)
	switch {
	case err == nil:
		el.Status.SetDeploymentConditions(existingDeployment.Status.Conditions)
		el.Status.SetExistsCondition(v1alpha1.DeploymentExists, nil)

		// Determine if reconciliation has to occur
		updated := reconcileObjectMeta(&existingDeployment.ObjectMeta, deployment.ObjectMeta)
		if *existingDeployment.Spec.Replicas != *deployment.Spec.Replicas {
			if el.Spec.Resources.KubernetesResource != nil {
				if el.Spec.Resources.KubernetesResource.Replicas != nil {
					existingDeployment.Spec.Replicas = deployment.Spec.Replicas
					updated = true
				}
				// if no replicas found as part of el.Spec then replicas from existingDeployment will be considered
			}
		}
		if existingDeployment.Spec.Selector != deployment.Spec.Selector {
			existingDeployment.Spec.Selector = deployment.Spec.Selector
			updated = true
		}
		if !reflect.DeepEqual(existingDeployment.Spec.Template.Labels, deployment.Spec.Template.Labels) {
			existingDeployment.Spec.Template.Labels = deployment.Spec.Template.Labels
			updated = true
		}
		if !reflect.DeepEqual(existingDeployment.Spec.Template.Annotations, deployment.Spec.Template.Annotations) {
			existingDeployment.Spec.Template.Annotations = deployment.Spec.Template.Annotations
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
			if !reflect.DeepEqual(existingDeployment.Spec.Template.Spec.Containers[0].LivenessProbe, container.LivenessProbe) {
				existingDeployment.Spec.Template.Spec.Containers[0].LivenessProbe = container.LivenessProbe
				updated = true
			}
			if !reflect.DeepEqual(existingDeployment.Spec.Template.Spec.Containers[0].ReadinessProbe, container.ReadinessProbe) {
				existingDeployment.Spec.Template.Spec.Containers[0].ReadinessProbe = container.ReadinessProbe
				updated = true
			}
			if existingDeployment.Spec.Template.Spec.Containers[0].Command != nil {
				existingDeployment.Spec.Template.Spec.Containers[0].Command = nil
				updated = true
			}
			if !reflect.DeepEqual(existingDeployment.Spec.Template.Spec.Containers[0].Resources, container.Resources) {
				existingDeployment.Spec.Template.Spec.Containers[0].Resources = container.Resources
				updated = true
			}
			if !reflect.DeepEqual(existingDeployment.Spec.Template.Spec.Containers[0].Env, container.Env) {
				existingDeployment.Spec.Template.Spec.Containers[0].Env = container.Env
				updated = true
			}
			if !reflect.DeepEqual(existingDeployment.Spec.Template.Spec.Containers[0].VolumeMounts, deployment.Spec.Template.Spec.Containers[0].VolumeMounts) {
				existingDeployment.Spec.Template.Spec.Containers[0].VolumeMounts = container.VolumeMounts
				updated = true
			}
			if !reflect.DeepEqual(existingDeployment.Spec.Template.Spec.Volumes, deployment.Spec.Template.Spec.Volumes) {
				existingDeployment.Spec.Template.Spec.Volumes = deployment.Spec.Template.Spec.Volumes
				updated = true
			}
			if !reflect.DeepEqual(existingDeployment.Spec.Template.Spec.SecurityContext, deployment.Spec.Template.Spec.SecurityContext) && *r.config.SetSecurityContext {
				existingDeployment.Spec.Template.Spec.SecurityContext = deployment.Spec.Template.Spec.SecurityContext
				updated = true
			}
		}
		if updated {
			if _, err := r.KubeClientSet.AppsV1().Deployments(el.Namespace).Update(ctx, existingDeployment, metav1.UpdateOptions{}); err != nil {
				logger.Errorf("Error updating EventListener Deployment: %s", err)
				return err
			}
			logger.Infof("Updated EventListener Deployment %s in Namespace %s", existingDeployment.Name, el.Namespace)
		}
	case errors.IsNotFound(err):
		// Create the EventListener Deployment
		deployment, err = r.KubeClientSet.AppsV1().Deployments(el.Namespace).Create(ctx, deployment, metav1.CreateOptions{})
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

func (r *Reconciler) reconcileCustomObject(ctx context.Context, logger *zap.SugaredLogger, el *v1alpha1.EventListener) error {
	// check logging config, create if it doesn't exist
	if err := r.reconcileLoggingConfig(ctx, logger, el); err != nil {
		logger.Error(err)
		return err
	}

	original := &duckv1.WithPod{}
	decoder := json.NewDecoder(bytes.NewBuffer(el.Spec.Resources.CustomResource.Raw))
	if err := decoder.Decode(&original); err != nil {
		logger.Errorf("unable to decode object", err)
		return err
	}

	customObjectData := original.DeepCopy()

	namespace := original.GetNamespace()
	// Default the resource creation to the EventListenerNamespace if not found in the resource object
	if namespace == "" {
		namespace = el.GetNamespace()
	}

	container := getContainer(el, r.config, original)
	container.VolumeMounts = []corev1.VolumeMount{{
		Name:      "config-logging",
		MountPath: "/etc/config-logging",
		ReadOnly:  true,
	}}

	container.Env = append(container.Env, corev1.EnvVar{
		Name: "SYSTEM_NAMESPACE",
		// Cannot use FieldRef here because Knative Serving mask that field under feature gate
		// https://github.com/knative/serving/blob/master/pkg/apis/config/features.go#L48
		Value: el.Namespace,
	})
	container.Env = append(container.Env, corev1.EnvVar{
		Name:  "CONFIG_OBSERVABILITY_NAME",
		Value: metrics.ConfigMapName(),
	})
	container.Env = append(container.Env, corev1.EnvVar{
		Name:  "METRICS_DOMAIN",
		Value: triggersMetricsDomain,
	})
	// METRICS_PROMETHEUS_PORT defines the port exposed by the EventListener metrics endpoint
	// env METRICS_PROMETHEUS_PORT set by controller
	container.Env = append(container.Env, corev1.EnvVar{
		Name:  "METRICS_PROMETHEUS_PORT",
		Value: os.Getenv("METRICS_PROMETHEUS_PORT"),
	})

	podlabels := mergeMaps(el.Labels, GenerateResourceLabels(el.Name, r.config.StaticResourceLabels))

	podlabels = mergeMaps(podlabels, customObjectData.Labels)

	original.Labels = podlabels
	original.Annotations = customObjectData.Annotations
	original.Spec.Template.ObjectMeta = metav1.ObjectMeta{
		Name:        customObjectData.Spec.Template.Name,
		Labels:      customObjectData.Spec.Template.Labels,
		Annotations: customObjectData.Spec.Template.Annotations,
	}
	original.Spec.Template.Spec = corev1.PodSpec{
		Tolerations:        customObjectData.Spec.Template.Spec.Tolerations,
		NodeSelector:       customObjectData.Spec.Template.Spec.NodeSelector,
		ServiceAccountName: customObjectData.Spec.Template.Spec.ServiceAccountName,
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
	}
	marshaledData, err := json.Marshal(original)
	if err != nil {
		logger.Errorf("failed to marshal custom object", err)
		return err
	}
	data := new(unstructured.Unstructured)
	if err := data.UnmarshalJSON(marshaledData); err != nil {
		logger.Errorf("failed to unmarshal to unstructured object", err)
		return err
	}

	if data.GetName() == "" {
		data.SetName(el.Status.Configuration.GeneratedResourceName)
	}
	gvr, _ := meta.UnsafeGuessKindToResource(data.GetObjectKind().GroupVersionKind())

	data.SetOwnerReferences([]metav1.OwnerReference{*el.GetOwnerReference()})

	var watchError error
	r.onlyOnce.Do(func() {
		watchError = r.podspecableTracker.WatchOnDynamicObject(ctx, gvr)
	})
	if watchError != nil {
		logger.Errorf("failed to watch on created custom object", watchError)
		return err
	}

	existingCustomObject, err := r.DynamicClientSet.Resource(gvr).Namespace(namespace).Get(ctx, data.GetName(), metav1.GetOptions{})
	switch {
	case err == nil:
		if _, err := r.deploymentLister.Deployments(el.Namespace).Get(el.Status.Configuration.GeneratedResourceName); err == nil {
			if err := r.KubeClientSet.AppsV1().Deployments(el.Namespace).Delete(ctx, el.Status.Configuration.GeneratedResourceName,
				metav1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
				return err
			}
			if err = r.KubeClientSet.CoreV1().Services(el.Namespace).Delete(ctx, el.Status.Configuration.GeneratedResourceName,
				metav1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
				return err
			}
		}
		originalObject := &duckv1.WithPod{}
		existingObject := &duckv1.WithPod{}
		originalData, e := data.MarshalJSON()
		if e != nil {
			return e
		}
		if e := json.Unmarshal(originalData, &originalObject); e != nil {
			return e
		}
		updatedData, e := existingCustomObject.MarshalJSON()
		if e != nil {
			return e
		}
		if e := json.Unmarshal(updatedData, &existingObject); e != nil {
			return e
		}
		// Determine if reconciliation has to occur
		updated := reconcileObjectMeta(&existingObject.ObjectMeta, originalObject.ObjectMeta)
		if !reflect.DeepEqual(existingObject.Spec.Template.Name, originalObject.Spec.Template.Name) {
			existingObject.Spec.Template.Name = originalObject.Spec.Template.Name
			updated = true
		}
		if !reflect.DeepEqual(existingObject.Spec.Template.Labels, originalObject.Spec.Template.Labels) {
			existingObject.Spec.Template.Labels = originalObject.Spec.Template.Labels
			updated = true
		}
		if !reflect.DeepEqual(existingObject.Spec.Template.Annotations, originalObject.Spec.Template.Annotations) {
			existingObject.Spec.Template.Annotations = originalObject.Spec.Template.Annotations
			updated = true
		}
		if existingObject.Spec.Template.Spec.ServiceAccountName != originalObject.Spec.Template.Spec.ServiceAccountName {
			existingObject.Spec.Template.Spec.ServiceAccountName = originalObject.Spec.Template.Spec.ServiceAccountName
			updated = true
		}
		if !reflect.DeepEqual(existingObject.Spec.Template.Spec.Tolerations, originalObject.Spec.Template.Spec.Tolerations) {
			existingObject.Spec.Template.Spec.Tolerations = originalObject.Spec.Template.Spec.Tolerations
			updated = true
		}
		if !reflect.DeepEqual(existingObject.Spec.Template.Spec.NodeSelector, originalObject.Spec.Template.Spec.NodeSelector) {
			existingObject.Spec.Template.Spec.NodeSelector = originalObject.Spec.Template.Spec.NodeSelector
			updated = true
		}
		if len(existingObject.Spec.Template.Spec.Containers) == 0 ||
			len(existingObject.Spec.Template.Spec.Containers) > 1 {
			existingObject.Spec.Template.Spec.Containers = []corev1.Container{container}
			updated = true
		} else {
			if existingObject.Spec.Template.Spec.Containers[0].Name != container.Name {
				existingObject.Spec.Template.Spec.Containers[0].Name = container.Name
				updated = true
			}
			if existingObject.Spec.Template.Spec.Containers[0].Image != container.Image {
				existingObject.Spec.Template.Spec.Containers[0].Image = container.Image
				updated = true
			}
			if !reflect.DeepEqual(existingObject.Spec.Template.Spec.Containers[0].Ports, container.Ports) {
				existingObject.Spec.Template.Spec.Containers[0].Ports = container.Ports
				updated = true
			}
			if !reflect.DeepEqual(existingObject.Spec.Template.Spec.Containers[0].Args, container.Args) {
				existingObject.Spec.Template.Spec.Containers[0].Args = container.Args
				updated = true
			}
			if existingObject.Spec.Template.Spec.Containers[0].Command != nil {
				existingObject.Spec.Template.Spec.Containers[0].Command = nil
				updated = true
			}
			if !reflect.DeepEqual(existingObject.Spec.Template.Spec.Containers[0].Resources, container.Resources) {
				existingObject.Spec.Template.Spec.Containers[0].Resources = container.Resources
				updated = true
			}
			if !reflect.DeepEqual(existingObject.Spec.Template.Spec.Containers[0].Env, container.Env) {
				existingObject.Spec.Template.Spec.Containers[0].Env = container.Env
				updated = true
			}
			if !reflect.DeepEqual(existingObject.Spec.Template.Spec.Containers[0].VolumeMounts, originalObject.Spec.Template.Spec.Containers[0].VolumeMounts) {
				existingObject.Spec.Template.Spec.Containers[0].VolumeMounts = container.VolumeMounts
				updated = true
			}
			if !reflect.DeepEqual(existingObject.Spec.Template.Spec.Volumes, originalObject.Spec.Template.Spec.Volumes) {
				existingObject.Spec.Template.Spec.Volumes = originalObject.Spec.Template.Spec.Volumes
				updated = true
			}
		}

		// if dynamicduck.ReconcileCustomObject(existingCustomObject, data) {
		if updated {
			existingMarshaledData, err := json.Marshal(existingObject)
			if err != nil {
				logger.Errorf("failed to marshal custom object", err)
				return err
			}
			existingCustomObject = new(unstructured.Unstructured)
			if err := existingCustomObject.UnmarshalJSON(existingMarshaledData); err != nil {
				logger.Errorf("failed to unmarshal to unstructured object", err)
				return err
			}
			if _, err := r.DynamicClientSet.Resource(gvr).Namespace(namespace).Update(ctx, existingCustomObject, metav1.UpdateOptions{}); err != nil {
				logger.Errorf("error updating to eventListener custom object: %v", err)
				return err
			}
			logger.Infof("Updated EventListener Custom Object %s in Namespace %s", data.GetName(), el.Namespace)
		}

		customConditions, url, err := dynamicduck.GetConditions(existingCustomObject)
		if customConditions == nil {
			// No status in the created object, it is weird but let's not fail
			logger.Warn("empty status for the created custom object")
			return nil
		}
		if err != nil {
			return err
		}
		el.Status.SetConditionsForDynamicObjects(customConditions)
		if url != nil {
			el.Status.SetAddress(strings.Split(fmt.Sprintf("%v", url), "//")[1])
		}
	case errors.IsNotFound(err):
		createDynamicObject, err := r.DynamicClientSet.Resource(gvr).Namespace(namespace).Create(ctx, data, metav1.CreateOptions{})
		if err != nil {
			logger.Errorf("Error creating EventListener Dynamic object: ", err)
			return err
		}
		logger.Infof("Created EventListener Deployment %s in Namespace %s", createDynamicObject.GetName(), el.Namespace)
	default:
		logger.Error(err)
		return err
	}
	return nil
}

func getDeployment(el *v1alpha1.EventListener, container corev1.Container, c Config) *appsv1.Deployment {
	var (
		tolerations                          []corev1.Toleration
		nodeSelector, annotations, podlabels map[string]string
		serviceAccountName                   string
		securityContext                      corev1.PodSecurityContext
	)
	podlabels = mergeMaps(el.Labels, GenerateResourceLabels(el.Name, c.StaticResourceLabels))

	serviceAccountName = el.Spec.ServiceAccountName

	vol := []corev1.Volume{{
		Name: "config-logging",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: eventListenerConfigMapName,
				},
			},
		},
	}}

	for _, v := range container.Env {
		// If TLS related env are set then mount secret volume which will be used while starting the eventlistener.
		if v.Name == "TLS_CERT" {
			vol = append(vol, corev1.Volume{
				Name: "https-connection",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: v.ValueFrom.SecretKeyRef.Name,
					},
				},
			})
		}
	}
	var replicas = ptr.Int32(1)
	if el.Spec.Resources.KubernetesResource != nil {
		if el.Spec.Resources.KubernetesResource.Replicas != nil {
			replicas = el.Spec.Resources.KubernetesResource.Replicas
		}
		if len(el.Spec.Resources.KubernetesResource.Template.Spec.Tolerations) != 0 {
			tolerations = el.Spec.Resources.KubernetesResource.Template.Spec.Tolerations
		}
		if len(el.Spec.Resources.KubernetesResource.Template.Spec.NodeSelector) != 0 {
			nodeSelector = el.Spec.Resources.KubernetesResource.Template.Spec.NodeSelector
		}
		if el.Spec.Resources.KubernetesResource.Template.Spec.ServiceAccountName != "" {
			serviceAccountName = el.Spec.Resources.KubernetesResource.Template.Spec.ServiceAccountName
		}
		annotations = el.Spec.Resources.KubernetesResource.Template.Annotations
		podlabels = mergeMaps(podlabels, el.Spec.Resources.KubernetesResource.Template.Labels)
	}

	if *c.SetSecurityContext {
		securityContext = corev1.PodSecurityContext{
			RunAsNonRoot: ptr.Bool(true),
			RunAsUser:    ptr.Int64(65532),
		}
	}

	return &appsv1.Deployment{
		ObjectMeta: generateObjectMeta(el, c.StaticResourceLabels),
		Spec: appsv1.DeploymentSpec{
			Replicas: replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: GenerateResourceLabels(el.Name, c.StaticResourceLabels),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      podlabels,
					Annotations: annotations,
				},
				Spec: corev1.PodSpec{
					Tolerations:        tolerations,
					NodeSelector:       nodeSelector,
					ServiceAccountName: serviceAccountName,
					Containers:         []corev1.Container{container},
					Volumes:            vol,
					SecurityContext:    &securityContext,
				},
			},
		},
	}
}

func addCertsForSecureConnection(container corev1.Container, c Config) corev1.Container {
	var elCert, elKey string
	certEnv := map[string]*corev1.EnvVarSource{}
	for i := range container.Env {
		certEnv[container.Env[i].Name] = container.Env[i].ValueFrom
	}
	var scheme corev1.URIScheme
	if v, ok := certEnv["TLS_CERT"]; ok {
		elCert = "/etc/triggers/tls/" + v.SecretKeyRef.Key
	} else {
		elCert = ""
	}
	if v, ok := certEnv["TLS_KEY"]; ok {
		elKey = "/etc/triggers/tls/" + v.SecretKeyRef.Key
	} else {
		elKey = ""
	}

	if elCert != "" && elKey != "" {
		scheme = corev1.URISchemeHTTPS
		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
			Name:      "https-connection",
			ReadOnly:  true,
			MountPath: "/etc/triggers/tls",
		})
	} else {
		scheme = corev1.URISchemeHTTP
	}
	container.LivenessProbe = &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   "/live",
				Scheme: scheme,
				Port:   intstr.FromInt(eventListenerContainerPort),
			},
		},
		PeriodSeconds:    int32(*c.PeriodSeconds),
		FailureThreshold: int32(*c.FailureThreshold),
	}
	container.ReadinessProbe = &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   "/live",
				Scheme: scheme,
				Port:   intstr.FromInt(eventListenerContainerPort),
			},
		},
		PeriodSeconds:    int32(*c.PeriodSeconds),
		FailureThreshold: int32(*c.FailureThreshold),
	}
	container.Args = append(container.Args, "--tls-cert="+elCert, "--tls-key="+elKey)
	return container
}

func getContainer(el *v1alpha1.EventListener, c Config, pod *duckv1.WithPod) corev1.Container {
	var resources corev1.ResourceRequirements
	env := []corev1.EnvVar{}
	if el.Spec.Resources.KubernetesResource != nil {
		if len(el.Spec.Resources.KubernetesResource.Template.Spec.Containers) != 0 {
			resources = el.Spec.Resources.KubernetesResource.Template.Spec.Containers[0].Resources
			env = append(env, el.Spec.Resources.KubernetesResource.Template.Spec.Containers[0].Env...)
		}
	}
	// handle env and resources for custom object
	if pod != nil {
		if len(pod.Spec.Template.Spec.Containers) == 1 {
			for i := range pod.Spec.Template.Spec.Containers[0].Env {
				env = append(env, pod.Spec.Template.Spec.Containers[0].Env[i])
			}
			resources = pod.Spec.Template.Spec.Containers[0].Resources
		}
	}

	isMultiNS := false
	if len(el.Spec.NamespaceSelector.MatchNames) != 0 {
		isMultiNS = true
	}

	return corev1.Container{
		Name:  "event-listener",
		Image: *c.Image,
		Ports: []corev1.ContainerPort{{
			ContainerPort: int32(eventListenerContainerPort),
			Protocol:      corev1.ProtocolTCP,
		}},
		Resources: resources,
		Args: []string{
			"--el-name=" + el.Name,
			"--el-namespace=" + el.Namespace,
			"--port=" + strconv.Itoa(eventListenerContainerPort),
			"--readtimeout=" + strconv.FormatInt(*c.ReadTimeOut, 10),
			"--writetimeout=" + strconv.FormatInt(*c.WriteTimeOut, 10),
			"--idletimeout=" + strconv.FormatInt(*c.IdleTimeOut, 10),
			"--timeouthandler=" + strconv.FormatInt(*c.TimeOutHandler, 10),
			"--is-multi-ns=" + strconv.FormatBool(isMultiNS),
		},
		Env: env,
	}
}

func getServicePort(el *v1alpha1.EventListener, c Config) corev1.ServicePort {
	var elCert, elKey string

	servicePortName := eventListenerServicePortName
	servicePortPort := *c.Port

	certEnv := map[string]*corev1.EnvVarSource{}
	if el.Spec.Resources.KubernetesResource != nil {
		if len(el.Spec.Resources.KubernetesResource.Template.Spec.Containers) != 0 {
			for i := range el.Spec.Resources.KubernetesResource.Template.Spec.Containers[0].Env {
				certEnv[el.Spec.Resources.KubernetesResource.Template.Spec.Containers[0].Env[i].Name] =
					el.Spec.Resources.KubernetesResource.Template.Spec.Containers[0].Env[i].ValueFrom
			}
		}
	}

	if v, ok := certEnv["TLS_CERT"]; ok {
		elCert = v.SecretKeyRef.Key
	} else {
		elCert = ""
	}
	if v, ok := certEnv["TLS_KEY"]; ok {
		elKey = v.SecretKeyRef.Key
	} else {
		elKey = ""
	}

	if elCert != "" && elKey != "" {
		servicePortName = eventListenerServiceTLSPortName
		if *c.Port == DefaultPort {
			// We return port 8443 if TLS is enabled and the default HTTP port is set.
			// This effectively makes 8443 the default HTTPS port unless a user explicitly sets a different port.
			servicePortPort = 8443
		}
	}

	return corev1.ServicePort{
		Name:     servicePortName,
		Protocol: corev1.ProtocolTCP,
		Port:     int32(servicePortPort),
		TargetPort: intstr.IntOrString{
			IntVal: int32(eventListenerContainerPort),
		},
	}
}

// GenerateResourceLabels generates the labels to be used on all generated resources.
func GenerateResourceLabels(eventListenerName string, staticResourceLabels map[string]string) map[string]string {
	resourceLabels := make(map[string]string, len(staticResourceLabels)+1)
	for k, v := range staticResourceLabels {
		resourceLabels[k] = v
	}
	resourceLabels["eventlistener"] = eventListenerName
	return resourceLabels
}

// generateObjectMeta generates the object meta that should be used by all
// resources generated by the EventListener reconciler
func generateObjectMeta(el *v1alpha1.EventListener, staticResourceLabels map[string]string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace:       el.Namespace,
		Name:            el.Status.Configuration.GeneratedResourceName,
		OwnerReferences: []metav1.OwnerReference{*el.GetOwnerReference()},
		Labels:          mergeMaps(el.Labels, GenerateResourceLabels(el.Name, staticResourceLabels)),
		Annotations:     el.Annotations,
	}
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

// mergeMaps merges the values in the passed maps into a new map.
// Values within m2 potentially clobber m1 values.
func mergeMaps(m1, m2 map[string]string) map[string]string {
	merged := make(map[string]string, len(m1)+len(m2))
	for k, v := range m1 {
		merged[k] = v
	}
	for k, v := range m2 {
		merged[k] = v
	}
	return merged
}

func defaultObservabilityConfigMap() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: metrics.ConfigMapName()},
		Data: map[string]string{
			// TODO: Better nonempty config
			"_example": "See tekton-pipelines namespace for valid values",
		},
	}
}
