package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/takymt/cflcli/internal/config"
)

func assertBasicAuth(t *testing.T, r *http.Request, wantUser, wantPass string) {
	t.Helper()

	user, pass, ok := r.BasicAuth()
	if !ok || user != wantUser || pass != wantPass {
		t.Fatalf("unexpected auth: ok=%v user=%q", ok, user)
	}
}

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

func TestListPages_QueryAndAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wiki/api/v2/pages" {
			http.NotFound(w, r)
			return
		}
		if got := r.URL.Query().Get("space-id"); got != "SPACE-1" {
			t.Fatalf("space-id=%q", got)
		}
		if got := r.URL.Query().Get("limit"); got != "25" {
			t.Fatalf("limit=%q", got)
		}
		if got := r.URL.Query().Get("cursor"); got != "CURSOR-1" {
			t.Fatalf("cursor=%q", got)
		}
		if got := r.URL.Query().Get("sort"); got != "-created-date" {
			t.Fatalf("sort=%q", got)
		}
		if got := r.URL.Query()["status"]; len(got) != 1 || got[0] != "current" {
			t.Fatalf("status=%v", got)
		}
		assertBasicAuth(t, r, "u@example.com", "token")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"id":"1","title":"A","status":"current","spaceId":"S1"}],"_links":{"next":"NEXT-1"}}`))
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

	result, err := cli.ListPages("SPACE-1", 25, "CURSOR-1", []string{"current"}, "-created-date")
	if err != nil {
		t.Fatalf("ListPages: %v", err)
	}
	if len(result.Results) != 1 || result.Results[0].ID != "1" || result.Links.Next != "NEXT-1" {
		t.Fatalf("unexpected result: %+v", result)
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

func TestCreatePage_Request(t *testing.T) {
	var gotBody struct {
		SpaceID  string `json:"spaceId"`
		Status   string `json:"status"`
		Title    string `json:"title"`
		ParentID string `json:"parentId"`
		Body     struct {
			Storage struct {
				Representation string `json:"representation"`
				Value          string `json:"value"`
			} `json:"storage"`
		} `json:"body"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method=%q", r.Method)
		}
		if r.URL.Path != "/wiki/api/v2/pages" {
			http.NotFound(w, r)
			return
		}
		if contentType := r.Header.Get("Content-Type"); contentType != "application/json" {
			t.Fatalf("content-type=%q", contentType)
		}
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(bodyBytes, &gotBody); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"10","title":"New Doc","status":"current","spaceId":"SPACE-1"}`))
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

	created, err := cli.CreatePage("SPACE-1", "New Doc", "<p>Hello</p>", "22")
	if err != nil {
		t.Fatalf("CreatePage: %v", err)
	}
	if created.ID != "10" || created.Title != "New Doc" {
		t.Fatalf("unexpected created page: %+v", created)
	}
	if gotBody.SpaceID != "SPACE-1" || gotBody.Status != "current" || gotBody.Title != "New Doc" || gotBody.ParentID != "22" {
		t.Fatalf("unexpected create payload: %+v", gotBody)
	}
	if gotBody.Body.Storage.Representation != "storage" || gotBody.Body.Storage.Value != "<p>Hello</p>" {
		t.Fatalf("unexpected create body payload: %+v", gotBody.Body.Storage)
	}
}

func TestUpdatePage_Request(t *testing.T) {
	var gotBody struct {
		ID       string `json:"id"`
		Status   string `json:"status"`
		Title    string `json:"title"`
		ParentID string `json:"parentId"`
		Body     struct {
			Storage struct {
				Representation string `json:"representation"`
				Value          string `json:"value"`
			} `json:"storage"`
		} `json:"body"`
		Version struct {
			Number int `json:"number"`
		} `json:"version"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("method=%q", r.Method)
		}
		if r.URL.Path != "/wiki/api/v2/pages/123" {
			http.NotFound(w, r)
			return
		}
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(bodyBytes, &gotBody); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"123","title":"Updated Doc","status":"current","spaceId":"SPACE-1"}`))
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

	updated, err := cli.UpdatePage("123", "Updated Doc", "<p>Updated</p>", "55", 8)
	if err != nil {
		t.Fatalf("UpdatePage: %v", err)
	}
	if updated.ID != "123" || updated.Title != "Updated Doc" {
		t.Fatalf("unexpected updated page: %+v", updated)
	}
	if gotBody.ID != "123" || gotBody.Status != "current" || gotBody.Title != "Updated Doc" || gotBody.ParentID != "55" {
		t.Fatalf("unexpected update payload: %+v", gotBody)
	}
	if gotBody.Version.Number != 8 {
		t.Fatalf("unexpected update version payload: %+v", gotBody.Version)
	}
	if gotBody.Body.Storage.Representation != "storage" || gotBody.Body.Storage.Value != "<p>Updated</p>" {
		t.Fatalf("unexpected update body payload: %+v", gotBody.Body.Storage)
	}
}

func TestDeletePage_Request(t *testing.T) {
	var gotPath string
	var gotMethod string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
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

	if err := cli.DeletePage("123"); err != nil {
		t.Fatalf("DeletePage: %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Fatalf("method=%q want %q", gotMethod, http.MethodDelete)
	}
	if gotPath != "/wiki/api/v2/pages/123" {
		t.Fatalf("path=%q", gotPath)
	}
}

func TestGetPage_Query(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wiki/api/v2/pages/123" {
			http.NotFound(w, r)
			return
		}
		if got := r.URL.Query().Get("body-format"); got != "storage" {
			t.Fatalf("body-format=%q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"123","title":"Doc","status":"current","spaceId":"S1","body":{"storage":{"representation":"storage","value":"<p>Hello</p>"}}}`))
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

	page, err := cli.GetPage("123")
	if err != nil {
		t.Fatalf("GetPage: %v", err)
	}
	if page.ID != "123" || page.Title != "Doc" {
		t.Fatalf("unexpected page: %+v", page)
	}
	if page.Body.Storage.Representation != "storage" || page.Body.Storage.Value != "<p>Hello</p>" {
		t.Fatalf("unexpected page body: %+v", page.Body.Storage)
	}
}
