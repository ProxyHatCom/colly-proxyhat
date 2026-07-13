package collyproxyhat

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
)

// Gateway connection constants for the ProxyHat residential/mobile network.
const (
	// Gateway is the ProxyHat proxy gateway host.
	Gateway = "gate.proxyhat.com"
	// PortHTTP is the gateway port for the HTTP(S) proxy protocol.
	PortHTTP = 8080
	// PortSOCKS5 is the gateway port for the SOCKS5 proxy protocol.
	PortSOCKS5 = 1080
)

// Protocol selects the proxy transport used to reach the gateway.
type Protocol string

const (
	// HTTP routes through the gateway's HTTP proxy (port 8080). Default.
	HTTP Protocol = "http"
	// SOCKS5 routes through the gateway's SOCKS5 proxy (port 1080).
	SOCKS5 Protocol = "socks5"
)

// slug normalises a targeting value into the gateway's token form:
// lower-cased, trimmed, with internal whitespace collapsed to underscores.
func slug(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), "_")
}

// randomHex returns n random bytes encoded as a 2*n-character lowercase hex
// string. A sticky session id is a routing key, not a secret.
func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// buildProxyUsername builds a gateway username from a sub-user's base
// proxy_username plus targeting tokens, in ProxyHat's fixed order:
//
//	<base>-country-<iso>[-region-<slug>][-city-<slug>][-sid-<16hex>-ttl-<dur>][-filter-<tier>]
//
// A rotating rotator passes an empty cfg.sid so no sid/ttl tokens are added and
// the gateway hands out a fresh IP per connection; a sticky rotator passes a sid
// minted once so every request pins the same exit IP for the TTL.
func buildProxyUsername(base string, cfg *config) string {
	country := strings.ToLower(strings.TrimSpace(cfg.country))
	if country == "" {
		country = "any"
	}
	parts := []string{strings.TrimSpace(base), "country", country}

	if cfg.region != "" && strings.ToLower(strings.TrimSpace(cfg.region)) != "any" {
		parts = append(parts, "region", slug(cfg.region))
	}
	if cfg.city != "" {
		parts = append(parts, "city", slug(cfg.city))
	}
	if cfg.sticky && cfg.sid != "" {
		ttl := strings.TrimSpace(cfg.stickyTTL)
		if ttl == "" {
			ttl = "30m"
		}
		parts = append(parts, "sid", cfg.sid, "ttl", ttl)
	}
	filter := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(cfg.filter)), "filter-")
	if filter != "" && filter != "none" {
		parts = append(parts, "filter", filter)
	}

	return strings.Join(parts, "-")
}

// buildConnectionURL builds the full gateway proxy URL for a resolved rotator,
// e.g. http://<user>-country-us:<pass>@gate.proxyhat.com:8080. The userinfo is
// picked up by Go's http.Transport, which sends it as Proxy-Authorization.
func buildConnectionURL(cfg *config, creds Credentials) *url.URL {
	proto := cfg.protocol
	if proto == "" {
		proto = HTTP
	}
	port := PortHTTP
	if proto == SOCKS5 {
		port = PortSOCKS5
	}
	username := buildProxyUsername(creds.Username, cfg)
	return &url.URL{
		Scheme: string(proto),
		User:   url.UserPassword(username, creds.Password),
		Host:   fmt.Sprintf("%s:%d", Gateway, port),
	}
}
