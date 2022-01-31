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
package main

import (
	"flag"
	"os"

	"github.com/tektoncd/triggers/pkg/reconciler/clusterinterceptor"
	elresources "github.com/tektoncd/triggers/pkg/reconciler/eventlistener/resources"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	filteredinformerfactory "knative.dev/pkg/client/injection/kube/informers/factory/filtered"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/injection/sharedmain"
	"knative.dev/pkg/signals"

	"github.com/tektoncd/triggers/pkg/reconciler/eventlistener"
)

const (
	// ControllerLogKey is the name of the logger for the controller cmd
	ControllerLogKey = "controller"
)

var (
	image                 = flag.String("el-image", elresources.DefaultImage, "The container image for the EventListener Pod.")
	port                  = flag.Int("el-port", elresources.DefaultPort, "The container port for the EventListener to listen on.")
	setSecurityContext    = flag.Bool("el-security-context", elresources.DefaultSetSecurityContext, "Add a security context to the event listener deployment.")
	setEventListenerEvent = flag.String("el-events", elresources.DefaultEventListenerEvent, "Enable events for event listener deployment.")
	readTimeOut           = flag.Int64("el-readtimeout", elresources.DefaultReadTimeout, "The read timeout for EventListener Server.")
	writeTimeOut          = flag.Int64("el-writetimeout", elresources.DefaultWriteTimeout, "The write timeout for EventListener Server.")
	idleTimeOut           = flag.Int64("el-idletimeout", elresources.DefaultIdleTimeout, "The idle timeout for EventListener Server.")
	timeOutHandler        = flag.Int64("el-timeouthandler", elresources.DefaultTimeOutHandler, "The timeout for Timeout Handler of EventListener Server.")
	periodSeconds         = flag.Int("period-seconds", elresources.DefaultPeriodSeconds, "The Period Seconds for the EventListener Liveness and Readiness Probes.")
	failureThreshold      = flag.Int("failure-threshold", elresources.DefaultFailureThreshold, "The Failure Threshold for the EventListener Liveness and Readiness Probes.")

	staticResourceLabels = elresources.DefaultStaticResourceLabels
	systemNamespace      = os.Getenv("SYSTEM_NAMESPACE")
)

func main() {
	cfg := injection.ParseAndGetRESTConfigOrDie()

	c := elresources.Config{
		Image:                 image,
		Port:                  port,
		SetSecurityContext:    setSecurityContext,
		SetEventListenerEvent: setEventListenerEvent,
		ReadTimeOut:           readTimeOut,
		WriteTimeOut:          writeTimeOut,
		IdleTimeOut:           idleTimeOut,
		TimeOutHandler:        timeOutHandler,
		PeriodSeconds:         periodSeconds,
		FailureThreshold:      failureThreshold,

		StaticResourceLabels: staticResourceLabels,
		SystemNamespace:      systemNamespace,
	}

	ctx := injection.WithNamespaceScope(signals.NewContext(), corev1.NamespaceAll)
	ctx = filteredinformerfactory.WithSelectors(ctx, labels.FormatLabels(elresources.DefaultStaticResourceLabels))
	sharedmain.MainWithConfig(
		ctx,
		ControllerLogKey,
		cfg,
		eventlistener.NewController(c),
		clusterinterceptor.NewController(),
	)
}
