/*
Copyright 2025 The Knative Authors

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

package resource

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDefault(t *testing.T) {
	r := Default("myservice")

	if r == "" {
		t.Fatalf("expected service name to not be empty")
	}

	if diff := cmp.Diff("myservice", r); diff != "" {
		t.Error("unexpected service name diff (-want +got):", diff)
	}
}

func TestDefaultWithEnvOverride(t *testing.T) {
	t.Setenv(otelServiceNameKey, "another")
	r := Default("myservice")

	if r == "" {
		t.Fatalf("expected service name to not be empty")
	}

	if diff := cmp.Diff("another", r); diff != "" {
		t.Error("unexpected service name diff (-want +got):", diff)
	}
}
