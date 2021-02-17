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
	// eventListenerServiceTLSPortName defines service TLS port name for EventListener Service
	eventListenerServiceTLSPortName = "https-listener"
	// eventListenerContainerPort defines the port exposed by the EventListener Container
	eventListenerContainerPort = 8000
	// GeneratedResourcePrefix is the name prefix for resources generated in the
	// EventListener reconciler
	GeneratedResourcePrefix = "el"

	defaultConfig = `{"level": "info","development": false,"sampling": {"initial": 100,"thereafter": 100},"outputPaths": ["stdout"],"errorOutputPaths": ["stderr"],"encoding": "json","encoderConfig": {"timeKey": "ts","levelKey": "level","nameKey": "logger","callerKey": "caller","messageKey": "msg","stacktraceKey": "stacktrace","lineEnding": "","levelEncoder": "","timeEncoder": "iso8601","durationEncoder": "","callerEncoder": ""}}`
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

	// config is the configuration options that the Reconciler accepts.
	config Config
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

	deploymentReconcileError := r.reconcileDeployment(ctx, logger, el)
	serviceReconcileError := r.reconcileService(ctx, logger, el)

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
			Ports:    []corev1.ServicePort{servicePort},
		},
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

func (r *Reconciler) reconcileDeployment(ctx context.Context, logger *zap.SugaredLogger, el *v1alpha1.EventListener) error {
	// check logging config, create if it doesn't exist
	if err := r.reconcileLoggingConfig(ctx, logger, el); err != nil {
		logger.Error(err)
		return err
	}

	container := getContainer(el, r.config)
	deployment := getDeployment(el, r.config)

	existingDeployment, err := r.deploymentLister.Deployments(el.Namespace).Get(el.Status.Configuration.GeneratedResourceName)
	switch {
	case err == nil:
		el.Status.SetDeploymentConditions(existingDeployment.Status.Conditions)
		el.Status.SetExistsCondition(v1alpha1.DeploymentExists, nil)

		// Determine if reconciliation has to occur
		updated := reconcileObjectMeta(&existingDeployment.ObjectMeta, deployment.ObjectMeta)
		if *existingDeployment.Spec.Replicas != *deployment.Spec.Replicas {
			if el.Spec.Replicas != nil {
				existingDeployment.Spec.Replicas = deployment.Spec.Replicas
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

func getDeployment(el *v1alpha1.EventListener, c Config) *appsv1.Deployment {
	var replicas = ptr.Int32(1)
	if el.Spec.Replicas != nil {
		replicas = el.Spec.Replicas
	}
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

	container := getContainer(el, c)
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
	if el.Spec.Resources.KubernetesResource != nil {
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

func getContainer(el *v1alpha1.EventListener, c Config) corev1.Container {
	var (
		elCert, elKey string
		resources     corev1.ResourceRequirements
	)

	vMount := []corev1.VolumeMount{{
		Name:      "config-logging",
		MountPath: "/etc/config-logging",
	}}

	env := []corev1.EnvVar{{
		Name: "SYSTEM_NAMESPACE",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.namespace",
			},
		},
	}}

	certEnv := map[string]*corev1.EnvVarSource{}
	if el.Spec.Resources.KubernetesResource != nil {
		if len(el.Spec.Resources.KubernetesResource.Template.Spec.Containers) != 0 {
			resources = el.Spec.Resources.KubernetesResource.Template.Spec.Containers[0].Resources
			for i, e := range el.Spec.Resources.KubernetesResource.Template.Spec.Containers[0].Env {
				env = append(env, e)
				certEnv[el.Spec.Resources.KubernetesResource.Template.Spec.Containers[0].Env[i].Name] =
					el.Spec.Resources.KubernetesResource.Template.Spec.Containers[0].Env[i].ValueFrom
			}
		}
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
		vMount = append(vMount, corev1.VolumeMount{
			Name:      "https-connection",
			ReadOnly:  true,
			MountPath: "/etc/triggers/tls",
		})
	} else {
		scheme = corev1.URISchemeHTTP
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
		LivenessProbe: &corev1.Probe{
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/live",
					Scheme: scheme,
					Port:   intstr.FromInt(eventListenerContainerPort),
				},
			},
			PeriodSeconds:    int32(*c.PeriodSeconds),
			FailureThreshold: int32(*c.FailureThreshold),
		},
		ReadinessProbe: &corev1.Probe{
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/live",
					Scheme: scheme,
					Port:   intstr.FromInt(eventListenerContainerPort),
				},
			},
			PeriodSeconds:    int32(*c.PeriodSeconds),
			FailureThreshold: int32(*c.FailureThreshold),
		},
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
			"--tls-cert=" + elCert,
			"--tls-key=" + elKey,
		},
		VolumeMounts: vMount,
		Env:          env,
	}
}

func getServicePort(el *v1alpha1.EventListener, c Config) corev1.ServicePort {
	var (
		elCert, elKey string
	)

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
