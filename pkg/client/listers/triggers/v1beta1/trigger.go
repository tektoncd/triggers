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
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/listers"
	"k8s.io/client-go/tools/cache"
)

// TriggerLister helps list Triggers.
// All objects returned here must be treated as read-only.
type TriggerLister interface {
	// List lists all Triggers in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1beta1.Trigger, err error)
	// Triggers returns an object that can list and get Triggers.
	Triggers(namespace string) TriggerNamespaceLister
	TriggerListerExpansion
}

// triggerLister implements the TriggerLister interface.
type triggerLister struct {
	listers.ResourceIndexer[*v1beta1.Trigger]
}

// NewTriggerLister returns a new TriggerLister.
func NewTriggerLister(indexer cache.Indexer) TriggerLister {
	return &triggerLister{listers.New[*v1beta1.Trigger](indexer, v1beta1.Resource("trigger"))}
}

// Triggers returns an object that can list and get Triggers.
func (s *triggerLister) Triggers(namespace string) TriggerNamespaceLister {
	return triggerNamespaceLister{listers.NewNamespaced[*v1beta1.Trigger](s.ResourceIndexer, namespace)}
}

// TriggerNamespaceLister helps list and get Triggers.
// All objects returned here must be treated as read-only.
type TriggerNamespaceLister interface {
	// List lists all Triggers in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1beta1.Trigger, err error)
	// Get retrieves the Trigger from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1beta1.Trigger, error)
	TriggerNamespaceListerExpansion
}

// triggerNamespaceLister implements the TriggerNamespaceLister
// interface.
type triggerNamespaceLister struct {
	listers.ResourceIndexer[*v1beta1.Trigger]
}
