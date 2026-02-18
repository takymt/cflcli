package output

import "github.com/takymt/cflcli/internal/model"

// PageListOutput is used for JSON output with request metadata.
type PageListOutput struct {
	Request PageListRequest `json:"request"`
	Next    string          `json:"next"`
	Results []model.Page    `json:"results"`
}

// PageListRequest captures page list inputs for JSON output.
type PageListRequest struct {
	SpaceID string `json:"space_id,omitempty"`
	Limit   int    `json:"limit"`
}
