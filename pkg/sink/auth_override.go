/*
Copyright 2020 The Tekton Authors

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
	"fmt"

	dynamicClientset "github.com/tektoncd/triggers/pkg/client/dynamic/clientset"
	"github.com/tektoncd/triggers/pkg/client/dynamic/clientset/tekton"
	"go.uber.org/zap"
	discoveryclient "k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

//AuthOverride is an interface that constructs a discovery client for the ServerResourceInterface
//and a dynamic client for the Tekton Resources, using the token provide as the bearer token in the
//REST config used to build those client.  The other non-credential related parameters for the
//REST client used are copied from the in cluster config of the event sink.
type AuthOverride interface {
	OverrideAuthentication(sa string,
		namespace string,
		log *zap.SugaredLogger,
		defaultDiscoveryClient discoveryclient.ServerResourcesInterface,
		defaultDynamicClient dynamic.Interface) (discoveryClient discoveryclient.ServerResourcesInterface,
		dynamicClient dynamic.Interface,
		err error)
}

type DefaultAuthOverride struct {
}

func (r DefaultAuthOverride) OverrideAuthentication(sa string,
	namespace string,
	log *zap.SugaredLogger,
	defaultDiscoverClient discoveryclient.ServerResourcesInterface,
	defaultDynamicClient dynamic.Interface) (discoveryClient discoveryclient.ServerResourcesInterface,
	dynamicClient dynamic.Interface,
	err error) {
	dynamicClient = defaultDynamicClient
	discoveryClient = defaultDiscoverClient
	clusterConfig, err := rest.InClusterConfig()
	if err != nil {
		log.Errorf("overrideAuthentication: problem getting in cluster config: %#v\n", err)
		return
	}
	clusterConfig.Impersonate = rest.ImpersonationConfig{
		UserName: fmt.Sprintf("system:serviceaccount:%s:%s", namespace, sa),
	}
	dc, err := dynamic.NewForConfig(clusterConfig)
	if err != nil {
		log.Errorf("overrideAuthentication: problem getting dynamic client set: %#v\n", err)
		return
	}
	kubeClient, err := kubernetes.NewForConfig(clusterConfig)
	if err != nil {
		log.Errorf("overrideAuthentication: problem getting kube client: %#v\n", err)
		return
	}
	dynamicClient = dynamicClientset.New(tekton.WithClient(dc))
	discoveryClient = kubeClient.Discovery()

	return
}
