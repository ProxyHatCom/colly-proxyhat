package collyproxyhat

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// defaultBaseURL is the ProxyHat management API base used for API-key auto-pick.
const defaultBaseURL = "https://api.proxyhat.com/v1"

// Credentials are a resolved sub-user gateway login (proxy_username /
// proxy_password) — never the account API key.
type Credentials struct {
	Username string
	Password string
}

// apiSubUser mirrors the fields of a ProxyHat sub-user that we need to pick an
// active gateway login. It intentionally includes proxy_password and
// suspended_at, which the official go-sdk's SubUser type omits today (see
// credential resolution notes in the README) — hence this local shape.
type apiSubUser struct {
	UUID          string  `json:"uuid"`
	Name          *string `json:"name"`
	ProxyUsername string  `json:"proxy_username"`
	ProxyPassword string  `json:"proxy_password"`
	SuspendedAt   *string `json:"suspended_at"`
	TrafficLimit  int     `json:"traffic_limit"`
	UsedTraffic   int     `json:"used_traffic"`
}

func env(name string) string {
	return strings.TrimSpace(os.Getenv(name))
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

// resolveCredentials resolves gateway credentials from explicit options or the
// environment. Precedence: explicit Username/Password (or PROXYHAT_USERNAME /
// PROXYHAT_PASSWORD) win; otherwise an API key (PROXYHAT_API_KEY) looks up the
// account's sub-users and picks an active one with remaining traffic — or the
// one named by SubUser / PROXYHAT_SUBUSER.
func resolveCredentials(ctx context.Context, cfg *config) (Credentials, error) {
	username := firstNonEmpty(cfg.username, env("PROXYHAT_USERNAME"))
	password := firstNonEmpty(cfg.password, env("PROXYHAT_PASSWORD"))
	if username != "" && password != "" {
		return Credentials{Username: username, Password: password}, nil
	}

	apiKey := firstNonEmpty(cfg.apiKey, env("PROXYHAT_API_KEY"))
	if apiKey == "" {
		return Credentials{}, fmt.Errorf(
			"colly-proxyhat: no credentials. Pass Username+Password (or PROXYHAT_USERNAME / " +
				"PROXYHAT_PASSWORD), or an APIKey (PROXYHAT_API_KEY) to auto-select a sub-user")
	}

	baseURL := firstNonEmpty(cfg.baseURL, defaultBaseURL)
	list, err := fetchSubUsers(ctx, baseURL, apiKey)
	if err != nil {
		return Credentials{}, err
	}

	want := firstNonEmpty(cfg.subUser, env("PROXYHAT_SUBUSER"))
	chosen, err := pickSubUser(list, want)
	if err != nil {
		return Credentials{}, err
	}
	return Credentials{Username: chosen.ProxyUsername, Password: chosen.ProxyPassword}, nil
}

// pickSubUser applies the selection policy: a named sub-user wins when given;
// otherwise the first active one (not suspended, with remaining traffic) that
// carries usable gateway credentials.
func pickSubUser(list []apiSubUser, want string) (*apiSubUser, error) {
	if want != "" {
		for i := range list {
			s := &list[i]
			if s.UUID == want || (s.Name != nil && *s.Name == want) {
				if s.ProxyUsername == "" || s.ProxyPassword == "" {
					return nil, fmt.Errorf("colly-proxyhat: sub-user %q has no gateway credentials", want)
				}
				return s, nil
			}
		}
		return nil, fmt.Errorf("colly-proxyhat: no sub-user matched %q", want)
	}

	for i := range list {
		s := &list[i]
		active := s.SuspendedAt == nil || strings.TrimSpace(*s.SuspendedAt) == ""
		hasTraffic := s.TrafficLimit == 0 || s.UsedTraffic < s.TrafficLimit
		if active && hasTraffic && s.ProxyUsername != "" && s.ProxyPassword != "" {
			return s, nil
		}
	}
	return nil, fmt.Errorf(
		"colly-proxyhat: no usable sub-user found (all suspended or out of traffic). " +
			"Create one, top up, or pass SubUser")
}

// fetchSubUsers calls GET {baseURL}/sub-users with a Bearer API key.
func fetchSubUsers(ctx context.Context, baseURL, apiKey string) ([]apiSubUser, error) {
	reqURL := strings.TrimRight(baseURL, "/") + "/sub-users"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "colly-proxyhat/0.1.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("colly-proxyhat: sub-user lookup failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("colly-proxyhat: sub-user lookup failed: HTTP %d: %s",
			resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return decodeSubUsers(body)
}

// decodeSubUsers unwraps the API's optional {"payload":...} / {"data":...}
// envelope and decodes the sub-user list.
func decodeSubUsers(body []byte) ([]apiSubUser, error) {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err == nil {
		if raw, ok := envelope["payload"]; ok {
			return unmarshalSubUsers(raw)
		}
		if raw, ok := envelope["data"]; ok {
			return unmarshalSubUsers(raw)
		}
	}
	return unmarshalSubUsers(body)
}

func unmarshalSubUsers(raw []byte) ([]apiSubUser, error) {
	var list []apiSubUser
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, fmt.Errorf("colly-proxyhat: could not decode sub-users: %w", err)
	}
	return list, nil
}
