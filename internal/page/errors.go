package page

import "errors"

var (
	ErrFileAlreadyExists  = errors.New("file already exists")
	ErrMissingFrontmatter = errors.New("missing YAML frontmatter")
	ErrInvalidFrontmatter = errors.New("invalid frontmatter")
	ErrDuplicatePage      = errors.New("page with the same title already exists under the target parent")
	ErrRemoteNotFound     = errors.New("remote page not found")
)
