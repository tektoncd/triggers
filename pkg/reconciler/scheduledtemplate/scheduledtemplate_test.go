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

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/triggers/pkg/apis/config"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"github.com/tektoncd/triggers/pkg/reconciler/scheduledtemplate/resources"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clienttesting "k8s.io/client-go/testing"
	"knative.dev/pkg/apis"
	fakekubeclient "knative.dev/pkg/client/injection/kube/client/fake"
	"knative.dev/pkg/ptr"
	rtesting "knative.dev/pkg/reconciler/testing"
)

func TestReconcileKind_CreateCronJob(t *testing.T) {
	ctx, _ := rtesting.SetupFakeContext(t)
	kube := fakekubeclient.Get(ctx)
	r := &Reconciler{
		KubeClientSet: kube,
		Image:         "cronrunner:test",
	}

	st := &v1alpha1.ScheduledTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "daily",
			Namespace:  "default",
			UID:        "uid-1",
			Generation: 1,
		},
		Spec: v1alpha1.ScheduledTemplateSpec{
			Schedule: "0 0 * * *",
			Ref:      ptr.String("my-tt"),
		},
	}

	if err := r.ReconcileKind(ctx, st); err != nil {
		t.Fatalf("ReconcileKind() error = %v", err)
	}

	if st.Status.CronJobName != "st-daily" {
		t.Errorf("CronJobName = %q, want st-daily", st.Status.CronJobName)
	}
	if st.Spec.ServiceAccountName != config.DefaultServiceAccountValue {
		t.Errorf("ServiceAccountName after defaults = %q, want %q", st.Spec.ServiceAccountName, config.DefaultServiceAccountValue)
	}
	if st.Status.ObservedGeneration != 1 {
		t.Errorf("ObservedGeneration = %d, want 1", st.Status.ObservedGeneration)
	}

	cjCond := st.Status.GetCondition(v1alpha1.CronJobExists)
	if cjCond == nil || cjCond.Status != corev1.ConditionTrue {
		t.Fatalf("CronJobExists = %#v, want True", cjCond)
	}
	ready := st.Status.GetCondition(apis.ConditionReady)
	if ready == nil || ready.Status != corev1.ConditionTrue {
		t.Fatalf("Ready = %#v, want True", ready)
	}

	got, err := kube.BatchV1().CronJobs("default").Get(ctx, "st-daily", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get CronJob: %v", err)
	}
	if got.Spec.Schedule != "0 0 * * *" {
		t.Errorf("CronJob schedule = %q, want 0 0 * * *", got.Spec.Schedule)
	}
	if len(got.OwnerReferences) != 1 || got.OwnerReferences[0].UID != "uid-1" {
		t.Errorf("OwnerReferences = %#v, want uid-1", got.OwnerReferences)
	}
}

func TestReconcileKind_UpdateSchedule(t *testing.T) {
	ctx, _ := rtesting.SetupFakeContext(t)
	kube := fakekubeclient.Get(ctx)
	r := &Reconciler{
		KubeClientSet: kube,
		Image:         "cronrunner:test",
	}

	st := &v1alpha1.ScheduledTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "daily",
			Namespace:  "default",
			UID:        "uid-1",
			Generation: 1,
		},
		Spec: v1alpha1.ScheduledTemplateSpec{
			Schedule:           "0 0 * * *",
			ServiceAccountName: "sa",
			Ref:                ptr.String("my-tt"),
		},
	}
	if err := r.ReconcileKind(ctx, st); err != nil {
		t.Fatalf("first ReconcileKind() error = %v", err)
	}

	st.Spec.Schedule = "*/5 * * * *"
	st.Generation = 2
	if err := r.ReconcileKind(ctx, st); err != nil {
		t.Fatalf("second ReconcileKind() error = %v", err)
	}

	got, err := kube.BatchV1().CronJobs("default").Get(ctx, "st-daily", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get CronJob: %v", err)
	}
	if got.Spec.Schedule != "*/5 * * * *" {
		t.Errorf("CronJob schedule = %q, want */5 * * * *", got.Spec.Schedule)
	}
	if st.Status.ObservedGeneration != 2 {
		t.Errorf("ObservedGeneration = %d, want 2", st.Status.ObservedGeneration)
	}
}

