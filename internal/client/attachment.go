package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
)

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

func (c *Client) restAPIBaseURL() string {
	const v2Suffix = "/wiki/api/v2"
	if strings.HasSuffix(c.baseURL, v2Suffix) {
		return strings.TrimSuffix(c.baseURL, v2Suffix) + "/wiki/rest/api"
	}
	return c.baseURL
}
