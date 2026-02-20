package client

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/takymt/cflcli/internal/model"
)

// Page represents a page summary.
type Page = model.Page

// PageListResult represents the page list response.
type PageListResult = PageResult[Page]

// PageDetail represents a page detail response.
type PageDetail = model.PageDetail

// ListPages lists pages by space ID with pagination.
func (c *Client) ListPages(spaceID string, limit int, cursor string, statuses []string, sort string) (*PageListResult, error) {
	query := url.Values{}
	if spaceID != "" {
		query.Set("space-id", spaceID)
	}
	for _, status := range statuses {
		if status != "" {
			query.Add("status", status)
		}
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	if cursor != "" {
		query.Set("cursor", cursor)
	}
	if sort != "" {
		query.Set("sort", sort)
	}

	var result PageListResult
	if err := c.get("/pages", query, func(decoder *json.Decoder) error {
		return decoder.Decode(&result)
	}); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreatePage creates a page in storage format.
func (c *Client) CreatePage(spaceID, title, body, parentID string) (*Page, error) {
	spaceID = strings.TrimSpace(spaceID)
	if spaceID == "" {
		return nil, fmt.Errorf("space id is required")
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}

	req := model.PageCreateRequest{
		SpaceID: spaceID,
		Status:  "current",
		Title:   title,
		Body: model.PageCreateBody{
			Storage: model.BodyType{
				Representation: "storage",
				Value:          body,
			},
		},
	}
	if trimmedParentID := strings.TrimSpace(parentID); trimmedParentID != "" {
		req.ParentID = trimmedParentID
	}

	var result Page
	if err := c.post("/pages", url.Values{}, req, func(decoder *json.Decoder) error {
		return decoder.Decode(&result)
	}); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetPage gets a page by ID in storage body format.
func (c *Client) GetPage(pageID string) (*PageDetail, error) {
	pageID = strings.TrimSpace(pageID)
	if pageID == "" {
		return nil, fmt.Errorf("page id is required")
	}

	query := url.Values{}
	query.Set("body-format", "storage")

	var result PageDetail
	if err := c.get("/pages/"+url.PathEscape(pageID), query, func(decoder *json.Decoder) error {
		return decoder.Decode(&result)
	}); err != nil {
		return nil, err
	}
	return &result, nil
}
