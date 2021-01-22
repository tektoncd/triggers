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

package test

import (
	"context"
	"testing"

	// Link in the fakes so they get injected into injection.Fake
	fakepipelineclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	fakepipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client/fake"
	fakeresourceclientset "github.com/tektoncd/pipeline/pkg/client/resource/clientset/versioned/fake"
	fakeresourceclient "github.com/tektoncd/pipeline/pkg/client/resource/injection/client/fake"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	faketriggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned/fake"
	dynamicclientset "github.com/tektoncd/triggers/pkg/client/dynamic/clientset"
	"github.com/tektoncd/triggers/pkg/client/dynamic/clientset/tekton"
	faketriggersclient "github.com/tektoncd/triggers/pkg/client/injection/client/fake"
	fakeclustertriggerbindinginformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1alpha1/clustertriggerbinding/fake"
	fakeeventlistenerinformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1alpha1/eventlistener/fake"
	fakeinterceptorinformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1alpha1/interceptor/fake"
	faketriggerinformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1alpha1/trigger/fake"
	faketriggerbindinginformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1alpha1/triggerbinding/fake"
	faketriggertemplateinformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1alpha1/triggertemplate/fake"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"
	fakekubeclient "knative.dev/pkg/client/injection/kube/client/fake"
	fakedeployinformer "knative.dev/pkg/client/injection/kube/informers/apps/v1/deployment/fake"
	fakeconfigmapinformer "knative.dev/pkg/client/injection/kube/informers/core/v1/configmap/fake"
	fakepodinformer "knative.dev/pkg/client/injection/kube/informers/core/v1/pod/fake"
	fakesecretinformer "knative.dev/pkg/client/injection/kube/informers/core/v1/secret/fake"
	fakeserviceinformer "knative.dev/pkg/client/injection/kube/informers/core/v1/service/fake"
	fakeserviceaccountinformer "knative.dev/pkg/client/injection/kube/informers/core/v1/serviceaccount/fake"
	"knative.dev/pkg/controller"
)

// Resources represents the desired state of the system (i.e. existing resources)
// to seed controllers with.
type Resources struct {
	Namespaces             []*corev1.Namespace
	ClusterTriggerBindings []*v1alpha1.ClusterTriggerBinding
	EventListeners         []*v1alpha1.EventListener
	Interceptors           []*v1alpha1.Interceptor
	TriggerBindings        []*v1alpha1.TriggerBinding
	TriggerTemplates       []*v1alpha1.TriggerTemplate
	Triggers               []*v1alpha1.Trigger
	Deployments            []*appsv1.Deployment
	Services               []*corev1.Service
	ConfigMaps             []*corev1.ConfigMap
	Secrets                []*corev1.Secret
	ServiceAccounts        []*corev1.ServiceAccount
	Pods                   []*corev1.Pod
}

// Clients holds references to clients which are useful for reconciler tests.
type Clients struct {
	Kube     *fakekubeclientset.Clientset
	Triggers *faketriggersclientset.Clientset
	Pipeline *fakepipelineclientset.Clientset
	Resource *fakeresourceclientset.Clientset
	Dynamic  *dynamicclientset.Clientset
}

// Assets holds references to the controller and clients.
type Assets struct {
	Controller *controller.Impl
	Clients    Clients
}

