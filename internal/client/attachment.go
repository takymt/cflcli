package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type attachmentListResult struct {
	Results []attachmentItem `json:"results"`
}

type attachmentItem struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	DownloadLink string `json:"downloadLink"`
	Links        struct {
		Download string `json:"download"`
	} `json:"_links"`
}

// UpsertPageAttachment creates or updates a page attachment by filename.
func (c *Client) UpsertPageAttachment(pageID, filename, sourcePath string) error {
	pageID = strings.TrimSpace(pageID)
	if pageID == "" {
		return fmt.Errorf("page id is required")
	}
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return fmt.Errorf("attachment filename is required")
	}
	sourcePath = strings.TrimSpace(sourcePath)
	if sourcePath == "" {
		return fmt.Errorf("attachment source path is required")
	}

	file, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open attachment source: %w", err)
	}
	defer func() { _ = file.Close() }()

	var payload bytes.Buffer
	writer := multipart.NewWriter(&payload)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return fmt.Errorf("create multipart file field: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return fmt.Errorf("write multipart file body: %w", err)
	}
	if err := writer.WriteField("minorEdit", "true"); err != nil {
		return fmt.Errorf("write multipart field: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close multipart writer: %w", err)
	}

	u, err := url.Parse(c.restAPIBaseURL() + "/content/" + url.PathEscape(pageID) + "/child/attachment")
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(c.ctx, http.MethodPut, u.String(), &payload)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Atlassian-Token", "no-check")

	return c.do(req, func(decoder *json.Decoder) error {
		var body any
		return decoder.Decode(&body)
	})
}

// DownloadPageAttachmentByFilename downloads a page attachment binary by filename.
func (c *Client) DownloadPageAttachmentByFilename(pageID, filename string) ([]byte, error) {
	pageID = strings.TrimSpace(pageID)
	if pageID == "" {
		return nil, fmt.Errorf("page id is required")
	}
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return nil, fmt.Errorf("attachment filename is required")
	}

	listURL, err := url.Parse(c.baseURL + "/pages/" + url.PathEscape(pageID) + "/attachments")
	if err != nil {
		return nil, err
	}
	query := listURL.Query()
	query.Set("filename", filename)
	query.Set("status", "current")
	query.Set("limit", "1")
	listURL.RawQuery = query.Encode()

	listReq, err := http.NewRequestWithContext(c.ctx, http.MethodGet, listURL.String(), nil)
	if err != nil {
		return nil, err
	}

	var listResult attachmentListResult
	if err := c.do(listReq, func(decoder *json.Decoder) error {
		return decoder.Decode(&listResult)
	}); err != nil {
		return nil, err
	}
	if len(listResult.Results) == 0 {
		return nil, fmt.Errorf("attachment %q not found on page %q", filename, pageID)
	}

	downloadPath := strings.TrimSpace(listResult.Results[0].DownloadLink)
	if downloadPath == "" {
		downloadPath = strings.TrimSpace(listResult.Results[0].Links.Download)
	}
	if downloadPath == "" {
		return nil, fmt.Errorf("attachment %q has no download link", filename)
	}

	downloadURLs := c.resolveDownloadURLCandidates(downloadPath)
	var lastErr error
	for _, downloadURL := range downloadURLs {
		downloadReq, reqErr := http.NewRequestWithContext(c.ctx, http.MethodGet, downloadURL, nil)
		if reqErr != nil {
			lastErr = reqErr
			continue
		}

		content, dlErr := c.doRaw(downloadReq, "*/*")
		if dlErr == nil {
			return content, nil
		}

		var httpErr *HTTPError
		if !errors.As(dlErr, &httpErr) || httpErr.StatusCode != http.StatusNotFound {
			return nil, dlErr
		}
		lastErr = dlErr
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("attachment %q download link is invalid", filename)
}

func (c *Client) resolveDownloadURLCandidates(path string) []string {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	if strings.HasPrefix(path, "https://") || strings.HasPrefix(path, "http://") {
		return []string{path}
	}

	var candidates []string
	switch {
	case strings.HasPrefix(path, "/wiki/"):
		candidates = append(candidates, c.domainBaseURL()+path)
	case strings.HasPrefix(path, "/"):
		candidates = append(candidates, c.domainBaseURL()+path, c.siteBaseURL()+path)
	default:
		candidates = append(candidates, c.domainBaseURL()+"/"+path, c.siteBaseURL()+"/"+path)
	}

	seen := map[string]struct{}{}
	unique := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		unique = append(unique, candidate)
	}
	return unique
}

func (c *Client) restAPIBaseURL() string {
	const v2Suffix = "/wiki/api/v2"
	if strings.HasSuffix(c.baseURL, v2Suffix) {
		return strings.TrimSuffix(c.baseURL, v2Suffix) + "/wiki/rest/api"
	}
	return c.baseURL
}

func (c *Client) siteBaseURL() string {
	const v2Suffix = "/wiki/api/v2"
	if strings.HasSuffix(c.baseURL, v2Suffix) {
		return strings.TrimSuffix(c.baseURL, v2Suffix)
	}
	return strings.TrimSuffix(c.baseURL, "/")
}

func (c *Client) domainBaseURL() string {
	const v2Suffix = "/wiki/api/v2"
	if strings.HasSuffix(c.baseURL, v2Suffix) {
		return strings.TrimSuffix(strings.TrimSuffix(c.baseURL, v2Suffix), "/")
	}
	return strings.TrimSuffix(c.baseURL, "/")
}
