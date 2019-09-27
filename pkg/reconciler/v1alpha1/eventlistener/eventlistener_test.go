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
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/tektoncd/pipeline/pkg/system"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"github.com/tektoncd/triggers/test"
	bldr "github.com/tektoncd/triggers/test/builder"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/rand"
	k8stest "k8s.io/client-go/testing"
	"knative.dev/pkg/apis"
	duckv1alpha1 "knative.dev/pkg/apis/duck/v1alpha1"
	"knative.dev/pkg/configmap"
	fakekubeclient "knative.dev/pkg/injection/clients/kubeclient/fake"
	rtesting "knative.dev/pkg/reconciler/testing"
)

func init() {
	rand.Seed(0)
	generatedResourceName = fmt.Sprintf("%s-cbhtc", eventListenerName)
	eventListener0 = bldr.EventListener(eventListenerName, namespace,
		bldr.EventListenerSpec(
			bldr.EventListenerServiceAccount("sa"),
		),
		bldr.EventListenerStatus(
			bldr.EventListenerConfig(generatedResourceName),
		),
	)
}

var (
	generatedResourceName    string
	ignoreLastTransitionTime = cmpopts.IgnoreTypes(apis.Condition{}.LastTransitionTime.Inner.Time)

	// 0 indicates pre-reconciliation
	eventListener0    *v1alpha1.EventListener
	eventListenerName = "my-eventlistener"
	namespace         = "tekton-pipelines"
	namespaceResource = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	reconcileKey                 = fmt.Sprintf("%s/%s", namespace, eventListenerName)
	updateLabel                  = map[string]string{"update": "true"}
	updatedSa                    = "updatedSa"
	deploymentAvailableCondition = appsv1.DeploymentCondition{
		Type:    appsv1.DeploymentAvailable,
		Status:  corev1.ConditionTrue,
		Message: "Deployment has minimum availability",
		Reason:  "MinimumReplicasAvailable",
	}
	deploymentProgressingCondition = appsv1.DeploymentCondition{
		Type:    appsv1.DeploymentProgressing,
		Status:  corev1.ConditionTrue,
		Message: fmt.Sprintf("ReplicaSet \"%s\" has successfully progressed.", eventListenerName),
		Reason:  "NewReplicaSetAvailable",
	}
)

// getEventListenerTestAssets returns TestAssets that have been seeded with the
// given TestResources r where r represents the state of the system
func getEventListenerTestAssets(t *testing.T, r test.TestResources) (test.TestAssets, context.CancelFunc) {
	t.Helper()
	ctx, _ := rtesting.SetupFakeContext(t)
	ctx, cancel := context.WithCancel(ctx)
	kubeClient := fakekubeclient.Get(ctx)
	// Fake client reactor chain ignores non handled reactors until v1.40.0
	// Test deployment/service resource should set their defaults
	// See: https://github.com/kubernetes/kubernetes/pull/73601
	kubeClient.PrependReactor("create", "deployments",
		func(action k8stest.Action) (bool, runtime.Object, error) {
			deployment := action.(k8stest.CreateActionImpl).GetObject().(*appsv1.Deployment)
			// Only add conditions when they don't exist
			// Test seeding expected resources "creates", which would create duplicates
			if len(deployment.Status.Conditions) == 0 {
				deployment.Status.Conditions = append(deployment.Status.Conditions, deploymentAvailableCondition)
				deployment.Status.Conditions = append(deployment.Status.Conditions, deploymentProgressingCondition)
			}
			// Pass modified resource and react using the default catch all reactor
			return kubeClient.ReactionChain[len(kubeClient.ReactionChain)-1].React(action)
		})
	clients := test.SeedTestResources(t, ctx, r)
	cmw := configmap.NewInformedWatcher(clients.Kube, system.GetNamespace())
	return test.TestAssets{
		Controller: NewController(ctx, cmw),
		Clients:    clients,
	}, cancel
}

