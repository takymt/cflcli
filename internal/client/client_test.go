package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/takymt/cflcli/internal/config"
)

func TestNew_Validation(t *testing.T) {
	testCases := []struct {
		name        string
		ctx         context.Context
		profile     *config.Profile
		token       string
		wantContain string
	}{
		{
			name:        "nil profile",
			ctx:         context.Background(),
			profile:     nil,
			token:       "token",
			wantContain: "profile is required",
		},
		{
			name:        "nil context",
			ctx:         nil,
			profile:     &config.Profile{Domain: "example.atlassian.net", User: "user@example.com"},
			token:       "token",
			wantContain: "context is required",
		},
		{
			name:        "missing domain",
			ctx:         context.Background(),
			profile:     &config.Profile{Domain: "", User: "user@example.com"},
			token:       "token",
			wantContain: "profile domain is required",
		},
		{
			name:        "missing user",
			ctx:         context.Background(),
			profile:     &config.Profile{Domain: "example.atlassian.net", User: ""},
			token:       "token",
			wantContain: "profile user is required",
		},
		{
			name:        "missing token",
			ctx:         context.Background(),
			profile:     &config.Profile{Domain: "example.atlassian.net", User: "user@example.com"},
			token:       "",
			wantContain: "CONFLUENCE_API_TOKEN is required",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := New(tc.ctx, tc.profile, tc.token)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantContain) {
				t.Fatalf("expected error containing %q, got %v", tc.wantContain, err)
			}
		})
	}
}

func TestNew_BaseURL(t *testing.T) {
	testCases := []struct {
		name    string
		domain  string
		wantURL string
	}{
		{
			name:    "domain without scheme",
			domain:  "example.atlassian.net",
			wantURL: "https://example.atlassian.net/wiki/api/v2",
		},
		{
			name:    "domain with scheme",
			domain:  "https://example.atlassian.net/",
			wantURL: "https://example.atlassian.net/wiki/api/v2",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			c, err := New(context.Background(), &config.Profile{
				Domain: tc.domain,
				User:   "user@example.com",
			}, "token")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if c.baseURL != tc.wantURL {
				t.Fatalf("unexpected baseURL: got %q want %q", c.baseURL, tc.wantURL)
			}
		})
	}
}

func TestGet_UsesHTTPContract(t *testing.T) {
	var gotMethod string
	var gotPath string
	var gotQuery string
	var gotAuth string
	var gotAccept string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotAuth = r.Header.Get("Authorization")
		gotAccept = r.Header.Get("Accept")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer server.Close()

	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server url failed: %v", err)
	}

	c := &Client{
		ctx:        context.Background(),
		baseURL:    parsed.String(),
		httpClient: server.Client(),
		user:       "user@example.com",
		token:      "token",
	}

	query := url.Values{}
	query.Set("a", "1")
	query.Set("b", "2")
	var payload map[string]any
	err = c.get("/resource", query, func(dec *json.Decoder) error {
		return dec.Decode(&payload)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotMethod != http.MethodGet {
		t.Fatalf("unexpected method: %s", gotMethod)
	}
	if gotPath != "/resource" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	values, err := url.ParseQuery(gotQuery)
	if err != nil {
		t.Fatalf("parse query failed: %v", err)
	}
	if values.Get("a") != "1" || values.Get("b") != "2" {
		t.Fatalf("unexpected query: %s", gotQuery)
	}
	if !strings.HasPrefix(gotAuth, "Basic ") {
		t.Fatalf("expected basic auth header, got %q", gotAuth)
	}
	if gotAccept != "application/json" {
		t.Fatalf("expected accept header application/json, got %q", gotAccept)
	}
	if payload["ok"] != true {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestDo_HTTPErrorAndDecodeError(t *testing.T) {
	t.Run("returns error for non-2xx", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"message":"rate limited"}`))
		}))
		defer server.Close()

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
		if err != nil {
			t.Fatalf("build request failed: %v", err)
		}
		c := &Client{
			ctx:        context.Background(),
			baseURL:    server.URL,
			httpClient: server.Client(),
			user:       "user@example.com",
			token:      "token",
		}

		err = c.do(req, func(*json.Decoder) error { return nil })
		if err == nil {
			t.Fatalf("expected http error, got nil")
		}
		msg := err.Error()
		if !strings.Contains(msg, "429") || !strings.Contains(msg, "rate limited") {
			t.Fatalf("unexpected error message: %s", msg)
		}
	})

	t.Run("propagates decode error on 2xx", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"ok":true}`))
		}))
		defer server.Close()

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
		if err != nil {
			t.Fatalf("build request failed: %v", err)
		}
		c := &Client{
			ctx:        context.Background(),
			baseURL:    server.URL,
			httpClient: server.Client(),
			user:       "user@example.com",
			token:      "token",
		}

		wantErr := errors.New("decode failed")
		err = c.do(req, func(*json.Decoder) error { return wantErr })
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected decode error, got %v", err)
		}
	})
}
