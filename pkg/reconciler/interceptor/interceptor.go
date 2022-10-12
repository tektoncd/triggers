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

package interceptor

import (
	"context"

	"github.com/tektoncd/triggers/pkg/apis/triggers/contexts"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	interceptorreconciler "github.com/tektoncd/triggers/pkg/client/injection/reconciler/triggers/v1alpha1/interceptor"
	v1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/logging"
	pkgreconciler "knative.dev/pkg/reconciler"
)

const ControllerName = "Interceptor"

// Reconciler implements controller.Reconciler for Configuration resources.
type Reconciler struct {
}

var (
	// Check that our Reconciler implements interceptorreconciler.Interface
	_ interceptorreconciler.Interface = (*Reconciler)(nil)
)

func (r *Reconciler) ReconcileKind(ctx context.Context, it *v1alpha1.Interceptor) pkgreconciler.Event {
	logger := logging.FromContext(ctx)
	if it.Status.Address == nil { // Initialize Address if needed
		it.Status.Address = &v1.Addressable{}
	}
	if contexts.IsUpgradeViaDefaulting(ctx) { // Set defaults
		it.SetDefaults(ctx)
	}
	url, err := it.ResolveAddress()
	logger.Debugf("Resolved Address is %s", url)
	if err != nil {
		return err
	}
	it.Status.Address.URL = url
	return nil
}
