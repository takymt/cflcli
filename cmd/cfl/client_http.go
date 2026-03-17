package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/takymt/cflcli/internal/auth"
	"github.com/takymt/cflcli/internal/page"
)

type httpClient struct {
	siteBaseURL       string
	apiBaseURL        string
	attachmentBaseURL string
	authHeader        string
	http              *http.Client
}

type clientConfig struct {
	domain string
	email  string
	token  string
}

func loadClientConfig() (clientConfig, error) {
	creds, err := auth.ResolveRuntimeCredentials(auth.NewXDGConfigStore())
	if err != nil {
		return clientConfig{}, err
	}

	return clientConfig{
		domain: creds.Domain,
		email:  creds.Email,
		token:  creds.APIToken,
	}, nil
}

func newHTTPClient(cfg clientConfig, httpTransport *http.Client) (page.Client, error) {
	if cfg.domain == "" {
		return nil, errors.New("domain is required")
	}
	if cfg.email == "" {
		return nil, errors.New("email is required")
	}
	if cfg.token == "" {
		return nil, errors.New("token is required")
	}

	siteBaseURL := auth.SiteBaseURL(cfg.domain)
	apiBaseURL := siteBaseURL + "/wiki/api/v2"
	attachmentBaseURL := siteBaseURL + "/wiki/rest/api"
	if httpTransport == nil {
		httpTransport = &http.Client{}
	}

	return &httpClient{
		siteBaseURL:       siteBaseURL,
		apiBaseURL:        apiBaseURL,
		attachmentBaseURL: attachmentBaseURL,
		authHeader:        "Basic " + base64.StdEncoding.EncodeToString([]byte(cfg.email+":"+cfg.token)),
		http:              httpTransport,
	}, nil
}

func (c *httpClient) SiteBaseURL() string {
	return c.siteBaseURL
}

func (c *httpClient) ResolveSpaceIDByKey(ctx context.Context, spaceKey string) (string, error) {
	endpoint := c.apiBaseURL + "/spaces?keys=" + url.QueryEscape(spaceKey) + "&limit=1"
	var response struct {
		Results []struct {
			ID string `json:"id"`
		} `json:"results"`
	}
	if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, &response); err != nil {
		return "", err
	}
	if len(response.Results) == 0 || response.Results[0].ID == "" {
		return "", fmt.Errorf("space key %q not found", spaceKey)
	}
	return response.Results[0].ID, nil
}

func (c *httpClient) ResolveSpaceRootPage(ctx context.Context, spaceID string) (string, error) {
	var response struct {
		HomepageID string `json:"homepageId"`
	}

	if err := c.doJSON(ctx, http.MethodGet, c.apiBaseURL+"/spaces/"+spaceID, nil, &response); err != nil {
		return "", err
	}
	if response.HomepageID == "" {
		return "", errors.New("space homepage id was empty")
	}
	return response.HomepageID, nil
}

func (c *httpClient) PageExists(ctx context.Context, spaceID string, parentID string, title string) (bool, error) {
	nextURL := c.apiBaseURL + "/pages?limit=250&space-id=" + url.QueryEscape(spaceID) + "&title=" + url.QueryEscape(title)

	for nextURL != "" {
		var response struct {
			Results []struct {
				ID       string `json:"id"`
				ParentID string `json:"parentId"`
				Title    string `json:"title"`
			} `json:"results"`
			Links struct {
				Next string `json:"next"`
			} `json:"_links"`
		}

		if err := c.doJSON(ctx, http.MethodGet, nextURL, nil, &response); err != nil {
			return false, err
		}
		for _, result := range response.Results {
			if result.Title == title && result.ParentID == parentID {
				return true, nil
			}
		}
		nextURL = c.resolveNext(response.Links.Next)
	}

	return false, nil
}

func (c *httpClient) CreatePage(ctx context.Context, spaceID string, parentID string, title string, body string) (page.Page, error) {
	payload := map[string]any{
		"spaceId":  spaceID,
		"status":   "current",
		"title":    title,
		"parentId": parentID,
		"body": map[string]string{
			"representation": "storage",
			"value":          body,
		},
	}

	var response apiPage
	if err := c.doJSON(ctx, http.MethodPost, c.apiBaseURL+"/pages", payload, &response); err != nil {
		return page.Page{}, err
	}
	return c.toPage(&response), nil
}

func (c *httpClient) UpdatePage(ctx context.Context, pageID string, title string, body string) (page.Page, error) {
	current, err := c.getPage(ctx, pageID)
	if err != nil {
		return page.Page{}, err
	}

	payload := map[string]any{
		"id":       pageID,
		"status":   "current",
		"title":    title,
		"parentId": current.ParentID,
		"spaceId":  current.SpaceID,
		"body": map[string]string{
			"representation": "storage",
			"value":          body,
		},
		"version": map[string]int{
			"number": current.Version.Number + 1,
		},
	}

	var response apiPage
	if err := c.doJSON(ctx, http.MethodPut, c.apiBaseURL+"/pages/"+pageID, payload, &response); err != nil {
		return page.Page{}, err
	}
	return c.toPage(&response), nil
}

