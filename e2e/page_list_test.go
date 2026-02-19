package e2e

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/takymt/cflcli/internal/config"
)

const pageListTestToken = "token-for-page-list-tests"

type fakeConfluenceResponse struct {
	spacesStatus int
	spacesBody   string
	pagesStatus  int
	pagesBody    string
}

type capturedRequest struct {
	Method        string
	Path          string
	RawQuery      string
	Authorization string
}

type fakeConfluenceServer struct {
	server *httptest.Server
	mu     sync.Mutex
	log    []capturedRequest
	spaces struct {
		status int
		body   string
	}
	pages struct {
		status int
		body   string
	}
}

func newFakeConfluenceServer(t *testing.T, resp fakeConfluenceResponse) *fakeConfluenceServer {
	t.Helper()

	s := &fakeConfluenceServer{}
	if resp.spacesStatus == 0 {
		resp.spacesStatus = http.StatusOK
	}
	if resp.pagesStatus == 0 {
		resp.pagesStatus = http.StatusOK
	}
	if resp.spacesBody == "" {
		resp.spacesBody = `{"results":[{"id":"SPACE-ID-1","key":"WORK"}]}`
	}
	if resp.pagesBody == "" {
		resp.pagesBody = `{"results":[{"id":"1","title":"Hello","status":"current","space":{"id":"SPACE-ID-1"}}],"_links":{"next":"cursor-1"}}`
	}
	s.spaces.status = resp.spacesStatus
	s.spaces.body = resp.spacesBody
	s.pages.status = resp.pagesStatus
	s.pages.body = resp.pagesBody

	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.record(r)
		switch {
		case strings.HasSuffix(r.URL.Path, "/spaces"):
			writeFakeJSON(w, s.spaces.status, s.spaces.body)
		case strings.HasSuffix(r.URL.Path, "/pages"):
			writeFakeJSON(w, s.pages.status, s.pages.body)
		default:
			writeFakeJSON(w, http.StatusNotFound, `{"message":"not found"}`)
		}
	}))
	return s
}

func writeFakeJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

func (s *fakeConfluenceServer) record(r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.log = append(s.log, capturedRequest{
		Method:        r.Method,
		Path:          r.URL.Path,
		RawQuery:      r.URL.RawQuery,
		Authorization: r.Header.Get("Authorization"),
	})
}

func (s *fakeConfluenceServer) Close() {
	s.server.Close()
}

func (s *fakeConfluenceServer) URL() string {
	return s.server.URL
}

func (s *fakeConfluenceServer) requestsTo(pathSuffix string) []capturedRequest {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]capturedRequest, 0, len(s.log))
	for _, req := range s.log {
		if strings.HasSuffix(req.Path, pathSuffix) {
			result = append(result, req)
		}
	}
	return result
}

func writePageListConfig(t *testing.T, xdgConfigHome string, current string, profiles []config.Profile) {
	t.Helper()

	cfg := &config.Config{
		Current:  current,
		Profiles: profiles,
	}
	if err := cfg.SaveTo(configPath(xdgConfigHome)); err != nil {
		t.Fatalf("save config failed: %v", err)
	}
}

func pageTestProfile(name string, domain string, user string, spaceKey string) config.Profile {
	return config.Profile{
		Name:     name,
		Domain:   domain,
		User:     user,
		SpaceKey: spaceKey,
		Output:   "table",
	}
}

func assertNoTokenLeak(t *testing.T, out string, token string) {
	t.Helper()
	if token == "" {
		return
	}
	if strings.Contains(out, token) {
		t.Fatalf("token leaked in output: %s", out)
	}
}

func assertBasicAuth(t *testing.T, authHeader string, user string, token string) {
	t.Helper()
	if !strings.HasPrefix(authHeader, "Basic ") {
		t.Fatalf("expected Basic auth header, got: %s", authHeader)
	}
	raw := strings.TrimPrefix(authHeader, "Basic ")
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		t.Fatalf("decode auth header failed: %v", err)
	}
	expected := user + ":" + token
	if string(decoded) != expected {
		t.Fatalf("unexpected credentials: got %q want %q", string(decoded), expected)
	}
}

