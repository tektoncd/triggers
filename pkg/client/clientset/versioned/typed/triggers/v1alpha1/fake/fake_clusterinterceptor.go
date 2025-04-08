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
	v1alpha1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	triggersv1alpha1 "github.com/tektoncd/triggers/pkg/client/clientset/versioned/typed/triggers/v1alpha1"
	gentype "k8s.io/client-go/gentype"
)

// fakeClusterInterceptors implements ClusterInterceptorInterface
type fakeClusterInterceptors struct {
	*gentype.FakeClientWithList[*v1alpha1.ClusterInterceptor, *v1alpha1.ClusterInterceptorList]
	Fake *FakeTriggersV1alpha1
}

func newFakeClusterInterceptors(fake *FakeTriggersV1alpha1) triggersv1alpha1.ClusterInterceptorInterface {
	return &fakeClusterInterceptors{
		gentype.NewFakeClientWithList[*v1alpha1.ClusterInterceptor, *v1alpha1.ClusterInterceptorList](
			fake.Fake,
			"",
			v1alpha1.SchemeGroupVersion.WithResource("clusterinterceptors"),
			v1alpha1.SchemeGroupVersion.WithKind("ClusterInterceptor"),
			func() *v1alpha1.ClusterInterceptor { return &v1alpha1.ClusterInterceptor{} },
			func() *v1alpha1.ClusterInterceptorList { return &v1alpha1.ClusterInterceptorList{} },
			func(dst, src *v1alpha1.ClusterInterceptorList) { dst.ListMeta = src.ListMeta },
			func(list *v1alpha1.ClusterInterceptorList) []*v1alpha1.ClusterInterceptor {
				return gentype.ToPointerSlice(list.Items)
			},
			func(list *v1alpha1.ClusterInterceptorList, items []*v1alpha1.ClusterInterceptor) {
				list.Items = gentype.FromPointerSlice(items)
			},
		),
		fake,
	}
}