func Test_reconcileService(t *testing.T) {
	eventListener1 := eventListener0.DeepCopy()
	eventListener1.Status.SetExistsCondition(v1alpha1.ServiceExists, nil)
	eventListener1.Status.Address = &duckv1alpha1.Addressable{
		Hostname: listenerHostname(generatedResourceName, namespace),
	}

	eventListener2 := eventListener1.DeepCopy()
	eventListener2.Labels = updateLabel

	service1 := &corev1.Service{
		ObjectMeta: GeneratedObjectMeta(eventListener0),
		Spec: corev1.ServiceSpec{
			Selector: GeneratedResourceLabels,
			Type:     corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				servicePort,
			},
		},
	}
	service2 := service1.DeepCopy()
	service2.Labels = mergeLabels(GeneratedResourceLabels, updateLabel)
	service2.Spec.Selector = mergeLabels(GeneratedResourceLabels, updateLabel)

	tests := []struct {
		name           string
		startResources test.TestResources
		endResources   test.TestResources
	}{
		{
			name: "create-service",
			startResources: test.TestResources{
				Namespaces:     []*corev1.Namespace{namespaceResource},
				EventListeners: []*v1alpha1.EventListener{eventListener0},
			},
			endResources: test.TestResources{
				EventListeners: []*v1alpha1.EventListener{eventListener1},
				Services:       []*corev1.Service{service1},
			},
		},
		{
			name: "eventlistener-label-update",
			startResources: test.TestResources{
				Namespaces:     []*corev1.Namespace{namespaceResource},
				EventListeners: []*v1alpha1.EventListener{eventListener2},
				Services:       []*corev1.Service{service1},
			},
			endResources: test.TestResources{
				EventListeners: []*v1alpha1.EventListener{eventListener2},
				Services:       []*corev1.Service{service2},
			},
		},
		{
			name: "service-label-update",
			startResources: test.TestResources{
				Namespaces:     []*corev1.Namespace{namespaceResource},
				EventListeners: []*v1alpha1.EventListener{eventListener1},
				Services:       []*corev1.Service{service2},
			},
			endResources: test.TestResources{
				Namespaces:     []*corev1.Namespace{namespaceResource},
				EventListeners: []*v1alpha1.EventListener{eventListener1},
				Services:       []*corev1.Service{service1},
			},
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			// Setup
			testAssets, cancel := getEventListenerTestAssets(t, tests[i].startResources)
			defer cancel()

			// Run Reconcile
			err := testAssets.Controller.Reconciler.(*Reconciler).reconcileService(tests[i].startResources.EventListeners[0])
			if err != nil {
				t.Errorf("eventlistener.Reconcile() returned error: %s", err)
				return
			}
			// Grab test resource results
			actualEndResources, err := test.GetTestResourcesFromClients(testAssets.Clients)
			if err != nil {
				t.Fatal(err)
			}
			// Compare services
			// Semantic equality since VolatileTime will not match using cmp.Diff
			if diff := cmp.Diff(tests[i].endResources.Services, actualEndResources.Services, ignoreLastTransitionTime); diff != "" {
				t.Errorf("eventlistener.Reconcile() equality mismatch. Diff request body: -want +got: %s", diff)
			}

			// Compare EventListener
			// The updates to EventListener are not persisted within reconcileService
			if diff := cmp.Diff(tests[i].endResources.EventListeners[0], tests[i].startResources.EventListeners[0], ignoreLastTransitionTime); diff != "" {
				t.Errorf("eventlistener.Reconcile() equality mismatch. Diff request body: -want +got: %s", diff)
			}
		})
	}
}

