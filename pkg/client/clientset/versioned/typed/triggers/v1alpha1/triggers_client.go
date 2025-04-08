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

package v1alpha1

import (
	http "net/http"

	triggersv1alpha1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	scheme "github.com/tektoncd/triggers/pkg/client/clientset/versioned/scheme"
	rest "k8s.io/client-go/rest"
)

type TriggersV1alpha1Interface interface {
	RESTClient() rest.Interface
	ClusterInterceptorsGetter
	ClusterTriggerBindingsGetter
	EventListenersGetter
	InterceptorsGetter
	TriggersGetter
	TriggerBindingsGetter
	TriggerTemplatesGetter
}

// TriggersV1alpha1Client is used to interact with features provided by the triggers.tekton.dev group.
type TriggersV1alpha1Client struct {
	restClient rest.Interface
}

func (c *TriggersV1alpha1Client) ClusterInterceptors() ClusterInterceptorInterface {
	return newClusterInterceptors(c)
}

func (c *TriggersV1alpha1Client) ClusterTriggerBindings() ClusterTriggerBindingInterface {
	return newClusterTriggerBindings(c)
}

func (c *TriggersV1alpha1Client) EventListeners(namespace string) EventListenerInterface {
	return newEventListeners(c, namespace)
}

func (c *TriggersV1alpha1Client) Interceptors(namespace string) InterceptorInterface {
	return newInterceptors(c, namespace)
}

func (c *TriggersV1alpha1Client) Triggers(namespace string) TriggerInterface {
	return newTriggers(c, namespace)
}

func (c *TriggersV1alpha1Client) TriggerBindings(namespace string) TriggerBindingInterface {
	return newTriggerBindings(c, namespace)
}

func (c *TriggersV1alpha1Client) TriggerTemplates(namespace string) TriggerTemplateInterface {
	return newTriggerTemplates(c, namespace)
}

// NewForConfig creates a new TriggersV1alpha1Client for the given config.
// NewForConfig is equivalent to NewForConfigAndClient(c, httpClient),
// where httpClient was generated with rest.HTTPClientFor(c).
func NewForConfig(c *rest.Config) (*TriggersV1alpha1Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	httpClient, err := rest.HTTPClientFor(&config)
	if err != nil {
		return nil, err
	}
	return NewForConfigAndClient(&config, httpClient)
}

// NewForConfigAndClient creates a new TriggersV1alpha1Client for the given config and http client.
// Note the http client provided takes precedence over the configured transport values.
func NewForConfigAndClient(c *rest.Config, h *http.Client) (*TriggersV1alpha1Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientForConfigAndClient(&config, h)
	if err != nil {
		return nil, err
	}
	return &TriggersV1alpha1Client{client}, nil
}

// NewForConfigOrDie creates a new TriggersV1alpha1Client for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *TriggersV1alpha1Client {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new TriggersV1alpha1Client for the given RESTClient.
func New(c rest.Interface) *TriggersV1alpha1Client {
	return &TriggersV1alpha1Client{c}
}

func setConfigDefaults(config *rest.Config) error {
	gv := triggersv1alpha1.SchemeGroupVersion
	config.GroupVersion = &gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = rest.CodecFactoryForGeneratedClient(scheme.Scheme, scheme.Codecs).WithoutConversion()

	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	return nil
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *TriggersV1alpha1Client) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
