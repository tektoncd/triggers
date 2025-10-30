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

	triggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
)

type Bootstrapper struct {
	config           *Config
	kubeClient       kubernetes.Interface
	triggersClient   triggersclientset.Interface
	installer        *Installer
	rbacManager      *RBACManager
	templatesManager *TemplatesManager
	githubManager    *GitHubManager
}

// New creates a new bootstrapper
func New(config *Config) (*Bootstrapper, error) {
	var (
		kubeClient     kubernetes.Interface
		triggersClient triggersclientset.Interface
		err            error
	)
	kubeClient, err = kubernetes.NewForConfig(config.KubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	triggersClient, err = triggersclientset.NewForConfig(config.KubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create triggers client: %w", err)
	}

	// Create component managers
	installer := NewInstaller(kubeClient, config)
	rbacManager := NewRBACManager(kubeClient, config)
	templatesManager := NewTemplatesManager(triggersClient, config)
	githubManager := NewGitHubManager(config)

	return &Bootstrapper{
		config:           config,
		kubeClient:       kubeClient,
		triggersClient:   triggersClient,
		installer:        installer,
		rbacManager:      rbacManager,
		templatesManager: templatesManager,
		githubManager:    githubManager,
	}, nil
}

// Run executes the bootstrap process
func (b *Bootstrapper) Run(ctx context.Context) error {
	// Check and install Tekton Pipelines
	if err := b.installer.InstallTektonPipelines(ctx); err != nil {
		return fmt.Errorf("failed to ensure pipelines: %w", err)
	}

	// Install Tekton Triggers
	if err := b.installer.InstallTriggers(ctx); err != nil {
		return fmt.Errorf("failed to install triggers: %w", err)
	}

	// Create/verify namespace
	if err := b.rbacManager.CreateNamespace(ctx); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	// Set up RBAC
	if err := b.rbacManager.SetupRBAC(ctx); err != nil {
		return fmt.Errorf("failed to setup RBAC: %w", err)
	}

	// Create Trigger resources (EventListener, TriggerTemplate, TriggerBinding)
	if err := b.templatesManager.CreateTriggerResources(ctx); err != nil {
		return fmt.Errorf("failed to create trigger resources: %w", err)
	}

	// Create examples (Pipeline)
	if err := b.templatesManager.CreateExamples(ctx); err != nil {
		return fmt.Errorf("failed to create examples: %w", err)
	}

	// Check existing webhooks
	if b.config.GitHubToken != "" {
		if err := b.githubManager.SetupWebhook(ctx); err != nil {
			return fmt.Errorf("failed to setup webhook: %w", err)
		}
	}
	if err := b.rbacManager.CreateWebhookSecret(ctx, b.githubManager.GetWebhookSecret()); err != nil {
		return fmt.Errorf("failed to create webhook secret: %w", err)
	}
	return nil
}
