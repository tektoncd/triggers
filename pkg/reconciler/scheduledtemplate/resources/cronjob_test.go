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

package resources_test

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/triggers/pkg/apis/config"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"github.com/tektoncd/triggers/pkg/reconciler/scheduledtemplate/resources"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"knative.dev/pkg/kmeta"
	"knative.dev/pkg/ptr"
)

func defaultDefaults() *config.Defaults {
	return &config.Defaults{
		DefaultServiceAccount: config.DefaultServiceAccountValue,
		DefaultRunAsUser:      65532,
		DefaultRunAsGroup:     65532,
		DefaultFSGroup:        65532,
		DefaultRunAsNonRoot:   true,
	}
}

func TestCronJobName(t *testing.T) {
	if got, want := resources.CronJobName("daily"), "st-daily"; got != want {
		t.Errorf("CronJobName() = %q, want %q", got, want)
	}

	prefix := strings.Repeat("a", 50)
	a := resources.CronJobName(prefix + "one")
	b := resources.CronJobName(prefix + "two")
	if len(a) > 52 {
		t.Errorf("CronJobName length = %d, want <= 52 (%q)", len(a), a)
	}
	if len(b) > 52 {
		t.Errorf("CronJobName length = %d, want <= 52 (%q)", len(b), b)
	}
	if a == b {
		t.Errorf("expected distinct CronJob names for different long ScheduledTemplate names; both %q", a)
	}
	if !strings.HasPrefix(a, "st-") || !strings.Contains(a, "-") {
		t.Errorf("unexpected hashed name format: %q", a)
	}
	if errs := validation.IsDNS1123Subdomain(a); len(errs) > 0 {
		t.Errorf("CronJobName %q failed DNS1123 subdomain validation: %v", a, errs)
	}
}

func TestCronJobName_DottedBoundary(t *testing.T) {
	// Valid ST name: 39 a's, a dot, then enough b's that st-<name> exceeds 52.
	stName := strings.Repeat("a", 39) + "." + strings.Repeat("b", 20)
	got := resources.CronJobName(stName)
	if len(got) > 52 {
		t.Errorf("CronJobName length = %d, want <= 52 (%q)", len(got), got)
	}
	if strings.Contains(got, ".-") {
		t.Errorf("CronJobName %q contains invalid '.-' boundary", got)
	}
	if errs := validation.IsDNS1123Subdomain(got); len(errs) > 0 {
		t.Errorf("CronJobName %q failed DNS1123 subdomain validation: %v", got, errs)
	}

	// Truncation ending on '-' must also be cleaned.
	stNameDash := strings.Repeat("a", 39) + "-" + strings.Repeat("b", 20)
	gotDash := resources.CronJobName(stNameDash)
	if errs := validation.IsDNS1123Subdomain(gotDash); len(errs) > 0 {
		t.Errorf("CronJobName %q failed DNS1123 subdomain validation: %v", gotDash, errs)
	}
}

