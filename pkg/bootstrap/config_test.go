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
	"testing"

	"k8s.io/client-go/rest"
)

func TestConfig(t *testing.T) {
	config := &Config{
		Namespace:      "test-namespace",
		Provider:       "github",
		InstallDeps:    true,
		CreateExamples: true,
		KubeConfig:     &rest.Config{},
		GitHubRepo:     "owner/repo",
		GitHubToken:    "test-token",
		PublicDomain:   "example.com",
		WebhookSecret:  "secret",
	}

	if config.Namespace != "test-namespace" {
		t.Errorf("Config.Namespace = %v, want test-namespace", config.Namespace)
	}
	if config.Provider != "github" {
		t.Errorf("Config.Provider = %v, want github", config.Provider)
	}
	if !config.InstallDeps {
		t.Error("Config.InstallDeps should be true")
	}
	if !config.CreateExamples {
		t.Error("Config.CreateExamples should be true")
	}
	if config.GitHubRepo != "owner/repo" {
		t.Errorf("Config.GitHubRepo = %v, want owner/repo", config.GitHubRepo)
	}
	if config.GitHubToken != "test-token" {
		t.Errorf("Config.GitHubToken = %v, want test-token", config.GitHubToken)
	}
	if config.PublicDomain != "example.com" {
		t.Errorf("Config.PublicDomain = %v, want example.com", config.PublicDomain)
	}
	if config.WebhookSecret != "secret" {
		t.Errorf("Config.WebhookSecret = %v, want secret", config.WebhookSecret)
	}
}

func TestTemplateData(t *testing.T) {
	data := &TemplateData{
		Namespace:      "test-namespace",
		Name:           "test-name",
		ServiceAccount: "test-sa",
		Provider:       "github",
	}

	if data.Namespace != "test-namespace" {
		t.Errorf("TemplateData.Namespace = %v, want test-namespace", data.Namespace)
	}
	if data.Name != "test-name" {
		t.Errorf("TemplateData.Name = %v, want test-name", data.Name)
	}
	if data.ServiceAccount != "test-sa" {
		t.Errorf("TemplateData.ServiceAccount = %v, want test-sa", data.ServiceAccount)
	}
	if data.Provider != "github" {
		t.Errorf("TemplateData.Provider = %v, want github", data.Provider)
	}
}
