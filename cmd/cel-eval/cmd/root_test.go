package cmd

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestEvalCEL(t *testing.T) {
	out := new(bytes.Buffer)
	if err := evalCEL(context.TODO(), out, "../testdata/expression.txt", "../testdata/http.txt"); err != nil {
		t.Fatalf("evalCEL: %v", err)
	}

	want := "true"
	if diff := cmp.Diff(want, out.String()); diff != "" {
		t.Errorf("-want +got: %s", diff)
	}
}
