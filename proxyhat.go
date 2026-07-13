// Package collyproxyhat routes Colly (github.com/gocolly/colly/v2) crawlers
// through the ProxyHat residential/mobile proxy network.
//
// It resolves a sub-user's gateway credentials (explicitly, from the
// environment, or by auto-selecting an active sub-user with an API key), then
// hands Colly a colly.ProxyFunc that returns the ProxyHat gateway URL — with the
// targeting encoded into the proxy username — for every request.
//
//	rotator, err := collyproxyhat.New(
//		collyproxyhat.Country("us"),
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//	c := colly.NewCollector()
//	c.SetProxyFunc(rotator.ProxyFunc())
//
// By default the rotator uses a stable username so the gateway rotates the exit
// IP per connection. Add Sticky (or StickyTTL) to pin one residential IP for the
// rotator's lifetime.
package collyproxyhat

import (
	"context"
	"net/http"
	"net/url"

	"github.com/gocolly/colly/v2"
)

// Rotator holds a resolved ProxyHat gateway URL and exposes it as a
// colly.ProxyFunc. Construct one with New.
//
// The gateway URL is built once at construction: with Sticky, a single session
// id is minted so every request pins the same exit IP; otherwise the stable
// username lets the gateway rotate the IP per connection. Either way ProxyFunc
// is safe for concurrent use — it returns the same immutable *url.URL.
type Rotator struct {
	proxyURL *url.URL
}

// New resolves credentials and targeting into a Rotator. It performs a network
// call only when credentials must be auto-selected from an API key (see
// APIKey); with explicit Username/Password or the matching environment
// variables it is fully offline.
func New(opts ...Option) (*Rotator, error) {
	return NewContext(context.Background(), opts...)
}

// NewContext is New with a caller-supplied context governing the optional
// API-key sub-user lookup.
func NewContext(ctx context.Context, opts ...Option) (*Rotator, error) {
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}

	creds, err := resolveCredentials(ctx, cfg)
	if err != nil {
		return nil, err
	}

	if cfg.sticky {
		sid, err := randomHex(8)
		if err != nil {
			return nil, err
		}
		cfg.sid = sid
	}

	return &Rotator{proxyURL: buildConnectionURL(cfg, creds)}, nil
}

// ProxyFunc returns a colly.ProxyFunc that yields the ProxyHat gateway URL for
// every request. Wire it with (*colly.Collector).SetProxyFunc.
func (r *Rotator) ProxyFunc() colly.ProxyFunc {
	u := r.proxyURL
	return func(_ *http.Request) (*url.URL, error) {
		return u, nil
	}
}

// URL returns the resolved gateway proxy URL (userinfo carries the targeting
// username and password). Useful for inspection and testing.
func (r *Rotator) URL() *url.URL {
	return r.proxyURL
}

// ProxyHatRotator is a one-liner: it constructs a Rotator and returns its
// colly.ProxyFunc, for direct use with SetProxyFunc.
//
//	fn, err := collyproxyhat.ProxyHatRotator(collyproxyhat.Country("de"))
//	if err != nil {
//		log.Fatal(err)
//	}
//	c.SetProxyFunc(fn)
func ProxyHatRotator(opts ...Option) (colly.ProxyFunc, error) {
	r, err := New(opts...)
	if err != nil {
		return nil, err
	}
	return r.ProxyFunc(), nil
}
