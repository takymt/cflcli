package page

import (
	"context"
	"errors"
	"fmt"
	"os"
)

type Service struct {
	Remote Remote
}

func (s Service) NewPage(ctx context.Context, path, spaceID, parentID string) (Result, error) {
	if _, err := os.Stat(path); err == nil {
		return Result{}, ErrFileAlreadyExists
	} else if !errors.Is(err, os.ErrNotExist) {
		return Result{}, err
	}

	title := TitleFromPath(path)
	resolvedParentID := parentID
	var err error
	if resolvedParentID == "" {
		resolvedParentID, err = s.Remote.ResolveRootPageID(ctx, spaceID)
		if err != nil {
			return Result{}, err
		}
	}

	exists, err := s.Remote.PageExists(ctx, spaceID, resolvedParentID, title)
	if err != nil {
		return Result{}, err
	}
	if exists {
		return Result{}, ErrDuplicatePage
	}

	page, err := s.Remote.CreatePage(ctx, CreatePageInput{
		SpaceID:  spaceID,
		ParentID: resolvedParentID,
		Title:    title,
		Body:     "",
	})
	if err != nil {
		return Result{}, err
	}

	if err := WriteNewDocument(path, page.SpaceID, page.ID, page.ParentID); err != nil {
		return Result{}, fmt.Errorf("write document: %w", err)
	}

	return Result{Action: "created", Page: page}, nil
}

func (s Service) SyncPage(ctx context.Context, path string) (Result, error) {
	doc, err := ParseDocument(path)
	if err != nil {
		return Result{}, err
	}

	body, err := ConvertMarkdown(doc.Body)
	if err != nil {
		return Result{}, err
	}

	page, err := s.Remote.UpdatePage(ctx, UpdatePageInput{
		PageID:   doc.PageID,
		SpaceID:  doc.SpaceID,
		ParentID: doc.ParentID,
		Title:    doc.Title,
		Body:     body,
	})
	if err != nil {
		return Result{}, err
	}

	return Result{Action: "updated", Page: page}, nil
}
