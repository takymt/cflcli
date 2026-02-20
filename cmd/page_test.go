package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

func writeTempBodyFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "body.xhtml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write body file: %v", err)
	}
	return path
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

func TestRunPageGetWithConfig_WritesStorageBody(t *testing.T) {
	srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wiki/api/v2/pages/123" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"123","title":"Doc","status":"current","spaceId":"SPACE-1","body":{"storage":{"representation":"storage","value":"<p>Hello</p>"}}}`))
	}))

	t.Setenv("CONFLUENCE_API_TOKEN", "token")

	cfg := newPageListConfig(srv.URL, "WORK")

	var out bytes.Buffer
	err := RunPageGetWithConfig(&out, "123", cfg)
	if err != nil {
		t.Fatalf("RunPageGetWithConfig: %v", err)
	}
	if out.String() != "<p>Hello</p>" {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestRunPageGetWithConfig_NotFound(t *testing.T) {
	srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wiki/api/v2/pages/999" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"not found"}`))
	}))

	t.Setenv("CONFLUENCE_API_TOKEN", "token")

	cfg := newPageListConfig(srv.URL, "WORK")

	err := RunPageGetWithConfig(&bytes.Buffer{}, "999", cfg)
	if err == nil || !strings.Contains(err.Error(), `page "999" not found`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunPageCreateWithConfig_Table_UsesSpaceKey(t *testing.T) {
	var gotSpacesQuery string
	var gotPayload struct {
		SpaceID string `json:"spaceId"`
		Body    struct {
			Storage struct {
				Value string `json:"value"`
			} `json:"storage"`
		} `json:"body"`
	}

	srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wiki/api/v2/spaces":
			gotSpacesQuery = r.URL.RawQuery
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[{"id":"SPACE-1","key":"WORK"}]}`))
		case "/wiki/api/v2/pages":
			if r.Method != http.MethodPost {
				t.Fatalf("method=%q", r.Method)
			}
			if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
				t.Fatalf("decode create payload: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"10","title":"New Doc","status":"current","spaceId":"SPACE-1"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	setOutputMode(t, "table")
	t.Setenv("CONFLUENCE_API_TOKEN", "token")

	cfg := newPageListConfig(srv.URL, "WORK")
	opts := &pageCreateOptions{
		Title:    "New Doc",
		BodyFile: writeTempBodyFile(t, "Hello **world**"),
	}

	var out bytes.Buffer
	if err := RunPageCreateWithConfig(&out, opts, cfg); err != nil {
		t.Fatalf("RunPageCreateWithConfig: %v", err)
	}

	if !strings.Contains(gotSpacesQuery, "keys=WORK") {
		t.Fatalf("unexpected spaces query: %q", gotSpacesQuery)
	}
	if gotPayload.SpaceID != "SPACE-1" {
		t.Fatalf("unexpected payload: %+v", gotPayload)
	}
	if !strings.Contains(gotPayload.Body.Storage.Value, "<strong>world</strong>") {
		t.Fatalf("unexpected markdown conversion: %q", gotPayload.Body.Storage.Value)
	}

	raw := out.String()
	if !strings.Contains(raw, `Created page "New Doc" (id: "10").`) {
		t.Fatalf("unexpected output: %q", raw)
	}
}

func TestRunPageCreateWithConfig_JSON_UsesSpaceID(t *testing.T) {
	var gotPayload struct {
		SpaceID  string `json:"spaceId"`
		ParentID string `json:"parentId"`
		Body     struct {
			Storage struct {
				Value string `json:"value"`
			} `json:"storage"`
		} `json:"body"`
	}

	srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wiki/api/v2/spaces":
			t.Fatalf("unexpected spaces lookup for explicit --space-id")
		case "/wiki/api/v2/pages":
			if r.Method != http.MethodPost {
				t.Fatalf("method=%q", r.Method)
			}
			if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
				t.Fatalf("decode create payload: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"11","title":"Child Doc","status":"current","spaceId":"SPACE-99"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	setOutputMode(t, "json")
	t.Setenv("CONFLUENCE_API_TOKEN", "token")

	cfg := newPageListConfig(srv.URL, "WORK")
	opts := &pageCreateOptions{
		Title:      "Child Doc",
		BodyFile:   writeTempBodyFile(t, "<p>child</p>"),
		BodyFormat: "storage",
		ParentID:   "55",
		SpaceID:    "SPACE-99",
	}

	var out bytes.Buffer
	if err := RunPageCreateWithConfig(&out, opts, cfg); err != nil {
		t.Fatalf("RunPageCreateWithConfig: %v", err)
	}

	if gotPayload.SpaceID != "SPACE-99" || gotPayload.ParentID != "55" {
		t.Fatalf("unexpected payload: %+v", gotPayload)
	}
	if gotPayload.Body.Storage.Value != "<p>child</p>" {
		t.Fatalf("unexpected storage passthrough: %q", gotPayload.Body.Storage.Value)
	}

	var created struct {
		ID      string `json:"id"`
		Title   string `json:"title"`
		Status  string `json:"status"`
		SpaceID string `json:"spaceId"`
	}
	if err := json.Unmarshal(out.Bytes(), &created); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if created.ID != "11" || created.Title != "Child Doc" || created.SpaceID != "SPACE-99" {
		t.Fatalf("unexpected output: %+v", created)
	}
}

func TestRunPageCreateWithConfig_Validation(t *testing.T) {
	t.Setenv("CONFLUENCE_API_TOKEN", "token")

	cfg := newPageListConfig("example.atlassian.net", "WORK")
	testCases := []struct {
		name       string
		opts       *pageCreateOptions
		wantErrSub string
	}{
		{
			name: "missing title",
			opts: &pageCreateOptions{
				BodyFile: writeTempBodyFile(t, "<p>Hello</p>"),
			},
			wantErrSub: "--title is required",
		},
		{
			name: "missing body file",
			opts: &pageCreateOptions{
				Title: "Doc",
			},
			wantErrSub: "--body-file is required",
		},
		{
			name: "body file read error",
			opts: &pageCreateOptions{
				Title:    "Doc",
				BodyFile: filepath.Join(t.TempDir(), "missing.xhtml"),
			},
			wantErrSub: "read body file",
		},
		{
			name: "space selectors are mutually exclusive",
			opts: &pageCreateOptions{
				Title:    "Doc",
				BodyFile: writeTempBodyFile(t, "<p>Hello</p>"),
				SpaceID:  "1",
				SpaceKey: "WORK",
			},
			wantErrSub: "mutually exclusive",
		},
		{
			name: "invalid body format",
			opts: &pageCreateOptions{
				Title:      "Doc",
				BodyFile:   writeTempBodyFile(t, "<p>Hello</p>"),
				BodyFormat: "wiki",
			},
			wantErrSub: "invalid --body-format",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := RunPageCreateWithConfig(&bytes.Buffer{}, tc.opts, cfg)
			if err == nil || !strings.Contains(err.Error(), tc.wantErrSub) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestRunPageCreateWithConfig_UsesFrontMatterTitleWhenFlagEmpty(t *testing.T) {
	var gotPayload struct {
		Title string `json:"title"`
		Body  struct {
			Storage struct {
				Value string `json:"value"`
			} `json:"storage"`
		} `json:"body"`
	}

	srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wiki/api/v2/spaces":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[{"id":"SPACE-1","key":"WORK"}]}`))
		case "/wiki/api/v2/pages":
			if r.Method != http.MethodPost {
				t.Fatalf("method=%q", r.Method)
			}
			if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
				t.Fatalf("decode create payload: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"10","title":"Frontmatter Title","status":"current","spaceId":"SPACE-1"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	setOutputMode(t, "table")
	t.Setenv("CONFLUENCE_API_TOKEN", "token")

	cfg := newPageListConfig(srv.URL, "WORK")
	opts := &pageCreateOptions{
		BodyFile: writeTempBodyFile(t, strings.Join([]string{
			"---",
			"title: Frontmatter Title",
			"---",
			"",
			"Hello **world**",
		}, "\n")),
	}

	var out bytes.Buffer
	if err := RunPageCreateWithConfig(&out, opts, cfg); err != nil {
		t.Fatalf("RunPageCreateWithConfig: %v", err)
	}

	if gotPayload.Title != "Frontmatter Title" {
		t.Fatalf("title=%q want %q", gotPayload.Title, "Frontmatter Title")
	}
	if strings.Contains(gotPayload.Body.Storage.Value, "Frontmatter Title") {
		t.Fatalf("frontmatter title leaked to body: %q", gotPayload.Body.Storage.Value)
	}
	if !strings.Contains(gotPayload.Body.Storage.Value, "<strong>world</strong>") {
		t.Fatalf("unexpected markdown conversion: %q", gotPayload.Body.Storage.Value)
	}
	if !strings.Contains(out.String(), `Created page "Frontmatter Title" (id: "10").`) {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestRunPageCreateWithConfig_FlagTitleOverridesFrontMatter(t *testing.T) {
	var gotPayload struct {
		Title string `json:"title"`
	}

	srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wiki/api/v2/spaces":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[{"id":"SPACE-1","key":"WORK"}]}`))
		case "/wiki/api/v2/pages":
			if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
				t.Fatalf("decode create payload: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"10","title":"CLI Title","status":"current","spaceId":"SPACE-1"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	setOutputMode(t, "table")
	t.Setenv("CONFLUENCE_API_TOKEN", "token")

	cfg := newPageListConfig(srv.URL, "WORK")
	opts := &pageCreateOptions{
		Title: "CLI Title",
		BodyFile: writeTempBodyFile(t, strings.Join([]string{
			"---",
			"title: Frontmatter Title",
			"---",
			"",
			"body",
		}, "\n")),
	}

	if err := RunPageCreateWithConfig(&bytes.Buffer{}, opts, cfg); err != nil {
		t.Fatalf("RunPageCreateWithConfig: %v", err)
	}
	if gotPayload.Title != "CLI Title" {
		t.Fatalf("title=%q want %q", gotPayload.Title, "CLI Title")
	}
}

func TestRunPageCreateWithConfig_InvalidFrontMatter(t *testing.T) {
	t.Setenv("CONFLUENCE_API_TOKEN", "token")

	cfg := newPageListConfig("example.atlassian.net", "WORK")
	opts := &pageCreateOptions{
		BodyFile: writeTempBodyFile(t, strings.Join([]string{
			"---",
			"title: Broken",
			"body without closing delimiter",
		}, "\n")),
	}

	err := RunPageCreateWithConfig(&bytes.Buffer{}, opts, cfg)
	if err == nil || !strings.Contains(err.Error(), "invalid frontmatter") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunPageUpdateWithConfig_JSON_AutoVersion(t *testing.T) {
	var gotVersion int
	var gotBodyValue string

	srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wiki/api/v2/pages/123":
			if r.Method == http.MethodGet {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"id":"123","title":"Old","status":"current","spaceId":"SPACE-1","version":{"number":7},"body":{"storage":{"representation":"storage","value":"<p>old</p>"}}}`))
				return
			}
			if r.Method == http.MethodPut {
				var payload struct {
					Version struct {
						Number int `json:"number"`
					} `json:"version"`
					Body struct {
						Storage struct {
							Value string `json:"value"`
						} `json:"storage"`
					} `json:"body"`
				}
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Fatalf("decode update payload: %v", err)
				}
				gotVersion = payload.Version.Number
				gotBodyValue = payload.Body.Storage.Value
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"id":"123","title":"Updated","status":"current","spaceId":"SPACE-1"}`))
				return
			}
			t.Fatalf("unexpected method: %s", r.Method)
		default:
			http.NotFound(w, r)
		}
	}))
	setOutputMode(t, "json")
	t.Setenv("CONFLUENCE_API_TOKEN", "token")

	cfg := newPageListConfig(srv.URL, "WORK")
	opts := &pageUpdateOptions{
		PageID:   "123",
		Title:    "Updated",
		BodyFile: writeTempBodyFile(t, "updated **body**"),
	}

	var out bytes.Buffer
	if err := RunPageUpdateWithConfig(&out, opts, cfg); err != nil {
		t.Fatalf("RunPageUpdateWithConfig: %v", err)
	}
	if gotVersion != 8 {
		t.Fatalf("version=%d want 8", gotVersion)
	}
	if !strings.Contains(gotBodyValue, "<strong>body</strong>") {
		t.Fatalf("unexpected body conversion: %q", gotBodyValue)
	}

	var payload struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if payload.ID != "123" || payload.Title != "Updated" {
		t.Fatalf("unexpected output: %+v", payload)
	}
}

