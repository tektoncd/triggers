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
)

// TemplatesManager handles creation of Trigger templates and resources
type TemplatesManager struct {
	triggersClient triggersclientset.Interface
	config         *Config
}

// NewTemplatesManager creates a new templates manager
func NewTemplatesManager(triggersClient triggersclientset.Interface, config *Config) *TemplatesManager {
	return &TemplatesManager{
		triggersClient: triggersClient,
		config:         config,
	}
}

// CreateTriggerResources creates Trigger resources from getting-started docs
func (t *TemplatesManager) CreateTriggerResources(ctx context.Context) error {
	return t.applyGettingStartedTriggers(ctx)
}

// CreateExamples creates the Pipeline from getting-started docs
func (t *TemplatesManager) CreateExamples(ctx context.Context) error {
	return t.applyGettingStartedPipeline(ctx)
}

// applyGettingStartedTriggers applies triggers.yaml from getting-started docs
func (t *TemplatesManager) applyGettingStartedTriggers(ctx context.Context) error {
	if err := applyFileFromGitHub("docs/getting-started/triggers.yaml"); err != nil {
		return fmt.Errorf("failed to apply triggers.yaml: %w", err)
	}
	return nil
}

// applyGettingStartedPipeline applies pipeline.yaml from getting-started docs
func (t *TemplatesManager) applyGettingStartedPipeline(ctx context.Context) error {
	if err := applyFileFromGitHub("docs/getting-started/pipeline.yaml"); err != nil {
		return fmt.Errorf("failed to apply pipeline.yaml: %w", err)
	}
	return nil
}
