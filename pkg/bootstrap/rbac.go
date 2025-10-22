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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type RBACManager struct {
	kubeClient kubernetes.Interface
	config     *Config
}

func NewRBACManager(kubeClient kubernetes.Interface, config *Config) *RBACManager {
	return &RBACManager{
		kubeClient: kubeClient,
		config:     config,
	}
}

// CreateNamespace creates the target namespace if it doesn't exist
func (r *RBACManager) CreateNamespace(ctx context.Context) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.config.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/part-of": "tekton-triggers",
				"tekton.dev/bootstrap":      "true",
			},
		},
	}

	_, err := r.kubeClient.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}
	return nil
}

// SetupRBAC creates necessary service accounts, roles, and bindings
func (r *RBACManager) SetupRBAC(ctx context.Context) error {
	// Apply admin-role.yaml (ServiceAccount + RoleBinding + ClusterRoleBinding)
	if err := r.applyGettingStartedRBAC(ctx); err != nil {
		return err
	}

	// Apply webhook-role.yaml (Role + ServiceAccount + RoleBinding for webhook tasks)
	if err := r.applyWebhookRBAC(ctx); err != nil {
		return err
	}

	return nil
}

// applyGettingStartedRBAC applies the admin-role.yaml from getting-started docs
func (r *RBACManager) applyGettingStartedRBAC(ctx context.Context) error {
	if err := applyFileFromGitHub("docs/getting-started/rbac/admin-role.yaml"); err != nil {
		return fmt.Errorf("failed to apply admin-role.yaml: %w", err)
	}

	if err := applyFileFromGitHub("docs/getting-started/rbac/clusterrolebinding.yaml"); err != nil {
		return fmt.Errorf("failed to apply clusterrolebinding.yaml: %w", err)
	}
	return nil
}

// applyWebhookRBAC applies webhook-role.yaml from getting-started docs
func (r *RBACManager) applyWebhookRBAC(ctx context.Context) error {
	if err := applyFileFromGitHub("docs/getting-started/rbac/webhook-role.yaml"); err != nil {
		return fmt.Errorf("failed to apply webhook-role.yaml: %w", err)
	}
	return nil
}

// CreateWebhookSecret creates a Kubernetes secret for webhook authentication
func (r *RBACManager) CreateWebhookSecret(ctx context.Context, webhookSecret string) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "github-webhook-secret",
			Namespace: r.config.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/part-of": "tekton-triggers",
				"tekton.dev/bootstrap":      "true",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"webhook-secret": []byte(webhookSecret),
		},
	}

	_, err := r.kubeClient.CoreV1().Secrets(r.config.Namespace).Create(ctx, secret, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to create webhook secret: %w", err)
	}

	return nil
}
