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

// ClusterTriggerBindingsGetter has a method to return a ClusterTriggerBindingInterface.
// A group's client should implement this interface.
type ClusterTriggerBindingsGetter interface {
	ClusterTriggerBindings() ClusterTriggerBindingInterface
}

// ClusterTriggerBindingInterface has methods to work with ClusterTriggerBinding resources.
type ClusterTriggerBindingInterface interface {
	Create(ctx context.Context, clusterTriggerBinding *triggersv1beta1.ClusterTriggerBinding, opts v1.CreateOptions) (*triggersv1beta1.ClusterTriggerBinding, error)
	Update(ctx context.Context, clusterTriggerBinding *triggersv1beta1.ClusterTriggerBinding, opts v1.UpdateOptions) (*triggersv1beta1.ClusterTriggerBinding, error)
	// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
	UpdateStatus(ctx context.Context, clusterTriggerBinding *triggersv1beta1.ClusterTriggerBinding, opts v1.UpdateOptions) (*triggersv1beta1.ClusterTriggerBinding, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*triggersv1beta1.ClusterTriggerBinding, error)
	List(ctx context.Context, opts v1.ListOptions) (*triggersv1beta1.ClusterTriggerBindingList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *triggersv1beta1.ClusterTriggerBinding, err error)
	ClusterTriggerBindingExpansion
}

// clusterTriggerBindings implements ClusterTriggerBindingInterface
type clusterTriggerBindings struct {
	*gentype.ClientWithList[*triggersv1beta1.ClusterTriggerBinding, *triggersv1beta1.ClusterTriggerBindingList]
}

// newClusterTriggerBindings returns a ClusterTriggerBindings
func newClusterTriggerBindings(c *TriggersV1beta1Client) *clusterTriggerBindings {
	return &clusterTriggerBindings{
		gentype.NewClientWithList[*triggersv1beta1.ClusterTriggerBinding, *triggersv1beta1.ClusterTriggerBindingList](
			"clustertriggerbindings",
			c.RESTClient(),
			scheme.ParameterCodec,
			"",
			func() *triggersv1beta1.ClusterTriggerBinding { return &triggersv1beta1.ClusterTriggerBinding{} },
			func() *triggersv1beta1.ClusterTriggerBindingList { return &triggersv1beta1.ClusterTriggerBindingList{} },
		),
	}
}
