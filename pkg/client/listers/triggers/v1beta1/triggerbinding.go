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

// Code generated by lister-gen. DO NOT EDIT.

package v1beta1

import (
	v1beta1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// TriggerBindingLister helps list TriggerBindings.
// All objects returned here must be treated as read-only.
type TriggerBindingLister interface {
	// List lists all TriggerBindings in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1beta1.TriggerBinding, err error)
	// TriggerBindings returns an object that can list and get TriggerBindings.
	TriggerBindings(namespace string) TriggerBindingNamespaceLister
	TriggerBindingListerExpansion
}

// triggerBindingLister implements the TriggerBindingLister interface.
type triggerBindingLister struct {
	indexer cache.Indexer
}

// NewTriggerBindingLister returns a new TriggerBindingLister.
func NewTriggerBindingLister(indexer cache.Indexer) TriggerBindingLister {
	return &triggerBindingLister{indexer: indexer}
}

// List lists all TriggerBindings in the indexer.
func (s *triggerBindingLister) List(selector labels.Selector) (ret []*v1beta1.TriggerBinding, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1beta1.TriggerBinding))
	})
	return ret, err
}

// TriggerBindings returns an object that can list and get TriggerBindings.
func (s *triggerBindingLister) TriggerBindings(namespace string) TriggerBindingNamespaceLister {
	return triggerBindingNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// TriggerBindingNamespaceLister helps list and get TriggerBindings.
// All objects returned here must be treated as read-only.
type TriggerBindingNamespaceLister interface {
	// List lists all TriggerBindings in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1beta1.TriggerBinding, err error)
	// Get retrieves the TriggerBinding from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1beta1.TriggerBinding, error)
	TriggerBindingNamespaceListerExpansion
}

// triggerBindingNamespaceLister implements the TriggerBindingNamespaceLister
// interface.
type triggerBindingNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all TriggerBindings in the indexer for a given namespace.
func (s triggerBindingNamespaceLister) List(selector labels.Selector) (ret []*v1beta1.TriggerBinding, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1beta1.TriggerBinding))
	})
	return ret, err
}

// Get retrieves the TriggerBinding from the indexer for a given namespace and name.
func (s triggerBindingNamespaceLister) Get(name string) (*v1beta1.TriggerBinding, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1beta1.Resource("triggerbinding"), name)
	}
	return obj.(*v1beta1.TriggerBinding), nil
}
