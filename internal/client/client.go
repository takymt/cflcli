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
	baseURL    string
	httpClient *http.Client
	user       string
	token      string
}

// HTTPError represents a non-2xx response from Confluence API.
type HTTPError struct {
	StatusCode int
	Status     string
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("request failed: %s: %s", e.Status, e.Body)
}

// DefaultHTTPClient is used when no custom client is provided.
var DefaultHTTPClient = http.DefaultClient

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
	if strings.HasPrefix(base, "http://") || strings.HasPrefix(base, "https://") {
		baseURL := fmt.Sprintf("%s/wiki/api/v2", base)
		return &Client{
			ctx:        ctx,
			baseURL:    baseURL,
			httpClient: DefaultHTTPClient,
			user:       profile.User,
			token:      token,
		}, nil
	}
	baseURL := fmt.Sprintf("https://%s/wiki/api/v2", base)

	return &Client{
		ctx:        ctx,
		baseURL:    baseURL,
		httpClient: DefaultHTTPClient,
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
		return &HTTPError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       strings.TrimSpace(string(body)),
		}
	}

	return decode(json.NewDecoder(resp.Body))
}

func (c *Client) get(path string, query url.Values, decode func(*json.Decoder) error) error {
	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return err
	}
	u.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(c.ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	return c.do(req, decode)
}
