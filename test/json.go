package test

import (
	"encoding/json"
	"testing"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// ToV1JSON is a wrapper around json.Marshal to easily convert to the Kubernetes apiextensionsv1.JSON type
func ToV1JSON(t testing.TB, v interface{}) apiextensionsv1.JSON {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal() failed: %s", err)
	}
	return apiextensionsv1.JSON{
		Raw: b,
	}
}
