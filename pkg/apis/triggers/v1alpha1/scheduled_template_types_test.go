/*
Copyright 2024 The Tekton Authors

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

package v1alpha1_test

import (
	"errors"
	"testing"

	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/apis"
)

func TestScheduledTemplateGetGroupVersionKind(t *testing.T) {
	st := &v1alpha1.ScheduledTemplate{}
	want := v1alpha1.SchemeGroupVersion.WithKind("ScheduledTemplate")
	if got := st.GetGroupVersionKind(); got != want {
		t.Errorf("GetGroupVersionKind() = %v, want %v", got, want)
	}
}

func TestScheduledTemplateStatusInitializeConditions(t *testing.T) {
	sts := &v1alpha1.ScheduledTemplateStatus{}
	sts.InitializeConditions()
	for _, cond := range []apis.ConditionType{v1alpha1.CronJobExists, apis.ConditionReady} {
		c := sts.GetCondition(cond)
		if c == nil {
			t.Errorf("expected condition %s to be initialized", cond)
			continue
		}
		if c.Status != corev1.ConditionFalse {
			t.Errorf("condition %s status = %s, want False", cond, c.Status)
		}
	}
}

func TestScheduledTemplateStatusSetExistsCondition(t *testing.T) {
	sts := &v1alpha1.ScheduledTemplateStatus{}
	sts.SetExistsCondition(v1alpha1.CronJobExists, nil)
	c := sts.GetCondition(v1alpha1.CronJobExists)
	if c == nil || c.Status != corev1.ConditionTrue {
		t.Fatalf("expected CronJobExists True, got %#v", c)
	}

	sts.SetExistsCondition(v1alpha1.CronJobExists, errors.New("missing"))
	c = sts.GetCondition(v1alpha1.CronJobExists)
	if c == nil || c.Status != corev1.ConditionFalse {
		t.Fatalf("expected CronJobExists False, got %#v", c)
	}
}

func TestScheduledTemplateStatusSetReadyCondition(t *testing.T) {
	sts := &v1alpha1.ScheduledTemplateStatus{}
	sts.SetExistsCondition(v1alpha1.CronJobExists, errors.New("missing"))
	sts.SetReadyCondition()
	ready := sts.GetCondition(apis.ConditionReady)
	if ready == nil || ready.Status != corev1.ConditionFalse {
		t.Fatalf("expected Ready False when CronJob missing, got %#v", ready)
	}

	sts.SetExistsCondition(v1alpha1.CronJobExists, nil)
	sts.SetReadyCondition()
	ready = sts.GetCondition(apis.ConditionReady)
	if ready == nil || ready.Status != corev1.ConditionTrue {
		t.Fatalf("expected Ready True when CronJob exists, got %#v", ready)
	}
}
