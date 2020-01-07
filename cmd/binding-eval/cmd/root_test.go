package cmd

import (
	"bytes"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestEvalBinding(t *testing.T) {
	out := new(bytes.Buffer)
	if err := evalBinding(out, "../testdata/triggerbinding.yaml", "../testdata/http.txt"); err != nil {
		t.Fatalf("evalBinding: %v", err)
	}

	want := `[
  {
    "name": "bar",
    "value": "tacocat"
  },
  {
    "name": "foo",
    "value": "body"
  }
]
`
	if diff := cmp.Diff(want, out.String()); diff != "" {
		t.Errorf("-want +got: %s", diff)
	}
}
