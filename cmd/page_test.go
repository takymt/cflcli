package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/takymt/cflcli/internal/client"
	"github.com/takymt/cflcli/internal/config"
)

func setupPageListServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	originalHTTPClient := client.DefaultHTTPClient
	client.DefaultHTTPClient = srv.Client()
	t.Cleanup(func() { client.DefaultHTTPClient = originalHTTPClient })

	return srv
}

func setOutputMode(t *testing.T, mode string) {
	t.Helper()

	originalOutput := OutputFlag()
	SetOutputFlag(mode)
	t.Cleanup(func() { SetOutputFlag(originalOutput) })
}

func newPageListConfig(domain, spaceKey string) *config.Config {
	return &config.Config{
		Current: "work",
		Profiles: []config.Profile{
			{Name: "work", Domain: domain, User: "user@example.com", SpaceKey: spaceKey},
		},
	}
}

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

func TestResolvePageListStatuses(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		opts       *PageListOptions
		want       []string
		wantShow   bool
		wantErrSub string
	}{
		{
			name:     "default current when status is not specified",
			opts:     &PageListOptions{},
			want:     []string{"current"},
			wantShow: false,
		},
		{
			name: "single valid status",
			opts: &PageListOptions{
				StatusSpecified: true,
				Status:          "archived",
			},
			want:     []string{"archived"},
			wantShow: true,
		},
		{
			name: "multiple valid statuses with spaces",
			opts: &PageListOptions{
				StatusSpecified: true,
				Status:          " current , archived ",
			},
			want:     []string{"current", "archived"},
			wantShow: true,
		},
		{
			name: "empty status element",
			opts: &PageListOptions{
				StatusSpecified: true,
				Status:          "current,,archived",
			},
			wantErrSub: "must not be empty",
		},
		{
			name: "invalid status value",
			opts: &PageListOptions{
				StatusSpecified: true,
				Status:          "draft",
			},
			wantErrSub: "invalid status",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, gotShow, err := resolvePageListStatuses(tc.opts)
			if tc.wantErrSub != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErrSub) {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolvePageListStatuses: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("statuses=%v want %v", got, tc.want)
			}
			if gotShow != tc.wantShow {
				t.Fatalf("showStatus=%v want %v", gotShow, tc.wantShow)
			}
		})
	}
}

func TestResolvePageListSort(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		opts       *PageListOptions
		want       string
		wantErrSub string
	}{
		{
			name: "unspecified sort",
			opts: &PageListOptions{},
			want: "",
		},
		{
			name: "valid sort",
			opts: &PageListOptions{Sort: "-modified-date"},
			want: "-modified-date",
		},
		{
			name: "valid sort with surrounding spaces",
			opts: &PageListOptions{Sort: " title "},
			want: "title",
		},
		{
			name:       "invalid sort",
			opts:       &PageListOptions{Sort: "updated"},
			wantErrSub: "invalid sort",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := resolvePageListSort(tc.opts)
			if tc.wantErrSub != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErrSub) {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolvePageListSort: %v", err)
			}
			if got != tc.want {
				t.Fatalf("sort=%q want %q", got, tc.want)
			}
		})
	}
}

func TestRunPageListWithConfig_JSON_ResolvesSpaceKey(t *testing.T) {
	var gotSpacesQuery string
	var gotPagesQuery string

	srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	setOutputMode(t, "json")

	t.Setenv("CONFLUENCE_API_TOKEN", "token")

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

func TestRunPageListWithConfig_Table_UsesProfileSpaceKey(t *testing.T) {
	var gotSpacesQuery string
	var gotPagesQuery string

	srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wiki/api/v2/spaces":
			gotSpacesQuery = r.URL.RawQuery
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[{"id":"SPACE-9","key":"WORK"}]}`))
		case "/wiki/api/v2/pages":
			gotPagesQuery = r.URL.RawQuery
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[{"id":"9","title":"Doc","status":"current","spaceId":"SPACE-9"}],"_links":{}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	setOutputMode(t, "table")

	t.Setenv("CONFLUENCE_API_TOKEN", "token")

	cfg := newPageListConfig(srv.URL, "WORK")

	var out bytes.Buffer
	err := RunPageListWithConfig(&out, &PageListOptions{Limit: 1}, cfg)
	if err != nil {
		t.Fatalf("RunPageListWithConfig: %v", err)
	}

	raw := out.String()
	if !strings.Contains(raw, "Doc") {
		t.Fatalf("unexpected table output: %q", raw)
	}
	if !strings.Contains(gotSpacesQuery, "keys=WORK") {
		t.Fatalf("unexpected spaces query: %q", gotSpacesQuery)
	}
	if !strings.Contains(gotPagesQuery, "status=current") {
		t.Fatalf("unexpected pages query: %q", gotPagesQuery)
	}
	if strings.Contains(gotPagesQuery, "sort=") {
		t.Fatalf("unexpected pages query: %q", gotPagesQuery)
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

	t.Setenv("CONFLUENCE_API_TOKEN", "token")

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

func TestRunPageListWithConfig_JSON_UsesCursor(t *testing.T) {
	var gotPagesQuery string

	srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wiki/api/v2/spaces":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[{"id":"SPACE-1","key":"WORK"}]}`))
		case "/wiki/api/v2/pages":
			gotPagesQuery = r.URL.RawQuery
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[],"_links":{"next":"cursor-3"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	setOutputMode(t, "json")

	t.Setenv("CONFLUENCE_API_TOKEN", "token")

	cfg := newPageListConfig(srv.URL, "WORK")

	var out bytes.Buffer
	err := RunPageListWithConfig(&out, &PageListOptions{
		Limit:  10,
		Cursor: "cursor-2",
	}, cfg)
	if err != nil {
		t.Fatalf("RunPageListWithConfig: %v", err)
	}

	if !strings.Contains(gotPagesQuery, "cursor=cursor-2") {
		t.Fatalf("unexpected pages query: %q", gotPagesQuery)
	}

	var payload struct {
		Next string `json:"next"`
	}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if payload.Next != "cursor-3" {
		t.Fatalf("unexpected next: %q", payload.Next)
	}
}

func TestRunPageListWithConfig_JSON_UsesSort(t *testing.T) {
	var gotSort string

	srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wiki/api/v2/spaces":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[{"id":"SPACE-1","key":"WORK"}]}`))
		case "/wiki/api/v2/pages":
			gotSort = r.URL.Query().Get("sort")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[],"_links":{"next":"cursor-3"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	setOutputMode(t, "json")

	t.Setenv("CONFLUENCE_API_TOKEN", "token")

	cfg := newPageListConfig(srv.URL, "WORK")

	var out bytes.Buffer
	err := RunPageListWithConfig(&out, &PageListOptions{
		Limit: 10,
		Sort:  "-created-date",
	}, cfg)
	if err != nil {
		t.Fatalf("RunPageListWithConfig: %v", err)
	}

	if gotSort != "-created-date" {
		t.Fatalf("sort=%q want %q", gotSort, "-created-date")
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

func TestPageListSortFlagErrorShowsAllowedValues(t *testing.T) {
	rootCmd := NewRootCmd()
	rootCmd.SetOut(io.Discard)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"page", "list", "--sort"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "allowed values: "+pageListAllowedSortValues) {
		t.Fatalf("unexpected error: %v", err)
	}
}
