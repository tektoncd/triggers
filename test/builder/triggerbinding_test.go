package builder

import (
	"github.com/google/go-cmp/cmp"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestTriggerBindingBuilder(t *testing.T) {
	tests := []struct {
		name    string
		normal  *v1alpha1.TriggerBinding
		builder *v1alpha1.TriggerBinding
	}{
		{
			name:    "Empty",
			normal:  &v1alpha1.TriggerBinding{},
			builder: TriggerBinding("", ""),
		},
		{
			name: "Name and Namespace",
			normal: &v1alpha1.TriggerBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "namespace",
				},
			},
			builder: TriggerBinding("name", "namespace"),
		},
		{
			name: "One InputParam",
			normal: &v1alpha1.TriggerBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "namespace",
				},
				Spec: v1alpha1.TriggerBindingSpec{
					InputParams: []pipelinev1.ParamSpec{
						pipelinev1.ParamSpec{
							Name:        "param1",
							Description: "description1",
							Default: &pipelinev1.ArrayOrString{
								StringVal: "value1",
								Type:      pipelinev1.ParamTypeString,
							},
						},
					},
				},
			},
			builder: TriggerBinding("name", "namespace",
				TriggerBindingSpec(
					TriggerBindingInputParam("param1", "description1", "value1"),
				),
			),
		},
		{
			name: "Two InputParams",
			normal: &v1alpha1.TriggerBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "namespace",
				},
				Spec: v1alpha1.TriggerBindingSpec{
					InputParams: []pipelinev1.ParamSpec{
						pipelinev1.ParamSpec{
							Name:        "param1",
							Description: "description1",
							Default: &pipelinev1.ArrayOrString{
								StringVal: "value1",
								Type:      pipelinev1.ParamTypeString,
							},
						},
						pipelinev1.ParamSpec{
							Name:        "param2",
							Description: "description2",
							Default: &pipelinev1.ArrayOrString{
								StringVal: "value2",
								Type:      pipelinev1.ParamTypeString,
							},
						},
					},
				},
			},
			builder: TriggerBinding("name", "namespace",
				TriggerBindingSpec(
					TriggerBindingInputParam("param1", "description1", "value1"),
					TriggerBindingInputParam("param2", "description2", "value2"),
				),
			),
		},
		{
			name: "One OutputParam",
			normal: &v1alpha1.TriggerBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "namespace",
				},
				Spec: v1alpha1.TriggerBindingSpec{
					OutputParams: []pipelinev1.Param{
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
			builder: TriggerBinding("name", "namespace",
				TriggerBindingSpec(
					TriggerBindingOutputParam("param1", "value1"),
				),
			),
		},
		{
			name: "Two OutputParams",
			normal: &v1alpha1.TriggerBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "namespace",
				},
				Spec: v1alpha1.TriggerBindingSpec{
					OutputParams: []pipelinev1.Param{
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
			builder: TriggerBinding("name", "namespace",
				TriggerBindingSpec(
					TriggerBindingOutputParam("param1", "value1"),
					TriggerBindingOutputParam("param2", "value2"),
				),
			),
		},
		{
			name: "Two InputParams and Two OutputParams",
			normal: &v1alpha1.TriggerBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "namespace",
				},
				Spec: v1alpha1.TriggerBindingSpec{
					InputParams: []pipelinev1.ParamSpec{
						pipelinev1.ParamSpec{
							Name:        "param1",
							Description: "description1",
							Default: &pipelinev1.ArrayOrString{
								StringVal: "value1",
								Type:      pipelinev1.ParamTypeString,
							},
						},
						pipelinev1.ParamSpec{
							Name:        "param2",
							Description: "description2",
							Default: &pipelinev1.ArrayOrString{
								StringVal: "value2",
								Type:      pipelinev1.ParamTypeString,
							},
						},
					},
					OutputParams: []pipelinev1.Param{
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
			builder: TriggerBinding("name", "namespace",
				TriggerBindingSpec(
					TriggerBindingInputParam("param1", "description1", "value1"),
					TriggerBindingInputParam("param2", "description2", "value2"),
					TriggerBindingOutputParam("param1", "value1"),
					TriggerBindingOutputParam("param2", "value2"),
				),
			),
		},
		{
			name: "Extra Meta",
			normal: &v1alpha1.TriggerBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "namespace",
					Labels: map[string]string{
						"key": "value",
					},
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "TriggerBinding",
					APIVersion: "v1alpha1",
				},
				Spec: v1alpha1.TriggerBindingSpec{},
			},
			builder: TriggerBinding("name", "namespace",
				TriggerBindingMeta(
					TypeMeta("TriggerBinding", "v1alpha1"),
					Label("key", "value"),
				),
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.normal, tt.builder); diff != "" {
				t.Errorf("TriggerBinding(): -want +got: %s", diff)
			}
		})
	}
}
