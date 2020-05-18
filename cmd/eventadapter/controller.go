package main

import (
	"context"
	"flag"
	"net/http"
	"sync"

	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"

	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	eventlistenerinformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1alpha1/eventlistener"
)

var (
	configName = flag.String("configName", "event-config-adapter", "event adapter config name. ")
	configNs   = flag.String("configNamespace", "default", "event adapter config namespace. ")
)

var eventListenerMapInfo sync.Map

// NewController creates a new instance of an EventListener controller.
func NewController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {

	go RunAdapter(ctx)

	logger := logging.FromContext(ctx)
	kubeclientset := kubeclient.Get(ctx)
	eventListenerInformer := eventlistenerinformer.Get(ctx)

	impl := controller.NewImpl(&EmptyReconcile{}, logger, ControllerLogKey)

	cm, err := kubeclientset.CoreV1().ConfigMaps(*configNs).Get(*configName, metav1.GetOptions{})
	if err != nil {
		logger.Fatalf("get configmap info %s/%s failed %v. ", *configName, *configNs, err)
	}

	err = yaml.Unmarshal([]byte(cm.Data[ConfigKey]), &CurrentConfig)
	if err != nil {
		logger.Fatalf("load configmap info %s/%s failed %v. ", *configName, *configNs, err)
	}
	logger.Info("load config ok: ", CurrentConfig)

	http.HandleFunc("/-/reload", func(writer http.ResponseWriter, request *http.Request) {
		cm, err := kubeclientset.CoreV1().ConfigMaps(*configNs).Get(*configName, metav1.GetOptions{})
		if err != nil {
			logger.Fatalf("get configmap info %s/%s failed %v. ", *configName, *configNs, err)
		}

		err = yaml.Unmarshal([]byte(cm.Data[ConfigKey]), &CurrentConfig)
		if err != nil {
			logger.Fatalf("load configmap info %s/%s failed %v. ", *configName, *configNs, err)
		}
		logger.Info("load config ok: ", CurrentConfig)
		return
	})

	go func() {
		err = http.ListenAndServe(":9910", nil)
		if err != nil {
			logger.Fatalf("start config port 9091 failed %v. ", err)
		}
	}()

	logger.Info("Setting up event handlers")
	eventListenerInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: SyncEventListener,
		UpdateFunc: func(oldObj, newObj interface{}) {
			SyncEventListener(newObj)
		},
		DeleteFunc: SyncEventListener,
	})

	return impl
}

func SyncEventListener(obj interface{}) {
	eventListener, ok := obj.(*v1alpha1.EventListener)
	if !ok {
		return
	}

	key := GetListenerKey(eventListener.Name, eventListener.Namespace)

	if eventListener.DeletionTimestamp != nil {
		eventListenerMapInfo.Delete(key)
		return
	}
	eventListenerMapInfo.Store(key, eventListener)
}

type EmptyReconcile struct {
}

func (t EmptyReconcile) Reconcile(ctx context.Context, key string) error {
	return nil
}
