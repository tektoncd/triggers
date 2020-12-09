package test

import (
	"crypto/hmac"
	"crypto/sha1" //nolint
	"encoding/hex"
	"fmt"
	"testing"
)

// HMACHeader generates a X-Hub-Signature header given a secret token and the request body
// See https://developer.github.com/webhooks/securing/#validating-payloads-from-github
func HMACHeader(t testing.TB, secret string, body []byte) string {
	t.Helper()
	h := hmac.New(sha1.New, []byte(secret))
	_, err := h.Write(body)
	if err != nil {
		t.Fatalf("HMACHeader fail: %s", err)
	}
	return fmt.Sprintf("sha1=%s", hex.EncodeToString(h.Sum(nil)))
}
