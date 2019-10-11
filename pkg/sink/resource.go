/*
Copyright 2019 The Tekton Authors

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

package sink

import (
	"fmt"

	v1alpha1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

type resource struct {
	resources []string
	recorder  record.EventRecorder
	source    *v1alpha1.EventListener
}

var _ cache.Store = (*resource)(nil)

func (a *resource) Add(obj interface{}) error {
	return nil
}

func (a *resource) Update(obj interface{}) error {
	if obj == nil {
		return fmt.Errorf("resource can not be nil")
	}
	object := obj.(*unstructured.Unstructured)
	resourceName := object.GetName()
	resourceKind := object.GetKind()
	if contains(a.resources, resourceName) {
		a.recorder.Eventf(a.source, corev1.EventTypeWarning, "Dependency Changed ", "Resource: %s updated", resourceKind+"/"+resourceName)
	}

	return nil
}

func (a *resource) Delete(obj interface{}) error {
	if obj == nil {
		return fmt.Errorf("resource can not be nil")
	}
	object := obj.(*unstructured.Unstructured)
	resourceName := object.GetName()
	resourceKind := object.GetKind()
	if contains(a.resources, resourceName) {
		a.recorder.Eventf(a.source, corev1.EventTypeWarning, "Dependency Changed ", "Resource: %s deleted", resourceKind+"/"+resourceName)
	}

	return nil
}

// Stub cache.Store impl

// Implements cache.Store
func (a *resource) List() []interface{} {
	return nil
}

// Implements cache.Store
func (a *resource) ListKeys() []string {
	return nil
}

// Implements cache.Store
func (a *resource) Get(obj interface{}) (item interface{}, exists bool, err error) {
	return nil, false, nil
}

// Implements cache.Store
func (a *resource) GetByKey(key string) (item interface{}, exists bool, err error) {
	return nil, false, nil
}

// Implements cache.Store
func (a *resource) Replace([]interface{}, string) error {
	return nil
}

// Implements cache.Store
func (a *resource) Resync() error {
	return nil
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
