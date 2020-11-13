package test_test

import (
	"encoding/json"
	"testing"

	"github.com/tektoncd/triggers/test"
)

func TestHMACHeader(t *testing.T) {
	got, err := test.HMACHeader("secret", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("HMACHeader() error: %v", err)
	}
	// Generated from https://play.golang.org/p/OlkBawQQPiJ
	want := "sha1=5d61605c3feea9799210ddcb71307d4ba264225f"
	if want != got {
		t.Fatalf("HMACHeader(). Want: %s Got: %s", want, got)
	}
}
