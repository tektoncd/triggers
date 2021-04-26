package v1beta1_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
)

func TestParseTriggerID(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  []string
	}{{
		input: "namespaces/default/triggers/my-trigger",
		want:  []string{"default", "my-trigger"},
	}, {
		input: "",
		want:  []string{"", ""},
	}} {
		t.Run(tc.input, func(t *testing.T) {
			name, ns := v1beta1.ParseTriggerID(tc.input)
			if diff := cmp.Diff(tc.want, []string{name, ns}); diff != "" {
				t.Errorf("Errror ParseTriggerID (-want/+got): %s", diff)
			}
		})
	}
}
