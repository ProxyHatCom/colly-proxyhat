package collyproxyhat

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gocolly/colly/v2"
)

func TestNew_RotatingURL(t *testing.T) {
	r, err := New(Username("acme"), Password("s3cret"), Country("us"))
	if err != nil {
		t.Fatal(err)
	}
	u := r.URL()

	if u.Scheme != "http" {
		t.Fatalf("scheme = %q, want http", u.Scheme)
	}
	if u.Host != "gate.proxyhat.com:8080" {
		t.Fatalf("host = %q, want gate.proxyhat.com:8080", u.Host)
	}
	if got := u.User.Username(); got != "acme-country-us" {
		t.Fatalf("proxy username = %q, want acme-country-us", got)
	}
	if pw, _ := u.User.Password(); pw != "s3cret" {
		t.Fatalf("proxy password = %q, want s3cret", pw)
	}
	if strings.Contains(u.User.Username(), "sid") {
		t.Fatalf("rotating URL must not pin a sid: %q", u.User.Username())
	}
}

func TestNew_StickyURL(t *testing.T) {
	r, err := New(Username("acme"), Password("pw"), Country("de"), Sticky())
	if err != nil {
		t.Fatal(err)
	}
	name := r.URL().User.Username()
	if !strings.Contains(name, "-sid-") || !strings.Contains(name, "-ttl-30m") {
		t.Fatalf("sticky URL missing sid/ttl: %q", name)
	}

	// The same rotator pins one IP: ProxyFunc returns an identical URL each call.
	fn := r.ProxyFunc()
	u1, _ := fn(nil)
	u2, _ := fn(nil)
	if u1.String() != u2.String() {
		t.Fatalf("sticky ProxyFunc returned different URLs: %q vs %q", u1, u2)
	}

	// Two sticky rotators mint different session ids.
	r2, _ := New(Username("acme"), Password("pw"), Country("de"), Sticky())
	if r.URL().User.Username() == r2.URL().User.Username() {
		t.Fatalf("two sticky rotators shared a sid: %q", name)
	}
}

func TestNew_SOCKS5(t *testing.T) {
	r, err := New(Username("acme"), Password("pw"), WithSOCKS5())
	if err != nil {
		t.Fatal(err)
	}
	u := r.URL()
	if u.Scheme != "socks5" {
		t.Fatalf("scheme = %q, want socks5", u.Scheme)
	}
	if u.Host != "gate.proxyhat.com:1080" {
		t.Fatalf("host = %q, want gate.proxyhat.com:1080", u.Host)
	}
}

func TestProxyFunc_WiresIntoColly(t *testing.T) {
	fn, err := ProxyHatRotator(Username("acme"), Password("pw"), Country("gb"))
	if err != nil {
		t.Fatal(err)
	}

	// Compile-time + runtime proof it satisfies colly.ProxyFunc and wires in.
	var _ colly.ProxyFunc = fn
	c := colly.NewCollector()
	c.SetProxyFunc(fn)

	req := httptest.NewRequest("GET", "https://example.com", nil)
	u, err := fn(req)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(u.User.Username(), "acme-country-gb") {
		t.Fatalf("unexpected proxy username %q", u.User.Username())
	}
}

func TestNew_NoCredentials(t *testing.T) {
	// Ensure no ambient env leaks into this test.
	t.Setenv("PROXYHAT_USERNAME", "")
	t.Setenv("PROXYHAT_PASSWORD", "")
	t.Setenv("PROXYHAT_API_KEY", "")

	if _, err := New(Country("us")); err == nil {
		t.Fatal("expected an error when no credentials are provided")
	}
}

func TestNew_EnvCredentials(t *testing.T) {
	t.Setenv("PROXYHAT_USERNAME", "envuser")
	t.Setenv("PROXYHAT_PASSWORD", "envpass")
	t.Setenv("PROXYHAT_API_KEY", "")

	r, err := New(Country("fr"))
	if err != nil {
		t.Fatal(err)
	}
	if got := r.URL().User.Username(); got != "envuser-country-fr" {
		t.Fatalf("env username = %q, want envuser-country-fr", got)
	}
	if pw, _ := r.URL().User.Password(); pw != "envpass" {
		t.Fatalf("env password = %q, want envpass", pw)
	}
}

func TestNew_ExplicitOverridesEnv(t *testing.T) {
	t.Setenv("PROXYHAT_USERNAME", "envuser")
	t.Setenv("PROXYHAT_PASSWORD", "envpass")

	r, err := New(Username("optuser"), Password("optpass"))
	if err != nil {
		t.Fatal(err)
	}
	if got := r.URL().User.Username(); !strings.HasPrefix(got, "optuser-") {
		t.Fatalf("explicit option should win over env, got %q", got)
	}
}
