/*
Copyright 2026 The Tekton Authors

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

package cronrunner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/tektoncd/triggers/pkg/apis/triggers"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	triggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	"github.com/tektoncd/triggers/pkg/reconciler/events"
	"github.com/tektoncd/triggers/pkg/resources"
	"github.com/tektoncd/triggers/pkg/template"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
)

// Runner resolves a ScheduledTemplate and creates its resource templates.
type Runner struct {
	TriggersClient  triggersclientset.Interface
	DynamicClient   dynamic.Interface
	DiscoveryClient discovery.DiscoveryInterface
	Logger          *zap.SugaredLogger

	// CloudEventClient is optional; used in tests. When nil, an HTTP client is created as needed.
	CloudEventClient cloudevents.Client
}

// Run loads the ScheduledTemplate identified by namespace/name, resolves
// params (defaults only; empty event body), and creates each resource.
func (r *Runner) Run(ctx context.Context, namespace, name string) error {
	if namespace == "" || name == "" {
		return errors.New("namespace and name are required")
	}

	st, err := r.TriggersClient.TriggersV1alpha1().ScheduledTemplates(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get ScheduledTemplate %s/%s: %w", namespace, name, err)
	}

	executionID := template.UUID()
	r.Logger.Infow("resolving ScheduledTemplate",
		"namespace", namespace,
		"name", name,
		"executionID", executionID,
	)

	tt, err := r.resolveTriggerTemplate(ctx, st)
	if err != nil {
		r.emitCloudEvent(ctx, st, executionID, events.ScheduledTemplateFailedV1, cloudEventPayload{
			ScheduledTemplate: st.Name,
			Namespace:         st.Namespace,
			ExecutionID:       executionID,
			Error:             err.Error(),
		})
		return err
	}

	params, unresolved, err := template.ResolveParams(
		template.ResolvedTrigger{TriggerTemplate: tt},
		[]byte("{}"),
		http.Header{},
		nil,
		template.NewTriggerContext(executionID),
	)
	if err != nil {
		r.emitCloudEvent(ctx, st, executionID, events.ScheduledTemplateFailedV1, cloudEventPayload{
			ScheduledTemplate: st.Name,
			Namespace:         st.Namespace,
			ExecutionID:       executionID,
			Error:             err.Error(),
		})
		return fmt.Errorf("resolve params: %w", err)
	}
	if len(unresolved) > 0 {
		r.Logger.Warnf("ScheduledTemplate %q has TriggerTemplate params with no value or default: %v; rendered resources may retain literal $(tt.params.<name>) references and be rejected", name, unresolved)
	}

	resolved := template.ResolveResources(tt, params)
	labels := map[string]string{
		triggers.ScheduledTemplateLabelKey: st.Name,
		triggers.EventIDLabelKey:           executionID,
	}

	for i, rr := range resolved {
		if err := resources.CreateWithLabels(r.Logger, rr, st.Namespace, executionID, r.DiscoveryClient, r.DynamicClient, labels); err != nil {
			r.emitCloudEvent(ctx, st, executionID, events.ScheduledTemplateFailedV1, cloudEventPayload{
				ScheduledTemplate: st.Name,
				Namespace:         st.Namespace,
				ExecutionID:       executionID,
				Error:             err.Error(),
			})
			return fmt.Errorf("create resource[%d]: %w", i, err)
		}
	}

	r.emitCloudEvent(ctx, st, executionID, events.ScheduledTemplateSuccessfulV1, cloudEventPayload{
		ScheduledTemplate: st.Name,
		Namespace:         st.Namespace,
		ExecutionID:       executionID,
		ResourceCount:     len(resolved),
	})

	r.Logger.Infow("created resources from ScheduledTemplate",
		"namespace", namespace,
		"name", name,
		"count", len(resolved),
		"executionID", executionID,
	)
	return nil
}

func (r *Runner) emitCloudEvent(ctx context.Context, st *v1alpha1.ScheduledTemplate, executionID, eventType string, payload cloudEventPayload) {
	sendCloudEventToSink(ctx, r.Logger, st.Spec.CloudEventSink, st, executionID, eventType, payload, r.CloudEventClient)
}

func (r *Runner) resolveTriggerTemplate(ctx context.Context, st *v1alpha1.ScheduledTemplate) (*v1beta1.TriggerTemplate, error) {
	if st.Spec.Ref != nil {
		tt, err := r.TriggersClient.TriggersV1alpha1().TriggerTemplates(st.Namespace).Get(ctx, *st.Spec.Ref, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("get TriggerTemplate %s/%s: %w", st.Namespace, *st.Spec.Ref, err)
		}
		return toV1Beta1TriggerTemplate(tt)
	}

	inline := &v1alpha1.TriggerTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:        st.Name,
			Namespace:   st.Namespace,
			Annotations: st.Annotations,
		},
		Spec: st.Spec.TriggerTemplateSpec,
	}
	return toV1Beta1TriggerTemplate(inline)
}

func toV1Beta1TriggerTemplate(tt *v1alpha1.TriggerTemplate) (*v1beta1.TriggerTemplate, error) {
	b, err := json.Marshal(tt)
	if err != nil {
		return nil, fmt.Errorf("marshal TriggerTemplate: %w", err)
	}
	out := &v1beta1.TriggerTemplate{}
	if err := json.Unmarshal(b, out); err != nil {
		return nil, fmt.Errorf("unmarshal TriggerTemplate as v1beta1: %w", err)
	}
	return out, nil
}
