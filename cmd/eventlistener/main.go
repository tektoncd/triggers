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
	"fmt"
	"log"
	"net/http"
	"os"

	triggersClientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	"github.com/tektoncd/triggers/pkg/reconciler/v1alpha1/eventlistener"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

func handler(w http.ResponseWriter, r *http.Request) {
	log.Printf("EventListener pod received a request: %+v", r)
	eventListener, err := triggersClient.TektonV1alpha1().EventListeners(listenerNamespace).Get(listenerName, v1.GetOptions{})
	if err != nil {
		log.Fatal("Failed to find EventListener", err)
	}

	for _, trigger := range eventListener.Spec.Triggers {

		tb, err := triggersClient.TektonV1alpha1().TriggerBindings(listenerNamespace).Get(trigger.TriggerBinding.Name, v1.GetOptions{})
		if err != nil {
			log.Printf("Error getting TriggerBinding %s", trigger.TriggerBinding.Name)
		}
		log.Print(tb)
		tt, err := triggersClient.TektonV1alpha1().TriggerTemplates(listenerNamespace).Get(trigger.TriggerTemplate.Name, v1.GetOptions{})
		if err != nil {
			log.Printf("Error getting TriggerTemplate %s", trigger.TriggerTemplate.Name)
		}
		log.Print(tt)
		// TODO: Header matching
		// TODO: Conditionally match
		// TODO: Create resources
	}
}

var triggersClient *triggersClientset.Clientset
var listenerName string
var listenerNamespace string

func main() {
	log.Print("EventListener pod started")
	listenerName = os.Getenv("LISTENER_NAME")
	if listenerName == "" {
		log.Fatal("LISTENER_NAME env not found")
	}
	listenerNamespace = os.Getenv("LISTENER_NAMESPACE")
	if listenerNamespace == "" {
		log.Fatal("LISTENER_NAMESPACE env not found")
	}

	clusterConfig, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal("Failed to get in cluster config", err)
	}
	triggersClient = triggersClientset.NewForConfigOrDie(clusterConfig)
	log.Printf("Listen and serve on port %d", eventlistener.Port)
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", eventlistener.Port), nil))
}
