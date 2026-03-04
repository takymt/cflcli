package page

import (
	"context"
	"fmt"
	"sync"
)

type FakeRemote struct {
	mu        sync.Mutex
	nextID    int
	rootPages map[string]string
	pages     map[string]RemotePage
}

func NewFakeRemote() *FakeRemote {
	return &FakeRemote{
		nextID:    1000,
		rootPages: map[string]string{},
		pages:     map[string]RemotePage{},
	}
}

func (f *FakeRemote) SetRootPage(spaceID, pageID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rootPages[spaceID] = pageID
}

func (f *FakeRemote) SeedPage(page RemotePage) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pages[page.ID] = page
}

func (f *FakeRemote) ResolveRootPageID(_ context.Context, spaceID string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	pageID, ok := f.rootPages[spaceID]
	if !ok {
		return "", fmt.Errorf("unknown space root for %s", spaceID)
	}
	return pageID, nil
}

func (f *FakeRemote) PageExists(_ context.Context, spaceID, parentID, title string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, page := range f.pages {
		if page.SpaceID == spaceID && page.ParentID == parentID && page.Title == title {
			return true, nil
		}
	}
	return false, nil
}

func (f *FakeRemote) CreatePage(_ context.Context, input CreatePageInput) (RemotePage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id := fmt.Sprintf("%d", f.nextID)
	f.nextID++
	page := RemotePage{
		ID:       id,
		SpaceID:  input.SpaceID,
		ParentID: input.ParentID,
		Title:    input.Title,
		Body:     input.Body,
		URL:      fmt.Sprintf("https://example.atlassian.net/wiki/pages/viewpage.action?pageId=%s", id),
		Version:  1,
	}
	f.pages[id] = page
	return page, nil
}

func (f *FakeRemote) UpdatePage(_ context.Context, input UpdatePageInput) (RemotePage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	page, ok := f.pages[input.PageID]
	if !ok {
		return RemotePage{}, ErrRemoteNotFound
	}
	page.SpaceID = input.SpaceID
	page.ParentID = input.ParentID
	page.Title = input.Title
	page.Body = input.Body
	page.Version++
	f.pages[input.PageID] = page
	return page, nil
}
