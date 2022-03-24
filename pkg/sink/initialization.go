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
	"context"
	"flag"
	"time"

	triggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	"github.com/tektoncd/triggers/pkg/sink/cloudevent"
	"golang.org/x/xerrors"
	discoveryclient "k8s.io/client-go/discovery"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	restclient "k8s.io/client-go/rest"
)

const (
	// Flag definitions
	name        = "el-name"
	elNamespace = "el-namespace"
	port        = "port"
	isMultiNS   = "is-multi-ns"
)

var (
	nameFlag = flag.String("el-name", "",
		"The name of the EventListener resource for this sink.")
	namespaceFlag = flag.String("el-namespace", "",
		"The namespace of the EventListener resource for this sink.")
	portFlag = flag.String("port", "",
		"The port for the EventListener sink to listen on.")
	elReadTimeOut = flag.Int64("readtimeout", 5,
		"The read timeout for EventListener Server.")
	elWriteTimeOut = flag.Int64("writetimeout", 40,
		"The write timeout for EventListener Server.")
	elIdleTimeOut = flag.Int64("idletimeout", 30,
		"The idle timeout for EventListener Server.")
	elTimeOutHandler = flag.Int64("timeouthandler", 5,
		"The timeout for Timeout Handler of EventListener Server.")
	elHTTPClientReadTimeOut = flag.Int64("httpclient-readtimeout", 30,
		"The HTTP Client read timeout for EventListener Server.")
	elHTTPClientKeepAlive = flag.Int64("httpclient-keep-alive", 30,
		"The HTTP Client read timeout for EventListener Server.")
	elHTTPClientTLSHandshakeTimeout = flag.Int64("httpclient-tlshandshaketimeout", 10,
		"The HTTP Client read timeout for EventListener Server.")
	elHTTPClientResponseHeaderTimeout = flag.Int64("httpclient-responseheadertimeout", 10,
		"The HTTP Client read timeout for EventListener Server.")
	elHTTPClientExpectContinueTimeout = flag.Int64("httpclient-expectcontinuetimeout", 1,
		"The HTTP Client read timeout for EventListener Server.")
	isMultiNSFlag = flag.Bool("is-multi-ns", false,
		"Whether EventListener serve Multiple NS.")
	tlsCertFlag = flag.String("tls-cert", "",
		"The filename for the TLS certificate.")
	tlsKeyFlag = flag.String("tls-key", "",
		"The filename for the TLS key.")
	payloadValidation = flag.Bool("payload-validation", true,
		"Whether to disable payload validation or not.")
	cloudEventURI = flag.String("cloudevent-uri", "", "uri for cloudevent")
)

// Args define the arguments for Sink.
type Args struct {
	// ElName is the EventListener name.
	ElName string
	// ElNamespace is the EventListener namespace.
	ElNamespace string
	// Port is the port the Sink should listen on.
	Port string
	// ELReadTimeOut defines the read timeout for EventListener Server
	ELReadTimeOut time.Duration
	// ELWriteTimeOut defines the write timeout for EventListener Server
	ELWriteTimeOut time.Duration
	// ELIdleTimeOut defines the read timeout for EventListener Server
	ELIdleTimeOut time.Duration
	// ELTimeOutHandler defines the timeout for Timeout Handler of EventListener Server
	ELTimeOutHandler time.Duration
	// ElHTTPClientReadTimeOut defines the Read timeout for HTTP Client
	ElHTTPClientReadTimeOut time.Duration
	// ElHTTPClientKeepAlive defines the Keep Alive for HTTP Client
	ElHTTPClientKeepAlive time.Duration
	// ElTLSHandshakeTimeout defines the Handshake timeout for HTTP Client
	ElHTTPClientTLSHandshakeTimeout time.Duration
	// ElResponseHeaderTimeout defines the Response Header timeout for HTTP Client
	ElHTTPClientResponseHeaderTimeout time.Duration
	// ElExpectContinueTimeout defines the Expect timeout for HTTP Client
	ElHTTPClientExpectContinueTimeout time.Duration
	// IsMultiNS determines whether el functions as namespaced or clustered
	IsMultiNS bool
	// Key defines the filename for tls Key.
	Key string
	// Cert defines the filename for tls Cert.
	Cert string
	// PayloadValidation defines whether to validate payload or not
	PayloadValidation bool
	// CloudEventURI refers to the location where cloudevent data need to be send
	CloudEventURI string
}

// Clients define the set of client dependencies Sink requires.
type Clients struct {
	DiscoveryClient discoveryclient.DiscoveryInterface
	RESTClient      restclient.Interface
	TriggersClient  triggersclientset.Interface
	K8sClient       *kubeclientset.Clientset
	CEClient        cloudevent.CEClient
}

// GetArgs returns the flagged Args
func GetArgs() (Args, error) {
	flag.Parse()
	if *nameFlag == "" {
		return Args{}, xerrors.Errorf("-%s arg not found", name)
	}
	if *namespaceFlag == "" {
		return Args{}, xerrors.Errorf("-%s arg not found", elNamespace)
	}
	if *portFlag == "" {
		return Args{}, xerrors.Errorf("-%s arg not found", port)
	}

	return Args{
		ElName:                            *nameFlag,
		ElNamespace:                       *namespaceFlag,
		Port:                              *portFlag,
		IsMultiNS:                         *isMultiNSFlag,
		PayloadValidation:                 *payloadValidation,
		ELReadTimeOut:                     time.Duration(*elReadTimeOut),
		ELWriteTimeOut:                    time.Duration(*elWriteTimeOut),
		ELIdleTimeOut:                     time.Duration(*elIdleTimeOut),
		ELTimeOutHandler:                  time.Duration(*elTimeOutHandler),
		ElHTTPClientReadTimeOut:           time.Duration(*elHTTPClientReadTimeOut),
		ElHTTPClientKeepAlive:             time.Duration(*elHTTPClientKeepAlive),
		ElHTTPClientTLSHandshakeTimeout:   time.Duration(*elHTTPClientTLSHandshakeTimeout),
		ElHTTPClientResponseHeaderTimeout: time.Duration(*elHTTPClientResponseHeaderTimeout),
		ElHTTPClientExpectContinueTimeout: time.Duration(*elHTTPClientExpectContinueTimeout),
		Cert:                              *tlsCertFlag,
		Key:                               *tlsKeyFlag,
		CloudEventURI:                     *cloudEventURI,
	}, nil
}

// ConfigureClients returns the kubernetes and triggers clientsets
func ConfigureClients(ctx context.Context, clusterConfig *rest.Config) (Clients, error) {
	kubeClient, err := kubeclientset.NewForConfig(clusterConfig)
	if err != nil {
		return Clients{}, xerrors.Errorf("Failed to create KubeClient: %s", err)
	}
	triggersClient, err := triggersclientset.NewForConfig(clusterConfig)
	if err != nil {
		return Clients{}, xerrors.Errorf("Failed to create TriggersClient: %s", err)
	}
	ceClient := cloudevent.Get(ctx)
	return Clients{
		DiscoveryClient: kubeClient.Discovery(),
		RESTClient:      kubeClient.RESTClient(),
		TriggersClient:  triggersClient,
		K8sClient:       kubeClient,
		CEClient:        ceClient,
	}, nil
}
