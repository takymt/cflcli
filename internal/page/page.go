package page

import (
	"context"
	"errors"
)

// ErrNotFound indicates that the requested Confluence resource does not exist.
var ErrNotFound = errors.New("not found")

// Page is the observable remote state used by the CLI.
type Page struct {
	ID       string
	SpaceID  string
	ParentID string
	Title    string
	Body     string
	URL      string
}

// Client defines the Confluence operations required by the page commands.
type Client interface {
	SiteBaseURL() string
	ResolveSpaceIDByKey(ctx context.Context, spaceKey string) (string, error)
	ResolveSpaceRootPage(ctx context.Context, spaceID string) (string, error)
	PageExists(ctx context.Context, spaceID string, parentID string, title string) (bool, error)
	CreatePage(ctx context.Context, spaceID string, parentID string, title string, body string) (Page, error)
	UpdatePage(ctx context.Context, pageID string, title string, body string) (Page, error)
	PutAttachment(ctx context.Context, pageID string, filePath string) error
	DeleteAttachment(ctx context.Context, pageID string, filename string) error
}
