package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/takymt/cflcli/internal/config"
)

func TestNew(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		profile *config.Profile
		ctx     context.Context
		token   string
		wantErr bool
		wantURL string
	}{
		{
			name:    "valid domain",
			profile: &config.Profile{Name: "work", Domain: "example.atlassian.net", User: "u@example.com"},
			ctx:     context.Background(),
			token:   "token",
			wantURL: "https://example.atlassian.net/wiki/api/v2",
		},
		{
			name:    "valid full url",
			profile: &config.Profile{Name: "work", Domain: "https://example.atlassian.net", User: "u@example.com"},
			ctx:     context.Background(),
			token:   "token",
			wantURL: "https://example.atlassian.net/wiki/api/v2",
		},
		{name: "missing profile", ctx: context.Background(), token: "token", wantErr: true},
		{name: "missing context", profile: &config.Profile{Name: "work", Domain: "example.atlassian.net", User: "u@example.com"}, token: "token", wantErr: true},
		{name: "missing domain", profile: &config.Profile{Name: "work", User: "u@example.com"}, ctx: context.Background(), token: "token", wantErr: true},
		{name: "missing user", profile: &config.Profile{Name: "work", Domain: "example.atlassian.net"}, ctx: context.Background(), token: "token", wantErr: true},
		{name: "missing token", profile: &config.Profile{Name: "work", Domain: "example.atlassian.net", User: "u@example.com"}, ctx: context.Background(), wantErr: true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cli, err := New(tc.ctx, tc.profile, tc.token)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			if cli.baseURL != tc.wantURL {
				t.Fatalf("baseURL=%q want %q", cli.baseURL, tc.wantURL)
			}
		})
	}
}

func TestListPages_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("token invalid"))
	}))
	defer srv.Close()

	old := DefaultHTTPClient
	DefaultHTTPClient = srv.Client()
	t.Cleanup(func() { DefaultHTTPClient = old })

	cli, err := New(
		context.Background(),
		&config.Profile{Name: "work", Domain: srv.URL, User: "u@example.com"},
		"token",
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = cli.ListPages("", 10, "", []string{"current"}, "")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "401 Unauthorized") || !strings.Contains(err.Error(), "token invalid") {
		t.Fatalf("unexpected error: %v", err)
	}
}
