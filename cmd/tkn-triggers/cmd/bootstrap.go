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

package cmd

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/tektoncd/triggers/pkg/bootstrap"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	// Bootstrap flags
	kubeconfig string

	// GitHub integration flags
	githubRepo    string
	githubToken   string
	publicDomain  string
	webhookSecret string
)

// bootstrapCmd represents the bootstrap command
var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Bootstrap Tekton Triggers with all necessary resources",
	Long: `Bootstrap sets up Tekton Triggers with all necessary resources following
the getting-started guide. This includes:

- Installing Tekton Pipelines (if not present)
- Installing Tekton Triggers (if not present)  
- Creating the getting-started namespace
- Setting up RBAC (ServiceAccounts, Roles, RoleBindings)
- Creating EventListener, TriggerTemplate, TriggerBinding
- Creating example Pipeline and Tasks

Examples:
  # Basic bootstrap (auto-detects kubeconfig, creates all resources)
  tkn triggers bootstrap`,
	RunE: bootstrapRun,
}

func init() {
	bootstrapCmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file")
	bootstrapCmd.Flags().StringVar(&githubRepo, "github-repo", "", "GitHub repository (owner/repo)")
	bootstrapCmd.Flags().StringVar(&githubToken, "github-token", "", "GitHub personal access token")
	bootstrapCmd.Flags().StringVar(&publicDomain, "domain", "", "Public domain for webhooks (e.g. myapp.com)")
	bootstrapCmd.Flags().StringVar(&webhookSecret, "webhook-secret", "", "Webhook secret (auto-generated if empty)")
}

func bootstrapRun(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Create kubernetes config
	var config *rest.Config
	var err error
	if kubeconfig != "" {
		// Use kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		// try ~/.kube/config first, then in-cluster
		kubeconfigPath := os.Getenv("HOME") + "/.kube/config"
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			config, err = rest.InClusterConfig()
		}
	}
	if err != nil {
		return fmt.Errorf("failed to build kubeconfig: %w", err)
	}

	// Create bootstrap configuration for getting-started resources
	log.Println("== Setting up Tekton Dependencies...")
	cfg := &bootstrap.Config{
		Namespace:      "getting-started",
		Provider:       "github",
		InstallDeps:    true,
		CreateExamples: true,
		KubeConfig:     config,

		GitHubRepo:    "",
		GitHubToken:   "",
		PublicDomain:  "",
		WebhookSecret: "",
	}

	// Create bootstrapper
	bootstrapper, err := bootstrap.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create bootstrapper: %w", err)
	}

	// Run bootstrap (create namespace, RBAC, EventListener, Pipeline)
	if err := bootstrapper.Run(ctx); err != nil {
		return fmt.Errorf("bootstrap failed: %w", err)
	}

	log.Println("Tekton Dependencies are ready!")

	// run GitHub integration setup
	log.Println("\n== Configuring GitHub Integration ==")
	if err := runInteractiveSetup(); err != nil {
		log.Printf("⚠️  GitHub setup failed: %v\n", err)
		return nil
	}

	// GitHub webhook creation if have the required info
	if githubRepo != "" && githubToken != "" && publicDomain != "" {
		log.Println("== Creating GitHub webhook ==")
		githubConfig := &bootstrap.Config{
			Namespace:     "getting-started",
			GitHubRepo:    githubRepo,
			GitHubToken:   githubToken,
			PublicDomain:  publicDomain,
			WebhookSecret: webhookSecret,
			KubeConfig:    config,
		}

		// Create GitHub manager and setup webhook
		githubManager := bootstrap.NewGitHubManager(githubConfig)
		if err := githubManager.SetupWebhook(ctx); err != nil {
			return fmt.Errorf("GitHub webhook setup failed: %w", err)
		}
		log.Println("GitHub webhook created!")
	} else {
		log.Println("⚠️  Missing required information")
	}
	return nil
}