func TestPageListCLI_UsesProvidedSpaceIDAndBasicAuth(t *testing.T) {
	xdgConfigHome := t.TempDir()
	fake := newFakeConfluenceServer(t, fakeConfluenceResponse{})
	defer fake.Close()

	writePageListConfig(t, xdgConfigHome, "work", []config.Profile{
		pageTestProfile("work", fake.URL(), "user@example.com", "WORK"),
	})

	out, err := runCLIWithEnv(
		t,
		xdgConfigHome,
		"",
		map[string]string{"CONFLUENCE_API_TOKEN": pageListTestToken},
		"page",
		"list",
		"--space-id",
		"S-123",
		"--limit",
		"10",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}

	if len(fake.requestsTo("/spaces")) != 0 {
		t.Fatalf("did not expect /spaces call when --space-id is provided")
	}
	pagesReq := fake.requestsTo("/pages")
	if len(pagesReq) != 1 {
		t.Fatalf("expected one /pages call, got %d", len(pagesReq))
	}
	if pagesReq[0].Method != http.MethodGet {
		t.Fatalf("unexpected method: %s", pagesReq[0].Method)
	}

	query, err := url.ParseQuery(pagesReq[0].RawQuery)
	if err != nil {
		t.Fatalf("parse query failed: %v", err)
	}
	if query.Get("space-id") != "S-123" {
		t.Fatalf("unexpected space-id query: %s", pagesReq[0].RawQuery)
	}
	if query.Get("limit") != "10" {
		t.Fatalf("unexpected limit query: %s", pagesReq[0].RawQuery)
	}
	assertBasicAuth(t, pagesReq[0].Authorization, "user@example.com", pageListTestToken)
	if !strings.Contains(out, "Hello") {
		t.Fatalf("expected table output containing page title, got: %s", out)
	}
	assertNoTokenLeak(t, out, pageListTestToken)
}

func TestPageListCLI_JSONOutputIncludesNextCursor(t *testing.T) {
	xdgConfigHome := t.TempDir()
	fake := newFakeConfluenceServer(t, fakeConfluenceResponse{})
	defer fake.Close()

	writePageListConfig(t, xdgConfigHome, "work", []config.Profile{
		pageTestProfile("work", fake.URL(), "user@example.com", "WORK"),
	})

	out, err := runCLIWithEnv(
		t,
		xdgConfigHome,
		"",
		map[string]string{"CONFLUENCE_API_TOKEN": pageListTestToken},
		"--output",
		"json",
		"page",
		"list",
		"--space-id",
		"S-123",
		"--limit",
		"10",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("expected valid json output, got parse error: %v\n%s", err, out)
	}
	if payload["next"] != "cursor-1" {
		t.Fatalf("unexpected next cursor: %#v", payload["next"])
	}
	requestObj, ok := payload["request"].(map[string]any)
	if !ok {
		t.Fatalf("expected request object in json output, got: %#v", payload["request"])
	}
	if requestObj["space_id"] != "S-123" {
		t.Fatalf("unexpected request.space_id: %#v", requestObj["space_id"])
	}
	assertNoTokenLeak(t, out, pageListTestToken)
}

func TestPageListCLI_LimitValidation(t *testing.T) {
	testCases := []struct {
		name        string
		limit       int
		wantErr     bool
		wantAPICall bool
	}{
		{name: "too small", limit: 0, wantErr: true, wantAPICall: false},
		{name: "too large", limit: 251, wantErr: true, wantAPICall: false},
		{name: "lower bound", limit: 1, wantErr: false, wantAPICall: true},
		{name: "upper bound", limit: 250, wantErr: false, wantAPICall: true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			xdgConfigHome := t.TempDir()
			fake := newFakeConfluenceServer(t, fakeConfluenceResponse{})
			defer fake.Close()

			writePageListConfig(t, xdgConfigHome, "work", []config.Profile{
				pageTestProfile("work", fake.URL(), "user@example.com", "WORK"),
			})

			out, err := runCLIWithEnv(
				t,
				xdgConfigHome,
				"",
				map[string]string{"CONFLUENCE_API_TOKEN": pageListTestToken},
				"page",
				"list",
				"--space-id",
				"S-123",
				"--limit",
				strconv.Itoa(tc.limit),
			)

			if tc.wantErr && err == nil {
				t.Fatalf("expected validation error, got success: %s", out)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v\n%s", err, out)
			}
			if tc.wantErr && !strings.Contains(out, "limit must be between 1 and 250") {
				t.Fatalf("expected limit validation message, got: %s", out)
			}
			gotAPICall := len(fake.requestsTo("/pages")) > 0
			if gotAPICall != tc.wantAPICall {
				t.Fatalf("unexpected api call behavior: got pages_call=%t want %t", gotAPICall, tc.wantAPICall)
			}
			assertNoTokenLeak(t, out, pageListTestToken)
		})
	}
}