func Test_reconcileDeployment(t *testing.T) {
	eventListener1 := eventListener0.DeepCopy()
	eventListener1.Status.SetExistsCondition(v1alpha1.DeploymentExists, nil)
	eventListener1.Status.SetDeploymentConditions([]appsv1.DeploymentCondition{
		deploymentAvailableCondition,
		deploymentProgressingCondition,
	})

	eventListener2 := eventListener1.DeepCopy()
	eventListener2.Labels = updateLabel

	eventListener3 := eventListener1.DeepCopy()
	eventListener3.Status.SetCondition(&apis.Condition{
		Type: apis.ConditionType(appsv1.DeploymentReplicaFailure),
	})

	eventListener4 := eventListener1.DeepCopy()
	eventListener4.Spec.ServiceAccountName = updatedSa

	var replicas int32 = 1
	deployment1 := &appsv1.Deployment{
		ObjectMeta: GeneratedObjectMeta(eventListener0),
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: GeneratedResourceLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: GeneratedResourceLabels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: eventListener0.Spec.ServiceAccountName,
					Containers: []corev1.Container{
						{
							Name:  "event-listener",
							Image: *elImage,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: int32(Port),
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Args: []string{
								"-el-name", eventListenerName,
								"-el-namespace", namespace,
								"-port", strconv.Itoa(Port),
							},
						},
					},
				},
			},
		},
		Status: appsv1.DeploymentStatus{
			Conditions: []appsv1.DeploymentCondition{
				deploymentAvailableCondition,
				deploymentProgressingCondition,
			},
		},
	}

	deployment2 := deployment1.DeepCopy()
	deployment2.Labels = mergeLabels(GeneratedResourceLabels, updateLabel)
	deployment2.Spec.Selector.MatchLabels = mergeLabels(GeneratedResourceLabels, updateLabel)
	deployment2.Spec.Template.Labels = mergeLabels(GeneratedResourceLabels, updateLabel)

	deployment3 := deployment1.DeepCopy()
	var updateReplicas int32 = 5
	deployment3.Spec.Replicas = &updateReplicas

	deployment4 := deployment1.DeepCopy()
	deployment4.Spec.Template.Spec.ServiceAccountName = updatedSa

	tests := []struct {
		name           string
		startResources test.TestResources
		endResources   test.TestResources
	}{
		{
			name: "create-deployment",
			startResources: test.TestResources{
				Namespaces:     []*corev1.Namespace{namespaceResource},
				EventListeners: []*v1alpha1.EventListener{eventListener0},
			},
			endResources: test.TestResources{
				Namespaces:     []*corev1.Namespace{namespaceResource},
				EventListeners: []*v1alpha1.EventListener{eventListener1},
				Deployments:    []*appsv1.Deployment{deployment1},
			},
		},
		{
			name: "eventlistener-label-update",
			startResources: test.TestResources{
				Namespaces:     []*corev1.Namespace{namespaceResource},
				EventListeners: []*v1alpha1.EventListener{eventListener2},
				Deployments:    []*appsv1.Deployment{deployment1},
			},
			endResources: test.TestResources{
				Namespaces:     []*corev1.Namespace{namespaceResource},
				EventListeners: []*v1alpha1.EventListener{eventListener2},
				Deployments:    []*appsv1.Deployment{deployment2},
			},
		},
		{
			name: "deployment-label-update",
			startResources: test.TestResources{
				Namespaces:     []*corev1.Namespace{namespaceResource},
				EventListeners: []*v1alpha1.EventListener{eventListener1},
				Deployments:    []*appsv1.Deployment{deployment2},
			},
			endResources: test.TestResources{
				Namespaces:     []*corev1.Namespace{namespaceResource},
				EventListeners: []*v1alpha1.EventListener{eventListener1},
				Deployments:    []*appsv1.Deployment{deployment1},
			},
		},
		{
			name: "deployment-replica-update",
			startResources: test.TestResources{
				Namespaces:     []*corev1.Namespace{namespaceResource},
				EventListeners: []*v1alpha1.EventListener{eventListener1},
				Deployments:    []*appsv1.Deployment{deployment3},
			},
			endResources: test.TestResources{
				Namespaces:     []*corev1.Namespace{namespaceResource},
				EventListeners: []*v1alpha1.EventListener{eventListener1},
				Deployments:    []*appsv1.Deployment{deployment3},
			},
		},
		{
			name: "eventlistener-replica-failure-status-update",
			startResources: test.TestResources{
				Namespaces:     []*corev1.Namespace{namespaceResource},
				EventListeners: []*v1alpha1.EventListener{eventListener3},
				Deployments:    []*appsv1.Deployment{deployment1},
			},
			endResources: test.TestResources{
				Namespaces:     []*corev1.Namespace{namespaceResource},
				EventListeners: []*v1alpha1.EventListener{eventListener1},
				Deployments:    []*appsv1.Deployment{deployment1},
			},
		},
		{
			name: "eventlistener-serviceaccount-update",
			startResources: test.TestResources{
				Namespaces:     []*corev1.Namespace{namespaceResource},
				EventListeners: []*v1alpha1.EventListener{eventListener4},
				Deployments:    []*appsv1.Deployment{deployment1},
			},
			endResources: test.TestResources{
				Namespaces:     []*corev1.Namespace{namespaceResource},
				EventListeners: []*v1alpha1.EventListener{eventListener4},
				Deployments:    []*appsv1.Deployment{deployment4},
			},
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			// Setup
			testAssets, cancel := getEventListenerTestAssets(t, tests[i].startResources)
			defer cancel()

			// Run Reconcile
			err := testAssets.Controller.Reconciler.(*Reconciler).reconcileDeployment(tests[i].startResources.EventListeners[0])
			if err != nil {
				t.Errorf("eventlistener.Reconcile() returned error: %s", err)
				return
			}
			// Grab test resource results
			actualEndResources, err := test.GetTestResourcesFromClients(testAssets.Clients)
			if err != nil {
				t.Fatal(err)
			}
			// Compare Deployments
			// Semantic equality since VolatileTime will not match using cmp.Diff
			if !equality.Semantic.DeepEqual(tests[i].endResources.Deployments, actualEndResources.Deployments) {
				t.Error("eventlistener.Reconcile() equality mismatch. Ignore semantic time mismatch")
				diff := cmp.Diff(tests[i].endResources.Deployments, actualEndResources.Deployments)
				t.Errorf("Diff request body: -want +got: %s", diff)
			}
			// Compare EventListener
			// The updates to EventListener are not persisted within reconcileService
			if !equality.Semantic.DeepEqual(tests[i].endResources.EventListeners[0], tests[i].startResources.EventListeners[0]) {
				t.Error("eventlistener.Reconcile() equality mismatch. Ignore semantic time mismatch")
				diff := cmp.Diff(tests[i].endResources.EventListeners[0], tests[i].startResources.EventListeners[0])
				t.Errorf("Diff request body: -want +got: %s", diff)
			}
		})
	}
}

