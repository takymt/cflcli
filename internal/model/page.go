package model

// Page represents a Confluence page summary.
type Page struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Status  string `json:"status"`
	SpaceID string `json:"spaceId"`
}

// BodyType represents one body representation payload.
type BodyType struct {
	Representation string `json:"representation"`
	Value          string `json:"value"`
}

// PageBody contains page body payloads by representation.
type PageBody struct {
	Storage BodyType `json:"storage"`
}

// PageDetail represents a page detail response.
type PageDetail struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Status  string   `json:"status"`
	SpaceID string   `json:"spaceId"`
	Body    PageBody `json:"body"`
}

// PageCreateBody represents create request body payload in storage format.
type PageCreateBody struct {
	Storage BodyType `json:"storage"`
}

// PageCreateRequest represents page create request payload.
type PageCreateRequest struct {
	SpaceID  string         `json:"spaceId"`
	Status   string         `json:"status"`
	Title    string         `json:"title"`
	ParentID string         `json:"parentId,omitempty"`
	Body     PageCreateBody `json:"body"`
}
