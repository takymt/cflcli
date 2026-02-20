package model

// Page represents a Confluence page summary.
type Page struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Status  string `json:"status"`
	SpaceID string `json:"spaceId"`
}

// PageVersion represents a page version.
type PageVersion struct {
	Number int `json:"number"`
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
	ID      string      `json:"id"`
	Title   string      `json:"title"`
	Status  string      `json:"status"`
	SpaceID string      `json:"spaceId"`
	Version PageVersion `json:"version"`
	Body    PageBody    `json:"body"`
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

// PageUpdateRequestVersion represents version payload for page update.
type PageUpdateRequestVersion struct {
	Number int `json:"number"`
}

// PageUpdateRequest represents page update request payload.
type PageUpdateRequest struct {
	ID       string                   `json:"id"`
	Status   string                   `json:"status"`
	Title    string                   `json:"title"`
	ParentID string                   `json:"parentId,omitempty"`
	Body     PageCreateBody           `json:"body"`
	Version  PageUpdateRequestVersion `json:"version"`
}
