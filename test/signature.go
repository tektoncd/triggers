package test

import (
	"crypto/hmac"
	"crypto/sha1" //nolint
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"testing"
)

// HMACHeader generates a X-Hub-Signature header given a secret token and the request body
// See https://developer.github.com/webhooks/securing/#validating-payloads-from-github
// algorithm must be one of (sha1, sha256)
func HMACHeader(t testing.TB, secret string, body []byte, algorithm string) string {
	t.Helper()
	var h hash.Hash
	if algorithm == "sha1" {
		h = hmac.New(sha1.New, []byte(secret))
	} else if algorithm == "sha256" {
		h = hmac.New(sha256.New, []byte(secret))
	}
	_, err := h.Write(body)
	if err != nil {
		t.Fatalf("HMACHeader fail: %s", err)
	}
	return fmt.Sprintf("%s=%s", algorithm, hex.EncodeToString(h.Sum(nil)))
}
