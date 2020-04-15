package builder

import "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"

func Param(name, value string) v1alpha1.Param {
	return v1alpha1.Param{
		Name:  name,
		Value: value,
	}
}
