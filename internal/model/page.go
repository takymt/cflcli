package model

// Page represents a Confluence page summary.
type Page struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
	Space  struct {
		ID string `json:"id"`
	} `json:"space"`
}
