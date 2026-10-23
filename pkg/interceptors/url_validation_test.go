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

package interceptors

import (
	"net"
	"testing"

	"github.com/tektoncd/triggers/pkg/apis/config"
)

func TestURLValidator_Validate(t *testing.T) {
	// Stub DNS so the test never touches the network. Hostnames map to the
	// address a cluster operator could point an interceptor URL at.
	stub := map[string][]net.IP{
		"metadata.internal":  {net.ParseIP("169.254.169.254")},
		"db.internal":        {net.ParseIP("10.1.2.3")},
		"public.example.com": {net.ParseIP("93.184.216.34")},
		// A host that resolves to both a public and a private address must be
		// rejected: the private answer is enough to reach an internal service.
		"mixed.example.com": {net.ParseIP("93.184.216.34"), net.ParseIP("127.0.0.1")},
	}
	orig := lookupIP
	lookupIP = func(host string) ([]net.IP, error) {
		if ips, ok := stub[host]; ok {
			return ips, nil
		}
		return nil, &net.DNSError{Err: "no such host", Name: host, IsNotFound: true}
	}
	t.Cleanup(func() { lookupIP = orig })

	for _, tc := range []struct {
		name      string
		validator *URLValidator
		url       string
		wantErr   bool
	}{{
		name:      "nil validator allows everything",
		validator: nil,
		url:       "http://169.254.169.254/latest/meta-data/",
		wantErr:   false,
	}, {
		name:      "disabled guard allows private url",
		validator: &URLValidator{BlockPrivate: false},
		url:       "http://169.254.169.254/latest/meta-data/",
		wantErr:   false,
	}, {
		name:      "blocks link-local metadata IP literal",
		validator: &URLValidator{BlockPrivate: true},
		url:       "http://169.254.169.254/latest/meta-data/",
		wantErr:   true,
	}, {
		name:      "blocks loopback IP literal",
		validator: &URLValidator{BlockPrivate: true},
		url:       "http://127.0.0.1:8080/cel",
		wantErr:   true,
	}, {
		name:      "blocks RFC1918 IP literal",
		validator: &URLValidator{BlockPrivate: true},
		url:       "http://10.0.0.5/cel",
		wantErr:   true,
	}, {
		name:      "blocks hostname resolving to metadata IP",
		validator: &URLValidator{BlockPrivate: true},
		url:       "http://metadata.internal/latest/meta-data/",
		wantErr:   true,
	}, {
		name:      "blocks hostname resolving to private IP",
		validator: &URLValidator{BlockPrivate: true},
		url:       "http://db.internal/cel",
		wantErr:   true,
	}, {
		name:      "blocks host with any private resolved address",
		validator: &URLValidator{BlockPrivate: true},
		url:       "http://mixed.example.com/cel",
		wantErr:   true,
	}, {
		name:      "allows public host",
		validator: &URLValidator{BlockPrivate: true},
		url:       "http://public.example.com/cel",
		wantErr:   false,
	}, {
		name:      "allowlisted private IP is permitted",
		validator: &URLValidator{BlockPrivate: true, Allowlist: []string{"127.0.0.1"}},
		url:       "http://127.0.0.1:8080/cel",
		wantErr:   false,
	}, {
		name:      "allowlisted host is permitted even when private",
		validator: &URLValidator{BlockPrivate: true, Allowlist: []string{"db.internal"}},
		url:       "http://db.internal/cel",
		wantErr:   false,
	}, {
		name:      "allowlist entry with scheme and path still matches",
		validator: &URLValidator{BlockPrivate: true, Allowlist: []string{"https://db.internal/path"}},
		url:       "http://db.internal/cel",
		wantErr:   false,
	}, {
		name:      "unresolvable host errors when enabled",
		validator: &URLValidator{BlockPrivate: true},
		url:       "http://does-not-exist.invalid/cel",
		wantErr:   true,
	}, {
		name:      "url with no host errors when enabled",
		validator: &URLValidator{BlockPrivate: true},
		url:       "not_a_url",
		wantErr:   true,
	}} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.validator.Validate(tc.url)
			if tc.wantErr && err == nil {
				t.Fatalf("Validate(%q) expected an error but got nil", tc.url)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("Validate(%q) unexpected error: %v", tc.url, err)
			}
		})
	}
}

// TestEnterpriseHostAllowlistConfigKeyMatchesConfigPackage guards against the
// duplicated ConfigMap key drifting from the value the config package writes.
func TestEnterpriseHostAllowlistConfigKeyMatchesConfigPackage(t *testing.T) {
	cfg, err := config.NewCoreInterceptorsFromMap(map[string]string{
		enterpriseHostAllowlistConfigKey: "github.example.com",
	})
	if err != nil {
		t.Fatalf("NewCoreInterceptorsFromMap() error = %v", err)
	}
	if len(cfg.EnterpriseHostAllowlist) != 1 {
		t.Fatalf("expected the %q key to populate the enterprise host allowlist, got %#v",
			enterpriseHostAllowlistConfigKey, cfg.EnterpriseHostAllowlist)
	}
}
