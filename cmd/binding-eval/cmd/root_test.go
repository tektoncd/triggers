/*
Copyright 2020 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestEvalBindingWithCorrectContentLength(t *testing.T) {
	// Test with HTTP file that has correct content length header - expect to pass
	out := new(bytes.Buffer)
	if err := evalBinding(out, "../testdata/triggerbinding.yaml", "../testdata/http.txt"); err != nil {
		t.Fatalf("evalBinding with correct Content-Length should pass: %v", err)
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

func TestEvalBindingWithWrongContentLength(t *testing.T) {
	// Test with HTTP file that has wrong content length header - expect to fail
	out := new(bytes.Buffer)
	err := evalBinding(out, "../testdata/triggerbinding.yaml", "../testdata/http_wrong_content_length.txt")
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
	if err := evalBinding(out, "../testdata/triggerbinding.yaml", "../testdata/http_no_content_length.txt"); err != nil {
		t.Fatalf("evalBinding with no Content-Length should pass: %v", err)
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
