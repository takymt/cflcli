package page

import "context"

type RemotePage struct {
	ID       string
	SpaceID  string
	ParentID string
	Title    string
	Body     string
	URL      string
	Version  int
}

type CreatePageInput struct {
	SpaceID  string
	ParentID string
	Title    string
	Body     string
}

type UpdatePageInput struct {
	PageID   string
	SpaceID  string
	ParentID string
	Title    string
	Body     string
}

type Remote interface {
	ResolveRootPageID(ctx context.Context, spaceID string) (string, error)
	PageExists(ctx context.Context, spaceID, parentID, title string) (bool, error)
	CreatePage(ctx context.Context, input CreatePageInput) (RemotePage, error)
	UpdatePage(ctx context.Context, input UpdatePageInput) (RemotePage, error)
}

type Result struct {
	Action string
	Page   RemotePage
}
