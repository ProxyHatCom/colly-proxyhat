# colly-proxyhat

Route [Colly](https://go-colly.org) crawlers through [ProxyHat](https://proxyhat.com?utm_source=github&utm_medium=readme&utm_campaign=colly) residential proxies — rotating IPs, geo-targeting, and sticky sessions behind a one-line `SetProxyFunc`.

[![CI](https://github.com/ProxyHatCom/colly-proxyhat/actions/workflows/ci.yml/badge.svg)](https://github.com/ProxyHatCom/colly-proxyhat/actions/workflows/ci.yml)
[![Compatible with Colly latest](https://github.com/ProxyHatCom/colly-proxyhat/actions/workflows/compat.yml/badge.svg)](https://github.com/ProxyHatCom/colly-proxyhat/actions/workflows/compat.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/ProxyHatCom/colly-proxyhat.svg)](https://pkg.go.dev/github.com/ProxyHatCom/colly-proxyhat)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

> [!TIP]
> **Recommended proxies — [ProxyHat](https://proxyhat.com?utm_source=github&utm_medium=readme&utm_campaign=colly&utm_content=callout) residential IPs.** Every feature in this package is tested end-to-end against ProxyHat and works great. First-class integration; also works with any proxy, or none.


## Why

Crawling at scale from datacenter IPs gets you blocked and rate-limited. This package plugs ProxyHat's residential IPs (50M+ across 148+ countries) into any Colly collector through its first-class `SetProxyFunc` API — a fresh IP per request by default, one pinned IP when you want a session, and geo-targeting encoded straight into the proxy username. No custom transport, no boilerplate.

## Install

```bash
go get github.com/ProxyHatCom/colly-proxyhat
```

Requires Go 1.21+. `github.com/gocolly/colly/v2` is the only direct dependency.

## Quick start

```go
package main

import (
	"log"

	collyproxyhat "github.com/ProxyHatCom/colly-proxyhat"
	"github.com/gocolly/colly/v2"
)

func main() {
	// Reads PROXYHAT_USERNAME / PROXYHAT_PASSWORD (or PROXYHAT_API_KEY) from the env.
	rotator, err := collyproxyhat.New(
		collyproxyhat.Country("us"),
	)
	if err != nil {
		log.Fatal(err)
	}

	c := colly.NewCollector()
	c.SetProxyFunc(rotator.ProxyFunc())

	c.OnResponse(func(r *colly.Response) {
		log.Printf("%d %s", r.StatusCode, r.Request.URL)
	})
	_ = c.Visit("https://httpbin.org/ip")
	c.Wait()
}
```

Prefer the one-liner? `ProxyHatRotator` builds the rotator and hands back the `colly.ProxyFunc` directly:

```go
fn, err := collyproxyhat.ProxyHatRotator(collyproxyhat.Country("de"))
if err != nil {
	log.Fatal(err)
}
c.SetProxyFunc(fn)
```

Get an API key at [proxyhat.com](https://proxyhat.com?utm_source=github&utm_medium=readme&utm_campaign=colly).

## Credentials

Gateway auth uses a **sub-user's** `proxy_username` + `proxy_password` (not the account API key). Pass them explicitly or via environment variables — options win over env:

| Option | Env var | Notes |
|---|---|---|
| `Username(...)` | `PROXYHAT_USERNAME` | Explicit gateway `proxy_username` (skips the API) |
| `Password(...)` | `PROXYHAT_PASSWORD` | Explicit gateway `proxy_password` |
| `APIKey(...)` | `PROXYHAT_API_KEY` | Auto-selects an active sub-user with remaining traffic |
| `SubUser(...)` | `PROXYHAT_SUBUSER` | Pick a specific sub-user by uuid or name (with an API key) |

With an API key, `New` calls the management API (`GET https://api.proxyhat.com/v1/sub-users`) once at construction and picks the first sub-user that isn't suspended and still has traffic. Everything after that is offline. Use `NewContext(ctx, ...)` to govern that lookup with your own context.

## Targeting

```go
rotator, err := collyproxyhat.New(
	collyproxyhat.Country("us"),          // ISO code; omit for no country constraint
	collyproxyhat.Region("california"),   // state/region slug
	collyproxyhat.City("new_york"),       // city slug
	collyproxyhat.Filter("high"),         // AI IP-quality tier
	collyproxyhat.WithSOCKS5(),           // or WithProtocol(collyproxyhat.HTTP), the default
)
```

### Rotating vs. sticky

- **Rotating (default):** the rotator keeps a stable username, so the gateway hands out a **fresh residential IP per connection**. Ideal for spreading a crawl across many IPs.
- **Sticky:** `collyproxyhat.Sticky()` (or `StickyTTL("12h")`) mints a single sticky session id **once**, so every request from this rotator exits from the **same pinned IP** for the TTL (default `30m`) — handy for flows that must look like one continuous user.

```go
// One pinned IP for the lifetime of this rotator:
rotator, _ := collyproxyhat.New(collyproxyhat.Country("gb"), collyproxyhat.StickyTTL("30m"))
```

Want several concurrent sticky identities? Build several rotators — each mints its own session id and its own pinned IP — and route collectors accordingly.

## How it works

ProxyHat exposes a single gateway (`gate.proxyhat.com`, `8080` HTTP / `1080` SOCKS5). All targeting is encoded into the proxy **username** using ProxyHat's grammar:

```
<proxy_username>-country-<iso>[-region-<slug>][-city-<slug>][-sid-<16hex>-ttl-<dur>][-filter-<tier>]
```

`New` resolves your sub-user credentials, mints a sticky session id if requested, and builds a `*url.URL` like `http://<user>-country-us:<pass>@gate.proxyhat.com:8080` **once**. `ProxyFunc()` returns that URL for every request — the userinfo is picked up by Go's HTTP transport as `Proxy-Authorization`. A rotating username carries no `sid`, so the gateway rotates the exit IP per connection; a sticky username carries a fixed `sid`, so the IP stays pinned.

## License

MIT © [ProxyHat](https://proxyhat.com?utm_source=github&utm_medium=readme&utm_campaign=colly)
