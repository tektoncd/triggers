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
)

func TestNewGitHubManager(t *testing.T) {
	config := &Config{
		GitHubRepo:    "owner/repo",
		GitHubToken:   "test-token",
		PublicDomain:  "example.com",
		WebhookSecret: "secret",
	}

	manager := NewGitHubManager(config)

	if manager == nil {
		t.Fatal("NewGitHubManager() returned nil")
	}
	if manager.config == nil {
		t.Error("NewGitHubManager() config is nil")
	}
	if manager.client == nil {
		t.Error("NewGitHubManager() client is nil")
	}
}

func TestGetWebhookSecret(t *testing.T) {
	tests := []struct {
		name          string
		webhookSecret string
	}{
		{
			name:          "with secret",
			webhookSecret: "test-secret",
		},
		{
			name:          "empty secret",
			webhookSecret: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				WebhookSecret: tt.webhookSecret,
			}
			manager := NewGitHubManager(config)

			got := manager.GetWebhookSecret()
			if got != tt.webhookSecret {
				t.Errorf("GetWebhookSecret() = %v, want %v", got, tt.webhookSecret)
			}
		})
	}
}

func TestWebhookPayload(t *testing.T) {
	payload := WebhookPayload{
		Name:   "web",
		Active: true,
		Events: []string{"push", "pull_request"},
	}
	payload.Config.URL = "https://example.com/hooks"
	payload.Config.ContentType = "json"
	payload.Config.Secret = "secret"
	payload.Config.InsecureSSL = "0"

	if payload.Name != "web" {
		t.Errorf("WebhookPayload.Name = %v, want web", payload.Name)
	}
	if !payload.Active {
		t.Error("WebhookPayload.Active should be true")
	}
	if len(payload.Events) != 2 {
		t.Errorf("WebhookPayload.Events length = %v, want 2", len(payload.Events))
	}
	if payload.Config.URL != "https://example.com/hooks" {
		t.Errorf("WebhookPayload.Config.URL = %v, want https://example.com/hooks", payload.Config.URL)
	}
	if payload.Config.ContentType != "json" {
		t.Errorf("WebhookPayload.Config.ContentType = %v, want json", payload.Config.ContentType)
	}
	if payload.Config.Secret != "secret" {
		t.Errorf("WebhookPayload.Config.Secret = %v, want secret", payload.Config.Secret)
	}
	if payload.Config.InsecureSSL != "0" {
		t.Errorf("WebhookPayload.Config.InsecureSSL = %v, want 0", payload.Config.InsecureSSL)
	}
}