func TestReconcile(t *testing.T) {
	eventListener1 := bldr.EventListener(eventListenerName, namespace,
		bldr.EventListenerSpec(
			bldr.EventListenerServiceAccount("sa"),
		),
		bldr.EventListenerStatus(
			bldr.EventListenerConfig(generatedResourceName),
			bldr.EventListenerAddress(listenerHostname(generatedResourceName, namespace)),
			bldr.EventListenerCondition(
				v1alpha1.ServiceExists,
				corev1.ConditionTrue,
				"Service exists", "",
			),
			bldr.EventListenerCondition(
				v1alpha1.DeploymentExists,
				corev1.ConditionTrue,
				"Deployment exists", "",
			),
			bldr.EventListenerCondition(
				apis.ConditionType(appsv1.DeploymentAvailable),
				corev1.ConditionTrue,
				"Deployment has minimum availability",
				"MinimumReplicasAvailable",
			),
			bldr.EventListenerCondition(
				apis.ConditionType(appsv1.DeploymentProgressing),
				corev1.ConditionTrue,
				fmt.Sprintf("ReplicaSet \"%s\" has successfully progressed.", eventListenerName),
				"NewReplicaSetAvailable",
			),
		),
	)

	eventListener2 := eventListener1.DeepCopy()
	eventListener2.Labels = updateLabel

	eventListener3 := eventListener1.DeepCopy()
	eventListener3.Spec.ServiceAccountName = updatedSa

	var replicas int32 = 1
	deployment1 := &appsv1.Deployment{
		ObjectMeta: GeneratedObjectMeta(eventListener0),
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: GeneratedResourceLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: GeneratedResourceLabels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: eventListener0.Spec.ServiceAccountName,
					Containers: []corev1.Container{
						{
							Name:  "event-listener",
							Image: *elImage,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: int32(Port),
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Args: []string{
								"-el-name", eventListenerName,
								"-el-namespace", namespace,
								"-port", strconv.Itoa(Port),
							},
						},
					},
				},
			},
		},
		Status: appsv1.DeploymentStatus{
			Conditions: []appsv1.DeploymentCondition{
				deploymentAvailableCondition,
				deploymentProgressingCondition,
			},
		},
	}

	deployment2 := deployment1.DeepCopy()
	deployment2.Labels = mergeLabels(updateLabel, GeneratedResourceLabels)
	deployment2.Spec.Selector.MatchLabels = mergeLabels(updateLabel, GeneratedResourceLabels)
	deployment2.Spec.Template.Labels = mergeLabels(updateLabel, GeneratedResourceLabels)

	deployment3 := deployment1.DeepCopy()
	deployment3.Spec.Template.Spec.ServiceAccountName = updatedSa

	service1 := &corev1.Service{
		ObjectMeta: GeneratedObjectMeta(eventListener0),
		Spec: corev1.ServiceSpec{
			Selector: GeneratedResourceLabels,
			Type:     corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				servicePort,
			},
		},
	}

	service2 := service1.DeepCopy()
	service2.Labels = mergeLabels(updateLabel, GeneratedResourceLabels)
	service2.Spec.Selector = mergeLabels(updateLabel, GeneratedResourceLabels)

	tests := []struct {
		name           string
		key            string
		startResources test.TestResources
		endResources   test.TestResources
	}{
		{
			name: "create-eventlistener",
			key:  reconcileKey,
			startResources: test.TestResources{
				Namespaces:     []*corev1.Namespace{namespaceResource},
				EventListeners: []*v1alpha1.EventListener{eventListener0},
			},
			endResources: test.TestResources{
				Namespaces:     []*corev1.Namespace{namespaceResource},
				EventListeners: []*v1alpha1.EventListener{eventListener1},
				Deployments:    []*appsv1.Deployment{deployment1},
				Services:       []*corev1.Service{service1},
			},
		},
		{
			name: "update-eventlistener-labels",
			key:  reconcileKey,
			startResources: test.TestResources{
				Namespaces:     []*corev1.Namespace{namespaceResource},
				EventListeners: []*v1alpha1.EventListener{eventListener2},
				Deployments:    []*appsv1.Deployment{deployment1},
				Services:       []*corev1.Service{service1},
			},
			endResources: test.TestResources{
				Namespaces:     []*corev1.Namespace{namespaceResource},
				EventListeners: []*v1alpha1.EventListener{eventListener2},
				Deployments:    []*appsv1.Deployment{deployment2},
				Services:       []*corev1.Service{service2},
			},
		},
		{
			name: "update-eventlistener-serviceaccount",
			key:  reconcileKey,
			startResources: test.TestResources{
				Namespaces:     []*corev1.Namespace{namespaceResource},
				EventListeners: []*v1alpha1.EventListener{eventListener3},
				Deployments:    []*appsv1.Deployment{deployment1},
				Services:       []*corev1.Service{service1},
			},
			endResources: test.TestResources{
				Namespaces:     []*corev1.Namespace{namespaceResource},
				EventListeners: []*v1alpha1.EventListener{eventListener3},
				Deployments:    []*appsv1.Deployment{deployment3},
				Services:       []*corev1.Service{service1},
			},
		},
		{
			name:           "delete-eventlistener",
			key:            reconcileKey,
			startResources: test.TestResources{},
			endResources:   test.TestResources{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup with startResources
			testAssets, cancel := getEventListenerTestAssets(t, tt.startResources)
			defer cancel()

			// Run Reconcile
			err := testAssets.Controller.Reconciler.Reconcile(context.Background(), tt.key)
			if err != nil {
				t.Errorf("eventlistener.Reconcile() returned error: %s", err)
				return
			}
			// Grab test resource results
			actualEndResources, err := test.GetTestResourcesFromClients(testAssets.Clients)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tt.endResources, *actualEndResources, ignoreLastTransitionTime); diff != "" {
				t.Errorf("eventlistener.Reconcile() equality mismatch. Diff request body: -want +got: %s", diff)
			}
		})
	}
}

