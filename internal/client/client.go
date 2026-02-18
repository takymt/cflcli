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
	ctx        context.Context
	baseUrl    string
	httpClient *http.Client
	user       string
	token      string
}

// New creates a client from a profile and API token.
func New(ctx context.Context, profile *config.Profile, token string) (*Client, error) {
	if profile == nil {
		return nil, fmt.Errorf("profile is required")
	}
	if ctx == nil {
		return nil, fmt.Errorf("context is required")
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
	baseUrl := fmt.Sprintf("https://%s/wiki/api/v2", base)

	return &Client{
		ctx:        ctx,
		baseUrl:    baseUrl,
		httpClient: http.DefaultClient,
		user:       profile.User,
		token:      token,
	}, nil
}

func (c *Client) do(req *http.Request, decode func(*json.Decoder) error) error {
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

	return decode(json.NewDecoder(resp.Body))
}

func (c *Client) get(path string, query url.Values, decode func(*json.Decoder) error) error {
	u, err := url.Parse(c.baseUrl + path)
	if err != nil {
		return err
	}
	u.RawQuery = query.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	req = req.WithContext(c.ctx)
	return c.do(req, decode)
}
