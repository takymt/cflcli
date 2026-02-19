package e2e

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
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
