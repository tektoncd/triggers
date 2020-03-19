package builder

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

func TestParam(t *testing.T) {
	got := Param("foo", "bar")

	want := v1beta1.Param{
		Name: "foo",
		Value: v1beta1.ArrayOrString{
			Type:      v1beta1.ParamTypeString,
			StringVal: "bar",
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("-want/+got: %s", diff)
	}
}
