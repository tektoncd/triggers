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
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	githubBaseURL = "https://raw.githubusercontent.com/tektoncd/triggers/main"
)

func applyFileFromGitHub(filePath string) error {
	// Download from GitHub
	githubURL := fmt.Sprintf("%s/%s", githubBaseURL, filePath)

	// Create temporary file
	tmpFile, err := os.CreateTemp("", filepath.Base(filePath)+"-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name()) // Always clean up

	// Download content
	resp, err := http.Get(githubURL)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", githubURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download %s: HTTP %d", githubURL, resp.StatusCode)
	}

	// Write to temp file
	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write downloaded content: %w", err)
	}
	tmpFile.Close()

	// Apply with kubectl
	cmd := exec.Command("kubectl", "apply", "-f", tmpFile.Name(), "-n", "getting-started")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kubectl apply failed for %s: %w\nOutput: %s", filePath, err, string(output))
	}

	fmt.Printf("  Applied %s:\n%s", filePath, string(output))
	return nil
}
