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

package main

import (
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"github.com/tektoncd/triggers/pkg/interceptors/bitbucket"
	"github.com/tektoncd/triggers/pkg/interceptors/cel"
	"github.com/tektoncd/triggers/pkg/interceptors/github"
	"github.com/tektoncd/triggers/pkg/interceptors/gitlab"
	"github.com/tektoncd/triggers/pkg/interceptors/server"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/signals"

	kubeclient "knative.dev/pkg/client/injection/kube/client"
)

func main() {
	// We only need to enable injection here to initialize the kubeclient for DefaultSecretGetter
	// TODO: Have server.Main accept an InterceptorConstructor that can return an InterceptorInterface
	ctx := signals.NewContext()
	cfg := injection.ParseAndGetRESTConfigOrDie()
	ctx, _ = injection.EnableInjectionOrDie(ctx, cfg)
	secretGetter := interceptors.DefaultSecretGetter(kubeclient.Get(ctx).CoreV1())

	// the name here has to match the clusterInterceptor CRD name
	// and by convention is also the path within the server that will be used to serve this interceptor
	coreInterceptors := map[string]v1beta1.InterceptorInterface{
		"bitbucket": bitbucket.NewInterceptor(secretGetter),
		"cel":       cel.NewInterceptor(secretGetter),
		"github":    github.NewInterceptor(secretGetter),
		"gitlab":    gitlab.NewInterceptor(secretGetter),
	}

	options := server.Options{
		// TODO: Initialize from env vars?
		ServiceName: "tekton-triggers-core-interceptors",
		SecretName:  "tekton-triggers-core-interceptors-certs",
		Port:        8443,
	}
	// TODO: Split Main into New and Run
	server.Main(ctx, "core-interceptors", cfg, options, coreInterceptors)
}
