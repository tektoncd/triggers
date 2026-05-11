/*
Copyright 2026 The Tekton Authors

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

// Package tlsconfig provides TLS configuration parsing and management for the core interceptors server.
// It supports loading configuration from environment variables injected by the Tekton operator,
// and converts them to Go's tls.Config for use with the HTTPS server.
package tlsconfig

import (
	"crypto/tls"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

// Config holds TLS configuration that can be loaded from environment variables
type Config struct {
	MinTLSVersion    string
	CipherSuites     string
	CurvePreferences string
}

// New creates a Config from explicit values
func New(minVersion, cipherSuites, curvePreferences string) *Config {
	return &Config{
		MinTLSVersion:    minVersion,
		CipherSuites:     cipherSuites,
		CurvePreferences: curvePreferences,
	}
}

// LoadFromEnv loads TLS configuration from environment variables.
// This allows the configuration to be injected by the Tekton operator on OpenShift.
func LoadFromEnv() *Config {
	return New(
		os.Getenv("TLS_MIN_VERSION"),
		os.Getenv("TLS_CIPHER_SUITES"),
		os.Getenv("TLS_CURVE_PREFERENCES"),
	)
}

// ToTLSConfig converts the configuration to Go's tls.Config
func (c *Config) ToTLSConfig() (*tls.Config, error) {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	if c.MinTLSVersion != "" {
		minVersion, err := parseTLSVersion(c.MinTLSVersion)
		if err != nil {
			return nil, fmt.Errorf("invalid TLS_MIN_VERSION: %w", err)
		}
		tlsConfig.MinVersion = minVersion
	}

	if c.CipherSuites != "" {
		cipherSuites, err := parseCipherSuites(c.CipherSuites)
		if err != nil {
			return nil, fmt.Errorf("invalid TLS_CIPHER_SUITES: %w", err)
		}
		tlsConfig.CipherSuites = cipherSuites
	}

	if c.CurvePreferences != "" {
		curvePreferences, err := parseCurvePreferences(c.CurvePreferences)
		if err != nil {
			return nil, fmt.Errorf("invalid TLS_CURVE_PREFERENCES: %w", err)
		}
		tlsConfig.CurvePreferences = curvePreferences
	}

	return tlsConfig, nil
}

// parseTLSVersion converts a version string to tls.Version constant.
// Only TLS 1.2 and 1.3 are accepted; TLS 1.0 and 1.1 are rejected as insecure.
func parseTLSVersion(version string) (uint16, error) {
	if num, err := parseUintFlexible(version); err == nil {
		if num < tls.VersionTLS12 {
			return 0, fmt.Errorf("TLS version 0x%04x is below minimum allowed (TLS 1.2)", num)
		}
		return num, nil
	}

	switch version {
	case "1.2", "TLS1.2", "TLSv1.2", "VersionTLS12":
		return tls.VersionTLS12, nil
	case "1.3", "TLS1.3", "TLSv1.3", "VersionTLS13":
		return tls.VersionTLS13, nil
	default:
		return 0, fmt.Errorf("unsupported TLS version: %s (only TLS 1.2 and 1.3 are allowed)", version)
	}
}

// parseUintFlexible parses a uint16 from a string, supporting both decimal and hex (0x prefix).
func parseUintFlexible(s string) (uint16, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		num, err := strconv.ParseUint(s[2:], 16, 16)
		if err != nil {
			return 0, err
		}
		return uint16(num), nil
	}
	num, err := strconv.ParseUint(s, 10, 16)
	if err != nil {
		return 0, err
	}
	return uint16(num), nil
}

var (
	cipherSuiteCache     map[string]uint16
	cipherSuiteCacheOnce sync.Once
)

func getCipherSuiteMap() map[string]uint16 {
	cipherSuiteCacheOnce.Do(func() {
		cipherSuiteCache = make(map[string]uint16)
		for _, suite := range tls.CipherSuites() {
			cipherSuiteCache[suite.Name] = suite.ID
		}
	})
	return cipherSuiteCache
}

// parseCipherSuites parses a comma-separated list of cipher suites.
// Supports IANA cipher names (e.g., "TLS_AES_128_GCM_SHA256") and numeric IDs
// in both decimal (e.g., "4865") and hex (e.g., "0x1301").
// Returns an error if any entry is unrecognized.
func parseCipherSuites(ciphers string) ([]uint16, error) {
	if ciphers == "" {
		return nil, nil
	}

	cipherMap := getCipherSuiteMap()
	parts := strings.Split(ciphers, ",")
	result := make([]uint16, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if id, err := parseUintFlexible(part); err == nil {
			result = append(result, id)
			continue
		}

		if id, ok := cipherMap[part]; ok {
			result = append(result, id)
			continue
		}

		return nil, fmt.Errorf("unrecognized cipher suite: %q", part)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no cipher suites specified in: %s", ciphers)
	}

	return result, nil
}

var curveNameToID = map[string]tls.CurveID{
	"P256":      tls.CurveP256,
	"P384":      tls.CurveP384,
	"P521":      tls.CurveP521,
	"CurveP256": tls.CurveP256,
	"CurveP384": tls.CurveP384,
	"CurveP521": tls.CurveP521,
	"X25519":    tls.X25519,
	"P-256":     tls.CurveP256,
	"P-384":     tls.CurveP384,
	"P-521":     tls.CurveP521,

	// Post-Quantum hybrid: ML-KEM 768 combined with X25519 (Go 1.24+)
	"X25519MLKEM768": tls.X25519MLKEM768,
}

// parseCurvePreferences parses a comma-separated list of curve names.
// Supports curve names (e.g., "X25519", "P256") and numeric IDs
// in both decimal and hex (e.g., "0x001d").
// Returns an error if any entry is unrecognized.
func parseCurvePreferences(curves string) ([]tls.CurveID, error) {
	if curves == "" {
		return nil, nil
	}

	parts := strings.Split(curves, ",")
	result := make([]tls.CurveID, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if id, err := parseUintFlexible(part); err == nil {
			result = append(result, tls.CurveID(id))
			continue
		}

		if id, ok := curveNameToID[part]; ok {
			result = append(result, id)
			continue
		}

		return nil, fmt.Errorf("unrecognized curve: %q", part)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no curve preferences specified in: %s", curves)
	}

	return result, nil
}

// GetTLSVersionName returns a human-readable name for a TLS version
func GetTLSVersionName(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	case 0:
		return "default"
	default:
		return fmt.Sprintf("Unknown (0x%04x)", version)
	}
}

// FormatCipherSuites returns a human-readable string of cipher suite names
func FormatCipherSuites(ciphers []uint16) string {
	if len(ciphers) == 0 {
		return "default"
	}
	names := make([]string, len(ciphers))
	for i, id := range ciphers {
		names[i] = getCipherSuiteName(id)
	}
	return strings.Join(names, ", ")
}

// FormatCurvePreferences returns a human-readable string of curve names
func FormatCurvePreferences(curves []tls.CurveID) string {
	if len(curves) == 0 {
		return "default"
	}
	names := make([]string, len(curves))
	for i, id := range curves {
		names[i] = getCurveName(id)
	}
	return strings.Join(names, ", ")
}

func getCipherSuiteName(id uint16) string {
	for _, suite := range tls.CipherSuites() {
		if suite.ID == id {
			return suite.Name
		}
	}
	for _, suite := range tls.InsecureCipherSuites() {
		if suite.ID == id {
			return suite.Name
		}
	}
	return fmt.Sprintf("0x%04x", id)
}

func getCurveName(id tls.CurveID) string {
	for name, curveID := range curveNameToID {
		if curveID == id && !strings.HasPrefix(name, "Curve") && !strings.Contains(name, "-") {
			return name
		}
	}
	for name, curveID := range curveNameToID {
		if curveID == id {
			return name
		}
	}
	return fmt.Sprintf("0x%04x", uint16(id))
}
