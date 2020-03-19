package builder

import "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"

func Param(name, value string) v1beta1.Param {
	return v1beta1.Param{
		Name: name,
		Value: v1beta1.ArrayOrString{
			Type:      v1beta1.ParamTypeString,
			StringVal: value,
		},
	}
}