func Test_wrapError(t *testing.T) {
	tests := []struct {
		name           string
		error1, error2 error
		expectedError  error
	}{
		{
			name:          "Both error empty",
			error1:        nil,
			error2:        nil,
			expectedError: nil,
		},
		{
			name:          "Error one empty",
			error1:        nil,
			error2:        fmt.Errorf("Error"),
			expectedError: fmt.Errorf("Error"),
		},
		{
			name:          "Error two empty",
			error1:        fmt.Errorf("Error"),
			error2:        nil,
			expectedError: fmt.Errorf("Error"),
		},
		{
			name:          "Both errors",
			error1:        fmt.Errorf("Error1"),
			error2:        fmt.Errorf("Error2"),
			expectedError: fmt.Errorf("Error1 : Error2"),
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			actualError := wrapError(tests[i].error1, tests[i].error2)
			// Compare strings since errors have unexported fields that panic
			var expectedErrorString string
			var actualErrorString string
			if tests[i].expectedError != nil {
				expectedErrorString = tests[i].expectedError.Error()
			}
			if actualError != nil {
				actualErrorString = actualError.Error()
			}
			if diff := cmp.Diff(expectedErrorString, actualErrorString); diff != "" {
				t.Errorf("wrapError() did not return expected. -want, +got: %s", diff)
			}
		})
	}
}

