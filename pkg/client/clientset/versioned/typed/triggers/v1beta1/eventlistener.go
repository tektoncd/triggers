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

// Code generated by client-gen. DO NOT EDIT.

package v1beta1

import (
	context "context"

	triggersv1beta1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	scheme "github.com/tektoncd/triggers/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	gentype "k8s.io/client-go/gentype"
)

// EventListenersGetter has a method to return a EventListenerInterface.
// A group's client should implement this interface.
type EventListenersGetter interface {
	EventListeners(namespace string) EventListenerInterface
}

// EventListenerInterface has methods to work with EventListener resources.
type EventListenerInterface interface {
	Create(ctx context.Context, eventListener *triggersv1beta1.EventListener, opts v1.CreateOptions) (*triggersv1beta1.EventListener, error)
	Update(ctx context.Context, eventListener *triggersv1beta1.EventListener, opts v1.UpdateOptions) (*triggersv1beta1.EventListener, error)
	// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
	UpdateStatus(ctx context.Context, eventListener *triggersv1beta1.EventListener, opts v1.UpdateOptions) (*triggersv1beta1.EventListener, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*triggersv1beta1.EventListener, error)
	List(ctx context.Context, opts v1.ListOptions) (*triggersv1beta1.EventListenerList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *triggersv1beta1.EventListener, err error)
	EventListenerExpansion
}

// eventListeners implements EventListenerInterface
type eventListeners struct {
	*gentype.ClientWithList[*triggersv1beta1.EventListener, *triggersv1beta1.EventListenerList]
}

// newEventListeners returns a EventListeners
func newEventListeners(c *TriggersV1beta1Client, namespace string) *eventListeners {
	return &eventListeners{
		gentype.NewClientWithList[*triggersv1beta1.EventListener, *triggersv1beta1.EventListenerList](
			"eventlisteners",
			c.RESTClient(),
			scheme.ParameterCodec,
			namespace,
			func() *triggersv1beta1.EventListener { return &triggersv1beta1.EventListener{} },
			func() *triggersv1beta1.EventListenerList { return &triggersv1beta1.EventListenerList{} },
		),
	}
}
