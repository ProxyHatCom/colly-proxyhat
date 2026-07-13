package collyproxyhat

import (
	"regexp"
	"strings"
	"testing"
)

func TestBuildProxyUsername_RotatingDefault(t *testing.T) {
	got := buildProxyUsername("acme", &config{})
	if got != "acme-country-any" {
		t.Fatalf("default username = %q, want %q", got, "acme-country-any")
	}
	if strings.Contains(got, "sid") {
		t.Fatalf("rotating username must not contain a sid: %q", got)
	}
}

func TestBuildProxyUsername_Geo(t *testing.T) {
	got := buildProxyUsername("acme", &config{
		country: "US",
		region:  "California",
		city:    "New York",
		filter:  "high",
	})
	want := "acme-country-us-region-california-city-new_york-filter-high"
	if got != want {
		t.Fatalf("geo username = %q, want %q", got, want)
	}
}

func TestBuildProxyUsername_RegionAnyOmitted(t *testing.T) {
	got := buildProxyUsername("acme", &config{country: "us", region: "any"})
	if got != "acme-country-us" {
		t.Fatalf(`region "any" should be omitted: %q`, got)
	}
}

func TestBuildProxyUsername_FilterPrefixAndNone(t *testing.T) {
	if got := buildProxyUsername("acme", &config{filter: "filter-medium"}); got != "acme-country-any-filter-medium" {
		t.Fatalf("filter- prefix should be stripped: %q", got)
	}
	if got := buildProxyUsername("acme", &config{filter: "none"}); got != "acme-country-any" {
		t.Fatalf(`filter "none" should be omitted: %q`, got)
	}
}

var sidRe = regexp.MustCompile(`-sid-[0-9a-f]{16}-ttl-30m$`)

func TestBuildProxyUsername_StickyDefaultTTL(t *testing.T) {
	got := buildProxyUsername("acme", &config{country: "us", sticky: true, sid: "0123456789abcdef"})
	want := "acme-country-us-sid-0123456789abcdef-ttl-30m"
	if got != want {
		t.Fatalf("sticky username = %q, want %q", got, want)
	}
	if !sidRe.MatchString(got) {
		t.Fatalf("sticky username does not match sid/ttl grammar: %q", got)
	}
}

func TestBuildProxyUsername_StickyCustomTTL(t *testing.T) {
	got := buildProxyUsername("acme", &config{sticky: true, sid: "0123456789abcdef", stickyTTL: "12h"})
	if !strings.HasSuffix(got, "-sid-0123456789abcdef-ttl-12h") {
		t.Fatalf("custom ttl not honored: %q", got)
	}
}

func TestBuildProxyUsername_StickyWithoutSidIsRotating(t *testing.T) {
	// sticky flag but no minted sid (defensive): must not add sid/ttl tokens.
	got := buildProxyUsername("acme", &config{sticky: true})
	if strings.Contains(got, "sid") {
		t.Fatalf("sticky without sid must not add tokens: %q", got)
	}
}

func TestRandomHex(t *testing.T) {
	a, err := randomHex(8)
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != 16 {
		t.Fatalf("randomHex(8) length = %d, want 16", len(a))
	}
	b, _ := randomHex(8)
	if a == b {
		t.Fatalf("randomHex produced identical values %q", a)
	}
}