func Test_mergeLabels(t *testing.T) {
	tests := []struct {
		name           string
		l1, l2         map[string]string
		expectedLabels map[string]string
	}{
		{
			name:           "Both maps empty",
			l1:             nil,
			l2:             nil,
			expectedLabels: nil,
		},
		{
			name:           "Map one empty",
			l1:             nil,
			l2:             map[string]string{"k": "v"},
			expectedLabels: map[string]string{"k": "v"},
		},
		{
			name:           "Map two empty",
			l1:             map[string]string{"k": "v"},
			l2:             nil,
			expectedLabels: map[string]string{"k": "v"},
		},
		{
			name:           "Both maps",
			l1:             map[string]string{"k1": "v1"},
			l2:             map[string]string{"k2": "v2"},
			expectedLabels: map[string]string{"k1": "v1", "k2": "v2"},
		},
		{
			name:           "Both maps with clobber",
			l1:             map[string]string{"k1": "v1"},
			l2:             map[string]string{"k1": "v2"},
			expectedLabels: map[string]string{"k1": "v2"},
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			actualLabels := mergeLabels(tests[i].l1, tests[i].l2)
			if diff := cmp.Diff(tests[i].expectedLabels, actualLabels); diff != "" {
				t.Errorf("mergeLabels() did not return expected. -want, +got: %s", diff)
			}
		})
	}
}

func TestGeneratedObjectMeta(t *testing.T) {
	blockOwnerDeletion := true
	isController := true
	tests := []struct {
		name               string
		el                 *v1alpha1.EventListener
		expectedObjectMeta metav1.ObjectMeta
	}{
		{
			name: "Empty EventListener",
			el:   bldr.EventListener("name", ""),
			expectedObjectMeta: metav1.ObjectMeta{
				Namespace: "",
				Name:      "",
				OwnerReferences: []metav1.OwnerReference{{
					APIVersion:         "tekton.dev/v1alpha1",
					Kind:               "EventListener",
					Name:               "name",
					UID:                "",
					Controller:         &isController,
					BlockOwnerDeletion: &blockOwnerDeletion,
				}},
				Labels: GeneratedResourceLabels,
			},
		}, {
			name: "EventListener with Configuration",
			el: bldr.EventListener("name", "",
				bldr.EventListenerStatus(
					bldr.EventListenerConfig("generatedName"),
				),
			),
			expectedObjectMeta: metav1.ObjectMeta{
				Namespace: "",
				Name:      "generatedName",
				OwnerReferences: []metav1.OwnerReference{{
					APIVersion:         "tekton.dev/v1alpha1",
					Kind:               "EventListener",
					Name:               "name",
					UID:                "",
					Controller:         &isController,
					BlockOwnerDeletion: &blockOwnerDeletion,
				}},
				Labels: GeneratedResourceLabels,
			},
		}, {
			name: "EventListener with Labels",
			el: bldr.EventListener("name", "",
				bldr.EventListenerMeta(
					bldr.Label("k", "v"),
				),
			),
			expectedObjectMeta: metav1.ObjectMeta{
				Namespace: "",
				Name:      "",
				OwnerReferences: []metav1.OwnerReference{{
					APIVersion:         "tekton.dev/v1alpha1",
					Kind:               "EventListener",
					Name:               "name",
					UID:                "",
					Controller:         &isController,
					BlockOwnerDeletion: &blockOwnerDeletion,
				}},
				Labels: mergeLabels(map[string]string{"k": "v"}, GeneratedResourceLabels),
			},
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			actualObjectMeta := GeneratedObjectMeta(tests[i].el)
			if diff := cmp.Diff(tests[i].expectedObjectMeta, actualObjectMeta); diff != "" {
				t.Errorf("GeneratedObjectMeta() did not return expected. -want, +got: %s", diff)
			}
		})
	}
}
