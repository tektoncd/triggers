//go:build e2e

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

package test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	knativetest "knative.dev/pkg/test"
)

const (
	// triggerControllerMetricsPort is the Prometheus metrics port exposed by
	// the tekton-triggers controller (configured via config-observability-triggers).
	triggerControllerMetricsPort = "9000"
)

// scrapeTriggersControllerMetrics scrapes the /metrics endpoint of the
// tekton-triggers controller pod via the Kubernetes API server proxy.
func scrapeTriggersControllerMetrics(ctx context.Context, t *testing.T, c *clients) map[string]*dto.MetricFamily {
	t.Helper()

	pods, err := c.KubeClient.CoreV1().Pods(triggersNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=controller,app.kubernetes.io/part-of=tekton-triggers",
	})
	if err != nil {
		t.Fatalf("Failed to list triggers controller pods: %v", err)
	}

	var podName string
	for _, pod := range pods.Items {
		if pod.Status.Phase != corev1.PodRunning {
			continue
		}
		allReady := true
		for _, cs := range pod.Status.ContainerStatuses {
			if !cs.Ready {
				allReady = false
				break
			}
		}
		if allReady {
			podName = pod.Name
			break
		}
	}
	if podName == "" {
		t.Fatalf("No Running/Ready triggers controller pod found in namespace %s", triggersNamespace)
	}

	result := c.KubeClient.
		CoreV1().
		RESTClient().
		Get().
		Resource("pods").
		Name(podName + ":" + triggerControllerMetricsPort).
		Namespace(triggersNamespace).
		SubResource("proxy").
		Suffix("metrics").
		Do(ctx)

	body, err := result.Raw()
	if err != nil {
		t.Fatalf("Failed to scrape metrics from triggers controller: %v", err)
	}

	parser := expfmt.NewTextParser(model.LegacyValidation)
	families, err := parser.TextToMetricFamilies(strings.NewReader(string(body)))
	if err != nil {
		t.Fatalf("Failed to parse metrics: %v", err)
	}
	return families
}

// waitForTriggersMetric polls until the named metric family appears in the controller metrics.
func waitForTriggersMetric(ctx context.Context, t *testing.T, c *clients, metricName string, timeout time.Duration) map[string]*dto.MetricFamily {
	t.Helper()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		families := scrapeTriggersControllerMetrics(ctx, t, c)
		if _, ok := families[metricName]; ok {
			return families
		}
		select {
		case <-ctx.Done():
			t.Fatalf("Timed out waiting for metric %q to appear (waited %v): %v", metricName, timeout, ctx.Err())
			return nil
		case <-time.After(5 * time.Second):
		}
	}
}

// waitForGaugeAtLeast polls until the named gauge metric reaches at least min.
// The triggers controller reports gauge metrics every 60 s, so this may wait
// up to that interval before the value becomes non-zero.
func waitForGaugeAtLeast(ctx context.Context, t *testing.T, c *clients, metricName string, min float64, timeout time.Duration) map[string]*dto.MetricFamily {
	t.Helper()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		families := scrapeTriggersControllerMetrics(ctx, t, c)
		if gaugeValue(families, metricName) >= min {
			return families
		}
		select {
		case <-ctx.Done():
			t.Fatalf("Timed out waiting for %q >= %v (waited %v): %v", metricName, min, timeout, ctx.Err())
			return nil
		case <-time.After(5 * time.Second):
		}
	}
}

// gaugeValue returns the sum of gauge values for the given metric.
func gaugeValue(families map[string]*dto.MetricFamily, name string) float64 {
	fam, ok := families[name]
	if !ok {
		return 0
	}
	var total float64
	for _, m := range fam.GetMetric() {
		if g := m.GetGauge(); g != nil {
			total += g.GetValue()
		}
	}
	return total
}