func TestReconcileKind_SyncLastScheduleTime(t *testing.T) {
	ctx, _ := rtesting.SetupFakeContext(t)
	kube := fakekubeclient.Get(ctx)
	r := &Reconciler{
		KubeClientSet: kube,
		Image:         "cronrunner:test",
	}

	st := &v1alpha1.ScheduledTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "daily",
			Namespace: "default",
			UID:       "uid-1",
		},
		Spec: v1alpha1.ScheduledTemplateSpec{
			Schedule:           "* * * * *",
			ServiceAccountName: "sa",
			Ref:                ptr.String("my-tt"),
		},
	}
	if err := r.ReconcileKind(ctx, st); err != nil {
		t.Fatalf("ReconcileKind() error = %v", err)
	}

	cj, err := kube.BatchV1().CronJobs("default").Get(ctx, resources.CronJobName(st.Name), metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get CronJob: %v", err)
	}
	now := metav1.Now()
	success := metav1.NewTime(now.Add(time.Minute))
	cj.Status.LastScheduleTime = &now
	cj.Status.LastSuccessfulTime = &success
	cj.Status.Active = []corev1.ObjectReference{{Name: "job-1"}}
	if _, err := kube.BatchV1().CronJobs("default").UpdateStatus(ctx, cj, metav1.UpdateOptions{}); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	if err := r.ReconcileKind(ctx, st); err != nil {
		t.Fatalf("second ReconcileKind() error = %v", err)
	}
	if st.Status.LastScheduleTime == nil || !st.Status.LastScheduleTime.Equal(&now) {
		t.Errorf("LastScheduleTime = %#v, want %#v", st.Status.LastScheduleTime, now)
	}
	if st.Status.LastSuccessfulTime == nil || !st.Status.LastSuccessfulTime.Equal(&success) {
		t.Errorf("LastSuccessfulTime = %#v, want %#v", st.Status.LastSuccessfulTime, success)
	}
	if d := cmp.Diff([]corev1.ObjectReference{{Name: "job-1"}}, st.Status.Active); d != "" {
		t.Errorf("Active mismatch (-want +got):\n%s", d)
	}
}

func TestReconcileKind_RejectsUnownedCronJob(t *testing.T) {
	ctx, _ := rtesting.SetupFakeContext(t)
	kube := fakekubeclient.Get(ctx)
	r := &Reconciler{
		KubeClientSet: kube,
		Image:         "cronrunner:test",
	}

	// Pre-create a CronJob with the expected name but a different controller.
	orphan := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "st-daily",
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "triggers.tekton.dev/v1alpha1",
				Kind:       "ScheduledTemplate",
				Name:       "other",
				UID:        "uid-other",
				Controller: ptr.Bool(true),
			}},
		},
		Spec: batchv1.CronJobSpec{Schedule: "0 0 * * *"},
	}
	if _, err := kube.BatchV1().CronJobs("default").Create(ctx, orphan, metav1.CreateOptions{}); err != nil {
		t.Fatalf("Create orphan CronJob: %v", err)
	}

	st := &v1alpha1.ScheduledTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "daily",
			Namespace: "default",
			UID:       "uid-1",
		},
		Spec: v1alpha1.ScheduledTemplateSpec{
			Schedule:           "*/5 * * * *",
			ServiceAccountName: "sa",
			Ref:                ptr.String("my-tt"),
		},
	}
	if err := r.ReconcileKind(ctx, st); err == nil {
		t.Fatal("expected error when CronJob is not controlled by this ScheduledTemplate")
	}

	cjCond := st.Status.GetCondition(v1alpha1.CronJobExists)
	if cjCond == nil || cjCond.Status != corev1.ConditionFalse {
		t.Fatalf("CronJobExists = %#v, want False", cjCond)
	}
	ready := st.Status.GetCondition(apis.ConditionReady)
	if ready == nil || ready.Status != corev1.ConditionFalse {
		t.Fatalf("Ready = %#v, want False", ready)
	}

	got, err := kube.BatchV1().CronJobs("default").Get(ctx, "st-daily", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get CronJob: %v", err)
	}
	if got.Spec.Schedule != "0 0 * * *" {
		t.Errorf("CronJob was updated despite foreign owner; schedule = %q", got.Spec.Schedule)
	}
}

