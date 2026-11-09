// Copyright 2026 The Tekton Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build ignore

package main

import (
	"testing"

	"go.yaml.in/yaml/v3"
)

// Run with: go test ./hack/generate-dependabot.go ./hack/generate-dependabot_test.go
func TestGenerateDependabotCommitMessage(t *testing.T) {
	config := Config{
		ReleaseBranches: []string{"release-v0.35.x", "release-v0.34.x"},
		Ecosystems: []Ecosystem{
			{PackageEcosystem: "gomod", Directory: "/"},
			{PackageEcosystem: "github-actions", Directory: "/"},
		},
	}
	data, err := yaml.Marshal(generateDependabotConfig(config))
	if err != nil {
		t.Fatal(err)
	}
	var output struct {
		Updates []map[string]interface{} `yaml:"updates"`
	}
	if err := yaml.Unmarshal(data, &output); err != nil {
		t.Fatal(err)
	}
	if got, want := len(output.Updates), len(config.Ecosystems)*(1+len(config.ReleaseBranches)); got != want {
		t.Fatalf("got %d updates, want %d", got, want)
	}
	for i, update := range output.Updates {
		ecosystem := config.Ecosystems[i%len(config.Ecosystems)].PackageEcosystem
		if got := update["package-ecosystem"]; got != ecosystem {
			t.Errorf("update %d: ecosystem = %v, want %s", i, got, ecosystem)
		}
		message, present := update["commit-message"]
		if i < len(config.Ecosystems) {
			if present {
				t.Errorf("main %s: commit-message must be omitted, got %v", ecosystem, message)
			}
			continue
		}
		branch := config.ReleaseBranches[i/len(config.Ecosystems)-1]
		if got := update["target-branch"]; got != branch {
			t.Errorf("update %d: target-branch = %v, want %s", i, got, branch)
		}
		fields, ok := message.(map[string]interface{})
		if !ok || fields["prefix"] != "["+branch+"]" {
			t.Errorf("%s %s: commit-message = %v, want prefix [%s]", branch, ecosystem, message, branch)
		}
	}
}
