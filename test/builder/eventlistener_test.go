package builder

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
)

func TestEventListenerBuilder(t *testing.T) {
	tests := []struct {
		name    string
		normal  *v1alpha1.EventListener
		builder *v1alpha1.EventListener
	}{
		{
			name:    "Empty",
			normal:  &v1alpha1.EventListener{},
			builder: EventListener("", ""),
		},
		{
			name: "Name and Namespace",
			normal: &v1alpha1.EventListener{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "namespace",
				},
			},
			builder: EventListener("name", "namespace"),
		},
		{
			name: "No Triggers",
			normal: &v1alpha1.EventListener{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "namespace",
				},
				Spec: v1alpha1.EventListenerSpec{
					ServiceAccountName: "serviceAccount",
				},
			},
			builder: EventListener("name", "namespace",
				EventListenerSpec(
					EventListenerServiceAccount("serviceAccount"),
				),
			),
		},
		{
			name: "Status configuration",
			normal: &v1alpha1.EventListener{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "namespace",
				},
				Status: v1alpha1.EventListenerStatus{
					Configuration: v1alpha1.EventListenerConfig{
						GeneratedResourceName: "generatedName",
						Hostname:              "hostname",
					},
				},
			},
			builder: EventListener("name", "namespace",
				EventListenerStatus(
					EventListenerConfig("generatedName", "hostname"),
				),
			),
		},
		{
			name: "One Condition",
			normal: &v1alpha1.EventListener{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "namespace",
				},
				Status: v1alpha1.EventListenerStatus{
					Status: duckv1beta1.Status{
						Conditions: []apis.Condition{
							apis.Condition{
								Type:    v1alpha1.ServiceExists,
								Status:  corev1.ConditionTrue,
								Message: "Service exists",
							},
						},
					},
				},
			},
			builder: EventListener("name", "namespace",
				EventListenerStatus(
					EventListenerCondition(
						v1alpha1.ServiceExists,
						corev1.ConditionTrue,
						"Service exists", "",
					),
				),
			),
		},
		{
			name: "Two Condition",
			normal: &v1alpha1.EventListener{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "namespace",
				},
				Status: v1alpha1.EventListenerStatus{
					Status: duckv1beta1.Status{
						Conditions: []apis.Condition{
							apis.Condition{
								Type:    v1alpha1.DeploymentExists,
								Status:  corev1.ConditionTrue,
								Message: "Deployment exists",
							},
							apis.Condition{
								Type:    v1alpha1.ServiceExists,
								Status:  corev1.ConditionTrue,
								Message: "Service exists",
							}},
					},
				},
			},
			builder: EventListener("name", "namespace",
				EventListenerStatus(
					EventListenerCondition(
						v1alpha1.ServiceExists,
						corev1.ConditionTrue,
						"Service exists", "",
					),
					EventListenerCondition(
						v1alpha1.DeploymentExists,
						corev1.ConditionTrue,
						"Deployment exists", "",
					),
				),
			),
		},
		{
			name: "One Trigger",
			normal: &v1alpha1.EventListener{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "namespace",
				},
				Spec: v1alpha1.EventListenerSpec{
					ServiceAccountName: "serviceAccount",
					Triggers: []v1alpha1.EventListenerTrigger{
						v1alpha1.EventListenerTrigger{
							Binding: v1alpha1.EventListenerBinding{
								Name:       "tb1",
								APIVersion: "v1alpha1",
							},
							Template: v1alpha1.EventListenerTemplate{
								Name:       "tt1",
								APIVersion: "v1alpha1",
							},
						},
					},
				},
			},
			builder: EventListener("name", "namespace",
				EventListenerSpec(
					EventListenerServiceAccount("serviceAccount"),
					EventListenerTrigger("tb1", "tt1", "v1alpha1"),
				),
			),
		},
		{
			name: "One Trigger with One Param",
			normal: &v1alpha1.EventListener{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "namespace",
				},
				Spec: v1alpha1.EventListenerSpec{
					ServiceAccountName: "serviceAccount",
					Triggers: []v1alpha1.EventListenerTrigger{
						v1alpha1.EventListenerTrigger{
							Binding: v1alpha1.EventListenerBinding{
								Name:       "tb1",
								APIVersion: "v1alpha1",
							},
							Template: v1alpha1.EventListenerTemplate{
								Name:       "tt1",
								APIVersion: "v1alpha1",
							},
							Params: []pipelinev1.Param{
								pipelinev1.Param{
									Name: "param1",
									Value: pipelinev1.ArrayOrString{
										StringVal: "value1",
										Type:      pipelinev1.ParamTypeString,
									},
								},
							},
						},
					},
				},
			},
			builder: EventListener("name", "namespace",
				EventListenerSpec(
					EventListenerServiceAccount("serviceAccount"),
					EventListenerTrigger("tb1", "tt1", "v1alpha1",
						EventListenerTriggerParam("param1", "value1"),
					),
				),
			),
		},
		{
			name: "One Trigger with Two Params",
			normal: &v1alpha1.EventListener{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "namespace",
				},
				Spec: v1alpha1.EventListenerSpec{
					ServiceAccountName: "serviceAccount",
					Triggers: []v1alpha1.EventListenerTrigger{
						v1alpha1.EventListenerTrigger{
							Binding: v1alpha1.EventListenerBinding{
								Name:       "tb1",
								APIVersion: "v1alpha1",
							},
							Template: v1alpha1.EventListenerTemplate{
								Name:       "tt1",
								APIVersion: "v1alpha1",
							},
							Params: []pipelinev1.Param{
								pipelinev1.Param{
									Name: "param1",
									Value: pipelinev1.ArrayOrString{
										StringVal: "value1",
										Type:      pipelinev1.ParamTypeString,
									},
								},
								pipelinev1.Param{
									Name: "param2",
									Value: pipelinev1.ArrayOrString{
										StringVal: "value2",
										Type:      pipelinev1.ParamTypeString,
									},
								},
							},
						},
					},
				},
			},
			builder: EventListener("name", "namespace",
				EventListenerSpec(
					EventListenerServiceAccount("serviceAccount"),
					EventListenerTrigger("tb1", "tt1", "v1alpha1",
						EventListenerTriggerParam("param1", "value1"),
						EventListenerTriggerParam("param2", "value2"),
					),
				),
			),
		},
		{
			name: "Two Trigger with extra Meta",
			normal: &v1alpha1.EventListener{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "namespace",
					Labels: map[string]string{
						"key": "value",
					},
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "EventListener",
					APIVersion: "v1alpha1",
				},
				Spec: v1alpha1.EventListenerSpec{
					ServiceAccountName: "serviceAccount",
					Triggers: []v1alpha1.EventListenerTrigger{
						v1alpha1.EventListenerTrigger{
							Binding: v1alpha1.EventListenerBinding{
								Name:       "tb1",
								APIVersion: "v1alpha1",
							},
							Template: v1alpha1.EventListenerTemplate{
								Name:       "tt1",
								APIVersion: "v1alpha1",
							},
						},
						v1alpha1.EventListenerTrigger{
							Binding: v1alpha1.EventListenerBinding{
								Name:       "tb2",
								APIVersion: "v1alpha1",
							},
							Template: v1alpha1.EventListenerTemplate{
								Name:       "tt2",
								APIVersion: "v1alpha1",
							},
						},
					},
				},
			},
			builder: EventListener("name", "namespace",
				EventListenerMeta(
					TypeMeta("EventListener", "v1alpha1"),
					Label("key", "value"),
				),
				EventListenerSpec(
					EventListenerServiceAccount("serviceAccount"),
					EventListenerTrigger("tb1", "tt1", "v1alpha1"),
					EventListenerTrigger("tb2", "tt2", "v1alpha1"),
				),
			),
		},
		{
			name: "One Trigger with Validation & Name",
			normal: &v1alpha1.EventListener{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "namespace",
				},
				Spec: v1alpha1.EventListenerSpec{
					ServiceAccountName: "serviceAccount",
					Triggers: []v1alpha1.EventListenerTrigger{
						v1alpha1.EventListenerTrigger{
							Name: "foo-trig",
							TriggerValidate: &v1alpha1.TriggerValidate{
								TaskRef: pipelinev1.TaskRef{
									Name:       "bar",
									Kind:       pipelinev1.NamespacedTaskKind,
									APIVersion: "v1alpha1",
								},
								ServiceAccountName: "foo",
								Params: []pipelinev1.Param{
									{
										Name: "Secret",
										Value: pipelinev1.ArrayOrString{
											Type:      pipelinev1.ParamTypeString,
											StringVal: "github-secret",
										},
									},
									{
										Name: "Secret-Key",
										Value: pipelinev1.ArrayOrString{
											Type:      pipelinev1.ParamTypeString,
											StringVal: "github-secret-key",
										},
									},
								},
							},
							Binding: v1alpha1.EventListenerBinding{
								Name:       "tb1",
								APIVersion: "v1alpha1",
							},
							Template: v1alpha1.EventListenerTemplate{
								Name:       "tt1",
								APIVersion: "v1alpha1",
							},
							Params: []pipelinev1.Param{
								pipelinev1.Param{
									Name: "param1",
									Value: pipelinev1.ArrayOrString{
										StringVal: "value1",
										Type:      pipelinev1.ParamTypeString,
									},
								},
							},
						},
					},
				},
			},
			builder: EventListener("name", "namespace",
				EventListenerSpec(
					EventListenerServiceAccount("serviceAccount"),
					EventListenerTrigger("tb1", "tt1", "v1alpha1",
						EventListenerTriggerName("foo-trig"),
						EventListenerTriggerParam("param1", "value1"),
						EventListenerTriggerValidate(
							EventListenerTriggerValidateTaskRef("bar", "v1alpha1", pipelinev1.NamespacedTaskKind),
							EventListenerTriggerValidateServiceAccount("foo"),
							EventListenerTriggerValidateParam("Secret", "github-secret"),
							EventListenerTriggerValidateParam("Secret-Key", "github-secret-key"),
						),
					),
				),
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !equality.Semantic.DeepEqual(tt.normal, tt.builder) {
				t.Error("EventListener() builder equality mismatch. Ignore semantic time mismatch")
				diff := cmp.Diff(tt.normal, tt.builder)
				t.Errorf("Diff request body: -want +got: %s", diff)
			}
		})
	}
}
