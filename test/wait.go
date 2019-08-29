/*
Copyright 2019 The Tekton Authors.

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

/*
Poll Kubernetes resources

After creating Kubernetes resources or making changes to them, you will need to
wait for the system to realize those changes. You can use polling methods to
check the resources reach the desired state.

The `WaitFor*` functions use the Kubernetes
[`wait` package](https://godoc.org/k8s.io/apimachinery/pkg/util/wait). For
polling they use
[`PollImmediate`](https://godoc.org/k8s.io/apimachinery/pkg/util/wait#PollImmediate)
with a [`ConditionFunc`](https://godoc.org/k8s.io/apimachinery/pkg/util/wait#ConditionFunc)
callback function, which returns a `bool` to indicate if the polling should stop
and an `error` to indicate if there was an error.
*/

package test

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	interval = 1 * time.Second
	timeout  = 30 * time.Second
)

// WaitForDeploymentToExist polls for the existence of the Deployment called name
// in the specified namespace
func WaitForDeploymentToExist(c *clients, namespace, name string) error {
	return wait.PollImmediate(interval, timeout, func() (bool, error) {
		_, err := c.KubeClient.AppsV1().Deployments(namespace).Get(name, metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			return false, nil
		}
		return true, err
	})
}

// WaitForDeploymentToNotExist polls for the absence of the Deployment called
// name in the specified namespace
func WaitForDeploymentToNotExist(c *clients, namespace, name string) error {
	return wait.PollImmediate(interval, timeout, func() (bool, error) {
		_, err := c.KubeClient.AppsV1().Deployments(namespace).Get(name, metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
}

// WaitForServiceToExist polls for the existence of the Service called name in
// the specified namespace
func WaitForServiceToExist(c *clients, namespace, name string) error {
	return wait.PollImmediate(interval, timeout, func() (bool, error) {
		_, err := c.KubeClient.CoreV1().Services(namespace).Get(name, metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			return false, nil
		}
		return true, err
	})
}

// WaitForServiceToNotExist polls for the absence of the Service called name in
// the specified namespace
func WaitForServiceToNotExist(c *clients, namespace, name string) error {
	return wait.PollImmediate(interval, timeout, func() (bool, error) {
		_, err := c.KubeClient.CoreV1().Services(namespace).Get(name, metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
}

// WaitForTriggerTemplateToExist polls for the existence of the Deployment called name
// in the specified namespace
func WaitForTriggerTemplateToExist(c *clients, namespace, name string) error {
	return wait.PollImmediate(interval, timeout, func() (bool, error) {
		_, err := c.TriggersClient.TektonV1alpha1().TriggerTemplates(namespace).Get(name, metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			return false, nil
		}
		return true, err
	})
}

// WaitForPodRunning polls for the Pod called name in the specified namespace
// to be in the running phase
func WaitForPodRunning(c *clients, namespace, name string) error {
	return wait.PollImmediate(interval, timeout, func() (bool, error) {
		pod, err := c.KubeClient.CoreV1().Pods(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			return true, err
		}
		if pod.Status.Phase != corev1.PodRunning {
			return false, nil
		}
		return true, nil
	})
}

// WaitForExternalIP polls for the Service called svcName in the specified
// namespace to have a LoadBalancer status with an Ingress
func WaitForExternalIP(c *clients, namespace, svcName string) error {
	return wait.PollImmediate(interval, timeout, func() (bool, error) {
		svc, err := c.KubeClient.CoreV1().Services(namespace).Get(svcName, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			return true, err
		}
		if len(svc.Status.LoadBalancer.Ingress) == 0 {
			return false, nil
		}
		return true, nil
	})
}
