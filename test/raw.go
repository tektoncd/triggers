package test

import (
	"encoding/json"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
)

// RawExtenstion is a test helper to generate RawExtension objects for tests
func RawExtension(t testing.TB, a interface{}) runtime.RawExtension {
	t.Helper()
	b, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	return runtime.RawExtension{Raw: b}
}
