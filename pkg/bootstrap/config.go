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
	"k8s.io/client-go/rest"
)

type Config struct {
	// Namespace is the target namespace for Triggers resources
	Namespace string

	// Provider is the Git provider (currently only "github" is supported)
	Provider string

	// InstallDeps tells whether to install Tekton Pipelines
	InstallDeps bool

	// CreateExamples tells whether to create example resources
	CreateExamples bool

	// Kubernetes client configuration
	KubeConfig *rest.Config

	// configuration
	GitHubRepo    string // GitHub repo (owner/repo)
	GitHubToken   string // GitHub PAT
	PublicDomain  string // Public domain for webhooks
	WebhookSecret string // Webhook secret
}

type TemplateData struct {
	Namespace      string
	Name           string
	ServiceAccount string
	Provider       string
}
