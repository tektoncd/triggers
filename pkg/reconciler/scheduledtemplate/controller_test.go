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

package scheduledtemplate

import (
	"context"
	"testing"
	"time"

	"github.com/tektoncd/triggers/pkg/apis/config"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	_ "github.com/tektoncd/triggers/pkg/client/injection/client/fake"
	_ "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1alpha1/scheduledtemplate/fake"
	_ "github.com/tektoncd/triggers/pkg/client/injection/kube/informers/batch/v1/cronjob/fake"
	"github.com/tektoncd/triggers/pkg/reconciler/scheduledtemplate/resources"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	fakekubeclient "knative.dev/pkg/client/injection/kube/client/fake"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/ptr"
	rtesting "knative.dev/pkg/reconciler/testing"
)

func testConfigWatcher() configmap.Watcher {
	return configmap.NewStaticWatcher(
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: config.GetDefaultsConfigName()}},
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: config.GetFeatureFlagsConfigName()}},
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: config.GetCoreInterceptorsConfigName()}},
	)
}

func TestNewController_CronJobChildEventsEnqueue(t *testing.T) {
	ctx, informers := rtesting.SetupFakeContext(t)
	ctx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)

	// Register production handlers only — do not construct a parallel handler.
	impl := NewController("cronrunner:test")(ctx, testConfigWatcher())
	if err := controller.StartInformers(ctx.Done(), informers...); err != nil {
		t.Fatalf("StartInformers: %v", err)
	}

	st := &v1alpha1.ScheduledTemplate{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "triggers.tekton.dev/v1alpha1",
			Kind:       "ScheduledTemplate",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "daily",
			Namespace: "default",
			UID:       "uid-1",
		},
		Spec: v1alpha1.ScheduledTemplateSpec{
			Schedule:           "0 0 * * *",
			ServiceAccountName: "sa",
			Ref:                ptr.String("my-tt"),
		},
	}
	cj := resources.MakeCronJob(st, "cronrunner:test", config.FromContextOrDefaults(ctx).Defaults)
	kube := fakekubeclient.Get(ctx)

	if _, err := kube.BatchV1().CronJobs(st.Namespace).Create(ctx, cj, metav1.CreateOptions{}); err != nil {
		t.Fatalf("Create CronJob: %v", err)
	}
	waitForEnqueue(t, impl, "create")
	drainWorkQueue(impl)

	updated := cj.DeepCopy()
	updated.ResourceVersion = "2"
	now := metav1.Now()
	updated.Status.LastScheduleTime = &now
	if _, err := kube.BatchV1().CronJobs(st.Namespace).Update(ctx, updated, metav1.UpdateOptions{}); err != nil {
		t.Fatalf("Update CronJob: %v", err)
	}
	waitForEnqueue(t, impl, "update")
	drainWorkQueue(impl)

	if err := kube.BatchV1().CronJobs(st.Namespace).Delete(ctx, cj.Name, metav1.DeleteOptions{}); err != nil {
		t.Fatalf("Delete CronJob: %v", err)
	}
	waitForEnqueue(t, impl, "delete")
}

func waitForEnqueue(t *testing.T, impl *controller.Impl, what string) {
	t.Helper()
	if err := wait.PollUntilContextTimeout(context.Background(), 10*time.Millisecond, 3*time.Second, true, func(context.Context) (bool, error) {
		return impl.WorkQueue().Len() > 0, nil
	}); err != nil {
		t.Fatalf("expected CronJob %s to enqueue owning ScheduledTemplate via NewController handlers: %v", what, err)
	}
}

func drainWorkQueue(impl *controller.Impl) {
	for impl.WorkQueue().Len() > 0 {
		item, _ := impl.WorkQueue().Get()
		impl.WorkQueue().Done(item)
	}
	time.Sleep(10 * time.Millisecond)
	for impl.WorkQueue().Len() > 0 {
		item, _ := impl.WorkQueue().Get()
		impl.WorkQueue().Done(item)
	}
}
