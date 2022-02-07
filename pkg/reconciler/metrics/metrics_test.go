/*
Copyright 2022 The Tekton Authors

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

package metrics

import (
	"sync"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	fakeCIInformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1alpha1/clusterinterceptor/fake"
	fakeCTBInformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1beta1/clustertriggerbinding/fake"
	fakeELInformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1beta1/eventlistener/fake"
	fakeTBInformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1beta1/triggerbinding/fake"
	fakeTTInformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1beta1/triggertemplate/fake"
	"github.com/tektoncd/triggers/test"
	"knative.dev/pkg/metrics/metricstest"
	_ "knative.dev/pkg/metrics/testing"
)

func TestUninitializedMetrics(t *testing.T) {
	metrics := &Recorder{}
	ctx, _ := test.SetupFakeContext(t)

	metrics.countMetrics(ctx, 3, elCount)
	metricstest.CheckStatsNotReported(t, "eventlistener_count")

	metrics.countMetrics(ctx, 3, ctbCount)
	metricstest.CheckStatsNotReported(t, "clustertriggerbinding_count")

	metrics.countMetrics(ctx, 3, tbCount)
	metricstest.CheckStatsNotReported(t, "triggerbinding_count")

	metrics.countMetrics(ctx, 3, ttCount)
	metricstest.CheckStatsNotReported(t, "triggertemplate_count")

	metrics.countMetrics(ctx, 3, ciCount)
	metricstest.CheckStatsNotReported(t, "clusterinterceptor_count")

}

func TestCountMetrics(t *testing.T) {
	unregisterMetrics()
	ctx, _ := test.SetupFakeContext(t)
	ctx = WithClient(ctx)

	rec := Get(ctx)

	fakeELIn := fakeELInformer.Get(ctx)
	fakeCTBIn := fakeCTBInformer.Get(ctx)
	fakeTBIn := fakeTBInformer.Get(ctx)
	fakeTTIn := fakeTTInformer.Get(ctx)
	fakeCIIn := fakeCIInformer.Get(ctx)
	e1 := &v1beta1.EventListener{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-1",
			Namespace: "test",
		},
	}
	e2 := &v1beta1.EventListener{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-2",
			Namespace: "test",
		},
	}
	e3 := &v1beta1.EventListener{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-3",
			Namespace: "test",
		},
	}

	for _, el := range []*v1beta1.EventListener{e1, e2, e3} {
		if err := fakeELIn.Informer().GetIndexer().Add(el); err != nil {
			t.Fatalf("Adding EL to informer: %v", err)
		}
	}

	tt1 := &v1beta1.TriggerTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-1",
			Namespace: "test",
		},
	}

	tt2 := &v1beta1.TriggerTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-2",
			Namespace: "test",
		},
	}

	tt3 := &v1beta1.TriggerTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-3",
			Namespace: "test",
		},
	}

	for _, tt := range []*v1beta1.TriggerTemplate{tt1, tt2, tt3} {
		if err := fakeTTIn.Informer().GetIndexer().Add(tt); err != nil {
			t.Fatalf("Adding TT to informer: %v", err)
		}
	}

	tb1 := &v1beta1.TriggerBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-1",
			Namespace: "test",
		},
	}

	tb2 := &v1beta1.TriggerBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-2",
			Namespace: "test",
		},
	}

	tb3 := &v1beta1.TriggerBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-3",
			Namespace: "test",
		},
	}

	for _, tb := range []*v1beta1.TriggerBinding{tb1, tb2, tb3} {
		if err := fakeTBIn.Informer().GetIndexer().Add(tb); err != nil {
			t.Fatalf("Adding TB to informer: %v", err)
		}
	}

	ctb1 := &v1beta1.ClusterTriggerBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-1",
			Namespace: "test",
		},
	}

	ctb2 := &v1beta1.ClusterTriggerBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-2",
			Namespace: "test",
		},
	}

	ctb3 := &v1beta1.ClusterTriggerBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-3",
			Namespace: "test",
		},
	}

	for _, ctb := range []*v1beta1.ClusterTriggerBinding{ctb1, ctb2, ctb3} {
		if err := fakeCTBIn.Informer().GetIndexer().Add(ctb); err != nil {
			t.Fatalf("Adding CTB to informer: %v", err)
		}
	}

	ci1 := &v1alpha1.ClusterInterceptor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-1",
			Namespace: "test",
		},
	}

	ci2 := &v1alpha1.ClusterInterceptor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-2",
			Namespace: "test",
		},
	}

	ci3 := &v1alpha1.ClusterInterceptor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-3",
			Namespace: "test",
		},
	}

	for _, ci := range []*v1alpha1.ClusterInterceptor{ci1, ci2, ci3} {
		if err := fakeCIIn.Informer().GetIndexer().Add(ci); err != nil {
			t.Fatalf("Adding CI to informer: %v", err)
		}
	}

	li := listers{
		el:  fakeELIn.Lister(),
		ctb: fakeCTBIn.Lister(),
		tb:  fakeTBIn.Lister(),
		tt:  fakeTTIn.Lister(),
		ci:  fakeCIIn.Lister(),
	}

	rec.CountMetrics(ctx, li)
	metricstest.CheckLastValueData(t, elMetricsName, map[string]string{}, 3)
	metricstest.CheckLastValueData(t, ttMetricsName, map[string]string{}, 3)
	metricstest.CheckLastValueData(t, tbMetricsName, map[string]string{}, 3)
	metricstest.CheckLastValueData(t, ctbMetricsName, map[string]string{}, 3)
	metricstest.CheckLastValueData(t, ciMetricsName, map[string]string{}, 3)
}

func TestELCount(t *testing.T) {
	unregisterMetrics()
	ctx, _ := test.SetupFakeContext(t)
	ctx = WithClient(ctx)

	rec := Get(ctx)
	rec.countMetrics(ctx, float64(3), elCount)
	metricstest.CheckLastValueData(t, elMetricsName, map[string]string{}, 3)
}

func TestTTCount(t *testing.T) {
	unregisterMetrics()
	ctx, _ := test.SetupFakeContext(t)
	ctx = WithClient(ctx)

	rec := Get(ctx)
	rec.countMetrics(ctx, float64(3), ttCount)
	metricstest.CheckLastValueData(t, ttMetricsName, map[string]string{}, 3)
}

func TestTBCount(t *testing.T) {
	unregisterMetrics()
	ctx, _ := test.SetupFakeContext(t)
	ctx = WithClient(ctx)

	rec := Get(ctx)
	rec.countMetrics(ctx, float64(3), tbCount)
	metricstest.CheckLastValueData(t, tbMetricsName, map[string]string{}, 3)
}

func TestCTBCount(t *testing.T) {
	unregisterMetrics()
	ctx, _ := test.SetupFakeContext(t)
	ctx = WithClient(ctx)

	rec := Get(ctx)
	rec.countMetrics(ctx, float64(3), ctbCount)
	metricstest.CheckLastValueData(t, ctbMetricsName, map[string]string{}, 3)
}

func TestCICount(t *testing.T) {
	unregisterMetrics()
	ctx, _ := test.SetupFakeContext(t)
	ctx = WithClient(ctx)

	rec := Get(ctx)
	rec.countMetrics(ctx, float64(3), ciCount)
	metricstest.CheckLastValueData(t, ciMetricsName, map[string]string{}, 3)
}

func unregisterMetrics() {
	metricstest.Unregister(elMetricsName, tbMetricsName, ctbMetricsName, ttMetricsName, ciMetricsName)

	// Allow the recorder singleton to be recreated.
	once = sync.Once{}
	r = nil
	recorderErr = nil
}
