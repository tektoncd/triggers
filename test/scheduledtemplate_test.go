//go:build e2e
// +build e2e

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
	"fmt"
	"testing"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/triggers/pkg/apis/triggers"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	stresources "github.com/tektoncd/triggers/pkg/reconciler/scheduledtemplate/resources"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	knativetest "knative.dev/pkg/test"
)

func TestScheduledTemplateCreatesCronJobAndResources(t *testing.T) {
	c, namespace := setup(t)
	t.Cleanup(func() { tearDown(t, c, namespace) })
	knativetest.CleanupOnInterrupt(func() { tearDown(t, c, namespace) }, t.Logf)

	saName := "st-runner-sa"
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: saName, Namespace: namespace},
	}
	if _, err := c.KubeClient.CoreV1().ServiceAccounts(namespace).Create(context.Background(), sa, metav1.CreateOptions{}); err != nil {
		t.Fatalf("create ServiceAccount: %v", err)
	}

	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{Name: "st-runner-role", Namespace: namespace},
		Rules: []rbacv1.PolicyRule{{
			APIGroups: []string{"tekton.dev"},
			Resources: []string{"taskruns"},
			Verbs:     []string{"create"},
		}, {
			APIGroups: []string{"triggers.tekton.dev"},
			Resources: []string{"scheduledtemplates", "triggertemplates"},
			Verbs:     []string{"get", "list", "watch"},
		}},
	}
	if _, err := c.KubeClient.RbacV1().Roles(namespace).Create(context.Background(), role, metav1.CreateOptions{}); err != nil {
		t.Fatalf("create Role: %v", err)
	}
	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "st-runner-binding", Namespace: namespace},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      saName,
			Namespace: namespace,
		}},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     role.Name,
		},
	}
	if _, err := c.KubeClient.RbacV1().RoleBindings(namespace).Create(context.Background(), rb, metav1.CreateOptions{}); err != nil {
		t.Fatalf("create RoleBinding: %v", err)
	}

	stName := "st-e2e"
	st := &v1alpha1.ScheduledTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      stName,
			Namespace: namespace,
		},
		Spec: v1alpha1.ScheduledTemplateSpec{
			Schedule:           "0 0 1 1 *", // rare schedule; we fire a Job manually
			ServiceAccountName: saName,
			TriggerTemplateSpec: v1alpha1.TriggerTemplateSpec{
				ResourceTemplates: []v1alpha1.TriggerResourceTemplate{{
					RawExtension: RawExtension(t, pipelinev1.TaskRun{
						TypeMeta: metav1.TypeMeta{
							APIVersion: "tekton.dev/v1",
							Kind:       "TaskRun",
						},
						ObjectMeta: metav1.ObjectMeta{
							GenerateName: "st-e2e-tr-",
						},
						Spec: pipelinev1.TaskRunSpec{
							TaskSpec: &pipelinev1.TaskSpec{
								Steps: []pipelinev1.Step{{
									Name:    "echo",
									Image:   "busybox",
									Command: []string{"echo", "scheduled"},
								}},
							},
						},
					}),
				}},
			},
		},
	}
	if _, err := c.TriggersClient.TriggersV1alpha1().ScheduledTemplates(namespace).Create(context.Background(), st, metav1.CreateOptions{}); err != nil {
		t.Fatalf("create ScheduledTemplate: %v", err)
	}

	cronJobName := stresources.CronJobName(stName)
	if err := WaitFor(func() (bool, error) {
		got, err := c.TriggersClient.TriggersV1alpha1().ScheduledTemplates(namespace).Get(context.Background(), stName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		ready := got.Status.GetCondition(apis.ConditionReady)
		return ready != nil && ready.IsTrue() && got.Status.CronJobName == cronJobName, nil
	}); err != nil {
		t.Fatalf("ScheduledTemplate not ready: %v", err)
	}

	cj, err := c.KubeClient.BatchV1().CronJobs(namespace).Get(context.Background(), cronJobName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get CronJob: %v", err)
	}
	if cj.Spec.Schedule != st.Spec.Schedule {
		t.Errorf("CronJob schedule = %q, want %q", cj.Spec.Schedule, st.Spec.Schedule)
	}

	// Force a fire by creating a Job from the CronJob template (avoids waiting on cron).
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: cronJobName + "-",
			Namespace:    namespace,
			Labels:       cj.Spec.JobTemplate.Labels,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "batch/v1",
				Kind:       "CronJob",
				Name:       cj.Name,
				UID:        cj.UID,
			}},
		},
		Spec: cj.Spec.JobTemplate.Spec,
	}
	if _, err := c.KubeClient.BatchV1().Jobs(namespace).Create(context.Background(), job, metav1.CreateOptions{}); err != nil {
		t.Fatalf("create Job from CronJob template: %v", err)
	}

	stLabel := triggers.GroupName + triggers.ScheduledTemplateLabelKey
	if err := WaitFor(func() (bool, error) {
		trs, err := c.PipelineClient.TektonV1().TaskRuns(namespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", stLabel, stName),
		})
		if err != nil {
			return false, err
		}
		return len(trs.Items) > 0, nil
	}); err != nil {
		t.Fatalf("TaskRun from ScheduledTemplate not created: %v", err)
	}
}