func (c *httpClient) PutAttachment(ctx context.Context, pageID string, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, file); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}

	endpoint := c.attachmentBaseURL + "/content/" + pageID + "/child/attachment"
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, &body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", c.authHeader)
	req.Header.Set("X-Atlassian-Token", "nocheck")
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	raw, readErr := io.ReadAll(resp.Body)
	closeErr := resp.Body.Close()
	if readErr != nil {
		if closeErr != nil {
			return fmt.Errorf("read response body: %w", errors.Join(readErr, closeErr))
		}
		return readErr
	}
	if closeErr != nil {
		return closeErr
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("confluence API PUT %s failed: %s", endpoint, strings.TrimSpace(string(raw)))
	}
	return nil
}

func (c *httpClient) DeleteAttachment(ctx context.Context, pageID string, filename string) error {
	attachmentID, err := c.findAttachmentID(ctx, pageID, filename)
	if err != nil {
		return err
	}
	endpoint := c.attachmentBaseURL + "/content/" + pageID + "/child/attachment/" + attachmentID
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", c.authHeader)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	raw, readErr := io.ReadAll(resp.Body)
	closeErr := resp.Body.Close()
	if readErr != nil {
		if closeErr != nil {
			return fmt.Errorf("read response body: %w", errors.Join(readErr, closeErr))
		}
		return readErr
	}
	if closeErr != nil {
		return closeErr
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("confluence API DELETE %s failed: %s", endpoint, strings.TrimSpace(string(raw)))
	}
	return nil
}

func (c *httpClient) findAttachmentID(ctx context.Context, pageID string, filename string) (string, error) {
	endpoint := c.attachmentBaseURL + "/content/" + pageID + "/child/attachment?filename=" + url.QueryEscape(filename) + "&limit=1"
	var response struct {
		Results []struct {
			ID string `json:"id"`
		} `json:"results"`
	}
	if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, &response); err != nil {
		return "", err
	}
	if len(response.Results) == 0 || response.Results[0].ID == "" {
		return "", fmt.Errorf("attachment %q not found on page %s", filename, pageID)
	}
	return response.Results[0].ID, nil
}

func (c *httpClient) getPage(ctx context.Context, pageID string) (apiPage, error) {
	var response apiPage
	err := c.doJSON(ctx, http.MethodGet, c.apiBaseURL+"/pages/"+pageID, nil, &response)
	return response, err
}

func (c *httpClient) doJSON(ctx context.Context, method string, endpoint string, payload any, out any) error {
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", c.authHeader)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusNotFound {
		closeErr := resp.Body.Close()
		if closeErr != nil {
			return closeErr
		}
		return page.ErrNotFound
	}

	raw, err := io.ReadAll(resp.Body)
	closeErr := resp.Body.Close()
	if err != nil {
		if closeErr != nil {
			return fmt.Errorf("read response body: %w", errors.Join(err, closeErr))
		}
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("confluence API %s %s failed: %s", method, endpoint, strings.TrimSpace(string(raw)))
	}
	if out == nil || len(raw) == 0 {
		return nil
	}

	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decode confluence response: %w", err)
	}
	return nil
}

func (c *httpClient) resolveNext(next string) string {
	if next == "" {
		return ""
	}
	if strings.HasPrefix(next, "http://") || strings.HasPrefix(next, "https://") {
		return next
	}
	if strings.HasPrefix(next, "/") {
		return c.siteBaseURL + next
	}
	return c.apiBaseURL + "/" + next
}

func (c *httpClient) toPage(api *apiPage) page.Page {
	pageURL := c.siteBaseURL + "/wiki/pages/viewpage.action?pageId=" + api.ID
	if api.Links.WebUI != "" {
		switch {
		case strings.HasPrefix(api.Links.WebUI, "http://"), strings.HasPrefix(api.Links.WebUI, "https://"):
			pageURL = api.Links.WebUI
		case strings.HasPrefix(api.Links.WebUI, "/wiki/"):
			pageURL = c.siteBaseURL + api.Links.WebUI
		case strings.HasPrefix(api.Links.WebUI, "/"):
			pageURL = c.siteBaseURL + "/wiki" + api.Links.WebUI
		default:
			pageURL = c.siteBaseURL + "/wiki/" + api.Links.WebUI
		}
	}

	return page.Page{
		ID:       api.ID,
		SpaceID:  api.SpaceID,
		ParentID: api.ParentID,
		Title:    api.Title,
		Body:     api.Body.Storage.Value,
		URL:      pageURL,
	}
}

type apiPage struct {
	ID       string `json:"id"`
	SpaceID  string `json:"spaceId"`
	ParentID string `json:"parentId"`
	Title    string `json:"title"`
	Version  struct {
		Number int `json:"number"`
	} `json:"version"`
	Body struct {
		Storage struct {
			Value string `json:"value"`
		} `json:"storage"`
	} `json:"body"`
	Links struct {
		WebUI string `json:"webui"`
	} `json:"_links"`
}
