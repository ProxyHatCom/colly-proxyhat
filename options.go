package collyproxyhat

// config is the fully-resolved rotator configuration assembled from Options.
type config struct {
	// Targeting.
	country string
	region  string
	city    string
	filter  string

	// Session.
	sticky    bool
	stickyTTL string
	sid       string // minted once in New when sticky; empty for rotating

	// Transport.
	protocol Protocol

	// Credentials.
	username string
	password string
	apiKey   string
	subUser  string
	baseURL  string // management API base override (mainly for tests)
}

// Option configures a Rotator. Options are applied in order; explicit values
// always win over environment variables during credential resolution.
type Option func(*config)

// Country pins the exit country by ISO code (e.g. "us", "de"). Use "any" or omit
// for no country constraint (the default).
func Country(iso string) Option {
	return func(c *config) { c.country = iso }
}

// Region pins the exit region/state slug (e.g. "california"). Ignored when the
// value is empty or "any".
func Region(slug string) Option {
	return func(c *config) { c.region = slug }
}

// City pins the exit city slug (e.g. "new_york").
func City(slug string) Option {
	return func(c *config) { c.city = slug }
}

// Filter selects ProxyHat's AI IP-quality tier (e.g. "high", "medium",
// "high-speed-fast"). Accepts the value with or without a leading "filter-".
func Filter(tier string) Option {
	return func(c *config) { c.filter = tier }
}

// Sticky pins a single residential IP for the lifetime of the rotator by minting
// one sticky session id. Without it (the default) the rotator keeps a stable
// username so the gateway hands out a fresh IP per connection. Default TTL 30m;
// override with StickyTTL.
func Sticky() Option {
	return func(c *config) { c.sticky = true }
}

// StickyTTL sets the sticky-session lifetime (e.g. "30m", "12h") and implies
// Sticky.
func StickyTTL(ttl string) Option {
	return func(c *config) {
		c.sticky = true
		c.stickyTTL = ttl
	}
}

// WithProtocol selects the proxy protocol (HTTP, the default, or SOCKS5).
func WithProtocol(p Protocol) Option {
	return func(c *config) { c.protocol = p }
}

// WithSOCKS5 routes through the gateway's SOCKS5 proxy (port 1080).
func WithSOCKS5() Option {
	return func(c *config) { c.protocol = SOCKS5 }
}

// Username sets the gateway login (a sub-user's proxy_username) explicitly,
// skipping the management API. Pairs with Password.
func Username(u string) Option {
	return func(c *config) { c.username = u }
}

// Password sets the gateway password (a sub-user's proxy_password) explicitly.
func Password(p string) Option {
	return func(c *config) { c.password = p }
}

// APIKey sets a ProxyHat API key used to auto-select an active sub-user when no
// explicit Username/Password is given.
func APIKey(k string) Option {
	return func(c *config) { c.apiKey = k }
}

// SubUser picks a specific sub-user by uuid or name during API-key auto-select.
func SubUser(nameOrUUID string) Option {
	return func(c *config) { c.subUser = nameOrUUID }
}

// WithBaseURL overrides the management API base URL (default
// https://api.proxyhat.com/v1). Mainly useful for testing.
func WithBaseURL(url string) Option {
	return func(c *config) { c.baseURL = url }
}
