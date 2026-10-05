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
	"fmt"
	"net"
	"net/url"
	"strings"
)

// lookupIP is the hostname resolver used by URLValidator. It is a package
// variable so tests can stub resolution without reaching the network.
var lookupIP = net.LookupIP

// URLValidator optionally restricts the destinations that Execute is allowed to
// contact. It is the SSRF guard for interceptor URLs: an interceptor's
// clientConfig.url is operator-supplied, so without a check it can be pointed
// at cloud metadata endpoints (for example 169.254.169.254), loopback, or any
// internal service, and the EventListener will dispatch the request from its
// own network context.
//
// The guard is opt-in. A zero value (BlockPrivate false) performs no checks,
// which preserves the previous behaviour. When BlockPrivate is true, a URL is
// rejected if any resolved IP is loopback, link-local, private, or
// unspecified, unless its host is in Allowlist. The allowlist mirrors the
// existing github.enterprise-host-allowlist escape hatch so that legitimate
// in-cluster interceptor URLs on private ranges can still be reached.
type URLValidator struct {
	// BlockPrivate enables the guard. When false, Validate is a no-op.
	BlockPrivate bool
	// Allowlist holds normalized hostnames that are permitted even when they
	// resolve to an otherwise blocked address range.
	Allowlist []string
}

// Validate returns an error when the guard is enabled and rawURL resolves to a
// disallowed address. A nil receiver or a disabled guard accepts every URL.
func (v *URLValidator) Validate(rawURL string) error {
	if v == nil || !v.BlockPrivate {
		return nil
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("could not parse interceptor url %q: %w", rawURL, err)
	}

	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("interceptor url %q has no host", rawURL)
	}

	if v.hostAllowed(host) {
		return nil
	}

	// A literal IP needs no resolution; anything else is resolved so that a
	// hostname cannot be used to indirectly reach a blocked address.
	var ips []net.IP
	if literal := net.ParseIP(host); literal != nil {
		ips = []net.IP{literal}
	} else {
		resolved, err := lookupIP(host)
		if err != nil {
			return fmt.Errorf("could not resolve interceptor host %q: %w", host, err)
		}
		ips = resolved
	}

	for _, ip := range ips {
		if isBlockedIP(ip) {
			return fmt.Errorf("interceptor url %q resolves to a disallowed address %s; "+
				"add the host to the %q allowlist to permit it", rawURL, ip, enterpriseHostAllowlistConfigKey)
		}
	}

	return nil
}

func (v *URLValidator) hostAllowed(host string) bool {
	host = normalizeHost(host)
	for _, h := range v.Allowlist {
		if strings.EqualFold(host, normalizeHost(h)) {
			return true
		}
	}
	return false
}

// enterpriseHostAllowlistConfigKey names the ConfigMap key reused as the escape
// hatch for this guard. It is duplicated here (rather than imported from the
// config package) to avoid an import cycle; it is covered by a test that fails
// if the two drift apart.
const enterpriseHostAllowlistConfigKey = "github.enterprise-host-allowlist"

// isBlockedIP reports whether ip falls in a range that should never be reached
// through an operator-supplied interceptor URL. IsLinkLocalUnicast covers the
// 169.254.0.0/16 cloud metadata range.
func isBlockedIP(ip net.IP) bool {
	return ip.IsLoopback() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsPrivate() ||
		ip.IsUnspecified()
}

// normalizeHost strips any scheme or path so that an allowlist entry such as
// "https://host/path" still matches a bare "host".
func normalizeHost(host string) string {
	if parts := strings.SplitN(host, "://", 2); len(parts) == 2 {
		host = parts[1]
	}
	if idx := strings.IndexByte(host, '/'); idx >= 0 {
		host = host[:idx]
	}
	return host
}