func TestMakeCronJob(t *testing.T) {
	st := &v1alpha1.ScheduledTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "daily",
			Namespace: "default",
			UID:       "uid-1",
		},
		Spec: v1alpha1.ScheduledTemplateSpec{
			Schedule:           "0 0 * * *",
			ServiceAccountName: "tekton-triggers-sa",
		},
	}
	image := "cronrunner:test"
	got := resources.MakeCronJob(st, image, defaultDefaults())

	if got.Name != "st-daily" {
		t.Errorf("Name = %q, want st-daily", got.Name)
	}
	if got.Namespace != "default" {
		t.Errorf("Namespace = %q, want default", got.Namespace)
	}
	if got.Spec.Schedule != "0 0 * * *" {
		t.Errorf("Schedule = %q, want 0 0 * * *", got.Spec.Schedule)
	}

	wantOwner := *kmeta.NewControllerRef(st)
	if d := cmp.Diff([]metav1.OwnerReference{wantOwner}, got.OwnerReferences); d != "" {
		t.Errorf("OwnerReferences mismatch (-want +got):\n%s", d)
	}

	if got.Spec.JobTemplate.Spec.BackoffLimit == nil || *got.Spec.JobTemplate.Spec.BackoffLimit != 0 {
		t.Errorf("BackoffLimit = %v, want 0", got.Spec.JobTemplate.Spec.BackoffLimit)
	}

	podSpec := got.Spec.JobTemplate.Spec.Template.Spec
	if podSpec.ServiceAccountName != "tekton-triggers-sa" {
		t.Errorf("ServiceAccountName = %q, want tekton-triggers-sa", podSpec.ServiceAccountName)
	}
	if podSpec.RestartPolicy != corev1.RestartPolicyNever {
		t.Errorf("RestartPolicy = %q, want Never", podSpec.RestartPolicy)
	}
	wantPodSC := &corev1.PodSecurityContext{
		RunAsNonRoot: ptr.Bool(true),
		RunAsUser:    ptr.Int64(65532),
		RunAsGroup:   ptr.Int64(65532),
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
	}
	if d := cmp.Diff(wantPodSC, podSpec.SecurityContext); d != "" {
		t.Errorf("Pod SecurityContext mismatch (-want +got):\n%s", d)
	}
	if len(podSpec.Containers) != 1 {
		t.Fatalf("expected 1 container, got %d", len(podSpec.Containers))
	}
	c := podSpec.Containers[0]
	if c.Image != image {
		t.Errorf("Image = %q, want %q", c.Image, image)
	}
	wantEnv := []corev1.EnvVar{{
		Name:  resources.EnvScheduledTemplateName,
		Value: "daily",
	}, {
		Name:  resources.EnvScheduledTemplateNamespace,
		Value: "default",
	}}
	if d := cmp.Diff(wantEnv, c.Env); d != "" {
		t.Errorf("Env mismatch (-want +got):\n%s", d)
	}
	wantCSC := &corev1.SecurityContext{
		AllowPrivilegeEscalation: ptr.Bool(false),
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
		RunAsNonRoot: ptr.Bool(true),
		RunAsUser:    ptr.Int64(65532),
		RunAsGroup:   ptr.Int64(65532),
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
	}
	if d := cmp.Diff(wantCSC, c.SecurityContext); d != "" {
		t.Errorf("Container SecurityContext mismatch (-want +got):\n%s", d)
	}
	if got.Labels[resources.LabelScheduledTemplate] != "daily" {
		t.Errorf("label %s = %q, want daily", resources.LabelScheduledTemplate, got.Labels[resources.LabelScheduledTemplate])
	}
}

func TestMakeCronJob_OmitsEmptyIdentity(t *testing.T) {
	st := &v1alpha1.ScheduledTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "daily", Namespace: "default", UID: "uid-1"},
		Spec:       v1alpha1.ScheduledTemplateSpec{Schedule: "0 0 * * *", ServiceAccountName: "sa"},
	}
	defaults := defaultDefaults()
	defaults.IsDefaultRunAsUserEmpty = true
	defaults.IsDefaultRunAsGroupEmpty = true

	got := resources.MakeCronJob(st, "cronrunner:test", defaults)
	podSC := got.Spec.JobTemplate.Spec.Template.Spec.SecurityContext
	if podSC.RunAsUser != nil || podSC.RunAsGroup != nil {
		t.Errorf("expected omitted UID/GID, got user=%v group=%v", podSC.RunAsUser, podSC.RunAsGroup)
	}
	if podSC.RunAsNonRoot == nil || !*podSC.RunAsNonRoot {
		t.Errorf("RunAsNonRoot = %v, want true", podSC.RunAsNonRoot)
	}
	csc := got.Spec.JobTemplate.Spec.Template.Spec.Containers[0].SecurityContext
	if csc.RunAsUser != nil || csc.RunAsGroup != nil {
		t.Errorf("container expected omitted UID/GID, got user=%v group=%v", csc.RunAsUser, csc.RunAsGroup)
	}
	if csc.AllowPrivilegeEscalation == nil || *csc.AllowPrivilegeEscalation {
		t.Errorf("AllowPrivilegeEscalation = %v, want false", csc.AllowPrivilegeEscalation)
	}
}

func TestMakeCronJob_CustomIdentity(t *testing.T) {
	st := &v1alpha1.ScheduledTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "daily", Namespace: "default", UID: "uid-1"},
		Spec:       v1alpha1.ScheduledTemplateSpec{Schedule: "0 0 * * *", ServiceAccountName: "sa"},
	}
	defaults := defaultDefaults()
	defaults.DefaultRunAsUser = 1000710000
	defaults.DefaultRunAsGroup = 1000710000

	got := resources.MakeCronJob(st, "cronrunner:test", defaults)
	podSC := got.Spec.JobTemplate.Spec.Template.Spec.SecurityContext
	if podSC.RunAsUser == nil || *podSC.RunAsUser != 1000710000 {
		t.Errorf("RunAsUser = %v, want 1000710000", podSC.RunAsUser)
	}
	if podSC.RunAsGroup == nil || *podSC.RunAsGroup != 1000710000 {
		t.Errorf("RunAsGroup = %v, want 1000710000", podSC.RunAsGroup)
	}
}
