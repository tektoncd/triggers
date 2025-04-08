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

package fake

import (
	"context"

	v1beta1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeTriggers implements TriggerInterface
type FakeTriggers struct {
	Fake *FakeTriggersV1beta1
	ns   string
}

var triggersResource = v1beta1.SchemeGroupVersion.WithResource("triggers")

var triggersKind = v1beta1.SchemeGroupVersion.WithKind("Trigger")

// Get takes name of the trigger, and returns the corresponding trigger object, and an error if there is any.
func (c *FakeTriggers) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1beta1.Trigger, err error) {
	emptyResult := &v1beta1.Trigger{}
	obj, err := c.Fake.
		Invokes(testing.NewGetActionWithOptions(triggersResource, c.ns, name, options), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1beta1.Trigger), err
}

// List takes label and field selectors, and returns the list of Triggers that match those selectors.
func (c *FakeTriggers) List(ctx context.Context, opts v1.ListOptions) (result *v1beta1.TriggerList, err error) {
	emptyResult := &v1beta1.TriggerList{}
	obj, err := c.Fake.
		Invokes(testing.NewListActionWithOptions(triggersResource, triggersKind, c.ns, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1beta1.TriggerList{ListMeta: obj.(*v1beta1.TriggerList).ListMeta}
	for _, item := range obj.(*v1beta1.TriggerList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested triggers.
func (c *FakeTriggers) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchActionWithOptions(triggersResource, c.ns, opts))

}

// Create takes the representation of a trigger and creates it.  Returns the server's representation of the trigger, and an error, if there is any.
func (c *FakeTriggers) Create(ctx context.Context, trigger *v1beta1.Trigger, opts v1.CreateOptions) (result *v1beta1.Trigger, err error) {
	emptyResult := &v1beta1.Trigger{}
	obj, err := c.Fake.
		Invokes(testing.NewCreateActionWithOptions(triggersResource, c.ns, trigger, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1beta1.Trigger), err
}

// Update takes the representation of a trigger and updates it. Returns the server's representation of the trigger, and an error, if there is any.
func (c *FakeTriggers) Update(ctx context.Context, trigger *v1beta1.Trigger, opts v1.UpdateOptions) (result *v1beta1.Trigger, err error) {
	emptyResult := &v1beta1.Trigger{}
	obj, err := c.Fake.
		Invokes(testing.NewUpdateActionWithOptions(triggersResource, c.ns, trigger, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1beta1.Trigger), err
}

// Delete takes name of the trigger and deletes it. Returns an error if one occurs.
func (c *FakeTriggers) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(triggersResource, c.ns, name, opts), &v1beta1.Trigger{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeTriggers) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionActionWithOptions(triggersResource, c.ns, opts, listOpts)

	_, err := c.Fake.Invokes(action, &v1beta1.TriggerList{})
	return err
}

// Patch applies the patch and returns the patched trigger.
func (c *FakeTriggers) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1beta1.Trigger, err error) {
	emptyResult := &v1beta1.Trigger{}
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceActionWithOptions(triggersResource, c.ns, name, pt, data, opts, subresources...), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1beta1.Trigger), err
}
