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
	"fmt"

	"github.com/tektoncd/triggers/pkg/apis/config"
	"github.com/tektoncd/triggers/pkg/apis/triggers/contexts"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	scheduledtemplatereconciler "github.com/tektoncd/triggers/pkg/client/injection/reconciler/triggers/v1alpha1/scheduledtemplate"
	"github.com/tektoncd/triggers/pkg/reconciler/scheduledtemplate/resources"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"knative.dev/pkg/logging"
	pkgreconciler "knative.dev/pkg/reconciler"
)

const (
	// ControllerName is the name of the ScheduledTemplate controller.
	ControllerName = "ScheduledTemplate"

	// DefaultImage is the CronJob runner image used when not overridden.
	DefaultImage = "override-with-cronrunner:latest"
)

// Reconciler implements controller.Reconciler for ScheduledTemplate resources.
type Reconciler struct {
	KubeClientSet kubernetes.Interface
	Image         string
}

var _ scheduledtemplatereconciler.Interface = (*Reconciler)(nil)

// ReconcileKind compares the actual state with the desired, and attempts to
// converge the two.
func (r *Reconciler) ReconcileKind(ctx context.Context, st *v1alpha1.ScheduledTemplate) pkgreconciler.Event {
	st.Status.InitializeConditions()
	st.SetDefaults(contexts.WithUpgradeViaDefaulting(ctx))

	cronJobName := resources.CronJobName(st.Name)
	st.Status.CronJobName = cronJobName

	desired := resources.MakeCronJob(st, r.Image, config.FromContextOrDefaults(ctx).Defaults)

	existing, err := r.KubeClientSet.BatchV1().CronJobs(st.Namespace).Get(ctx, cronJobName, metav1.GetOptions{})
	switch {
	case err == nil:
		if !metav1.IsControlledBy(existing, st) {
			ownerErr := fmt.Errorf("CronJob %s/%s exists but is not controlled by ScheduledTemplate %s (uid=%s)",
				st.Namespace, cronJobName, st.Name, st.UID)
			logging.FromContext(ctx).Error(ownerErr)
			st.Status.SetExistsCondition(v1alpha1.CronJobExists, ownerErr)
			st.Status.SetReadyCondition()
			return ownerErr
		}

		st.Status.SetExistsCondition(v1alpha1.CronJobExists, nil)
		r.syncStatusFromCronJob(st, existing)

		if cronJobNeedsUpdate(existing, desired) {
			updated := applyOwnedCronJobFields(existing, desired)
			if _, err := r.KubeClientSet.BatchV1().CronJobs(st.Namespace).Update(ctx, updated, metav1.UpdateOptions{}); err != nil {
				logging.FromContext(ctx).Errorf("Error updating CronJob %s/%s: %v", st.Namespace, cronJobName, err)
				st.Status.SetExistsCondition(v1alpha1.CronJobExists, err)
				st.Status.SetReadyCondition()
				return err
			}
			logging.FromContext(ctx).Infof("Updated CronJob %s in Namespace %s", cronJobName, st.Namespace)
		}

	case apierrors.IsNotFound(err):
		created, createErr := r.KubeClientSet.BatchV1().CronJobs(st.Namespace).Create(ctx, desired, metav1.CreateOptions{})
		st.Status.SetExistsCondition(v1alpha1.CronJobExists, createErr)
		if createErr != nil {
			logging.FromContext(ctx).Errorf("Error creating CronJob %s/%s: %v", st.Namespace, cronJobName, createErr)
			st.Status.SetReadyCondition()
			return createErr
		}
		r.syncStatusFromCronJob(st, created)
		logging.FromContext(ctx).Infof("Created CronJob %s in Namespace %s", cronJobName, st.Namespace)

	default:
		logging.FromContext(ctx).Error(err)
		st.Status.SetExistsCondition(v1alpha1.CronJobExists, err)
		st.Status.SetReadyCondition()
		return err
	}

	st.Status.ObservedGeneration = st.Generation
	st.Status.SetReadyCondition()
	return nil
}

