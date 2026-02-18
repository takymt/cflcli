package client

// PageLinks contains pagination links from Confluence API.
type PageLinks struct {
	Next string `json:"next"`
}

// PageResult is a generic container for paginated responses.
type PageResult[T any] struct {
	Results []T       `json:"results"`
	Links   PageLinks `json:"_links"`
}
