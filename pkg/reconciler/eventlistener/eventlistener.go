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
	stdError "errors"
	"fmt"
	"strings"
	"sync"

	"github.com/tektoncd/triggers/pkg/apis/triggers/contexts"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	triggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	eventlistenerreconciler "github.com/tektoncd/triggers/pkg/client/injection/reconciler/triggers/v1beta1/eventlistener"
	listers "github.com/tektoncd/triggers/pkg/client/listers/triggers/v1beta1"
	dynamicduck "github.com/tektoncd/triggers/pkg/dynamic"
	"github.com/tektoncd/triggers/pkg/reconciler/eventlistener/resources"
	"golang.org/x/xerrors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	appsv1lister "k8s.io/client-go/listers/apps/v1"
	corev1lister "k8s.io/client-go/listers/core/v1"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/metrics"
	pkgreconciler "knative.dev/pkg/reconciler"
)

const (
	// ControllerName defines the name for EventListener Controller
	ControllerName = "EventListener"
	// eventListenerServicePortName defines service port name for EventListener Service
	eventListenerServicePortName = "http-listener"
	// eventListenerServiceTLSPortName defines service TLS port name for EventListener Service
	eventListenerServiceTLSPortName = "https-listener"
	// eventListenerMetricsPortName defines the metrics port name by the EventListener Container
	eventListenerMetricsPortName = "http-metrics"
	// eventListenerContainerPort defines service port for EventListener Service
	eventListenerContainerPort = 8080
	// eventListenerMetricsPort defines metrics port for EventListener Service
	eventListenerMetricsPort = 9000
	// GeneratedResourcePrefix is the name prefix for resources generated in the
	// EventListener reconciler
	GeneratedResourcePrefix = "el"

	defaultConfig = `{"level": "info","development": false,"sampling": {"initial": 100,"thereafter": 100},"outputPaths": ["stdout"],"errorOutputPaths": ["stderr"],"encoding": "json","encoderConfig": {"timeKey": "ts","levelKey": "level","nameKey": "logger","callerKey": "caller","messageKey": "msg","stacktraceKey": "stacktrace","lineEnding": "","levelEncoder": "","timeEncoder": "iso8601","durationEncoder": "","callerEncoder": ""}}`
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
	config             resources.Config
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
func (r *Reconciler) ReconcileKind(ctx context.Context, el *v1beta1.EventListener) pkgreconciler.Event {
	// Initial reconciliation
	el.Status.InitializeConditions()
	el.Status.Configuration.GeneratedResourceName = fmt.Sprintf("%s-%s", GeneratedResourcePrefix, el.Name)

	// We may be reading a version of the object that was stored at an older version
	// and may not have had all of the assumed default specified.
	el.SetDefaults(contexts.WithUpgradeViaDefaulting(ctx))

	if el.Spec.Resources.CustomResource != nil {
		kError := r.reconcileCustomObject(ctx, el)
		return wrapError(kError, nil)
	}
	deploymentReconcileError := r.reconcileDeployment(ctx, el)
	serviceReconcileError := r.reconcileService(ctx, el)
	if el.Spec.Resources.CustomResource == nil {
		el.Status.SetReadyCondition()
	}
	return wrapError(serviceReconcileError, deploymentReconcileError)
}

// FinalizeKind cleans up associated logging config maps when an EventListener is deleted
func (r *Reconciler) FinalizeKind(ctx context.Context, el *v1beta1.EventListener) pkgreconciler.Event {
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
	if err = r.KubeClientSet.CoreV1().ConfigMaps(el.Namespace).Delete(ctx, resources.EventListenerConfigMapName, metav1.DeleteOptions{}); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	logger.Infof("Deleted logging config map since last EventListener in the namespace %s was deleted", lastEL.Namespace)
	return nil
}

func (r *Reconciler) reconcileService(ctx context.Context, el *v1beta1.EventListener) error {
	service := resources.MakeService(el, r.config)

	existingService, err := r.serviceLister.Services(el.Namespace).Get(el.Status.Configuration.GeneratedResourceName)
	switch {
	case err == nil:
		el.Status.SetExistsCondition(v1beta1.ServiceExists, nil)
		el.Status.SetAddress(resources.ListenerHostname(el, r.config))

		// Copy over output spec fields.
		service.Spec.ClusterIP = existingService.Spec.ClusterIP

		// Copy any assigned NodePorts
		if service.Spec.Type == corev1.ServiceTypeNodePort &&
			existingService.Spec.Type == corev1.ServiceTypeNodePort {
			for i := range service.Spec.Ports {
				if i >= len(existingService.Spec.Ports) {
					break
				}
				service.Spec.Ports[i].NodePort = existingService.Spec.Ports[i].NodePort
			}
		}

		if !equality.Semantic.DeepEqual(existingService.Spec, service.Spec) ||
			!equality.Semantic.DeepEqual(existingService.Labels, service.Labels) ||
			!equality.Semantic.DeepEqual(existingService.Annotations, service.Annotations) {
			existingService = existingService.DeepCopy() // Don't modify the lister cache
			existingService.Labels = service.Labels
			existingService.Annotations = service.Annotations
			existingService.Spec = service.Spec
			if updated, err := r.KubeClientSet.CoreV1().Services(el.Namespace).Update(ctx, existingService, metav1.UpdateOptions{}); err != nil {
				logging.FromContext(ctx).Errorf("Error updating EventListener Service: %s", err)
				return err
			} else if existingService.ResourceVersion != updated.ResourceVersion {
				logging.FromContext(ctx).Infof("Updated EventListener Service %s in Namespace %s", existingService.Namespace, el.Namespace)
			}
		}

	case errors.IsNotFound(err):
		// Create the EventListener Service
		_, err = r.KubeClientSet.CoreV1().Services(el.Namespace).Create(ctx, service, metav1.CreateOptions{})
		el.Status.SetExistsCondition(v1beta1.ServiceExists, err)
		if err != nil {
			logging.FromContext(ctx).Errorf("Error creating EventListener Service: %s", err)
			return err
		}
		el.Status.SetAddress(resources.ListenerHostname(el, r.config))
		logging.FromContext(ctx).Infof("Created EventListener Service %s in Namespace %s", service.Name, el.Namespace)

	default:
		logging.FromContext(ctx).Error(err)
		return err
	}
	return nil
}

func (r *Reconciler) reconcileLoggingConfig(ctx context.Context, el *v1beta1.EventListener) error {
	if _, err := r.configmapLister.ConfigMaps(el.Namespace).Get(resources.EventListenerConfigMapName); errors.IsNotFound(err) {
		// create default config-logging ConfigMap
		if _, err := r.KubeClientSet.CoreV1().ConfigMaps(el.Namespace).Create(ctx, defaultLoggingConfigMap(), metav1.CreateOptions{}); err != nil {
			logging.FromContext(ctx).Errorf("Failed to create logging config: %s.  EventListener won't start.", err)
			return err
		}
	} else if err != nil {
		logging.FromContext(ctx).Errorf("Error retrieving ConfigMap %q: %s", resources.EventListenerConfigMapName, err)
		return err
	}
	return nil
}

func (r *Reconciler) reconcileObservabilityConfig(ctx context.Context, el *v1beta1.EventListener) error {
	if _, err := r.configmapLister.ConfigMaps(el.Namespace).Get(metrics.ConfigMapName()); errors.IsNotFound(err) {
		if _, err := r.KubeClientSet.CoreV1().ConfigMaps(el.Namespace).Create(ctx, defaultObservabilityConfigMap(), metav1.CreateOptions{}); err != nil {
			logging.FromContext(ctx).Errorf("Failed to create observability config: %s.  EventListener won't start.", err)
			return err
		}
	} else if err != nil {
		logging.FromContext(ctx).Errorf("Error retrieving ConfigMap %q: %s", metrics.ConfigMapName(), err)
		return err
	}
	return nil
}

func (r *Reconciler) reconcileDeployment(ctx context.Context, el *v1beta1.EventListener) error {
	// check logging config, create if it doesn't exist
	if err := r.reconcileLoggingConfig(ctx, el); err != nil {
		logging.FromContext(ctx).Error(err)
		return err
	}
	if err := r.reconcileObservabilityConfig(ctx, el); err != nil {
		logging.FromContext(ctx).Error(err)
		return err
	}

	deployment, err := resources.MakeDeployment(el, r.config)
	if err != nil {
		logging.FromContext(ctx).Error(err)
		return err
	}

	existingDeployment, err := r.deploymentLister.Deployments(el.Namespace).Get(el.Status.Configuration.GeneratedResourceName)
	switch {
	case err == nil:
		// TODO(mattmoor): Should this delete stuff for the CustomObject?  That path deletes Deployments,
		// so it seems asymmetrical for this path to not.

		el.Status.SetDeploymentConditions(existingDeployment.Status.Conditions)
		el.Status.SetExistsCondition(v1beta1.DeploymentExists, nil)

		// If the scale of the deployment is unspecified, then persist the current
		// scale of what is deployed.  This allows users to use HPA to automatically
		// (or manually themselves) scale the underlying deployment.
		if deployment.Spec.Replicas == nil {
			deployment.Spec.Replicas = existingDeployment.Spec.Replicas
		}

		if !equality.Semantic.DeepEqual(existingDeployment.Spec, deployment.Spec) ||
			!equality.Semantic.DeepEqual(existingDeployment.Labels, deployment.Labels) ||
			!equality.Semantic.DeepEqual(existingDeployment.Annotations, deployment.Annotations) {
			existingDeployment = existingDeployment.DeepCopy() // Don't modify the lister cache
			existingDeployment.Labels = deployment.Labels
			existingDeployment.Annotations = deployment.Annotations
			existingDeployment.Spec = deployment.Spec
			// If the spec/labels/annotations of what we want and got are different, then
			// issue an Update.  They may differ due to things like defaulting, so the
			// Update may not change anything, so only log if the ResourceVersion changes.
			if updated, err := r.KubeClientSet.AppsV1().Deployments(el.Namespace).Update(ctx, existingDeployment, metav1.UpdateOptions{}); err != nil {
				logging.FromContext(ctx).Errorf("Error updating EventListener Deployment: %s", err)
				return err
			} else if existingDeployment.ResourceVersion != updated.ResourceVersion {
				logging.FromContext(ctx).Infof("Updated EventListener Deployment %s in Namespace %s", existingDeployment.Name, el.Namespace)
			}
		}

	case errors.IsNotFound(err):
		// Create the EventListener Deployment
		deployment, err = r.KubeClientSet.AppsV1().Deployments(el.Namespace).Create(ctx, deployment, metav1.CreateOptions{})
		el.Status.SetExistsCondition(v1beta1.DeploymentExists, err)
		if err != nil {
			logging.FromContext(ctx).Errorf("Error creating EventListener Deployment: %s", err)
			return err
		}
		el.Status.SetDeploymentConditions(deployment.Status.Conditions)
		logging.FromContext(ctx).Infof("Created EventListener Deployment %s in Namespace %s", deployment.Name, el.Namespace)

	default:
		logging.FromContext(ctx).Error(err)
		return err
	}
	return nil
}

func (r *Reconciler) reconcileCustomObject(ctx context.Context, el *v1beta1.EventListener) error {
	// check logging config, create if it doesn't exist
	if err := r.reconcileLoggingConfig(ctx, el); err != nil {
		logging.FromContext(ctx).Error(err)
		return err
	}
	// TODO(mattmoor): This is missing reconciliation of the observability config, which reconcileDeployment has.
	// Given that the CustomObject passes the config name, this probably should have it, but it probably makes
	// the most sense to eliminate the configmap propagation altogether and translate that state into container
	// config (env, args).

	data, err := resources.MakeCustomObject(el, r.config)
	if err != nil {
		logging.FromContext(ctx).Errorf("unable to construct custom object", err)
		return err
	}

	gvr, _ := meta.UnsafeGuessKindToResource(data.GetObjectKind().GroupVersionKind())

	// TODO(mattmoor): Consider replacing this with duck.InformerFactory, it actually has a bug where
	// the podspecableTracker can only service a single GVR, despite multiple EventListener objects
	// being able to specify unique resource types (yikes).
	var watchError error
	r.onlyOnce.Do(func() {
		watchError = r.podspecableTracker.WatchOnDynamicObject(ctx, gvr)
	})
	if watchError != nil {
		logging.FromContext(ctx).Errorf("failed to watch on created custom object", watchError)
		return watchError
	}

	// TODO(mattmoor): We should look into using duck.InformerFactory to have this be a lister fetch.
	existingCustomObject, err := r.DynamicClientSet.Resource(gvr).Namespace(data.GetNamespace()).Get(ctx, data.GetName(), metav1.GetOptions{})
	switch {
	case err == nil:
		// Clean up any Deployments that may have existed for this listener.
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

		if !equality.Semantic.DeepEqual(data.GetLabels(), existingCustomObject.GetLabels()) ||
			!equality.Semantic.DeepEqual(data.GetAnnotations(), existingCustomObject.GetAnnotations()) ||
			!equality.Semantic.DeepEqual(data.Object["spec"], existingCustomObject.Object["spec"]) {
			data = data.DeepCopy()
			data.SetLabels(existingCustomObject.GetLabels())
			data.SetAnnotations(existingCustomObject.GetAnnotations())
			data.Object["spec"] = existingCustomObject.Object["spec"]

			if updated, err := r.DynamicClientSet.Resource(gvr).Namespace(data.GetNamespace()).Update(ctx, data, metav1.UpdateOptions{}); err != nil {
				logging.FromContext(ctx).Errorf("error updating to eventListener custom object: %v", err)
				return err
			} else if data.GetResourceVersion() != updated.GetResourceVersion() {
				logging.FromContext(ctx).Infof("Updated EventListener Custom Object %s in Namespace %s", data.GetName(), el.Namespace)
			}
		}

		// TODO(mattmoor): Consider replacing this stuff with the "addressable resolver"
		// from knative.dev/pkg, which is purpose built for this kind of thing.
		customConditions, url, err := dynamicduck.GetConditions(existingCustomObject)
		if customConditions == nil {
			// No status in the created object, it is weird but let's not fail
			logging.FromContext(ctx).Warn("empty status for the created custom object")
			return nil
		}
		if err != nil {
			return err
		}
		for _, cond := range customConditions {
			if cond.Type == apis.ConditionReady {
				if cond.Status != corev1.ConditionTrue {
					logging.FromContext(ctx).Warn("custom object is not yet ready")
					return stdError.New("custom object is not yet ready")
				}
			}
		}
		el.Status.SetConditionsForDynamicObjects(customConditions)
		if url != nil {
			el.Status.SetAddress(strings.Split(fmt.Sprintf("%v", url), "//")[1])
		}

	case errors.IsNotFound(err):
		createDynamicObject, err := r.DynamicClientSet.Resource(gvr).Namespace(data.GetNamespace()).Create(ctx, data, metav1.CreateOptions{})
		if err != nil {
			logging.FromContext(ctx).Errorf("Error creating EventListener Dynamic object: ", err)
			return err
		}
		logging.FromContext(ctx).Infof("Created EventListener Deployment %s in Namespace %s", createDynamicObject.GetName(), el.Namespace)

	default:
		logging.FromContext(ctx).Error(err)
		return err
	}
	return nil
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

func defaultLoggingConfigMap() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: resources.EventListenerConfigMapName},
		Data: map[string]string{
			"loglevel.eventlistener": "info",
			"zap-logger-config":      defaultConfig,
		},
	}
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
