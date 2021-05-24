/*
Copyright 2019 The Tekton Authors

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

package v1alpha1

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/apis/duck/v1beta1"
)

func TestSetGetCondition(t *testing.T) {
	tests := []struct {
		name               string
		conditions         []*apis.Condition
		expectedConditions int
	}{{
		name:               "No conditions",
		conditions:         []*apis.Condition{},
		expectedConditions: 0,
	}, {
		name: "One condition",
		conditions: []*apis.Condition{{
			Type:    "Some Type",
			Status:  corev1.ConditionTrue,
			Message: "Message",
		}},
		expectedConditions: 1,
	}, {
		name: "Two conditions",
		conditions: []*apis.Condition{{
			Type:    "Some Type1",
			Status:  corev1.ConditionTrue,
			Message: "Message1",
		}, {
			Type:    "Some Type2",
			Status:  corev1.ConditionFalse,
			Message: "Message2",
		}},
		expectedConditions: 2,
	}, {
		name: "Two conditions repeated",
		conditions: []*apis.Condition{{
			Type:    "Some Type1",
			Status:  corev1.ConditionTrue,
			Message: "Message1",
		}, {
			Type:    "Some Type1",
			Status:  corev1.ConditionFalse,
			Message: "Message2",
		}, {
			Type:    "Some Type2",
			Status:  corev1.ConditionTrue,
			Message: "Message1",
		}, {
			Type:    "Some Type2",
			Status:  corev1.ConditionFalse,
			Message: "Message2",
		}},
		expectedConditions: 2,
	},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			els := &EventListenerStatus{}
			for _, cond := range tests[i].conditions {
				els.SetCondition(cond)
				getCondition := els.GetCondition(cond.Type)
				if !equality.Semantic.DeepEqual(cond, getCondition) {
					t.Errorf("Get Condition %v does not equal expected %v\n", getCondition, cond)
				}
			}
			if len(els.Conditions) != tests[i].expectedConditions {
				t.Errorf("EventListener has %d conditions and expects %d\n", len(els.Conditions), tests[i].expectedConditions)
			}
		})
	}
}

func TestInitializeConditions(t *testing.T) {
	var conditionTypes = []apis.ConditionType{
		ServiceExists,
		DeploymentExists,
		apis.ConditionReady,
	}
	els := &EventListenerStatus{}
	els.InitializeConditions()
	if len(els.Conditions) != len(conditionTypes) {
		t.Error("InitializeConditions() did not initialize all conditions in EventlistenerStatus")
	}
	for _, condType := range conditionTypes {
		if els.GetCondition(condType).Status != corev1.ConditionFalse {
			t.Errorf("Condition not set to %s\n", corev1.ConditionFalse)
		}
	}
}

func TestSetExistsCondition(t *testing.T) {
	condType := apis.ConditionType("Cond")
	tests := []struct {
		name              string
		conditionType     apis.ConditionType
		err               error
		expectedCondition *apis.Condition
	}{{
		name:          "Condition with error",
		conditionType: condType,
		err:           fmt.Errorf("something bad"),
		expectedCondition: &apis.Condition{
			Type:    condType,
			Status:  corev1.ConditionFalse,
			Message: "something bad",
		},
	}, {
		name:          "Condition without error",
		conditionType: condType,
		err:           nil,
		expectedCondition: &apis.Condition{
			Type:    condType,
			Status:  corev1.ConditionTrue,
			Message: fmt.Sprintf("%s exists", condType),
		},
	},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			els := EventListenerStatus{}
			els.SetExistsCondition(tests[i].conditionType, tests[i].err)
			actualCond := els.GetCondition(condType)
			if !equality.Semantic.DeepEqual(tests[i].expectedCondition, actualCond) {
				t.Errorf("Get Condition %v does not equal expected %v\n", actualCond, tests[i].expectedCondition)
			}
		})
	}
}

func TestSetDeploymentConditions(t *testing.T) {
	tests := []struct {
		name                 string
		deploymentConditions []appsv1.DeploymentCondition
		initialStatus        *EventListenerStatus
		expectedStatus       *EventListenerStatus
	}{{
		name:                 "No Deployment Conditions",
		deploymentConditions: []appsv1.DeploymentCondition{},
		initialStatus:        &EventListenerStatus{},
		expectedStatus:       &EventListenerStatus{},
	}, {
		name: "One Deployment Condition",
		deploymentConditions: []appsv1.DeploymentCondition{{
			Type:    appsv1.DeploymentAvailable,
			Status:  corev1.ConditionTrue,
			Reason:  "Reason",
			Message: "Message",
		}},
		initialStatus: &EventListenerStatus{},
		expectedStatus: &EventListenerStatus{
			Status: duckv1.Status{
				Conditions: duckv1.Conditions{
					apis.Condition{
						Type:    apis.ConditionType(appsv1.DeploymentAvailable),
						Status:  corev1.ConditionTrue,
						Reason:  "Reason",
						Message: "Message",
					},
				},
			},
		},
	}, {
		name: "Two Deployment Conditions",
		deploymentConditions: []appsv1.DeploymentCondition{{
			Type:    appsv1.DeploymentAvailable,
			Status:  corev1.ConditionTrue,
			Reason:  "Reason",
			Message: "Message",
		}, {
			Type:    appsv1.DeploymentProgressing,
			Status:  corev1.ConditionTrue,
			Reason:  "Reason",
			Message: "Message",
		}},
		initialStatus: &EventListenerStatus{},
		expectedStatus: &EventListenerStatus{
			Status: duckv1.Status{
				Conditions: duckv1.Conditions{
					apis.Condition{
						Type:    apis.ConditionType(appsv1.DeploymentAvailable),
						Status:  corev1.ConditionTrue,
						Reason:  "Reason",
						Message: "Message",
					},
					apis.Condition{
						Type:    apis.ConditionType(appsv1.DeploymentProgressing),
						Status:  corev1.ConditionTrue,
						Reason:  "Reason",
						Message: "Message",
					},
				},
			},
		},
	}, {
		name: "Update Replica Condition",
		deploymentConditions: []appsv1.DeploymentCondition{{
			Type:    appsv1.DeploymentAvailable,
			Status:  corev1.ConditionTrue,
			Reason:  "Reason",
			Message: "Message",
		}, {
			Type:    appsv1.DeploymentProgressing,
			Status:  corev1.ConditionTrue,
			Reason:  "Reason",
			Message: "Message",
		}},
		initialStatus: &EventListenerStatus{
			Status: duckv1.Status{
				Conditions: duckv1.Conditions{
					apis.Condition{
						Type:    apis.ConditionType(appsv1.DeploymentReplicaFailure),
						Status:  corev1.ConditionTrue,
						Reason:  "Reason",
						Message: "Message",
					},
				},
			},
		},
		expectedStatus: &EventListenerStatus{
			Status: duckv1.Status{
				Conditions: duckv1.Conditions{
					apis.Condition{
						Type:    apis.ConditionType(appsv1.DeploymentAvailable),
						Status:  corev1.ConditionTrue,
						Reason:  "Reason",
						Message: "Message",
					},
					apis.Condition{
						Type:    apis.ConditionType(appsv1.DeploymentProgressing),
						Status:  corev1.ConditionTrue,
						Reason:  "Reason",
						Message: "Message",
					},
				},
			},
		},
	},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			tests[i].initialStatus.SetDeploymentConditions(tests[i].deploymentConditions)
			if !equality.Semantic.DeepEqual(tests[i].expectedStatus, tests[i].initialStatus) {
				t.Error("SetDeploymentConditions() equality mismatch. Ignore semantic time mismatch")
				diff := cmp.Diff(tests[i].expectedStatus, tests[i].initialStatus)
				t.Errorf("Diff request body (-want +got) = %s", diff)
			}
		})
	}
}

func TestSetConditionsForDynamicObjects(t *testing.T) {
	var status EventListenerStatus
	status.SetConditionsForDynamicObjects(v1beta1.Conditions{{
		Type:    apis.ConditionReady,
		Status:  corev1.ConditionTrue,
		Reason:  "Reason",
		Message: "Message",
	}})
	expected := EventListenerStatus{
		Status: duckv1.Status{
			Conditions: []apis.Condition{{
				Type:    apis.ConditionReady,
				Status:  corev1.ConditionTrue,
				Reason:  "Reason",
				Message: "Message",
			}},
		},
	}
	if diff := cmp.Diff(expected, status, cmpopts.IgnoreTypes(
		apis.Condition{}.LastTransitionTime.Inner.Time)); diff != "" {
		t.Fatalf("SetConditionsForDynamicObjects() error. Diff (-want/+got) : %s", diff)
	}
}

func TestEventListenerStatus_SetReadyCondition(t *testing.T) {
	for _, tc := range []struct {
		name    string
		initial *EventListenerStatus
		want    *EventListenerStatus
	}{{
		name: "set Ready to true when all dependent conditions are true",
		initial: &EventListenerStatus{
			Status: duckv1.Status{
				Conditions: []apis.Condition{{
					Type:    DeploymentExists,
					Status:  corev1.ConditionTrue,
					Message: "Deployment Exists",
				}, {
					Type:    ServiceExists,
					Status:  corev1.ConditionTrue,
					Message: "Service Exists",
				}, {
					Type:    apis.ConditionType(appsv1.DeploymentProgressing),
					Status:  corev1.ConditionTrue,
					Message: `ReplicaSet "el-example-655bf689f" has successfully progressed.`,
				}, {
					Type:    apis.ConditionType(appsv1.DeploymentAvailable),
					Status:  corev1.ConditionTrue,
					Message: "Deployment has minimum availability.",
				}},
			},
		},
		want: &EventListenerStatus{
			Status: duckv1.Status{
				Conditions: []apis.Condition{{
					Type:    DeploymentExists,
					Status:  corev1.ConditionTrue,
					Message: "Deployment Exists",
				}, {
					Type:    ServiceExists,
					Status:  corev1.ConditionTrue,
					Message: "Service Exists",
				}, {
					Type:    apis.ConditionType(appsv1.DeploymentProgressing),
					Status:  corev1.ConditionTrue,
					Message: `ReplicaSet "el-example-655bf689f" has successfully progressed.`,
				}, {
					Type:    apis.ConditionType(appsv1.DeploymentAvailable),
					Status:  corev1.ConditionTrue,
					Message: "Deployment has minimum availability.",
				}, {
					Type:    apis.ConditionReady,
					Status:  corev1.ConditionTrue,
					Message: "EventListener is ready",
				}},
			},
		},
	}, {
		name: "set Ready to false when at least one dependent condition is false",
		initial: &EventListenerStatus{
			Status: duckv1.Status{
				Conditions: []apis.Condition{{
					Type:    DeploymentExists,
					Status:  corev1.ConditionTrue,
					Message: "Deployment Exists",
				}, {
					Type:    ServiceExists,
					Status:  corev1.ConditionFalse,
					Message: "Service does not exist",
				}, {
					Type:    apis.ConditionType(appsv1.DeploymentProgressing),
					Status:  corev1.ConditionTrue,
					Message: `ReplicaSet "el-example-655bf689f" has successfully progressed.`,
				}, {
					Type:    apis.ConditionType(appsv1.DeploymentAvailable),
					Status:  corev1.ConditionTrue,
					Message: "Deployment has minimum availability.",
				}},
			},
		},
		want: &EventListenerStatus{
			Status: duckv1.Status{
				Conditions: []apis.Condition{{
					Type:    DeploymentExists,
					Status:  corev1.ConditionTrue,
					Message: "Deployment Exists",
				}, {
					Type:    ServiceExists,
					Status:  corev1.ConditionFalse,
					Message: "Service does not exist",
				}, {
					Type:    apis.ConditionType(appsv1.DeploymentProgressing),
					Status:  corev1.ConditionTrue,
					Message: `ReplicaSet "el-example-655bf689f" has successfully progressed.`,
				}, {
					Type:    apis.ConditionType(appsv1.DeploymentAvailable),
					Status:  corev1.ConditionTrue,
					Message: "Deployment has minimum availability.",
				}, {
					Type:    apis.ConditionReady,
					Status:  corev1.ConditionFalse,
					Message: "Condition Service has status: False with message: Service does not exist",
				}},
			},
		},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			tc.initial.SetReadyCondition()
			if diff := cmp.Diff(tc.want, tc.initial, cmpopts.IgnoreTypes(
				apis.Condition{}.LastTransitionTime.Inner.Time), cmpopts.SortSlices(func(x, y apis.Condition) bool {
				return x.Type < y.Type
			})); diff != "" {
				t.Fatalf("SetReadyCondition() mismatch. -want/+got: %s", diff)
			}
		})
	}
}
