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
	"log"
	"time"

	v1alpha1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	triggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

type Watcher struct {
	K8s                    dynamic.Interface
	TriggersClient         triggersclientset.Interface
	EventListenerName      string
	EventListenerNamespace string
	Recorder               record.EventRecorder
}

var (
	resyncPeriod = time.Duration(10 * time.Hour)
)

func (w Watcher) Start(stopCh <-chan struct{}) error {
	triggerBindings := []string{}
	triggerTemplates := []string{}

	el, err := w.TriggersClient.TektonV1alpha1().EventListeners(w.EventListenerNamespace).Get(w.EventListenerName, metav1.GetOptions{})
	if err != nil {
		log.Printf("Error getting EventListener %s in Namespace %s: %s", w.EventListenerName, w.EventListenerNamespace, err)
		return err
	}

	for _, trigger := range el.Spec.Triggers {
		if trigger.Binding != nil {
			triggerBindings = append(triggerBindings, trigger.Binding.Name)
		}
		triggerTemplates = append(triggerTemplates, trigger.Template.Name)
	}

	if len(triggerBindings) != 0 {
		TriggerBindings := v1alpha1.SchemeGroupVersion.WithResource("triggerbindings")
		w.start(TriggerBindings, triggerBindings, el, stopCh)
	}
	TriggerTemplates := v1alpha1.SchemeGroupVersion.WithResource("triggertemplates")
	w.start(TriggerTemplates, triggerTemplates, el, stopCh)

	return nil
}

func (w Watcher) start(gvr schema.GroupVersionResource, resources []string, source *v1alpha1.EventListener, stop <-chan struct{}) {
	d := &resource{
		resources: resources,
		recorder:  w.Recorder,
		source:    source,
	}

	lw := &cache.ListWatch{
		ListFunc:  asUnstructuredLister(w.K8s.Resource(gvr).Namespace(w.EventListenerNamespace).List),
		WatchFunc: asUnstructuredWatcher(w.K8s.Resource(gvr).Namespace(w.EventListenerNamespace).Watch),
	}

	reflector := cache.NewReflector(lw, &unstructured.Unstructured{}, d, resyncPeriod)
	go reflector.Run(stop)

}

type unstructuredLister func(metav1.ListOptions) (*unstructured.UnstructuredList, error)

func asUnstructuredLister(ulist unstructuredLister) cache.ListFunc {
	return func(opts metav1.ListOptions) (runtime.Object, error) {
		ul, err := ulist(opts)
		if err != nil {
			return nil, err
		}
		return ul, nil
	}
}

func asUnstructuredWatcher(wf cache.WatchFunc) cache.WatchFunc {
	return func(lo metav1.ListOptions) (watch.Interface, error) {
		return wf(lo)
	}
}
