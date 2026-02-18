package cmd

import (
	"bytes"
	"errors"
	"testing"

	"github.com/takymt/cflcli/internal/client"
	"github.com/takymt/cflcli/internal/config"
	"github.com/takymt/cflcli/internal/model"
)

type fakePageClient struct {
	result *client.PageListResult
	err    error

	spaceID string
	limit   int
	cursor  string
}

func (f *fakePageClient) ListPages(spaceID string, limit int, cursor string) (*client.PageListResult, error) {
	f.spaceID = spaceID
	f.limit = limit
	f.cursor = cursor
	return f.result, f.err
}

func TestRunPageList_LimitValidation(t *testing.T) {
	cfg := &config.Config{
		Current:  "work",
		Profiles: []config.Profile{{Name: "work", Domain: "example.atlassian.net", User: "user@example.com"}},
	}

	opts := &pageListOptions{limit: 0}
	err := runPageListWithConfig(&bytes.Buffer{}, opts, cfg)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestRunPageList_ProfileMissing(t *testing.T) {
	cfg := &config.Config{}
	opts := &pageListOptions{limit: 10}

	err := runPageListWithConfig(&bytes.Buffer{}, opts, cfg)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestRunPageList_ProfileFlagNotFound(t *testing.T) {
	cfg := &config.Config{
		Current:  "work",
		Profiles: []config.Profile{{Name: "work", Domain: "example.atlassian.net", User: "user@example.com"}},
	}

	prevProfile := profileFlag
	profileFlag = "missing"
	defer func() { profileFlag = prevProfile }()

	opts := &pageListOptions{limit: 10}
	err := runPageListWithConfig(&bytes.Buffer{}, opts, cfg)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestRunPageList_Success(t *testing.T) {
	cfg := &config.Config{
		Current:  "work",
		Profiles: []config.Profile{{Name: "work", Domain: "example.atlassian.net", User: "user@example.com"}},
	}

	fake := &fakePageClient{
		result: &client.PageListResult{Results: []model.Page{{ID: "1"}}},
	}
	prev := newClient
	prevOutput := outputFlag
	prevProfile := profileFlag
	newClient = func(_ *config.Profile, _ string) (pageLister, error) {
		return fake, nil
	}
	outputFlag = "json"
	profileFlag = ""
	defer func() {
		newClient = prev
		outputFlag = prevOutput
		profileFlag = prevProfile
	}()

	opts := &pageListOptions{spaceID: "SPACE", limit: 10}
	output := &bytes.Buffer{}
	if err := runPageListWithConfig(output, opts, cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output.Len() == 0 {
		t.Fatalf("expected output")
	}
	if fake.spaceID != "SPACE" || fake.limit != 10 {
		t.Fatalf("unexpected params: %v %v", fake.spaceID, fake.limit)
	}
}

func TestRunPageList_ClientError(t *testing.T) {
	cfg := &config.Config{
		Current:  "work",
		Profiles: []config.Profile{{Name: "work", Domain: "example.atlassian.net", User: "user@example.com"}},
	}

	fake := &fakePageClient{err: errors.New("boom")}
	prev := newClient
	prevOutput := outputFlag
	prevProfile := profileFlag
	newClient = func(_ *config.Profile, _ string) (pageLister, error) {
		return fake, nil
	}
	outputFlag = "json"
	profileFlag = ""
	defer func() {
		newClient = prev
		outputFlag = prevOutput
		profileFlag = prevProfile
	}()

	opts := &pageListOptions{limit: 10}
	if err := runPageListWithConfig(&bytes.Buffer{}, opts, cfg); err == nil {
		t.Fatalf("expected error")
	}
}

func TestRunPageList_CustomProfile(t *testing.T) {
	cfg := &config.Config{
		Current:  "work",
		Profiles: []config.Profile{{Name: "work", Domain: "example.atlassian.net", User: "user@example.com"}, {Name: "other", Domain: "example.atlassian.net", User: "other@example.com"}},
	}

	fake := &fakePageClient{result: &client.PageListResult{Results: []model.Page{{ID: "1"}}}}
	prev := newClient
	prevOutput := outputFlag
	prevProfile := profileFlag
	newClient = func(_ *config.Profile, _ string) (pageLister, error) {
		return fake, nil
	}
	outputFlag = "json"
	profileFlag = "other"
	defer func() {
		newClient = prev
		outputFlag = prevOutput
		profileFlag = prevProfile
	}()

	opts := &pageListOptions{limit: 10}
	if err := runPageListWithConfig(&bytes.Buffer{}, opts, cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