// cronJobNeedsUpdate reports whether controller-owned fields differ.
// It ignores Kubernetes-defaulted PodSpec fields (imagePullPolicy, dnsPolicy, etc.)
// so a defaulted CronJob does not look like drift.
func cronJobNeedsUpdate(existing, desired *batchv1.CronJob) bool {
	if !equality.Semantic.DeepEqual(existing.Labels, desired.Labels) {
		return true
	}
	if existing.Spec.Schedule != desired.Spec.Schedule {
		return true
	}
	if !equality.Semantic.DeepEqual(existing.Spec.JobTemplate.Spec.BackoffLimit, desired.Spec.JobTemplate.Spec.BackoffLimit) {
		return true
	}
	return !equality.Semantic.DeepEqual(ownedPodSpec(existing), ownedPodSpec(desired))
}

// ownedPodSpec extracts the PodSpec fields this controller manages.
func ownedPodSpec(cj *batchv1.CronJob) corev1.PodSpec {
	ps := cj.Spec.JobTemplate.Spec.Template.Spec
	out := corev1.PodSpec{
		ServiceAccountName: ps.ServiceAccountName,
		RestartPolicy:      ps.RestartPolicy,
		SecurityContext:    ps.SecurityContext.DeepCopy(),
	}
	if len(ps.Containers) == 0 {
		return out
	}
	c := ps.Containers[0]
	out.Containers = []corev1.Container{{
		Name:            c.Name,
		Image:           c.Image,
		Env:             append([]corev1.EnvVar(nil), c.Env...),
		SecurityContext: c.SecurityContext.DeepCopy(),
	}}
	return out
}

// applyOwnedCronJobFields copies controller-owned fields onto existing without
// replacing the full CronJobSpec (preserves suspend, history limits, etc.).
func applyOwnedCronJobFields(existing, desired *batchv1.CronJob) *batchv1.CronJob {
	updated := existing.DeepCopy()
	updated.Labels = desired.Labels
	updated.OwnerReferences = desired.OwnerReferences
	updated.Spec.Schedule = desired.Spec.Schedule
	updated.Spec.JobTemplate.ObjectMeta.Labels = desired.Spec.JobTemplate.ObjectMeta.Labels
	updated.Spec.JobTemplate.Spec.BackoffLimit = desired.Spec.JobTemplate.Spec.BackoffLimit
	updated.Spec.JobTemplate.Spec.Template.ObjectMeta.Labels = desired.Spec.JobTemplate.Spec.Template.ObjectMeta.Labels

	pod := &updated.Spec.JobTemplate.Spec.Template.Spec
	desiredPod := desired.Spec.JobTemplate.Spec.Template.Spec
	pod.ServiceAccountName = desiredPod.ServiceAccountName
	pod.RestartPolicy = desiredPod.RestartPolicy
	pod.SecurityContext = desiredPod.SecurityContext.DeepCopy()

	if len(desiredPod.Containers) > 0 {
		desiredC := desiredPod.Containers[0]
		if len(pod.Containers) == 0 {
			pod.Containers = []corev1.Container{{}}
		}
		pod.Containers[0].Name = desiredC.Name
		pod.Containers[0].Image = desiredC.Image
		pod.Containers[0].Env = append([]corev1.EnvVar(nil), desiredC.Env...)
		pod.Containers[0].SecurityContext = desiredC.SecurityContext.DeepCopy()
	}
	return updated
}

func (r *Reconciler) syncStatusFromCronJob(st *v1alpha1.ScheduledTemplate, cj *batchv1.CronJob) {
	if cj.Status.LastScheduleTime != nil {
		st.Status.LastScheduleTime = cj.Status.LastScheduleTime.DeepCopy()
	}
	if cj.Status.LastSuccessfulTime != nil {
		st.Status.LastSuccessfulTime = cj.Status.LastSuccessfulTime.DeepCopy()
	}
	if cj.Status.Active != nil {
		st.Status.Active = append([]corev1.ObjectReference(nil), cj.Status.Active...)
	} else {
		st.Status.Active = nil
	}
}
