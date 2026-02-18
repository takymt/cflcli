package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/takymt/cflcli/internal/config"
)

// Client is a minimal Confluence REST API v2 client.
type Client struct {
	baseURL    string
	httpClient *http.Client
	user       string
	token      string
}

// New creates a client from a profile and API token.
func New(profile *config.Profile, token string) (*Client, error) {
	if profile == nil {
		return nil, fmt.Errorf("profile is required")
	}
	if profile.Domain == "" {
		return nil, fmt.Errorf("profile domain is required")
	}
	if profile.User == "" {
		return nil, fmt.Errorf("profile user is required")
	}
	if token == "" {
		return nil, fmt.Errorf("CONFLUENCE_API_TOKEN is required")
	}

	base := strings.TrimSuffix(profile.Domain, "/")
	baseURL := fmt.Sprintf("https://%s/wiki/api/v2", base)

	return &Client{
		baseURL:    baseURL,
		httpClient: http.DefaultClient,
		user:       profile.User,
		token:      token,
	}, nil
}

// NewWithBaseURL creates a client with an explicit base URL (for tests).
func NewWithBaseURL(profile *config.Profile, token string, baseURL string) (*Client, error) {
	if profile == nil {
		return nil, fmt.Errorf("profile is required")
	}
	if profile.User == "" {
		return nil, fmt.Errorf("profile user is required")
	}
	if token == "" {
		return nil, fmt.Errorf("CONFLUENCE_API_TOKEN is required")
	}
	base := strings.TrimSuffix(baseURL, "/")
	return &Client{
		baseURL:    base,
		httpClient: http.DefaultClient,
		user:       profile.User,
		token:      token,
	}, nil
}

func (c *Client) do(req *http.Request, dest any) error {
	req.SetBasicAuth(c.user, c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	return json.NewDecoder(resp.Body).Decode(dest)
}

func (c *Client) get(path string, query url.Values, dest any) error {
	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return err
	}
	u.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	return c.do(req, dest)
}
