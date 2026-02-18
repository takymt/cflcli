package client

import (
	"net/url"
	"strconv"

	"github.com/takymt/cflcli/internal/model"
)

// Page represents a page summary.
type Page = model.Page

// PageListResult represents the page list response.
type PageListResult = PageResult[Page]

// ListPages lists pages by space ID with pagination.
func (c *Client) ListPages(spaceID string, limit int, cursor string) (*PageListResult, error) {
	query := url.Values{}
	if spaceID != "" {
		query.Set("space-id", spaceID)
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	if cursor != "" {
		query.Set("cursor", cursor)
	}

	var result PageListResult
	if err := c.get("/pages", query, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
