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

package tlsconfig

import (
	"crypto/tls"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestToTLSConfig(t *testing.T) {
	tests := []struct {
		name       string
		cfg        *Config
		wantMin    uint16
		wantSuites []uint16
		wantCurves []tls.CurveID
		wantErr    bool
	}{
		{
			name:    "empty config defaults to TLS 1.2",
			cfg:     New("", "", ""),
			wantMin: tls.VersionTLS12,
		},
		{
			name:       "operator intermediate profile",
			cfg:        New("VersionTLS12", "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384", "X25519,P-256,P-384"),
			wantMin:    tls.VersionTLS12,
			wantSuites: []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256, tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384},
			wantCurves: []tls.CurveID{tls.X25519, tls.CurveP256, tls.CurveP384},
		},
		{
			name:       "TLS 1.3 with modern ciphers",
			cfg:        New("VersionTLS13", "TLS_AES_128_GCM_SHA256,TLS_AES_256_GCM_SHA384", "X25519"),
			wantMin:    tls.VersionTLS13,
			wantSuites: []uint16{tls.TLS_AES_128_GCM_SHA256, tls.TLS_AES_256_GCM_SHA384},
			wantCurves: []tls.CurveID{tls.X25519},
		},
		{
			name:    "numeric decimal version 771 = TLS 1.2",
			cfg:     New("771", "", ""),
			wantMin: tls.VersionTLS12,
		},
		{
			name:    "numeric hex version 0x0303 = TLS 1.2",
			cfg:     New("0x0303", "", ""),
			wantMin: tls.VersionTLS12,
		},
		{
			name:    "version string format 1.3",
			cfg:     New("1.3", "", ""),
			wantMin: tls.VersionTLS13,
		},
		{
			name:    "TLS 1.0 is rejected as insecure",
			cfg:     New("VersionTLS10", "", ""),
			wantErr: true,
		},
		{
			name:    "TLS 1.1 is rejected as insecure",
			cfg:     New("1.1", "", ""),
			wantErr: true,
		},
		{
			name:    "numeric version below TLS 1.2 is rejected",
			cfg:     New("769", "", ""),
			wantErr: true,
		},
		{
			name:    "invalid version returns error",
			cfg:     New("invalid", "", ""),
			wantErr: true,
		},
		{
			name:    "unrecognized cipher returns error",
			cfg:     New("", "UNKNOWN_CIPHER", ""),
			wantErr: true,
		},
		{
			name:    "typo in cipher is caught",
			cfg:     New("", "TLS_AES_128_GCM_SHA256,TLS_TYPO_CIPHER", ""),
			wantErr: true,
		},
		{
			name:    "unrecognized curve returns error",
			cfg:     New("", "", "UNKNOWN_CURVE"),
			wantErr: true,
		},
		{
			name:    "typo in curve is caught",
			cfg:     New("", "", "X25519,INVALID"),
			wantErr: true,
		},
		{
			name:       "hex numeric cipher suite ID",
			cfg:        New("", "0x1301", ""),
			wantMin:    tls.VersionTLS12,
			wantSuites: []uint16{0x1301},
		},
		{
			name:       "hex numeric curve ID",
			cfg:        New("", "", "0x001d"),
			wantMin:    tls.VersionTLS12,
			wantCurves: []tls.CurveID{tls.CurveID(0x001d)},
		},
		{
			name:       "X25519MLKEM768 PQC curve",
			cfg:        New("", "", "X25519MLKEM768"),
			wantMin:    tls.VersionTLS12,
			wantCurves: []tls.CurveID{tls.X25519MLKEM768},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.cfg.ToTLSConfig()
			if (err != nil) != tt.wantErr {
				t.Fatalf("ToTLSConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got.MinVersion != tt.wantMin {
				t.Errorf("MinVersion = %d, want %d", got.MinVersion, tt.wantMin)
			}
			if diff := cmp.Diff(tt.wantSuites, got.CipherSuites); diff != "" {
				t.Errorf("CipherSuites mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantCurves, got.CurvePreferences); diff != "" {
				t.Errorf("CurvePreferences mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLoadFromEnv(t *testing.T) {
	os.Setenv("TLS_MIN_VERSION", "VersionTLS13")
	os.Setenv("TLS_CIPHER_SUITES", "TLS_AES_128_GCM_SHA256")
	os.Setenv("TLS_CURVE_PREFERENCES", "X25519,P-256")
	t.Cleanup(func() {
		os.Unsetenv("TLS_MIN_VERSION")
		os.Unsetenv("TLS_CIPHER_SUITES")
		os.Unsetenv("TLS_CURVE_PREFERENCES")
	})

	cfg := LoadFromEnv()
	if cfg.MinTLSVersion != "VersionTLS13" {
		t.Errorf("MinTLSVersion = %q, want %q", cfg.MinTLSVersion, "VersionTLS13")
	}
	if cfg.CipherSuites != "TLS_AES_128_GCM_SHA256" {
		t.Errorf("CipherSuites = %q, want %q", cfg.CipherSuites, "TLS_AES_128_GCM_SHA256")
	}
	if cfg.CurvePreferences != "X25519,P-256" {
		t.Errorf("CurvePreferences = %q, want %q", cfg.CurvePreferences, "X25519,P-256")
	}

	got, err := cfg.ToTLSConfig()
	if err != nil {
		t.Fatalf("ToTLSConfig() unexpected error: %v", err)
	}
	if got.MinVersion != tls.VersionTLS13 {
		t.Errorf("MinVersion = %d, want %d", got.MinVersion, tls.VersionTLS13)
	}
}

func TestFormatFunctions(t *testing.T) {
	if got := GetTLSVersionName(tls.VersionTLS12); got != "TLS 1.2" {
		t.Errorf("GetTLSVersionName(TLS12) = %q, want %q", got, "TLS 1.2")
	}
	if got := GetTLSVersionName(tls.VersionTLS13); got != "TLS 1.3" {
		t.Errorf("GetTLSVersionName(TLS13) = %q, want %q", got, "TLS 1.3")
	}
	if got := FormatCipherSuites(nil); got != "default" {
		t.Errorf("FormatCipherSuites(nil) = %q, want %q", got, "default")
	}
	if got := FormatCurvePreferences(nil); got != "default" {
		t.Errorf("FormatCurvePreferences(nil) = %q, want %q", got, "default")
	}
}
