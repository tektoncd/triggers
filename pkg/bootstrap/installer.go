/*
Copyright 2023 The Tekton Authors

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
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

type Installer struct {
	kubeClient kubernetes.Interface
	config     *Config
}

func NewInstaller(kubeClient kubernetes.Interface, config *Config) *Installer {
	return &Installer{
		kubeClient: kubeClient,
		config:     config,
	}
}

// InstallTriggers installs Tekton Triggers CRDs and controllers
func (i *Installer) InstallTriggers(ctx context.Context) error {
	// Check if Triggers is already installed
	if i.isTriggersInstalled(ctx) {
		log.Println("Tekton Triggers is already installed.")
		return nil
	}
	if err := i.downloadAndApplyTriggers(ctx); err != nil {
		return fmt.Errorf("failed to install Triggers: %w", err)
	}

	log.Println("Waiting for Tekton Triggers to be ready...")
	return i.waitForTriggersReady(ctx)
}

func (i *Installer) isTriggersInstalled(ctx context.Context) bool {
	if i.kubeClient == nil {
		return false
	}
	// Check for the existence of the Triggers namespace and deployments
	_, err := i.kubeClient.AppsV1().Deployments("tekton-pipelines").Get(ctx, "tekton-triggers-controller", metav1.GetOptions{})
	return err == nil
}

// waitForTriggersReady waits for Tekton Triggers components to be ready
func (i *Installer) waitForTriggersReady(ctx context.Context) error {
	// Wait for the tekton-pipelines namespace to exist
	err := wait.PollImmediate(2*time.Second, 5*time.Minute, func() (bool, error) {
		_, err := i.kubeClient.CoreV1().Namespaces().Get(ctx, "tekton-pipelines", metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return false, err
		}
		return err == nil, err
	})

	if err != nil {
		return fmt.Errorf("timeout waiting for tekton-pipelines namespace: %w", err)
	}
	time.Sleep(2 * time.Second)
	log.Println("Tekton Triggers is ready!")

	return nil
}

// InstallTektonPipelines installs Tekton Pipelines if requested and not present
func (i *Installer) InstallTektonPipelines(ctx context.Context) error {
	if !i.config.InstallDeps {
		// Check if Pipelines is installed, it's required for Triggers
		if !i.isPipelinesInstalled(ctx) {
			return errors.New("tekton Pipelines is not installed and is required for Triggers")
		}
		return nil
	}

	if i.isPipelinesInstalled(ctx) {
		log.Println("Tekton Pipelines is already installed.")
		return nil
	}

	if err := i.downloadAndApplyPipelines(ctx); err != nil {
		return fmt.Errorf("failed to install Pipelines: %w", err)
	}

	// Wait for Pipelines to be ready before continuing
	log.Println("Waiting for Pipelines to be ready, this may take a minute or two...")
	if err := i.waitForPipelinesReady(ctx); err != nil {
		return fmt.Errorf("failed to wait for Pipelines: %w", err)
	}

	log.Println("Tekton Pipelines is ready!")
	return nil
}

// isPipelinesInstalled checks if Tekton Pipelines is installed
func (i *Installer) isPipelinesInstalled(ctx context.Context) bool {
	if i.kubeClient == nil {
		return false // In dry-run, assume nothing is installed
	}
	_, err := i.kubeClient.AppsV1().Deployments("tekton-pipelines").Get(ctx, "tekton-pipelines-controller", metav1.GetOptions{})
	return err == nil
}

func (i *Installer) downloadAndApplyTriggers(ctx context.Context) error {
	triggersURL := "https://storage.googleapis.com/tekton-releases/triggers/latest/release.yaml"
	interceptorsURL := "https://storage.googleapis.com/tekton-releases/triggers/latest/interceptors.yaml"

	if err := i.downloadAndApplyManifest(ctx, triggersURL); err != nil {
		return fmt.Errorf("failed to apply triggers manifest: %w", err)
	}

	if err := i.downloadAndApplyManifest(ctx, interceptorsURL); err != nil {
		return fmt.Errorf("failed to apply interceptors manifest: %w", err)
	}

	return nil
}

func (i *Installer) downloadAndApplyPipelines(ctx context.Context) error {
	pipelinesURL := "https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml"

	if err := i.downloadAndApplyManifest(ctx, pipelinesURL); err != nil {
		return fmt.Errorf("failed to apply pipelines manifest: %w", err)
	}

	return nil
}

// downloadAndApplyManifest downloads a YAML manifest from URL and applies it to the cluster
func (i *Installer) downloadAndApplyManifest(ctx context.Context, url string) error {
	// Download the manifest
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download manifest: HTTP %d", resp.StatusCode)
	}

	// Read the YAML content
	yamlContent, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	// Apply the manifest using kubectl-like logic
	if err := i.applyYAMLManifest(ctx, yamlContent); err != nil {
		return fmt.Errorf("failed to apply manifest: %w", err)
	}

	return nil
}

// applyYAMLManifest applies a multi-document YAML manifest
func (i *Installer) applyYAMLManifest(ctx context.Context, yamlContent []byte) error {
	if err := i.applyManifestViaKubectl(ctx, yamlContent); err != nil {
		return fmt.Errorf("failed to apply manifest: %w", err)
	}

	return nil
}

// applyManifestViaKubectl applies YAML using kubectl apply
func (i *Installer) applyManifestViaKubectl(ctx context.Context, yamlContent []byte) error {
	// Write YAML to temporary file
	tmpFile, err := os.CreateTemp("", "tekton-manifest-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.Write(yamlContent); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}
	tmpFile.Close()

	kubeconfigFlag := ""
	if i.config.KubeConfig != nil {
		kubeconfigFlag = "--kubeconfig=" + os.Getenv("HOME") + "/.kube/config"
	}

	// #nosec G204 -- kubectl is a known binary, tmpFile.Name() is a controlled temp file path
	cmd := exec.CommandContext(ctx, "kubectl", "apply", "-f", tmpFile.Name())
	if kubeconfigFlag != "" {
		// #nosec G204 -- kubectl is a known binary, tmpFile.Name() is a controlled temp file path
		cmd = exec.CommandContext(ctx, "kubectl", kubeconfigFlag, "apply", "-f", tmpFile.Name())
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kubectl apply failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// waitForPipelinesReady waits for Tekton Pipelines to be fully ready
func (i *Installer) waitForPipelinesReady(ctx context.Context) error {
	if i.kubeClient == nil {
		return nil
	}

	// Wait for the tekton-pipelines namespace to exist and be active
	err := wait.PollImmediate(5*time.Second, 10*time.Minute, func() (bool, error) {
		ns, err := i.kubeClient.CoreV1().Namespaces().Get(ctx, "tekton-pipelines", metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		// Check if namespace is active
		if ns.Status.Phase != "Active" {
			return false, nil
		}

		return true, nil
	})

	if err != nil {
		return fmt.Errorf("timeout waiting for tekton-pipelines namespace: %w", err)
	}

	// Wait for Pipelines controller deployment to be ready
	err = wait.PollImmediate(5*time.Second, 10*time.Minute, func() (bool, error) {
		deployment, err := i.kubeClient.AppsV1().Deployments("tekton-pipelines").Get(ctx, "tekton-pipelines-controller", metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		// Check if deployment is ready
		if deployment.Status.ReadyReplicas == 0 || deployment.Status.ReadyReplicas < *deployment.Spec.Replicas {
			return false, nil
		}

		return true, nil
	})

	if err != nil {
		return fmt.Errorf("timeout waiting for tekton-pipelines-controller: %w", err)
	}

	// wait for webhook deployment to be ready
	err = wait.PollImmediate(5*time.Second, 10*time.Minute, func() (bool, error) {
		deployment, err := i.kubeClient.AppsV1().Deployments("tekton-pipelines").Get(ctx, "tekton-pipelines-webhook", metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		// Check if webhook deployment is ready
		if deployment.Status.ReadyReplicas == 0 || deployment.Status.ReadyReplicas < *deployment.Spec.Replicas {
			return false, nil
		}

		return true, nil
	})

	if err != nil {
		return fmt.Errorf("timeout waiting for tekton-pipelines: %w", err)
	}

	// buffer for webhook service to be fully responsive
	time.Sleep(10 * time.Second)
	return nil
}