func TestPageListCLI_ProfileResolution(t *testing.T) {
	testCases := []struct {
		name        string
		current     string
		profiles    []config.Profile
		args        []string
		wantContain string
	}{
		{
			name:    "specified profile not found",
			current: "work",
			profiles: []config.Profile{
				pageTestProfile("work", "http://example.invalid", "user@example.com", "WORK"),
			},
			args:        []string{"--profile", "missing", "page", "list", "--space-id", "S-123"},
			wantContain: `profile "missing" not found`,
		},
		{
			name:    "current profile missing",
			current: "",
			profiles: []config.Profile{
				pageTestProfile("work", "http://example.invalid", "user@example.com", "WORK"),
			},
			args:        []string{"page", "list", "--space-id", "S-123"},
			wantContain: "no current profile",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			xdgConfigHome := t.TempDir()
			fake := newFakeConfluenceServer(t, fakeConfluenceResponse{})
			defer fake.Close()

			profiles := make([]config.Profile, len(tc.profiles))
			copy(profiles, tc.profiles)
			if len(profiles) > 0 {
				profiles[0].Domain = fake.URL()
			}
			writePageListConfig(t, xdgConfigHome, tc.current, profiles)

			out, err := runCLIWithEnv(
				t,
				xdgConfigHome,
				"",
				map[string]string{"CONFLUENCE_API_TOKEN": pageListTestToken},
				tc.args...,
			)
			if err == nil {
				t.Fatalf("expected error, got success: %s", out)
			}
			if !strings.Contains(out, tc.wantContain) {
				t.Fatalf("expected output containing %q, got: %s", tc.wantContain, out)
			}
			if len(fake.requestsTo("/pages")) != 0 || len(fake.requestsTo("/spaces")) != 0 {
				t.Fatalf("did not expect API calls on profile resolution error")
			}
			assertNoTokenLeak(t, out, pageListTestToken)
		})
	}
}

func TestPageListCLI_RequiresAPIToken(t *testing.T) {
	xdgConfigHome := t.TempDir()
	fake := newFakeConfluenceServer(t, fakeConfluenceResponse{})
	defer fake.Close()

	writePageListConfig(t, xdgConfigHome, "work", []config.Profile{
		pageTestProfile("work", fake.URL(), "user@example.com", "WORK"),
	})

	out, err := runCLIWithEnv(
		t,
		xdgConfigHome,
		"",
		map[string]string{"CONFLUENCE_API_TOKEN": ""},
		"page",
		"list",
		"--space-id",
		"S-123",
	)
	if err == nil {
		t.Fatalf("expected error for missing token, got success: %s", out)
	}
	if !strings.Contains(strings.ToLower(out), "token") {
		t.Fatalf("expected token-related error message, got: %s", out)
	}
	if len(fake.requestsTo("/pages")) != 0 || len(fake.requestsTo("/spaces")) != 0 {
		t.Fatalf("did not expect API calls without token")
	}
}

func TestPageListCLI_OutputFormat(t *testing.T) {
	testCases := []struct {
		name        string
		args        []string
		wantErr     bool
		wantContain string
	}{
		{
			name:        "table output",
			args:        []string{"--output", "table", "page", "list", "--space-id", "S-123"},
			wantErr:     false,
			wantContain: "Hello",
		},
		{
			name:        "json output",
			args:        []string{"--output", "json", "page", "list", "--space-id", "S-123"},
			wantErr:     false,
			wantContain: `"results"`,
		},
		{
			name:        "unsupported output",
			args:        []string{"--output", "yaml", "page", "list", "--space-id", "S-123"},
			wantErr:     true,
			wantContain: "unsupported output format",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			xdgConfigHome := t.TempDir()
			fake := newFakeConfluenceServer(t, fakeConfluenceResponse{})
			defer fake.Close()

			writePageListConfig(t, xdgConfigHome, "work", []config.Profile{
				pageTestProfile("work", fake.URL(), "user@example.com", "WORK"),
			})

			out, err := runCLIWithEnv(
				t,
				xdgConfigHome,
				"",
				map[string]string{"CONFLUENCE_API_TOKEN": pageListTestToken},
				tc.args...,
			)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got success: %s", out)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v\n%s", err, out)
			}
			if !strings.Contains(out, tc.wantContain) {
				t.Fatalf("expected output containing %q, got: %s", tc.wantContain, out)
			}
			assertNoTokenLeak(t, out, pageListTestToken)
		})
	}
}