func TestReconcileKind_DefaultedCronJobNoUpdate(t *testing.T) {
	ctx, _ := rtesting.SetupFakeContext(t)
	kube := fakekubeclient.Get(ctx)
	r := &Reconciler{
		KubeClientSet: kube,
		Image:         "cronrunner:test",
	}

	st := &v1alpha1.ScheduledTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "daily",
			Namespace:  "default",
			UID:        "uid-1",
			Generation: 1,
		},
		Spec: v1alpha1.ScheduledTemplateSpec{
			Schedule:           "0 0 * * *",
			ServiceAccountName: "sa",
			Ref:                ptr.String("my-tt"),
		},
	}
	if err := r.ReconcileKind(ctx, st); err != nil {
		t.Fatalf("first ReconcileKind() error = %v", err)
	}

	cj, err := kube.BatchV1().CronJobs("default").Get(ctx, "st-daily", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get CronJob: %v", err)
	}
	// Simulate Kubernetes defaulting on the CronJob pod template.
	cj.Spec.JobTemplate.Spec.Template.Spec.DNSPolicy = corev1.DNSClusterFirst
	cj.Spec.JobTemplate.Spec.Template.Spec.SchedulerName = "default-scheduler"
	cj.Spec.JobTemplate.Spec.Template.Spec.Containers[0].ImagePullPolicy = corev1.PullIfNotPresent
	cj.Spec.JobTemplate.Spec.Template.Spec.Containers[0].TerminationMessagePath = "/dev/termination-log"
	cj.Spec.JobTemplate.Spec.Template.Spec.Containers[0].TerminationMessagePolicy = corev1.TerminationMessageReadFile
	if _, err := kube.BatchV1().CronJobs("default").Update(ctx, cj, metav1.UpdateOptions{}); err != nil {
		t.Fatalf("seed defaulted CronJob: %v", err)
	}

	kube.ClearActions()

	if err := r.ReconcileKind(ctx, st); err != nil {
		t.Fatalf("second ReconcileKind() error = %v", err)
	}

	for _, a := range kube.Actions() {
		if u, ok := a.(clienttesting.UpdateAction); ok && u.GetResource().Resource == "cronjobs" {
			t.Fatalf("unexpected CronJob update for defaulted-unchanged CronJob: %#v", a)
		}
	}
}

func TestReconcileKind_PreservesSuspendOnOwnedUpdate(t *testing.T) {
	ctx, _ := rtesting.SetupFakeContext(t)
	kube := fakekubeclient.Get(ctx)
	r := &Reconciler{
		KubeClientSet: kube,
		Image:         "cronrunner:test",
	}

	st := &v1alpha1.ScheduledTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "daily",
			Namespace:  "default",
			UID:        "uid-1",
			Generation: 1,
		},
		Spec: v1alpha1.ScheduledTemplateSpec{
			Schedule:           "0 0 * * *",
			ServiceAccountName: "sa",
			Ref:                ptr.String("my-tt"),
		},
	}
	if err := r.ReconcileKind(ctx, st); err != nil {
		t.Fatalf("first ReconcileKind() error = %v", err)
	}

	cj, err := kube.BatchV1().CronJobs("default").Get(ctx, "st-daily", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get CronJob: %v", err)
	}
	cj.Spec.Suspend = ptr.Bool(true)
	if _, err := kube.BatchV1().CronJobs("default").Update(ctx, cj, metav1.UpdateOptions{}); err != nil {
		t.Fatalf("set Suspend: %v", err)
	}

	st.Spec.Schedule = "*/5 * * * *"
	st.Generation = 2
	if err := r.ReconcileKind(ctx, st); err != nil {
		t.Fatalf("second ReconcileKind() error = %v", err)
	}

	got, err := kube.BatchV1().CronJobs("default").Get(ctx, "st-daily", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get CronJob: %v", err)
	}
	if got.Spec.Schedule != "*/5 * * * *" {
		t.Errorf("schedule = %q, want */5 * * * *", got.Spec.Schedule)
	}
	if got.Spec.Suspend == nil || !*got.Spec.Suspend {
		t.Errorf("Suspend = %v, want true (must be preserved across owned-field update)", got.Spec.Suspend)
	}
}

func TestCronJobNeedsUpdate_IgnoresPodDefaults(t *testing.T) {
	st := &v1alpha1.ScheduledTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "daily", Namespace: "default", UID: "uid-1"},
		Spec: v1alpha1.ScheduledTemplateSpec{
			Schedule:           "0 0 * * *",
			ServiceAccountName: "sa",
		},
	}
	desired := resources.MakeCronJob(st, "cronrunner:test", config.FromContextOrDefaults(context.Background()).Defaults)
	existing := desired.DeepCopy()
	existing.Spec.JobTemplate.Spec.Template.Spec.DNSPolicy = corev1.DNSClusterFirst
	existing.Spec.JobTemplate.Spec.Template.Spec.SchedulerName = "default-scheduler"
	existing.Spec.JobTemplate.Spec.Template.Spec.Containers[0].ImagePullPolicy = corev1.PullIfNotPresent

	if cronJobNeedsUpdate(existing, desired) {
		t.Fatal("defaulted PodSpec fields should not require an update")
	}

	existing.Spec.Schedule = "*/5 * * * *"
	if !cronJobNeedsUpdate(existing, desired) {
		t.Fatal("schedule drift should require an update")
	}
}
