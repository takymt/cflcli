package client

import (
	"encoding/json"
	"fmt"
	"net/url"
)

type spaceBulk struct {
	ID  string `json:"id"`
	Key string `json:"key"`
}

type spaceListResult struct {
	Results []spaceBulk `json:"results"`
}

type spaceDetail struct {
	ID  string `json:"id"`
	Key string `json:"key"`
}

// ResolveSpaceIDByKey resolves a space key to a unique space ID.
func (c *Client) ResolveSpaceIDByKey(spaceKey string) (string, error) {
	query := url.Values{}
	query.Set("keys", spaceKey)
	query.Set("limit", "2")

	var result spaceListResult
	if err := c.get("/spaces", query, func(decoder *json.Decoder) error {
		return decoder.Decode(&result)
	}); err != nil {
		return "", err
	}

	switch len(result.Results) {
	case 0:
		return "", fmt.Errorf("space key %q not found", spaceKey)
	case 1:
		return result.Results[0].ID, nil
	default:
		return "", fmt.Errorf("space key %q resolved to multiple spaces", spaceKey)
	}
}

// ResolveSpaceKeyByID resolves a space ID to its space key.
func (c *Client) ResolveSpaceKeyByID(spaceID string) (string, error) {
	if spaceID == "" {
		return "", fmt.Errorf("space id is required")
	}

	var result spaceDetail
	if err := c.get("/spaces/"+url.PathEscape(spaceID), url.Values{}, func(decoder *json.Decoder) error {
		return decoder.Decode(&result)
	}); err != nil {
		return "", err
	}
	if result.Key == "" {
		return "", fmt.Errorf("space id %q returned empty space key", spaceID)
	}

	return result.Key, nil
}
