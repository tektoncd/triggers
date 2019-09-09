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
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/pipeline/pkg/system"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"github.com/tektoncd/triggers/test"
	bldr "github.com/tektoncd/triggers/test/builder"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/configmap"
	rtesting "knative.dev/pkg/reconciler/testing"
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
	labels := map[string]string{"app": "my-eventlistener"}
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "tekton-pipelines",
		},
	}
	eventListener := bldr.EventListener("my-eventlistener", "tekton-pipelines",
		bldr.EventListenerSpec(
			bldr.EventListenerTrigger("my-triggerbinding", "my-triggertemplate", ""),
		),
	)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       "tekton-pipelines",
			Name:            "my-eventlistener",
			OwnerReferences: []metav1.OwnerReference{*eventListener.GetOwnerReference()},
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
									ContainerPort: int32(Port),
								},
							},
							Args: []string{
								"-el-name", "my-eventlistener",
								"-el-namespace", "tekton-pipelines",
								"-port", strconv.Itoa(Port),
							},
						},
					},
				},
			},
		},
	}
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       "tekton-pipelines",
			Name:            "my-eventlistener",
			OwnerReferences: []metav1.OwnerReference{*eventListener.GetOwnerReference()},
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
	serviceWithClusterIp := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       "tekton-pipelines",
			Name:            "my-eventlistener",
			OwnerReferences: []metav1.OwnerReference{*eventListener.GetOwnerReference()},
			Labels:          labels,
		},
		Spec: corev1.ServiceSpec{
			Selector:  labels,
			Type:      corev1.ServiceTypeLoadBalancer,
			ClusterIP: "some_ip_address",
			Ports: []corev1.ServicePort{
				{
					Protocol: corev1.ProtocolTCP,
					Port:     int32(Port),
				},
			},
		},
	}

	tests := []struct {
		name               string
		key                string
		testResourcesStart test.TestResources
		testResourcesEnd   test.TestResources
	}{{
		name:               "delete-eventlistener",
		key:                "tekton-pipelines/my-eventlistener",
		testResourcesStart: test.TestResources{},
		testResourcesEnd:   test.TestResources{},
	}, {
		name: "create-eventlistener",
		key:  "tekton-pipelines/my-eventlistener",
		testResourcesStart: test.TestResources{
			Namespaces:     []*corev1.Namespace{namespace},
			EventListeners: []*v1alpha1.EventListener{eventListener},
		},
		testResourcesEnd: test.TestResources{
			Namespaces:     []*corev1.Namespace{namespace},
			EventListeners: []*v1alpha1.EventListener{eventListener},
			Deployments:    []*appsv1.Deployment{deployment},
			Services:       []*corev1.Service{service},
		},
	}, {
		name: "update-eventlistener",
		key:  "tekton-pipelines/my-eventlistener",
		testResourcesStart: test.TestResources{
			Namespaces:     []*corev1.Namespace{namespace},
			EventListeners: []*v1alpha1.EventListener{eventListener},
			Deployments:    []*appsv1.Deployment{deployment},
			Services:       []*corev1.Service{serviceWithClusterIp},
		},
		testResourcesEnd: test.TestResources{
			Namespaces:     []*corev1.Namespace{namespace},
			EventListeners: []*v1alpha1.EventListener{eventListener},
			Deployments:    []*appsv1.Deployment{deployment},
			Services:       []*corev1.Service{serviceWithClusterIp},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup with testResourcesStart
			testAssets, cancel := getEventListenerTestAssets(t, tt.testResourcesStart)
			defer cancel()

			// Run Reconcile
			err := testAssets.Controller.Reconciler.Reconcile(context.Background(), tt.key)
			if err != nil {
				t.Errorf("eventlistener.Reconcile() returned error: %s", err)
				return
			}

			// Check current resources match endTestResources
			actualTestResourcesEnd, err := test.GetTestResourcesFromClients(testAssets.Clients)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tt.testResourcesEnd, *actualTestResourcesEnd); diff != "" {
				t.Errorf("eventlistener.Reconcile(): -want +got: %s", diff)
			}
		})
	}
}
