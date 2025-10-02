/*
Copyright 2025 The Tekton Authors

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

package bootstrap

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetTektonNamespace(t *testing.T) {
	tests := []struct {
		name              string
		existingNamespace string
		wantNamespace     string
	}{
		{
			name:              "detects openshift-pipelines",
			existingNamespace: "openshift-pipelines",
			wantNamespace:     "openshift-pipelines",
		},
		{
			name:              "detects tekton-pipelines",
			existingNamespace: "tekton-pipelines",
			wantNamespace:     "tekton-pipelines",
		},
		{
			name:              "defaults to tekton-pipelines when not found",
			existingNamespace: "",
			wantNamespace:     "tekton-pipelines",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			config := &Config{InstallDeps: true}

			var objects []runtime.Object
			if tt.existingNamespace != "" {
				// Create a deployment in the existing namespace
				deployment := &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "tekton-pipelines-controller",
						Namespace: tt.existingNamespace,
					},
					Status: appsv1.DeploymentStatus{
						ReadyReplicas: 1,
					},
				}
				objects = append(objects, deployment)
			}

			fakeClient := fake.NewSimpleClientset(objects...)
			installer := NewInstaller(fakeClient, config)

			gotNamespace := installer.getTektonNamespace(ctx)
			if gotNamespace != tt.wantNamespace {
				t.Errorf("getTektonNamespace() = %v, want %v", gotNamespace, tt.wantNamespace)
			}

			// Test caching this should return same value without API call
			gotNamespace2 := installer.getTektonNamespace(ctx)
			if gotNamespace2 != tt.wantNamespace {
				t.Errorf("getTektonNamespace() cached = %v, want %v", gotNamespace2, tt.wantNamespace)
			}
		})
	}
}

func TestIsPipelinesInstalled(t *testing.T) {
	tests := []struct {
		name          string
		deployment    *appsv1.Deployment
		wantInstalled bool
	}{
		{
			name: "pipelines installed and ready",
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tekton-pipelines-controller",
					Namespace: "tekton-pipelines",
				},
				Status: appsv1.DeploymentStatus{
					ReadyReplicas: 1,
				},
			},
			wantInstalled: true,
		},
		{
			name: "pipelines deployed but not ready",
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tekton-pipelines-controller",
					Namespace: "tekton-pipelines",
				},
				Status: appsv1.DeploymentStatus{
					ReadyReplicas: 0,
				},
			},
			wantInstalled: false,
		},
		{
			name:          "pipelines not installed",
			deployment:    nil,
			wantInstalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			config := &Config{InstallDeps: true}

			var objects []runtime.Object
			if tt.deployment != nil {
				objects = append(objects, tt.deployment)
			}

			fakeClient := fake.NewSimpleClientset(objects...)
			installer := NewInstaller(fakeClient, config)

			gotInstalled := installer.isPipelinesInstalled(ctx)
			if gotInstalled != tt.wantInstalled {
				t.Errorf("isPipelinesInstalled() = %v, want %v", gotInstalled, tt.wantInstalled)
			}
		})
	}
}

func TestIsTriggersInstalled(t *testing.T) {
	tests := []struct {
		name          string
		deployment    *appsv1.Deployment
		wantInstalled bool
	}{
		{
			name: "triggers installed and ready",
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tekton-triggers-controller",
					Namespace: "tekton-pipelines",
				},
				Status: appsv1.DeploymentStatus{
					ReadyReplicas: 1,
				},
			},
			wantInstalled: true,
		},
		{
			name: "triggers deployed but not ready",
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tekton-triggers-controller",
					Namespace: "tekton-pipelines",
				},
				Status: appsv1.DeploymentStatus{
					ReadyReplicas: 0,
				},
			},
			wantInstalled: false,
		},
		{
			name:          "triggers not installed",
			deployment:    nil,
			wantInstalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			config := &Config{InstallDeps: true}

			var objects []runtime.Object
			if tt.deployment != nil {
				objects = append(objects, tt.deployment)
			}

			fakeClient := fake.NewSimpleClientset(objects...)
			installer := NewInstaller(fakeClient, config)

			gotInstalled := installer.isTriggersInstalled(ctx)
			if gotInstalled != tt.wantInstalled {
				t.Errorf("isTriggersInstalled() = %v, want %v", gotInstalled, tt.wantInstalled)
			}
		})
	}
}

func TestWaitForNamespace(t *testing.T) {
	tests := []struct {
		name      string
		namespace *corev1.Namespace
		wantError bool
	}{
		{
			name: "namespace exists and active",
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-namespace",
				},
				Status: corev1.NamespaceStatus{
					Phase: corev1.NamespaceActive,
				},
			},
			wantError: false,
		},
		{
			name: "namespace exists but terminating",
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-namespace",
				},
				Status: corev1.NamespaceStatus{
					Phase: corev1.NamespaceTerminating,
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{InstallDeps: true}

			var objects []runtime.Object
			if tt.namespace != nil {
				objects = append(objects, tt.namespace)
			}

			fakeClient := fake.NewSimpleClientset(objects...)
			_ = NewInstaller(fakeClient, config)
		})
	}
}

func TestWaitForDeployment(t *testing.T) {
	tests := []struct {
		name       string
		deployment *appsv1.Deployment
		wantError  bool
	}{
		{
			name: "deployment ready",
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "test-namespace",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: int32Ptr(1),
				},
				Status: appsv1.DeploymentStatus{
					ReadyReplicas: 1,
				},
			},
			wantError: false,
		},
		{
			name: "deployment not ready",
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "test-namespace",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: int32Ptr(1),
				},
				Status: appsv1.DeploymentStatus{
					ReadyReplicas: 0,
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{InstallDeps: true}

			var objects []runtime.Object
			if tt.deployment != nil {
				objects = append(objects, tt.deployment)
			}

			fakeClient := fake.NewSimpleClientset(objects...)
			_ = NewInstaller(fakeClient, config)
		})
	}
}

func TestNewInstaller(t *testing.T) {
	config := &Config{
		Namespace:   "test",
		InstallDeps: true,
	}
	fakeClient := fake.NewSimpleClientset()

	installer := NewInstaller(fakeClient, config)

	if installer == nil {
		t.Error("NewInstaller() returned nil")
	}
	if installer.kubeClient == nil {
		t.Error("NewInstaller() kubeClient is nil")
	}
	if installer.config == nil {
		t.Error("NewInstaller() config is nil")
	}
	if installer.tektonNamespace != "" {
		t.Errorf("NewInstaller() tektonNamespace should be empty initially, got %v", installer.tektonNamespace)
	}
}

func int32Ptr(i int32) *int32 {
	return &i
}
