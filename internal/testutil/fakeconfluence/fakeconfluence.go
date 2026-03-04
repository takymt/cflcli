package fakeconfluence

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

type Space struct {
	ID         string
	HomepageID string
}

type Page struct {
	ID       string
	SpaceID  string
	ParentID string
	Title    string
	Body     string
	Version  int
	Status   string
}

type Fake struct {
	mu      sync.Mutex
	nextID  int64
	spaces  map[string]Space
	pages   map[string]Page
	baseURL string
}

func New(baseURL string) *Fake {
	return &Fake{
		nextID:  1000,
		spaces:  map[string]Space{},
		pages:   map[string]Page{},
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

func (f *Fake) AddSpace(id, homepageID string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.spaces[id] = Space{
		ID:         id,
		HomepageID: homepageID,
	}
}

func (f *Fake) AddPage(page Page) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if page.Version == 0 {
		page.Version = 1
	}
	if page.Status == "" {
		page.Status = "current"
	}
	f.pages[page.ID] = page
}

func (f *Fake) SnapshotPages() []Page {
	f.mu.Lock()
	defer f.mu.Unlock()

	out := make([]Page, 0, len(f.pages))
	for _, page := range f.pages {
		out = append(out, page)
	}
	return out
}

func (f *Fake) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/wiki/api/v2/spaces/") && !strings.Contains(r.URL.Path, "/pages"):
		f.handleGetSpace(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/wiki/api/v2/spaces/") && strings.HasSuffix(r.URL.Path, "/pages"):
		f.handleGetPagesInSpace(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/wiki/api/v2/pages":
		f.handleCreatePage(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/wiki/api/v2/pages/"):
		f.handleGetPage(w, r)
	case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/wiki/api/v2/pages/"):
		f.handleUpdatePage(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (f *Fake) handleGetSpace(w http.ResponseWriter, r *http.Request) {
	spaceID := strings.TrimPrefix(r.URL.Path, "/wiki/api/v2/spaces/")

	f.mu.Lock()
	space, ok := f.spaces[spaceID]
	f.mu.Unlock()
	if !ok {
		http.NotFound(w, r)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":         space.ID,
		"homepageId": space.HomepageID,
		"_links": map[string]any{
			"base": f.baseURL,
		},
	})
}

func (f *Fake) handleGetPagesInSpace(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/wiki/api/v2/spaces/")
	spaceID := strings.TrimSuffix(path, "/pages")
	title := r.URL.Query().Get("title")

	f.mu.Lock()
	defer f.mu.Unlock()

	if _, ok := f.spaces[spaceID]; !ok {
		http.NotFound(w, r)
		return
	}

	results := make([]map[string]any, 0)
	for _, page := range f.pages {
		if page.SpaceID != spaceID {
			continue
		}
		if title != "" && page.Title != title {
			continue
		}

		results = append(results, map[string]any{
			"id":       page.ID,
			"status":   page.Status,
			"title":    page.Title,
			"spaceId":  page.SpaceID,
			"parentId": page.ParentID,
			"version": map[string]any{
				"number": page.Version,
			},
			"body": map[string]any{
				"storage": map[string]any{
					"value": page.Body,
				},
			},
			"_links": map[string]any{
				"webui": fmt.Sprintf("/wiki/pages/viewpage.action?pageId=%s", page.ID),
			},
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"results": results,
		"_links": map[string]any{
			"base": f.baseURL,
		},
	})
}

func (f *Fake) handleCreatePage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SpaceID  string `json:"spaceId"`
		Status   string `json:"status"`
		Title    string `json:"title"`
		ParentID string `json:"parentId"`
		Body     struct {
			Representation string `json:"representation"`
			Value          string `json:"value"`
		} `json:"body"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if _, ok := f.spaces[req.SpaceID]; !ok {
		http.NotFound(w, r)
		return
	}

	for _, page := range f.pages {
		if page.SpaceID == req.SpaceID && page.ParentID == req.ParentID && page.Title == req.Title {
			http.Error(w, "duplicate title", http.StatusConflict)
			return
		}
	}

	id := strconv.FormatInt(f.nextID, 10)
	f.nextID++

	page := Page{
		ID:       id,
		SpaceID:  req.SpaceID,
		ParentID: req.ParentID,
		Title:    req.Title,
		Body:     req.Body.Value,
		Version:  1,
		Status:   defaultStatus(req.Status),
	}
	f.pages[id] = page

	writeJSON(w, http.StatusOK, pageResponse(page, f.baseURL))
}

func (f *Fake) handleGetPage(w http.ResponseWriter, r *http.Request) {
	pageID := strings.TrimPrefix(r.URL.Path, "/wiki/api/v2/pages/")

	f.mu.Lock()
	page, ok := f.pages[pageID]
	f.mu.Unlock()
	if !ok {
		http.NotFound(w, r)
		return
	}

	writeJSON(w, http.StatusOK, pageResponse(page, f.baseURL))
}

func (f *Fake) handleUpdatePage(w http.ResponseWriter, r *http.Request) {
	pageID := strings.TrimPrefix(r.URL.Path, "/wiki/api/v2/pages/")

	var req struct {
		ID       string `json:"id"`
		Status   string `json:"status"`
		Title    string `json:"title"`
		SpaceID  string `json:"spaceId"`
		ParentID string `json:"parentId"`
		Body     struct {
			Representation string `json:"representation"`
			Value          string `json:"value"`
		} `json:"body"`
		Version struct {
			Number int `json:"number"`
		} `json:"version"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	page, ok := f.pages[pageID]
	if !ok {
		http.NotFound(w, r)
		return
	}

	if req.ID != "" && req.ID != pageID {
		http.Error(w, "id mismatch", http.StatusBadRequest)
		return
	}
	if req.Version.Number != page.Version+1 {
		http.Error(w, "invalid version", http.StatusConflict)
		return
	}

	page.Title = req.Title
	page.Body = req.Body.Value
	if req.SpaceID != "" {
		page.SpaceID = req.SpaceID
	}
	if req.ParentID != "" {
		page.ParentID = req.ParentID
	}
	page.Status = defaultStatus(req.Status)
	page.Version = req.Version.Number

	f.pages[pageID] = page
	writeJSON(w, http.StatusOK, pageResponse(page, f.baseURL))
}

func defaultStatus(status string) string {
	if status == "" {
		return "current"
	}
	return status
}

func pageResponse(page Page, baseURL string) map[string]any {
	return map[string]any{
		"id":       page.ID,
		"status":   page.Status,
		"title":    page.Title,
		"spaceId":  page.SpaceID,
		"parentId": page.ParentID,
		"version": map[string]any{
			"number": page.Version,
		},
		"body": map[string]any{
			"storage": map[string]any{
				"value": page.Body,
			},
		},
		"_links": map[string]any{
			"base":  baseURL,
			"webui": fmt.Sprintf("/wiki/pages/viewpage.action?pageId=%s", page.ID),
		},
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
