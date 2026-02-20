//go:build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/takymt/cflcli/internal/client"
	"github.com/takymt/cflcli/internal/config"
)

func integrationProfile(t *testing.T) (*config.Profile, string) {
	t.Helper()

	domain := strings.TrimSpace(os.Getenv("CFL_IT_DOMAIN"))
	user := strings.TrimSpace(os.Getenv("CFL_IT_USER"))
	token := strings.TrimSpace(os.Getenv("CONFLUENCE_API_TOKEN"))

	if domain == "" || user == "" || token == "" {
		t.Skip("set CFL_IT_DOMAIN, CFL_IT_USER, and CONFLUENCE_API_TOKEN for integration tests")
	}

	profile := &config.Profile{
		Name:   "integration",
		Domain: domain,
		User:   user,
	}

	return profile, token
}

func cleanupCreatedPage(t *testing.T, profile *config.Profile, token, pageID string) {
	t.Helper()

	cleanupCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cleanupClient, err := client.New(cleanupCtx, profile, token)
	if err != nil {
		t.Fatalf("cleanup client.New: %v", err)
	}
	if err := cleanupClient.DeletePage(pageID); err != nil {
		t.Fatalf("cleanup DeletePage(%q): %v", pageID, err)
	}
}

func TestPageCreateSmoke(t *testing.T) {
	if os.Getenv("CFL_IT_ENABLE_CREATE") != "1" {
		t.Skip("set CFL_IT_ENABLE_CREATE=1 to run page create integration test")
	}

	profile, token := integrationProfile(t)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cli, err := client.New(ctx, profile, token)
	if err != nil {
		t.Fatalf("client.New: %v", err)
	}

	spaceID := strings.TrimSpace(os.Getenv("CFL_IT_SPACE_ID"))
	if spaceID == "" {
		t.Skip("set CFL_IT_SPACE_ID for page create integration test")
	}

	title := fmt.Sprintf("cfl-it-create-%d", time.Now().UnixNano())
	created, err := cli.CreatePage(spaceID, title, "<p>integration create smoke</p>", "")
	if err != nil {
		t.Fatalf("CreatePage: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("created page id is empty")
	}
	t.Cleanup(func() {
		cleanupCreatedPage(t, profile, token, created.ID)
	})
	if created.Title != title {
		t.Fatalf("title=%q want %q", created.Title, title)
	}
}

func TestPageUpdateSmoke(t *testing.T) {
	if os.Getenv("CFL_IT_ENABLE_UPDATE") != "1" {
		t.Skip("set CFL_IT_ENABLE_UPDATE=1 to run page update integration test")
	}

	profile, token := integrationProfile(t)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cli, err := client.New(ctx, profile, token)
	if err != nil {
		t.Fatalf("client.New: %v", err)
	}

	pageID := strings.TrimSpace(os.Getenv("CFL_IT_UPDATE_PAGE_ID"))
	if pageID == "" {
		t.Skip("set CFL_IT_UPDATE_PAGE_ID for page update integration test")
	}

	current, err := cli.GetPage(pageID)
	if err != nil {
		t.Fatalf("GetPage: %v", err)
	}
	if current.Version.Number < 1 {
		t.Fatalf("unexpected current version number: %d", current.Version.Number)
	}

	body := current.Body.Storage.Value
	if strings.TrimSpace(body) == "" {
		body = "<p>integration update smoke</p>"
	}

	updated, err := cli.UpdatePage(pageID, current.Title, body, "", current.Version.Number+1)
	if err != nil {
		t.Fatalf("UpdatePage: %v", err)
	}
	if updated.ID != pageID {
		t.Fatalf("id=%q want %q", updated.ID, pageID)
	}
}

func TestPageDeleteSmoke(t *testing.T) {
	if os.Getenv("CFL_IT_ENABLE_DELETE") != "1" {
		t.Skip("set CFL_IT_ENABLE_DELETE=1 to run page delete integration test")
	}

	profile, token := integrationProfile(t)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cli, err := client.New(ctx, profile, token)
	if err != nil {
		t.Fatalf("client.New: %v", err)
	}

	spaceID := strings.TrimSpace(os.Getenv("CFL_IT_SPACE_ID"))
	if spaceID == "" {
		t.Skip("set CFL_IT_SPACE_ID for page delete integration test")
	}

	title := fmt.Sprintf("cfl-it-delete-%d", time.Now().UnixNano())
	created, err := cli.CreatePage(spaceID, title, "<p>integration delete smoke</p>", "")
	if err != nil {
		t.Fatalf("CreatePage for delete smoke: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("created page id is empty")
	}

	if err := cli.DeletePage(created.ID); err != nil {
		t.Fatalf("DeletePage: %v", err)
	}
}
