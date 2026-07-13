package collyproxyhat

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func strptr(s string) *string { return &s }

func TestPickSubUser_AutoPickFirstActive(t *testing.T) {
	list := []apiSubUser{
		{UUID: "1", ProxyUsername: "u1", ProxyPassword: "p1", TrafficLimit: 0},
	}
	got, err := pickSubUser(list, "")
	if err != nil {
		t.Fatal(err)
	}
	if got.ProxyUsername != "u1" || got.ProxyPassword != "p1" {
		t.Fatalf("picked %+v", got)
	}
}

func TestPickSubUser_SkipsSuspendedAndExhausted(t *testing.T) {
	list := []apiSubUser{
		{UUID: "1", ProxyUsername: "susp", ProxyPassword: "p", SuspendedAt: strptr("2026-01-01T00:00:00Z")},
		{UUID: "2", ProxyUsername: "full", ProxyPassword: "p", TrafficLimit: 100, UsedTraffic: 100},
		{UUID: "3", ProxyUsername: "good", ProxyPassword: "p", TrafficLimit: 100, UsedTraffic: 10},
	}
	got, err := pickSubUser(list, "")
	if err != nil {
		t.Fatal(err)
	}
	if got.ProxyUsername != "good" {
		t.Fatalf("expected 'good', got %q", got.ProxyUsername)
	}
}

func TestPickSubUser_ByNameAndUUID(t *testing.T) {
	list := []apiSubUser{
		{UUID: "uuid-1", Name: strptr("alpha"), ProxyUsername: "ua", ProxyPassword: "pa"},
		{UUID: "uuid-2", Name: strptr("beta"), ProxyUsername: "ub", ProxyPassword: "pb"},
	}
	byName, err := pickSubUser(list, "beta")
	if err != nil || byName.ProxyUsername != "ub" {
		t.Fatalf("by name: %+v err=%v", byName, err)
	}
	byUUID, err := pickSubUser(list, "uuid-1")
	if err != nil || byUUID.ProxyUsername != "ua" {
		t.Fatalf("by uuid: %+v err=%v", byUUID, err)
	}
}

func TestPickSubUser_NoMatch(t *testing.T) {
	list := []apiSubUser{{UUID: "1", ProxyUsername: "u", ProxyPassword: "p"}}
	if _, err := pickSubUser(list, "missing"); err == nil {
		t.Fatal("expected error for unknown sub-user")
	}
}

func TestPickSubUser_NoneUsable(t *testing.T) {
	list := []apiSubUser{
		{UUID: "1", ProxyUsername: "susp", ProxyPassword: "p", SuspendedAt: strptr("2026-01-01")},
	}
	if _, err := pickSubUser(list, ""); err == nil {
		t.Fatal("expected error when no sub-user is usable")
	}
}

func TestDecodeSubUsers_Envelopes(t *testing.T) {
	cases := map[string]string{
		"bare array":       `[{"uuid":"1","proxy_username":"u","proxy_password":"p"}]`,
		"payload envelope": `{"payload":[{"uuid":"1","proxy_username":"u","proxy_password":"p"}]}`,
		"data envelope":    `{"data":[{"uuid":"1","proxy_username":"u","proxy_password":"p"}]}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			list, err := decodeSubUsers([]byte(body))
			if err != nil {
				t.Fatal(err)
			}
			if len(list) != 1 || list[0].ProxyUsername != "u" || list[0].ProxyPassword != "p" {
				t.Fatalf("decoded %+v", list)
			}
		})
	}
}

// TestResolveCredentials_APIKeyAutoPick exercises the full API-key path against a
// loopback test server (no external network).
func TestResolveCredentials_APIKeyAutoPick(t *testing.T) {
	t.Setenv("PROXYHAT_USERNAME", "")
	t.Setenv("PROXYHAT_PASSWORD", "")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("Authorization = %q, want Bearer test-key", got)
		}
		if r.URL.Path != "/sub-users" {
			t.Errorf("path = %q, want /sub-users", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"payload":[
			{"uuid":"1","proxy_username":"picked","proxy_password":"secret","traffic_limit":0}
		]}`))
	}))
	defer srv.Close()

	creds, err := resolveCredentials(context.Background(), &config{apiKey: "test-key", baseURL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	if creds.Username != "picked" || creds.Password != "secret" {
		t.Fatalf("resolved %+v", creds)
	}
}

func TestResolveCredentials_APIKeyHTTPError(t *testing.T) {
	t.Setenv("PROXYHAT_USERNAME", "")
	t.Setenv("PROXYHAT_PASSWORD", "")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"unauthorized"}`))
	}))
	defer srv.Close()

	if _, err := resolveCredentials(context.Background(), &config{apiKey: "bad", baseURL: srv.URL}); err == nil {
		t.Fatal("expected error on HTTP 401")
	}
}