func TestPageListCLI_ResolvesSpaceIDFromSpaceKey(t *testing.T) {
	xdgConfigHome := t.TempDir()
	fake := newFakeConfluenceServer(t, fakeConfluenceResponse{
		spacesBody: `{"results":[{"id":"SPACE-42","key":"WORK"}]}`,
	})
	defer fake.Close()

	writePageListConfig(t, xdgConfigHome, "work", []config.Profile{
		pageTestProfile("work", fake.URL(), "user@example.com", "WORK"),
	})

	out, err := runCLIWithEnv(
		t,
		xdgConfigHome,
		"",
		map[string]string{"CONFLUENCE_API_TOKEN": pageListTestToken},
		"page",
		"list",
		"--limit",
		"7",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}

	spaceReqs := fake.requestsTo("/spaces")
	if len(spaceReqs) != 1 {
		t.Fatalf("expected one /spaces request for space-key resolution, got %d", len(spaceReqs))
	}
	spaceQuery, err := url.ParseQuery(spaceReqs[0].RawQuery)
	if err != nil {
		t.Fatalf("parse /spaces query failed: %v", err)
	}
	if spaceQuery.Get("keys") != "WORK" {
		t.Fatalf("expected keys=WORK, got: %s", spaceReqs[0].RawQuery)
	}

	pagesReqs := fake.requestsTo("/pages")
	if len(pagesReqs) != 1 {
		t.Fatalf("expected one /pages request, got %d", len(pagesReqs))
	}
	pageQuery, err := url.ParseQuery(pagesReqs[0].RawQuery)
	if err != nil {
		t.Fatalf("parse /pages query failed: %v", err)
	}
	if pageQuery.Get("space-id") != "SPACE-42" {
		t.Fatalf("expected resolved space-id=SPACE-42, got: %s", pagesReqs[0].RawQuery)
	}
	if pageQuery.Get("limit") != "7" {
		t.Fatalf("expected limit=7, got: %s", pagesReqs[0].RawQuery)
	}
	assertNoTokenLeak(t, out, pageListTestToken)
}

func TestPageListCLI_RejectsWhenSpaceKeyResolutionFails(t *testing.T) {
	testCases := []struct {
		name        string
		resp        fakeConfluenceResponse
		wantContain string
	}{
		{
			name:        "space key not found",
			resp:        fakeConfluenceResponse{spacesBody: `{"results":[]}`},
			wantContain: "space key",
		},
		{
			name:        "space key resolves to multiple spaces",
			resp:        fakeConfluenceResponse{spacesBody: `{"results":[{"id":"S1","key":"WORK"},{"id":"S2","key":"WORK"}]}`},
			wantContain: "space key",
		},
		{
			name:        "resolve api forbidden",
			resp:        fakeConfluenceResponse{spacesStatus: http.StatusForbidden, spacesBody: `{"message":"forbidden"}`},
			wantContain: "403",
		},
		{
			name:        "resolve api invalid json",
			resp:        fakeConfluenceResponse{spacesBody: `{`},
			wantContain: "decode",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			xdgConfigHome := t.TempDir()
			fake := newFakeConfluenceServer(t, tc.resp)
			defer fake.Close()

			writePageListConfig(t, xdgConfigHome, "work", []config.Profile{
				pageTestProfile("work", fake.URL(), "user@example.com", "WORK"),
			})

			out, err := runCLIWithEnv(
				t,
				xdgConfigHome,
				"",
				map[string]string{"CONFLUENCE_API_TOKEN": pageListTestToken},
				"page",
				"list",
			)
			if err == nil {
				t.Fatalf("expected space key resolution error, got success: %s", out)
			}
			if !strings.Contains(strings.ToLower(out), strings.ToLower(tc.wantContain)) {
				t.Fatalf("expected output containing %q, got: %s", tc.wantContain, out)
			}
			if len(fake.requestsTo("/spaces")) != 1 {
				t.Fatalf("expected one /spaces call, got %d", len(fake.requestsTo("/spaces")))
			}
			if len(fake.requestsTo("/pages")) != 0 {
				t.Fatalf("did not expect /pages call when space resolution fails")
			}
			assertNoTokenLeak(t, out, pageListTestToken)
		})
	}
}

