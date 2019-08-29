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
	"flag"
	"golang.org/x/xerrors"

	triggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	discoveryclient "k8s.io/client-go/discovery"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	restclient "k8s.io/client-go/rest"
)

const (
	EL_NAME      = "el-name"
	EL_NAMESPACE = "el-namespace"
	PORT         = "port"
)

var (
	elName = flag.String(EL_NAME, "",
		"The name of the EventListener resource for this sink.")
	elNamespace = flag.String(EL_NAMESPACE, "",
		"The namespace of the EventListener resource for this sink.")
	port = flag.String(PORT, "",
		"The port for the EventListener sink to listen on.")
)

type SinkArgs struct {
	ElName      string
	ElNamespace string
	Port        string
}

type SinkClients struct {
	DiscoveryClient discoveryclient.DiscoveryInterface
	RESTClient      restclient.Interface
	TriggersClient  triggersclientset.Interface
}

// GetArgs returns the flagged SinkArgs
func GetArgs() (SinkArgs, error) {
	flag.Parse()
	if *elName == "" {
		return SinkArgs{}, xerrors.Errorf("-%s arg not found", EL_NAME)
	}
	if *elNamespace == "" {
		return SinkArgs{}, xerrors.Errorf("-%s arg not found", EL_NAMESPACE)
	}
	if *port == "" {
		return SinkArgs{}, xerrors.Errorf("-%s arg not found", PORT)
	}
	return SinkArgs{
		ElName:      *elName,
		ElNamespace: *elNamespace,
		Port:        *port,
	}, nil
}

// ConfigureClients returns the kubernetes and triggers clientsets
func ConfigureClients() (SinkClients, error) {
	clusterConfig, err := rest.InClusterConfig()
	if err != nil {
		return SinkClients{}, xerrors.Errorf("Failed to get in cluster config: %s", err)
	}
	kubeClient, err := kubeclientset.NewForConfig(clusterConfig)
	if err != nil {
		return SinkClients{}, xerrors.Errorf("Failed to create KubeClient: %s", err)
	}
	triggersClient, err := triggersclientset.NewForConfig(clusterConfig)
	if err != nil {
		return SinkClients{}, xerrors.Errorf("Failed to create TriggersClient: %s", err)
	}
	return SinkClients{
		DiscoveryClient: kubeClient.Discovery(),
		RESTClient:      kubeClient.RESTClient(),
		TriggersClient:  triggersClient,
	}, nil
}
