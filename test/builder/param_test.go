package builder

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
)

func TestParam(t *testing.T) {
	got := Param("foo", "bar")

	want := v1alpha1.Param{
		Name:  "foo",
		Value: "bar",
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("-want/+got: %s", diff)
	}
}