func TestPageListCLI_ExplicitSpaceIDSkipsSpaceKeyResolution(t *testing.T) {
	xdgConfigHome := t.TempDir()
	fake := newFakeConfluenceServer(t, fakeConfluenceResponse{})
	defer fake.Close()

	writePageListConfig(t, xdgConfigHome, "work", []config.Profile{
		pageTestProfile("work", fake.URL(), "user@example.com", "WORK"),
	})

	out, err := runCLIWithEnv(
		t,
		xdgConfigHome,
		"",
		map[string]string{"CONFLUENCE_API_TOKEN": pageListTestToken},
		"page",
		"list",
		"--space-id",
		"OVERRIDE-SPACE",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	if len(fake.requestsTo("/spaces")) != 0 {
		t.Fatalf("did not expect /spaces call when --space-id is provided")
	}
	pagesReqs := fake.requestsTo("/pages")
	if len(pagesReqs) != 1 {
		t.Fatalf("expected one /pages call, got %d", len(pagesReqs))
	}
	pageQuery, err := url.ParseQuery(pagesReqs[0].RawQuery)
	if err != nil {
		t.Fatalf("parse /pages query failed: %v", err)
	}
	if pageQuery.Get("space-id") != "OVERRIDE-SPACE" {
		t.Fatalf("unexpected /pages query: %s", pagesReqs[0].RawQuery)
	}
	assertNoTokenLeak(t, out, pageListTestToken)
}

func TestPageListCLI_HTTPErrorHandling(t *testing.T) {
	testCases := []struct {
		name        string
		status      int
		wantContain []string
	}{
		{
			name:        "authentication error",
			status:      http.StatusUnauthorized,
			wantContain: []string{"authentication"},
		},
		{
			name:        "permission error",
			status:      http.StatusForbidden,
			wantContain: []string{"permission"},
		},
		{
			name:        "rate limit error",
			status:      http.StatusTooManyRequests,
			wantContain: []string{"rate", "retry"},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			xdgConfigHome := t.TempDir()
			fake := newFakeConfluenceServer(t, fakeConfluenceResponse{
				pagesStatus: tc.status,
				pagesBody:   `{"message":"upstream error"}`,
			})
			defer fake.Close()

			writePageListConfig(t, xdgConfigHome, "work", []config.Profile{
				pageTestProfile("work", fake.URL(), "user@example.com", "WORK"),
			})

			out, err := runCLIWithEnv(
				t,
				xdgConfigHome,
				"",
				map[string]string{"CONFLUENCE_API_TOKEN": pageListTestToken},
				"page",
				"list",
				"--space-id",
				"S-123",
			)
			if err == nil {
				t.Fatalf("expected error for status %d, got success: %s", tc.status, out)
			}

			lower := strings.ToLower(out)
			for _, want := range tc.wantContain {
				if !strings.Contains(lower, strings.ToLower(want)) {
					t.Fatalf("expected output containing %q, got: %s", want, out)
				}
			}
			assertNoTokenLeak(t, out, pageListTestToken)
		})
	}
}

func TestPageListCLI_VerboseRedactsToken(t *testing.T) {
	xdgConfigHome := t.TempDir()
	fake := newFakeConfluenceServer(t, fakeConfluenceResponse{})
	defer fake.Close()

	writePageListConfig(t, xdgConfigHome, "work", []config.Profile{
		pageTestProfile("work", fake.URL(), "user@example.com", "WORK"),
	})

	out, err := runCLIWithEnv(
		t,
		xdgConfigHome,
		"",
		map[string]string{"CONFLUENCE_API_TOKEN": pageListTestToken},
		"--verbose",
		"page",
		"list",
		"--space-id",
		"S-123",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	assertNoTokenLeak(t, out, pageListTestToken)

	lower := strings.ToLower(out)
	if !strings.Contains(lower, "request") && !strings.Contains(lower, "get ") && !strings.Contains(lower, "/pages") {
		t.Fatalf("expected verbose diagnostics in output, got: %s", out)
	}
}
