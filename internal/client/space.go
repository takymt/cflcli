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
