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

func TestEvalBindingWithWrongContentLength(t *testing.T) {
	// Test with HTTP file that has a wrong (stale) content length header -
	// the actual length should be recomputed from the body, so this should pass.
	out := new(bytes.Buffer)
	if err := evalCEL(context.TODO(), out, "../testdata/expression.txt", "../testdata/http_wrong_content_length.txt"); err != nil {
		t.Fatalf("evalBinding with wrong Content-Length should pass: %v", err)
	}

	want := "true"
	if diff := cmp.Diff(want, out.String()); diff != "" {
		t.Errorf("-want +got: %s", diff)
	}
}

func TestEvalBindingWithNoContentLength(t *testing.T) {
	// Test with HTTP file that has no content length header - expect to pass
	out := new(bytes.Buffer)
	if err := evalCEL(context.TODO(), out, "../testdata/expression.txt", "../testdata/http_no_content_length.txt"); err != nil {
		t.Fatalf("evalBinding with no Content-Length should pass: %v", err)
	}

	want := "true"
	if diff := cmp.Diff(want, out.String()); diff != "" {
		t.Errorf("-want +got: %s", diff)
	}
}
