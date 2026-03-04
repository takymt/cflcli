package page

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type HTTPRemote struct {
	baseURL string
	email   string
	token   string
	client  *http.Client
}

func NewHTTPRemote(baseURL, email, token string, client *http.Client) *HTTPRemote {
	if client == nil {
		client = http.DefaultClient
	}
	return &HTTPRemote{
		baseURL: strings.TrimRight(baseURL, "/"),
		email:   email,
		token:   token,
		client:  client,
	}
}

func (r *HTTPRemote) ResolveRootPageID(ctx context.Context, spaceID string) (string, error) {
	var resp struct {
		HomepageID string `json:"homepageId"`
	}
	if err := r.doJSON(ctx, http.MethodGet, "/wiki/api/v2/spaces/"+spaceID, nil, &resp); err != nil {
		return "", err
	}
	return resp.HomepageID, nil
}

func (r *HTTPRemote) PageExists(ctx context.Context, spaceID, parentID, title string) (bool, error) {
	query := url.Values{}
	query.Set("title", title)

	var resp struct {
		Results []struct {
			ID       string `json:"id"`
			ParentID string `json:"parentId"`
			Title    string `json:"title"`
		} `json:"results"`
	}
	if err := r.doJSON(ctx, http.MethodGet, "/wiki/api/v2/spaces/"+spaceID+"/pages?"+query.Encode(), nil, &resp); err != nil {
		return false, err
	}
	for _, item := range resp.Results {
		if item.ParentID == parentID && item.Title == title {
			return true, nil
		}
	}
	return false, nil
}

func (r *HTTPRemote) CreatePage(ctx context.Context, input CreatePageInput) (RemotePage, error) {
	reqBody := map[string]any{
		"spaceId":  input.SpaceID,
		"parentId": input.ParentID,
		"title":    input.Title,
		"body": map[string]any{
			"representation": "storage",
			"value":          input.Body,
		},
	}

	var resp struct {
		ID       string `json:"id"`
		SpaceID  string `json:"spaceId"`
		ParentID string `json:"parentId"`
		Title    string `json:"title"`
		Version  struct {
			Number int `json:"number"`
		} `json:"version"`
	}
	if err := r.doJSON(ctx, http.MethodPost, "/wiki/api/v2/pages", reqBody, &resp); err != nil {
		return RemotePage{}, err
	}

	return RemotePage{
		ID:       resp.ID,
		SpaceID:  resp.SpaceID,
		ParentID: resp.ParentID,
		Title:    resp.Title,
		URL:      r.pageURL(resp.ID),
		Version:  resp.Version.Number,
		Body:     input.Body,
	}, nil
}

func (r *HTTPRemote) UpdatePage(ctx context.Context, input UpdatePageInput) (RemotePage, error) {
	current, err := r.getPage(ctx, input.PageID)
	if err != nil {
		return RemotePage{}, err
	}

	reqBody := map[string]any{
		"id":       input.PageID,
		"spaceId":  input.SpaceID,
		"parentId": input.ParentID,
		"title":    input.Title,
		"body": map[string]any{
			"representation": "storage",
			"value":          input.Body,
		},
		"version": map[string]any{
			"number": current.Version + 1,
		},
	}

	var resp struct {
		ID       string `json:"id"`
		SpaceID  string `json:"spaceId"`
		ParentID string `json:"parentId"`
		Title    string `json:"title"`
		Version  struct {
			Number int `json:"number"`
		} `json:"version"`
	}
	if err := r.doJSON(ctx, http.MethodPut, "/wiki/api/v2/pages/"+input.PageID, reqBody, &resp); err != nil {
		return RemotePage{}, err
	}

	return RemotePage{
		ID:       resp.ID,
		SpaceID:  resp.SpaceID,
		ParentID: resp.ParentID,
		Title:    resp.Title,
		URL:      r.pageURL(resp.ID),
		Version:  resp.Version.Number,
		Body:     input.Body,
	}, nil
}

func (r *HTTPRemote) getPage(ctx context.Context, pageID string) (RemotePage, error) {
	var resp struct {
		ID       string `json:"id"`
		SpaceID  string `json:"spaceId"`
		ParentID string `json:"parentId"`
		Title    string `json:"title"`
		Version  struct {
			Number int `json:"number"`
		} `json:"version"`
	}
	if err := r.doJSON(ctx, http.MethodGet, "/wiki/api/v2/pages/"+pageID, nil, &resp); err != nil {
		return RemotePage{}, err
	}
	return RemotePage{
		ID:       resp.ID,
		SpaceID:  resp.SpaceID,
		ParentID: resp.ParentID,
		Title:    resp.Title,
		URL:      r.pageURL(resp.ID),
		Version:  resp.Version.Number,
	}, nil
}

func (r *HTTPRemote) doJSON(ctx context.Context, method, endpoint string, requestBody any, responseBody any) error {
	fullURL := r.baseURL + endpoint

	var body io.Reader
	if requestBody != nil {
		payload, err := json.Marshal(requestBody)
		if err != nil {
			return err
		}
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return err
	}
	req.SetBasicAuth(r.email, r.token)
	req.Header.Set("Accept", "application/json")
	if requestBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return ErrRemoteNotFound
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("remote request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	if responseBody == nil {
		return nil
	}

	return json.NewDecoder(resp.Body).Decode(responseBody)
}

func (r *HTTPRemote) pageURL(pageID string) string {
	return fmt.Sprintf("%s/wiki/pages/viewpage.action?pageId=%s", r.baseURL, pageID)
}
