package cmd

import (
	"bytes"
	"context"
	"strings"
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
	// Test with HTTP file that has wrong content length header - expect to fail
	out := new(bytes.Buffer)
	err := evalCEL(context.TODO(), out, "../testdata/expression.txt", "../testdata/http_wrong_content_length.txt")
	if err == nil {
		t.Fatal("evalBinding with wrong Content-Length should fail, but it passed")
	}

	// Verify that the error is related to parsing the body
	expectedErrorSubstring := "unexpected end of JSON input"
	if !strings.Contains(err.Error(), expectedErrorSubstring) {
		t.Errorf("Expected error to contain %q, got: %v", expectedErrorSubstring, err)
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