// TestOTelMetrics is a consolidated e2e test for the OpenCensus-to-OpenTelemetry
// metrics migration in Triggers (PR #1934). It creates a minimal EventListener
// (with a TriggerBinding and a TriggerTemplate that produces a TaskRun) and
// scrapes the controller /metrics endpoint on port 9000 to verify all metric
// families are present and that infrastructure metrics use new OTel naming.
//
// Metrics verified:
//
//	Controller gauge metrics (registered at startup, appear immediately):
//	- controller_eventlistener_count
//	- controller_triggerbinding_count
//	- controller_clustertriggerbinding_count
//	- controller_triggertemplate_count
//	- controller_clusterinterceptor_count
//
//	Infrastructure metrics (OTel renamed):
//	- kn_workqueue_* prefix present
//	- go_* runtime metrics present
func TestOTelMetrics(t *testing.T) {
	c, namespace := setup(t)
	defer tearDown(t, c, namespace)
	knativetest.CleanupOnInterrupt(func() { tearDown(t, c, namespace) }, t.Logf)

	ctx := context.Background()

	// ========== Create minimal Triggers resources ==========

	// TriggerTemplate producing a minimal TaskRun.
	tr := pipelinev1.TaskRun{
		TypeMeta:   metav1.TypeMeta{APIVersion: "tekton.dev/v1", Kind: "TaskRun"},
		ObjectMeta: metav1.ObjectMeta{GenerateName: "otel-metrics-tr-", Namespace: namespace},
		Spec: pipelinev1.TaskRunSpec{
			TaskSpec: &pipelinev1.TaskSpec{
				Steps: []pipelinev1.Step{{
					Name:   "noop",
					Image:  "mirror.gcr.io/alpine",
					Script: "exit 0",
				}},
			},
		},
	}
	trBytes, err := json.Marshal(tr)
	if err != nil {
		t.Fatalf("Failed to marshal TaskRun template: %v", err)
	}

	_, err = c.TriggersClient.TriggersV1beta1().TriggerTemplates(namespace).Create(ctx,
		&triggersv1.TriggerTemplate{
			ObjectMeta: metav1.ObjectMeta{Name: "otel-metrics-tt"},
			Spec: triggersv1.TriggerTemplateSpec{
				ResourceTemplates: []triggersv1.TriggerResourceTemplate{
					{RawExtension: runtime.RawExtension{Raw: trBytes}},
				},
			},
		}, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create TriggerTemplate: %v", err)
	}

	_, err = c.TriggersClient.TriggersV1beta1().TriggerBindings(namespace).Create(ctx,
		&triggersv1.TriggerBinding{
			ObjectMeta: metav1.ObjectMeta{Name: "otel-metrics-tb"},
			Spec: triggersv1.TriggerBindingSpec{
				Params: []triggersv1.Param{{Name: "dummy", Value: "$(body)"}},
			},
		}, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create TriggerBinding: %v", err)
	}

	ttRef := "otel-metrics-tt"
	_, err = c.TriggersClient.TriggersV1beta1().EventListeners(namespace).Create(ctx,
		&triggersv1.EventListener{
			ObjectMeta: metav1.ObjectMeta{Name: "otel-metrics-el"},
			Spec: triggersv1.EventListenerSpec{
				ServiceAccountName: "default",
				Triggers: []triggersv1.EventListenerTrigger{{
					Bindings: []*triggersv1.TriggerSpecBinding{{
						Ref:  "otel-metrics-tb",
						Kind: triggersv1.NamespacedTriggerBindingKind,
					}},
					Template: &triggersv1.EventListenerTemplate{Ref: &ttRef},
				}},
			},
		}, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create EventListener: %v", err)
	}

	// ========== Scrape metrics ==========
	// Wait for kn_workqueue_adds_total which only appears after the controller
	// processes its first reconcile item. This prevents the kn_workqueue_*
	// assertions from being flaky: controller_eventlistener_count is registered
	// at startup (value 0) and may appear before any workqueue activity, but
	// kn_workqueue_* only appears after the first item is queued and processed.

	t.Log("Waiting for kn_workqueue_adds_total to appear")
	families := waitForTriggersMetric(ctx, t, c, "kn_workqueue_adds_total", 2*time.Minute)
	t.Logf("Scraped %d metric families from triggers controller", len(families))

	// Gauge value assertions — poll until controller_eventlistener_count >= 1
	// (covers the 60 s reporting period), then assert created-resource counts.
	t.Log("Waiting for controller_eventlistener_count >= 1 (up to 90s for 60s reporting period)")
	gaugeFamilies := waitForGaugeAtLeast(ctx, t, c, "controller_eventlistener_count", 1, 90*time.Second)
	t.Logf("Scraped %d metric families for gauge value assertions", len(gaugeFamilies))

	// Resources created: 1 EventListener, 1 TriggerBinding, 1 TriggerTemplate.
	gaugeTests := []struct {
		name       string
		metricName string
		wantMin    float64
	}{
		{name: "eventlistener_count", metricName: "controller_eventlistener_count", wantMin: 1},
		{name: "triggerbinding_count", metricName: "controller_triggerbinding_count", wantMin: 1},
		{name: "triggertemplate_count", metricName: "controller_triggertemplate_count", wantMin: 1},
	}
	for _, tt := range gaugeTests {
		t.Run(tt.name, func(t *testing.T) {
			v := gaugeValue(gaugeFamilies, tt.metricName)
			if v < tt.wantMin {
				t.Errorf("%s = %v, want >= %v", tt.metricName, v, tt.wantMin)
			}
			t.Logf("%s: %v", tt.metricName, v)
		})
	}

	// No ClusterTriggerBindings or ClusterInterceptors created — assert presence only.
	gaugeExistTests := []struct {
		name       string
		metricName string
	}{
		{name: "clustertriggerbinding_count", metricName: "controller_clustertriggerbinding_count"},
		{name: "clusterinterceptor_count", metricName: "controller_clusterinterceptor_count"},
	}
	for _, tt := range gaugeExistTests {
		t.Run(tt.name, func(t *testing.T) {
			if _, ok := gaugeFamilies[tt.metricName]; !ok {
				t.Errorf("%s not found", tt.metricName)
			}
		})
	}

	// Infrastructure metrics use new OTel naming.
	infraTests := []struct {
		name   string
		prefix string
		errMsg string
	}{
		{
			name:   "workqueue_uses_kn_prefix",
			prefix: "kn_workqueue_",
			errMsg: "Expected at least one kn_workqueue_* metric, found none",
		},
		{
			name:   "go_runtime_uses_standard_prefix",
			prefix: "go_",
			errMsg: "Expected standard go_* runtime metrics, found none",
		},
	}
	for _, tt := range infraTests {
		t.Run(tt.name, func(t *testing.T) {
			for name := range families {
				if strings.HasPrefix(name, tt.prefix) {
					return
				}
			}
			t.Error(tt.errMsg)
		})
	}

	// Old OC workqueue metrics must be absent.
	// TODO: Remove in a future release once no OC-based release is supported.
	for name := range families {
		if strings.HasPrefix(name, "tekton_triggers_controller_workqueue_") {
			t.Errorf("Old OC workqueue metric %q still present; expected kn_workqueue_* after OTel migration", name)
		}
	}
}
