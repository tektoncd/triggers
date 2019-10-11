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

	"github.com/tektoncd/triggers/pkg/sink"
)

func main() {
	log.Print("EventListener pod started")

	sinkArgs, err := sink.GetArgs()
	if err != nil {
		log.Fatal(err)
	}

	sinkClients, err := sink.ConfigureClients()
	if err != nil {
		log.Fatal(err)
	}

	// Create sink Resource
	r := sink.Resource{
		DiscoveryClient:        sinkClients.DiscoveryClient,
		RESTClient:             sinkClients.RESTClient,
		TriggersClient:         sinkClients.TriggersClient,
		PipelineClient:         sinkClients.PipelineClient,
		HTTPClient:             http.DefaultClient, // TODO: Use a different client since the default client has weird timeout values
		EventListenerName:      sinkArgs.ElName,
		EventListenerNamespace: sinkArgs.ElNamespace,
	}

	// Listen and serve
	log.Printf("Listen and serve on port %s", sinkArgs.Port)
	http.HandleFunc("/", r.HandleEvent)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", sinkArgs.Port), nil))
}
