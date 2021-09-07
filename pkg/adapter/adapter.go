/*
Copyright 2021 The Tekton Authors

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

package adapter

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	clusterinterceptorsinformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1alpha1/clusterinterceptor"
	clustertriggerbindingsinformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1beta1/clustertriggerbinding"
	eventlistenerinformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1beta1/eventlistener"
	triggersinformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1beta1/trigger"
	triggerbindingsinformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1beta1/triggerbinding"
	triggertemplatesinformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1beta1/triggertemplate"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/tektoncd/triggers/pkg/sink"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	"knative.dev/eventing/pkg/adapter/v2"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/injection/clients/dynamicclient"
	"knative.dev/pkg/logging"
)

type envConfig struct {
	adapter.EnvConfig
}

func NewEnvConfig() adapter.EnvConfigAccessor {
	return &envConfig{}
}

// sinker implements the adapter for an event listener.
type sinker struct {
	Logger    *zap.SugaredLogger
	Namespace string

	Args     sink.Args
	Clients  sink.Clients
	Recorder *sink.Recorder

	injCtx context.Context
}

var _ adapter.Adapter = (*sinker)(nil)

func (s *sinker) createRecorder(ctx context.Context, agentName string) record.EventRecorder {
	logger := logging.FromContext(ctx)

	recorder := controller.GetEventRecorder(ctx)
	if recorder == nil {
		// Create event broadcaster
		logger.Debug("Creating event broadcaster")
		eventBroadcaster := record.NewBroadcaster()
		watches := []watch.Interface{
			eventBroadcaster.StartLogging(logger.Named("event-broadcaster").Infof),
			eventBroadcaster.StartRecordingToSink(
				&v1.EventSinkImpl{Interface: s.Clients.K8sClient.CoreV1().Events("")}),
		}
		recorder = eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: agentName})
		go func() {
			<-ctx.Done()
			for _, w := range watches {
				w.Stop()
			}
		}()
	}

	return recorder
}

func (s *sinker) Start(ctx context.Context) error {
	// Create EventListener Sink
	r := sink.Sink{
		KubeClientSet:          kubeclient.Get(ctx),
		DiscoveryClient:        s.Clients.DiscoveryClient,
		DynamicClient:          dynamicclient.Get(ctx),
		TriggersClient:         s.Clients.TriggersClient,
		HTTPClient:             http.DefaultClient,
		EventListenerName:      s.Args.ElName,
		EventListenerNamespace: s.Args.ElNamespace,
		PayloadValidation:      s.Args.PayloadValidation,
		Logger:                 s.Logger,
		Recorder:               s.Recorder,
		Auth:                   sink.DefaultAuthOverride{},
		WGProcessTriggers:      &sync.WaitGroup{},
		EventRecorder:          s.createRecorder(s.injCtx, "EventListener"),

		// Register all the listers we'll need
		EventListenerLister:         eventlistenerinformer.Get(s.injCtx).Lister(),
		TriggerLister:               triggersinformer.Get(s.injCtx).Lister(),
		TriggerBindingLister:        triggerbindingsinformer.Get(s.injCtx).Lister(),
		ClusterTriggerBindingLister: clustertriggerbindingsinformer.Get(s.injCtx).Lister(),
		TriggerTemplateLister:       triggertemplatesinformer.Get(s.injCtx).Lister(),
		ClusterInterceptorLister:    clusterinterceptorsinformer.Get(s.injCtx).Lister(),
	}

	mux := http.NewServeMux()
	eventHandler := http.HandlerFunc(r.HandleEvent)
	metricsRecorder := &sink.MetricsHandler{Handler: r.IsValidPayload(eventHandler)}

	mux.HandleFunc("/", http.HandlerFunc(metricsRecorder.Intercept(r.NewMetricsRecorderInterceptor())))

	// For handling Liveness Probe
	// TODO(dibyom): Livness, metrics etc. should be on a separate port
	mux.HandleFunc("/live", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, "ok")
	})

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", s.Args.Port),
		ReadTimeout:  s.Args.ELReadTimeOut * time.Second,
		WriteTimeout: s.Args.ELWriteTimeOut * time.Second,
		IdleTimeout:  s.Args.ELIdleTimeOut * time.Second,
		Handler: http.TimeoutHandler(mux,
			s.Args.ELTimeOutHandler*time.Second, "EventListener Timeout!\n"),
	}

	if s.Args.Cert == "" && s.Args.Key == "" {
		if err := srv.ListenAndServe(); err != nil {
			return err
		}
	} else {
		if err := srv.ListenAndServeTLS(s.Args.Cert, s.Args.Key); err != nil {
			return err
		}
	}
	return nil
}

func New(sinkArgs sink.Args, sinkClients sink.Clients, recorder *sink.Recorder) adapter.AdapterConstructor {
	return func(ctx context.Context, processed adapter.EnvConfigAccessor, ceClient cloudevents.Client) adapter.Adapter {
		env := processed.(*envConfig)
		logger := logging.FromContext(ctx)

		return &sinker{
			Logger:    logger,
			Namespace: env.Namespace,
			Args:      sinkArgs,
			Clients:   sinkClients,
			Recorder:  recorder,
			injCtx:    ctx,
		}
	}
}
