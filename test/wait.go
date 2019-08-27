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

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	interval = 1 * time.Second
	timeout  = 30 * time.Second
)

// WaitForDeployment polls for the existence/non-existence of the specified Deployment
func WaitForDeployment(c *clients, namespace, name string, exists bool) error {
	return wait.PollImmediate(interval, timeout, func() (bool, error) {
		_, err := c.KubeClient.AppsV1().Deployments(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return !exists, nil
			}
			return exists, err
		}
		return true, nil
	})
}

// WaitForService polls for the existence/non-existence of the specified Service
func WaitForService(c *clients, namespace, name string, exists bool) error {
	return wait.PollImmediate(interval, timeout, func() (bool, error) {
		_, err := c.KubeClient.CoreV1().Services(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return !exists, nil
			}
			return exists, err
		}
		return true, nil
	})
}

// WaitForRole polls for the existence/non-existence of the specified Role
func WaitForRole(c *clients, namespace, name string, exists bool) error {
	return wait.PollImmediate(interval, timeout, func() (bool, error) {
		_, err := c.KubeClient.RbacV1().Roles(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return !exists, nil
			}
			return exists, err
		}
		return true, nil
	})
}

// WaitForRoleBinding polls for the existence/non-existence of the specified RoleBinding
func WaitForRoleBinding(c *clients, namespace, name string, exists bool) error {
	return wait.PollImmediate(interval, timeout, func() (bool, error) {
		_, err := c.KubeClient.RbacV1().RoleBindings(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return !exists, nil
			}
			return exists, err
		}
		return true, nil
	})
}
