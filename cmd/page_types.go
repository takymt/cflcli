package cmd

// PageListOptions holds options for page listing.
type PageListOptions struct {
	SpaceID         string
	SpaceKey        string
	Cursor          string
	Status          string
	Sort            string
	StatusSpecified bool
	Limit           int
}

type pageGetOptions struct {
	PageID string
}

type pageCreateOptions struct {
	Title      string
	BodyFile   string
	BodyFormat string
	ParentID   string
	SpaceID    string
	SpaceKey   string
}

type pageUpdateOptions struct {
	PageID     string
	Title      string
	BodyFile   string
	BodyFormat string
	ParentID   string
}

type pageDeleteOptions struct {
	PageID string
}

type pageBodyInput struct {
	StorageBody      string
	FrontMatterTitle string
}

var pageListAllowedStatuses = map[string]struct{}{
	"current":  {},
	"archived": {},
	"deleted":  {},
	"trashed":  {},
}

var pageListAllowedSorts = map[string]struct{}{
	"id":             {},
	"-id":            {},
	"created-date":   {},
	"-created-date":  {},
	"modified-date":  {},
	"-modified-date": {},
	"title":          {},
	"-title":         {},
}

const pageListAllowedSortValues = "id, -id, created-date, -created-date, modified-date, -modified-date, title, -title"
const pageBodyFormatValues = "markdown, storage"
