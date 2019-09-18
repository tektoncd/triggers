package builder

import (
	"github.com/google/go-cmp/cmp"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.normal, tt.builder); diff != "" {
				t.Errorf("EventListener(): -want +got: %s", diff)
			}
		})
	}
}
