package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/takymt/cflcli/internal/client"
	"github.com/takymt/cflcli/internal/config"
)

func TestValidatePageListLimit(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		limit   int
		wantErr bool
	}{
		{name: "min", limit: 1},
		{name: "max", limit: 250},
		{name: "below min", limit: 0, wantErr: true},
		{name: "above max", limit: 251, wantErr: true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validatePageListLimit(tc.limit)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestResolveProfile(t *testing.T) {
	cfg := &config.Config{
		Current: "default",
		Profiles: []config.Profile{
			{Name: "default", Domain: "example.atlassian.net", User: "default@example.com"},
			{Name: "work", Domain: "work.atlassian.net", User: "work@example.com"},
		},
	}

	original := ProfileFlag()
	t.Cleanup(func() { SetProfileFlag(original) })

	t.Run("profile flag", func(t *testing.T) {
		SetProfileFlag("work")
		got, err := resolveProfile(cfg)
		if err != nil {
			t.Fatalf("resolveProfile: %v", err)
		}
		if got.Name != "work" {
			t.Fatalf("got %q want %q", got.Name, "work")
		}
	})

	t.Run("current profile", func(t *testing.T) {
		SetProfileFlag("")
		got, err := resolveProfile(cfg)
		if err != nil {
			t.Fatalf("resolveProfile: %v", err)
		}
		if got.Name != "default" {
			t.Fatalf("got %q want %q", got.Name, "default")
		}
	})

	t.Run("missing flagged profile", func(t *testing.T) {
		SetProfileFlag("missing")
		_, err := resolveProfile(cfg)
		if err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("missing current profile", func(t *testing.T) {
		SetProfileFlag("")
		emptyCfg := &config.Config{}
		_, err := resolveProfile(emptyCfg)
		if err == nil {
			t.Fatalf("expected error")
		}
	})
}

func TestRunPageListWithConfig_JSON_ResolvesSpaceKey(t *testing.T) {
	var gotSpacesQuery string
	var gotPagesQuery string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, password, ok := r.BasicAuth()
		if !ok || user != "user@example.com" || password != "token" {
			t.Fatalf("unexpected basic auth: ok=%v user=%q", ok, user)
		}

		switch r.URL.Path {
		case "/wiki/api/v2/spaces":
			gotSpacesQuery = r.URL.RawQuery
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[{"id":"SPACE-1","key":"WORK"}]}`))
		case "/wiki/api/v2/pages":
			gotPagesQuery = r.URL.RawQuery
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[{"id":"1","title":"T","status":"current","spaceId":"SPACE-1"}],"_links":{"next":"cursor-2"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	originalHTTPClient := client.DefaultHTTPClient
	client.DefaultHTTPClient = srv.Client()
	t.Cleanup(func() { client.DefaultHTTPClient = originalHTTPClient })

	originalOutput := OutputFlag()
	SetOutputFlag("json")
	t.Cleanup(func() { SetOutputFlag(originalOutput) })

	t.Setenv("CONFLUENCE_API_TOKEN", "token")

	cfg := &config.Config{
		Current: "work",
		Profiles: []config.Profile{
			{Name: "work", Domain: srv.URL, User: "user@example.com", SpaceKey: "WORK"},
		},
	}

	var out bytes.Buffer
	err := RunPageListWithConfig(&out, &PageListOptions{SpaceKey: "WORK", Limit: 2}, cfg)
	if err != nil {
		t.Fatalf("RunPageListWithConfig: %v", err)
	}

	if !strings.Contains(gotSpacesQuery, "keys=WORK") {
		t.Fatalf("unexpected spaces query: %q", gotSpacesQuery)
	}
	if !strings.Contains(gotPagesQuery, "space-id=SPACE-1") || !strings.Contains(gotPagesQuery, "limit=2") {
		t.Fatalf("unexpected pages query: %q", gotPagesQuery)
	}

	var payload struct {
		Request struct {
			SpaceID string `json:"space_id"`
			Limit   int    `json:"limit"`
		} `json:"request"`
		Next    string `json:"next"`
		Results []struct {
			ID string `json:"id"`
		} `json:"results"`
	}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if payload.Request.SpaceID != "SPACE-1" || payload.Request.Limit != 2 || payload.Next != "cursor-2" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if len(payload.Results) != 1 || payload.Results[0].ID != "1" {
		t.Fatalf("unexpected results: %+v", payload.Results)
	}
}

func TestRunPageListWithConfig_Table_UsesProfileSpaceKey(t *testing.T) {
	var gotSpacesQuery string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wiki/api/v2/spaces":
			gotSpacesQuery = r.URL.RawQuery
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[{"id":"SPACE-9","key":"WORK"}]}`))
		case "/wiki/api/v2/pages":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[{"id":"9","title":"Doc","status":"current","spaceId":"SPACE-9"}],"_links":{}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	originalHTTPClient := client.DefaultHTTPClient
	client.DefaultHTTPClient = srv.Client()
	t.Cleanup(func() { client.DefaultHTTPClient = originalHTTPClient })

	originalOutput := OutputFlag()
	SetOutputFlag("table")
	t.Cleanup(func() { SetOutputFlag(originalOutput) })

	t.Setenv("CONFLUENCE_API_TOKEN", "token")

	cfg := &config.Config{
		Current: "work",
		Profiles: []config.Profile{
			{Name: "work", Domain: srv.URL, User: "user@example.com", SpaceKey: "WORK"},
		},
	}

	var out bytes.Buffer
	err := RunPageListWithConfig(&out, &PageListOptions{Limit: 1}, cfg)
	if err != nil {
		t.Fatalf("RunPageListWithConfig: %v", err)
	}

	raw := out.String()
	if !strings.Contains(raw, "Doc") || !strings.Contains(raw, "SPACE-9") {
		t.Fatalf("unexpected table output: %q", raw)
	}
	if !strings.Contains(gotSpacesQuery, "keys=WORK") {
		t.Fatalf("unexpected spaces query: %q", gotSpacesQuery)
	}
}

func TestRunPageListWithConfig_SpaceSelectorErrors(t *testing.T) {
	t.Setenv("CONFLUENCE_API_TOKEN", "token")

	t.Run("space-id and space-key are mutually exclusive", func(t *testing.T) {
		cfg := &config.Config{
			Current: "work",
			Profiles: []config.Profile{
				{Name: "work", Domain: "example.atlassian.net", User: "user@example.com"},
			},
		}

		err := RunPageListWithConfig(&bytes.Buffer{}, &PageListOptions{
			SpaceID:  "123",
			SpaceKey: "WORK",
			Limit:    1,
		}, cfg)
		if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("space key missing from options and profile", func(t *testing.T) {
		cfg := &config.Config{
			Current: "work",
			Profiles: []config.Profile{
				{Name: "work", Domain: "example.atlassian.net", User: "user@example.com"},
			},
		}

		err := RunPageListWithConfig(&bytes.Buffer{}, &PageListOptions{Limit: 1}, cfg)
		if err == nil || !strings.Contains(err.Error(), "space key is required") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("space key not found", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/wiki/api/v2/spaces" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[]}`))
		}))
		defer srv.Close()

		originalHTTPClient := client.DefaultHTTPClient
		client.DefaultHTTPClient = srv.Client()
		t.Cleanup(func() { client.DefaultHTTPClient = originalHTTPClient })

		cfg := &config.Config{
			Current: "work",
			Profiles: []config.Profile{
				{Name: "work", Domain: srv.URL, User: "user@example.com"},
			},
		}

		err := RunPageListWithConfig(&bytes.Buffer{}, &PageListOptions{SpaceKey: "WORK", Limit: 1}, cfg)
		if err == nil || !strings.Contains(err.Error(), "not found") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("space key resolved to multiple spaces", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/wiki/api/v2/spaces" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[{"id":"1","key":"WORK"},{"id":"2","key":"WORK"}]}`))
		}))
		defer srv.Close()

		originalHTTPClient := client.DefaultHTTPClient
		client.DefaultHTTPClient = srv.Client()
		t.Cleanup(func() { client.DefaultHTTPClient = originalHTTPClient })

		cfg := &config.Config{
			Current: "work",
			Profiles: []config.Profile{
				{Name: "work", Domain: srv.URL, User: "user@example.com"},
			},
		}

		err := RunPageListWithConfig(&bytes.Buffer{}, &PageListOptions{SpaceKey: "WORK", Limit: 1}, cfg)
		if err == nil || !strings.Contains(err.Error(), "multiple spaces") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("space key resolve api error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/wiki/api/v2/spaces" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"message":"forbidden"}`))
		}))
		defer srv.Close()

		originalHTTPClient := client.DefaultHTTPClient
		client.DefaultHTTPClient = srv.Client()
		t.Cleanup(func() { client.DefaultHTTPClient = originalHTTPClient })

		cfg := &config.Config{
			Current: "work",
			Profiles: []config.Profile{
				{Name: "work", Domain: srv.URL, User: "user@example.com"},
			},
		}

		err := RunPageListWithConfig(&bytes.Buffer{}, &PageListOptions{SpaceKey: "WORK", Limit: 1}, cfg)
		if err == nil || !strings.Contains(err.Error(), "403 Forbidden") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
