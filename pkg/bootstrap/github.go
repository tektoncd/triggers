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
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type GitHubManager struct {
	config *Config
	client *http.Client
}

func NewGitHubManager(config *Config) *GitHubManager {
	return &GitHubManager{
		config: config,
		client: &http.Client{},
	}
}

// WebhookPayload represents the GitHub webhook payload
type WebhookPayload struct {
	Name   string        `json:"name"`
	Config WebhookConfig `json:"config"`
	Events []string      `json:"events"`
	Active bool          `json:"active"`
}

// WebhookConfig represents webhook configuration
type WebhookConfig struct {
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	Secret      string `json:"secret,omitempty"`
	InsecureSSL string `json:"insecure_ssl"`
}

// SetupWebhook creates a GitHub webhook for the repository
func (g *GitHubManager) SetupWebhook(ctx context.Context) error {
	// Generate webhook secret
	secret := g.config.WebhookSecret
	if secret == "" {
		var err error
		secret, err = generateWebhookSecret()
		if err != nil {
			return fmt.Errorf("failed to generate webhook secret: %w", err)
		}
		g.config.WebhookSecret = secret
	}

	// Create webhook
	webhookURL := fmt.Sprintf("https://%s/hooks", g.config.PublicDomain)

	payload := WebhookPayload{
		Name: "web",
		Config: WebhookConfig{
			URL:         webhookURL,
			ContentType: "json",
			Secret:      secret,
			InsecureSSL: "0", // Always use SSL in production
		},
		Events: []string{"push", "pull_request"},
		Active: true,
	}

	if err := g.createWebhook(ctx, payload); err != nil {
		return fmt.Errorf("failed to create webhook: %w", err)
	}

	return nil
}

// createWebhook makes the API call to create the webhook
func (g *GitHubManager) createWebhook(ctx context.Context, payload WebhookPayload) error {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/hooks", g.config.GitHubRepo)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "token "+g.config.GitHubToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusUnprocessableEntity && strings.Contains(string(body), "Hook already exists") {
			return errors.New("webhook already exists")
		}
		return fmt.Errorf("GitHub API error: %s - %s", resp.Status, string(body))
	}

	return nil
}

// generateWebhookSecret generates a secure random webhook secret
func generateWebhookSecret() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// GetWebhookSecret returns the webhook secret for storing in Kubernetes
func (g *GitHubManager) GetWebhookSecret() string {
	return g.config.WebhookSecret
}