// SeedResources returns Clients populated with the given Resources
// nolint: golint
func SeedResources(t *testing.T, ctx context.Context, r Resources) Clients {
	t.Helper()
	dynamicClient := fakedynamic.NewSimpleDynamicClient(runtime.NewScheme())
	c := Clients{
		Kube:     fakekubeclient.Get(ctx),
		Triggers: faketriggersclient.Get(ctx),
		Pipeline: fakepipelineclient.Get(ctx),
		Resource: fakeresourceclient.Get(ctx),
		Dynamic:  dynamicclientset.New(tekton.WithClient(dynamicClient)),
	}

	// Teach Kube clients about the Tekton resources (needed by discovery client when creating resources)
	AddTektonResources(c.Kube)

	// Setup fake informer for reconciler tests
	ctbInformer := fakeclustertriggerbindinginformer.Get(ctx)
	elInformer := fakeeventlistenerinformer.Get(ctx)
	icInformer := fakeinterceptorinformer.Get(ctx)
	ttInformer := faketriggertemplateinformer.Get(ctx)
	tbInformer := faketriggerbindinginformer.Get(ctx)
	trInformer := faketriggerinformer.Get(ctx)
	deployInformer := fakedeployinformer.Get(ctx)
	serviceInformer := fakeserviceinformer.Get(ctx)
	configMapInformer := fakeconfigmapinformer.Get(ctx)
	secretInformer := fakesecretinformer.Get(ctx)
	saInformer := fakeserviceaccountinformer.Get(ctx)
	podInformer := fakepodinformer.Get(ctx)

	// Create Namespaces
	for _, ns := range r.Namespaces {
		if _, err := c.Kube.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	// Create test Resources
	for _, ctb := range r.ClusterTriggerBindings {
		if err := ctbInformer.Informer().GetIndexer().Add(ctb); err != nil {
			t.Fatal(err)
		}
		if _, err := c.Triggers.TriggersV1alpha1().ClusterTriggerBindings().Create(context.Background(), ctb, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	for _, el := range r.EventListeners {
		if err := elInformer.Informer().GetIndexer().Add(el); err != nil {
			t.Fatal(err)
		}
		if _, err := c.Triggers.TriggersV1alpha1().EventListeners(el.Namespace).Create(context.Background(), el, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	for _, ic := range r.Interceptors {
		if err := icInformer.Informer().GetIndexer().Add(ic); err != nil {
			t.Fatal(err)
		}
		if _, err := c.Triggers.TriggersV1alpha1().Interceptors().Create(context.Background(), ic, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	for _, tb := range r.TriggerBindings {
		if err := tbInformer.Informer().GetIndexer().Add(tb); err != nil {
			t.Fatal(err)
		}
		if _, err := c.Triggers.TriggersV1alpha1().TriggerBindings(tb.Namespace).Create(context.Background(), tb, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	for _, tt := range r.TriggerTemplates {
		if err := ttInformer.Informer().GetIndexer().Add(tt); err != nil {
			t.Fatal(err)
		}
		if _, err := c.Triggers.TriggersV1alpha1().TriggerTemplates(tt.Namespace).Create(context.Background(), tt, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	for _, tr := range r.Triggers {
		if err := trInformer.Informer().GetIndexer().Add(tr); err != nil {
			t.Fatal(err)
		}
		if _, err := c.Triggers.TriggersV1alpha1().Triggers(tr.Namespace).Create(context.Background(), tr, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	for _, d := range r.Deployments {
		if err := deployInformer.Informer().GetIndexer().Add(d); err != nil {
			t.Fatal(err)
		}
		if _, err := c.Kube.AppsV1().Deployments(d.Namespace).Create(context.Background(), d, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	for _, svc := range r.Services {
		if err := serviceInformer.Informer().GetIndexer().Add(svc); err != nil {
			t.Fatal(err)
		}

		if _, err := c.Kube.CoreV1().Services(svc.Namespace).Create(context.Background(), svc, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}

	for _, cfg := range r.ConfigMaps {
		if err := configMapInformer.Informer().GetIndexer().Add(cfg); err != nil {
			t.Fatal(err)
		}
		if _, err := c.Kube.CoreV1().ConfigMaps(cfg.Namespace).Create(context.Background(), cfg, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	for _, s := range r.Secrets {
		if err := secretInformer.Informer().GetIndexer().Add(s); err != nil {
			t.Fatal(err)
		}
		if _, err := c.Kube.CoreV1().Secrets(s.Namespace).Create(context.Background(), s, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	for _, sa := range r.ServiceAccounts {
		if err := saInformer.Informer().GetIndexer().Add(sa); err != nil {
			t.Fatal(err)
		}
		if _, err := c.Kube.CoreV1().ServiceAccounts(sa.Namespace).Create(context.Background(), sa, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	for _, pod := range r.Pods {
		if err := podInformer.Informer().GetIndexer().Add(pod); err != nil {
			t.Fatal(err)
		}
		if _, err := c.Kube.CoreV1().Pods(pod.Namespace).Create(context.Background(), pod, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}

	c.Kube.ClearActions()
	c.Triggers.ClearActions()
	c.Pipeline.ClearActions()
	return c
}

// GetResourcesFromClients returns the Resources in the Clients provided
// Precondition: all Namespaces used in Resources must be listed in Resources.Namespaces
// nolint: golint
func GetResourcesFromClients(c Clients) (*Resources, error) {
	testResources := &Resources{}
	// Add ClusterTriggerBindings
	ctbList, err := c.Triggers.TriggersV1alpha1().ClusterTriggerBindings().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, ctb := range ctbList.Items {
		testResources.ClusterTriggerBindings = append(testResources.ClusterTriggerBindings, ctb.DeepCopy())
	}
	nsList, err := c.Kube.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, ns := range nsList.Items {
		// Add Namespace
		testResources.Namespaces = append(testResources.Namespaces, ns.DeepCopy())
		// Add EventListeners
		elList, err := c.Triggers.TriggersV1alpha1().EventListeners(ns.Name).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, el := range elList.Items {
			testResources.EventListeners = append(testResources.EventListeners, el.DeepCopy())
		}
		// Add TriggerBindings
		tbList, err := c.Triggers.TriggersV1alpha1().TriggerBindings(ns.Name).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, tb := range tbList.Items {
			testResources.TriggerBindings = append(testResources.TriggerBindings, tb.DeepCopy())
		}
		// Add TriggerTemplates
		ttList, err := c.Triggers.TriggersV1alpha1().TriggerTemplates(ns.Name).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, tt := range ttList.Items {
			testResources.TriggerTemplates = append(testResources.TriggerTemplates, tt.DeepCopy())
		}
		// Add Deployments
		dList, err := c.Kube.AppsV1().Deployments(ns.Name).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, d := range dList.Items {
			testResources.Deployments = append(testResources.Deployments, d.DeepCopy())
		}
		// Add Services
		svcList, err := c.Kube.CoreV1().Services(ns.Name).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, svc := range svcList.Items {
			testResources.Services = append(testResources.Services, svc.DeepCopy())
		}
		// Add ConfigMaps
		cfgList, err := c.Kube.CoreV1().ConfigMaps(ns.Name).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, cfg := range cfgList.Items {
			testResources.ConfigMaps = append(testResources.ConfigMaps, cfg.DeepCopy())
		}
		// Add Secrets
		secretsList, err := c.Kube.CoreV1().Secrets(ns.Name).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, s := range secretsList.Items {
			testResources.Secrets = append(testResources.Secrets, s.DeepCopy())
		}
		// Get Triggers
		trList, err := c.Triggers.TriggersV1alpha1().Triggers(ns.Name).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, tr := range trList.Items {
			testResources.Triggers = append(testResources.Triggers, tr.DeepCopy())
		}
		// Get Pods
		podList, err := c.Kube.CoreV1().Pods(ns.Name).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, pod := range podList.Items {
			testResources.Pods = append(testResources.Pods, pod.DeepCopy())
		}

	}
	return testResources, nil
}
