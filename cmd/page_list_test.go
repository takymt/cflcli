package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/takymt/cflcli/internal/config"
)

func TestRunPageListWithConfig_JSON_ResolvesSpaceKey(t *testing.T) {
	var gotSpacesQuery string
	var gotPagesQuery string

	srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	setOutputMode(t, "json")

	t.Setenv("CFL_API_TOKEN", "token")

	cfg := newPageListConfig(srv.URL, "WORK")

	var out bytes.Buffer
	err := RunPageListWithConfig(&out, &PageListOptions{SpaceKey: "WORK", Limit: 2}, cfg)
	if err != nil {
		t.Fatalf("RunPageListWithConfig: %v", err)
	}

	if !strings.Contains(gotSpacesQuery, "keys=WORK") {
		t.Fatalf("unexpected spaces query: %q", gotSpacesQuery)
	}
	if !strings.Contains(gotPagesQuery, "space-id=SPACE-1") ||
		!strings.Contains(gotPagesQuery, "limit=2") ||
		!strings.Contains(gotPagesQuery, "status=current") {
		t.Fatalf("unexpected pages query: %q", gotPagesQuery)
	}
	if strings.Contains(gotPagesQuery, "cursor=") {
		t.Fatalf("unexpected cursor query: %q", gotPagesQuery)
	}
	if strings.Contains(gotPagesQuery, "sort=") {
		t.Fatalf("unexpected sort query: %q", gotPagesQuery)
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

func TestRunPageListWithConfig_Table_ShowsStatusWhenExplicitStatus(t *testing.T) {
	var gotStatuses []string

	srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wiki/api/v2/spaces":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[{"id":"SPACE-1","key":"WORK"}]}`))
		case "/wiki/api/v2/pages":
			gotStatuses = r.URL.Query()["status"]
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[{"id":"1","title":"Doc","status":"archived","spaceId":"SPACE-1"}],"_links":{}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	setOutputMode(t, "table")

	t.Setenv("CFL_API_TOKEN", "token")

	cfg := newPageListConfig(srv.URL, "WORK")

	var out bytes.Buffer
	err := RunPageListWithConfig(&out, &PageListOptions{
		Limit:           1,
		Status:          "current,archived",
		StatusSpecified: true,
	}, cfg)
	if err != nil {
		t.Fatalf("RunPageListWithConfig: %v", err)
	}

	raw := out.String()
	if !strings.Contains(raw, "STATUS") || !strings.Contains(raw, "archived") {
		t.Fatalf("missing status output: %q", raw)
	}
	if len(gotStatuses) != 2 || gotStatuses[0] != "current" || gotStatuses[1] != "archived" {
		t.Fatalf("unexpected statuses: %v", gotStatuses)
	}
}

func TestRunPageListWithConfig_SpaceSelectorErrors(t *testing.T) {
	t.Setenv("CFL_API_TOKEN", "token")

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

	t.Run("space key not found", func(t *testing.T) {
		srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/wiki/api/v2/spaces" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[]}`))
		}))
		cfg := newPageListConfig(srv.URL, "")

		err := RunPageListWithConfig(&bytes.Buffer{}, &PageListOptions{SpaceKey: "WORK", Limit: 1}, cfg)
		if err == nil || !strings.Contains(err.Error(), "not found") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("invalid cursor returns actionable error", func(t *testing.T) {
		srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/wiki/api/v2/pages" {
				http.NotFound(w, r)
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"message":"invalid cursor"}`))
		}))
		cfg := newPageListConfig(srv.URL, "")

		err := RunPageListWithConfig(&bytes.Buffer{}, &PageListOptions{
			SpaceID: "123",
			Limit:   1,
			Cursor:  "bad-cursor",
		}, cfg)
		if err == nil || !strings.Contains(err.Error(), "invalid or expired") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestRunPageListWithConfig_UsesRepoConfigSpaceIDAndDomain(t *testing.T) {
	var gotPagesQuery string

	srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wiki/api/v2/spaces":
			t.Fatalf("unexpected spaces lookup when repo config has space_id")
		case "/wiki/api/v2/pages":
			gotPagesQuery = r.URL.RawQuery
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[{"id":"77","title":"RepoDoc","status":"current","spaceId":"SPACE-77"}],"_links":{}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	setOutputMode(t, "table")
	t.Setenv("CFL_API_TOKEN", "token")

	repoDir := t.TempDir()
	writeRepoConfig(t, repoDir, fmt.Sprintf("domain = %q\nspace_id = \"SPACE-77\"\n", srv.URL))
	chdir(t, repoDir)

	cfg := newPageListConfig("profile.example.atlassian.net", "PROFILE")
	var out bytes.Buffer
	err := RunPageListWithConfig(&out, &PageListOptions{Limit: 1}, cfg)
	if err != nil {
		t.Fatalf("RunPageListWithConfig: %v", err)
	}
	if !strings.Contains(gotPagesQuery, "space-id=SPACE-77") {
		t.Fatalf("unexpected pages query: %q", gotPagesQuery)
	}
	if !strings.Contains(out.String(), "RepoDoc") {
		t.Fatalf("unexpected table output: %q", out.String())
	}
}

func TestRunPageListWithConfig_RepoConfigSpaceSelectorsAreMutuallyExclusive(t *testing.T) {
	t.Setenv("CFL_API_TOKEN", "token")

	repoDir := t.TempDir()
	writeRepoConfig(t, repoDir, "domain = \"example.atlassian.net\"\nspace_id = \"SPACE-1\"\nspace_key = \"WORK\"\n")
	chdir(t, repoDir)

	cfg := newPageListConfig("example.atlassian.net", "PROFILE")
	err := RunPageListWithConfig(&bytes.Buffer{}, &PageListOptions{Limit: 1}, cfg)
	if err == nil || !strings.Contains(err.Error(), "space_id and space_key") {
		t.Fatalf("unexpected error: %v", err)
	}
}
