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

package config

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewCoreInterceptorsFromMap(t *testing.T) {
	tests := []struct {
		name string
		data map[string]string
		want *CoreInterceptorsConfig
	}{{
		name: "empty map",
		data: map[string]string{},
		want: &CoreInterceptorsConfig{
			EnterpriseHostAllowlist: nil,
		},
	}, {
		name: "single host",
		data: map[string]string{
			"github.enterprise-host-allowlist": "github.mycompany.com",
		},
		want: &CoreInterceptorsConfig{
			EnterpriseHostAllowlist: []string{"github.mycompany.com"},
		},
	}, {
		name: "multiple hosts",
		data: map[string]string{
			"github.enterprise-host-allowlist": "github.mycompany.com, github.other.com",
		},
		want: &CoreInterceptorsConfig{
			EnterpriseHostAllowlist: []string{"github.mycompany.com", "github.other.com"},
		},
	},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewCoreInterceptorsFromMap(tt.data)
			if err != nil {
				t.Fatalf("NewCoreInterceptorsFromMap() error = %v", err)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("NewCoreInterceptorsFromMap() diff (-want/+got):\n%s", diff)
			}
		})
	}
}

func TestNewCoreInterceptorsFromConfigMap(t *testing.T) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "config-triggers-core-interceptors"},
		Data: map[string]string{
			"github.enterprise-host-allowlist": "github.example.com",
		},
	}
	got, err := NewCoreInterceptorsFromConfigMap(cm)
	if err != nil {
		t.Fatalf("NewCoreInterceptorsFromConfigMap() error = %v", err)
	}
	if len(got.EnterpriseHostAllowlist) != 1 || got.EnterpriseHostAllowlist[0] != "github.example.com" {
		t.Errorf("unexpected allowlist: %v", got.EnterpriseHostAllowlist)
	}
}

func TestCoreInterceptorsDeepCopy(t *testing.T) {
	orig := &CoreInterceptorsConfig{
		EnterpriseHostAllowlist: []string{"a", "b"},
	}
	cp := orig.DeepCopy()
	cp.EnterpriseHostAllowlist[0] = "mutated"
	if orig.EnterpriseHostAllowlist[0] != "a" {
		t.Error("DeepCopy did not isolate the slice")
	}
}

func TestCoreInterceptorsDeepCopyNil(t *testing.T) {
	var orig *CoreInterceptorsConfig
	if orig.DeepCopy() != nil {
		t.Error("DeepCopy of nil should return nil")
	}
}

func TestHostInEnterpriseHostAllowList(t *testing.T) {
	tests := []struct {
		name          string
		allowList     []string
		host          string
		expectAllowed bool
	}{
		{
			name:          "allowed and host only",
			allowList:     []string{"good-url.com"},
			host:          "good-url.com",
			expectAllowed: true,
		},
		{
			name:          "allowed and full URL in host",
			allowList:     []string{"good-url.com"},
			host:          "https://good-url.com/api",
			expectAllowed: true,
		},
		{
			name:          "allowed and host and path only",
			allowList:     []string{"good-url.com"},
			host:          "good-url.com/api",
			expectAllowed: true,
		},
		{
			name:          "allowed and scheme and host only",
			allowList:     []string{"good-url.com"},
			host:          "https://good-url.com",
			expectAllowed: true,
		},
		{
			name:          "disallowed and host only",
			allowList:     []string{"good-url.com"},
			host:          "bad-url.com",
			expectAllowed: false,
		},
		{
			name:          "disallowed and full URL in host",
			allowList:     []string{"good-url.com"},
			host:          "https://bad-url.com/api",
			expectAllowed: false,
		},
		{
			name:          "disallowed and host and path only",
			allowList:     []string{"good-url.com"},
			host:          "bad-url.com/api",
			expectAllowed: false,
		},
		{
			name:          "disallowed and scheme and host only",
			allowList:     []string{"good-url.com"},
			host:          "https://bad-url.com",
			expectAllowed: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			config := &CoreInterceptorsConfig{
				EnterpriseHostAllowlist: tc.allowList,
			}
			allowed := config.HostInEnterpriseHostAllowList(tc.host)
			if allowed != tc.expectAllowed {
				t.Fatalf("Expected allowed to be %v, got %v", tc.expectAllowed, allowed)
			}
		})
	}
}
