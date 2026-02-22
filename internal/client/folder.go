package client

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// Folder represents a Confluence folder.
type Folder struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	ParentID   string `json:"parentId"`
	ParentType string `json:"parentType"`
	SpaceID    string `json:"spaceId"`
}

// GetFolder gets a folder by ID.
func (c *Client) GetFolder(folderID string) (*Folder, error) {
	folderID = strings.TrimSpace(folderID)
	if folderID == "" {
		return nil, fmt.Errorf("folder id is required")
	}

	var result Folder
	if err := c.get("/folders/"+url.PathEscape(folderID), url.Values{}, func(decoder *json.Decoder) error {
		return decoder.Decode(&result)
	}); err != nil {
		return nil, err
	}
	return &result, nil
}