func TestRunPageUpdateWithConfig_UsesFrontMatterTitleWhenFlagEmpty(t *testing.T) {
	var gotTitle string
	var gotVersion int

	srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wiki/api/v2/pages/123":
			if r.Method == http.MethodGet {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"id":"123","title":"Old","status":"current","spaceId":"SPACE-1","version":{"number":7},"body":{"storage":{"representation":"storage","value":"<p>old</p>"}}}`))
				return
			}
			if r.Method == http.MethodPut {
				var payload struct {
					Title   string `json:"title"`
					Version struct {
						Number int `json:"number"`
					} `json:"version"`
					Body struct {
						Storage struct {
							Value string `json:"value"`
						} `json:"storage"`
					} `json:"body"`
				}
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Fatalf("decode update payload: %v", err)
				}
				gotTitle = payload.Title
				gotVersion = payload.Version.Number
				if strings.Contains(payload.Body.Storage.Value, "Frontmatter Update") {
					t.Fatalf("frontmatter title leaked to body: %q", payload.Body.Storage.Value)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"id":"123","title":"Frontmatter Update","status":"current","spaceId":"SPACE-1"}`))
				return
			}
			t.Fatalf("unexpected method: %s", r.Method)
		default:
			http.NotFound(w, r)
		}
	}))
	setOutputMode(t, "table")
	t.Setenv("CONFLUENCE_API_TOKEN", "token")

	cfg := newPageListConfig(srv.URL, "WORK")
	opts := &pageUpdateOptions{
		PageID: "123",
		BodyFile: writeTempBodyFile(t, strings.Join([]string{
			"---",
			"title: Frontmatter Update",
			"---",
			"",
			"updated body",
		}, "\n")),
	}

	var out bytes.Buffer
	if err := RunPageUpdateWithConfig(&out, opts, cfg); err != nil {
		t.Fatalf("RunPageUpdateWithConfig: %v", err)
	}
	if gotTitle != "Frontmatter Update" {
		t.Fatalf("title=%q want %q", gotTitle, "Frontmatter Update")
	}
	if gotVersion != 8 {
		t.Fatalf("version=%d want 8", gotVersion)
	}
	if !strings.Contains(out.String(), `Updated page "Frontmatter Update" (id: "123").`) {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestRunPageUpdateWithConfig_Table_WritesSummary(t *testing.T) {
	var gotVersion int

	srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wiki/api/v2/pages/123":
			if r.Method == http.MethodGet {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"id":"123","title":"Old","status":"current","spaceId":"SPACE-1","version":{"number":4},"body":{"storage":{"representation":"storage","value":"<p>old</p>"}}}`))
				return
			}
			if r.Method == http.MethodPut {
				var payload struct {
					Version struct {
						Number int `json:"number"`
					} `json:"version"`
				}
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Fatalf("decode update payload: %v", err)
				}
				gotVersion = payload.Version.Number
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"id":"123","title":"Updated","status":"current","spaceId":"SPACE-1"}`))
				return
			}
			t.Fatalf("unexpected method: %s", r.Method)
		default:
			http.NotFound(w, r)
		}
	}))
	setOutputMode(t, "table")
	t.Setenv("CONFLUENCE_API_TOKEN", "token")

	cfg := newPageListConfig(srv.URL, "WORK")
	opts := &pageUpdateOptions{
		PageID:   "123",
		Title:    "Updated",
		BodyFile: writeTempBodyFile(t, "updated"),
	}

	var out bytes.Buffer
	if err := RunPageUpdateWithConfig(&out, opts, cfg); err != nil {
		t.Fatalf("RunPageUpdateWithConfig: %v", err)
	}

	if gotVersion != 5 {
		t.Fatalf("version=%d want 5", gotVersion)
	}
	if !strings.Contains(out.String(), `Updated page "Updated" (id: "123").`) {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestRunPageUpdateWithConfig_Conflict(t *testing.T) {
	srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wiki/api/v2/pages/123":
			if r.Method == http.MethodGet {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"id":"123","title":"Old","status":"current","spaceId":"SPACE-1","version":{"number":7},"body":{"storage":{"representation":"storage","value":"<p>old</p>"}}}`))
				return
			}
			if r.Method == http.MethodPut {
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte(`{"message":"version conflict"}`))
				return
			}
			t.Fatalf("unexpected method: %s", r.Method)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Setenv("CONFLUENCE_API_TOKEN", "token")

	cfg := newPageListConfig(srv.URL, "WORK")
	opts := &pageUpdateOptions{
		PageID:   "123",
		Title:    "Updated",
		BodyFile: writeTempBodyFile(t, "updated"),
	}

	err := RunPageUpdateWithConfig(&bytes.Buffer{}, opts, cfg)
	if err == nil || !strings.Contains(err.Error(), "update conflict") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunPageDeleteWithConfig_Table(t *testing.T) {
	srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("method=%q", r.Method)
		}
		if r.URL.Path != "/wiki/api/v2/pages/123" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	setOutputMode(t, "table")
	t.Setenv("CONFLUENCE_API_TOKEN", "token")

	cfg := newPageListConfig(srv.URL, "WORK")

	var out bytes.Buffer
	if err := RunPageDeleteWithConfig(&out, "123", cfg); err != nil {
		t.Fatalf("RunPageDeleteWithConfig: %v", err)
	}

	if !strings.Contains(out.String(), `Deleted page "123".`) {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestRunPageDeleteWithConfig_JSON(t *testing.T) {
	srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("method=%q", r.Method)
		}
		if r.URL.Path != "/wiki/api/v2/pages/123" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	setOutputMode(t, "json")
	t.Setenv("CONFLUENCE_API_TOKEN", "token")

	cfg := newPageListConfig(srv.URL, "WORK")

	var out bytes.Buffer
	if err := RunPageDeleteWithConfig(&out, "123", cfg); err != nil {
		t.Fatalf("RunPageDeleteWithConfig: %v", err)
	}

	var payload struct {
		ID      string `json:"id"`
		Deleted bool   `json:"deleted"`
	}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if payload.ID != "123" || !payload.Deleted {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestRunPageDeleteWithConfig_NotFound(t *testing.T) {
	srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wiki/api/v2/pages/999" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"not found"}`))
	}))
	t.Setenv("CONFLUENCE_API_TOKEN", "token")

	cfg := newPageListConfig(srv.URL, "WORK")

	err := RunPageDeleteWithConfig(&bytes.Buffer{}, "999", cfg)
	if err == nil || !strings.Contains(err.Error(), `page "999" not found`) {
		t.Fatalf("unexpected error: %v", err)
	}
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
