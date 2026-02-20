//go:build integration

package integration

import (
	"context"
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

func TestPageListSmoke(t *testing.T) {
	profile, token := integrationProfile(t)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cli, err := client.New(ctx, profile, token)
	if err != nil {
		t.Fatalf("client.New: %v", err)
	}

	spaceID := strings.TrimSpace(os.Getenv("CFL_IT_SPACE_ID"))
	result, err := cli.ListPages(spaceID, 1, "", []string{"current"}, "id")
	if err != nil {
		t.Fatalf("ListPages with sort=id: %v", err)
	}
	if len(result.Results) > 1 {
		t.Fatalf("expected at most one result, got %d", len(result.Results))
	}
}

func TestPageGetSmoke(t *testing.T) {
	profile, token := integrationProfile(t)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cli, err := client.New(ctx, profile, token)
	if err != nil {
		t.Fatalf("client.New: %v", err)
	}

	spaceID := strings.TrimSpace(os.Getenv("CFL_IT_SPACE_ID"))
	list, err := cli.ListPages(spaceID, 1, "", []string{"current"}, "id")
	if err != nil {
		t.Fatalf("ListPages: %v", err)
	}
	if len(list.Results) == 0 {
		t.Skip("no current pages available for page get smoke test")
	}

	pageID := list.Results[0].ID
	page, err := cli.GetPage(pageID)
	if err != nil {
		t.Fatalf("GetPage: %v", err)
	}
	if page.ID != pageID {
		t.Fatalf("id=%q want %q", page.ID, pageID)
	}
	if page.Body.Storage.Representation != "" && page.Body.Storage.Representation != "storage" {
		t.Fatalf("unexpected body representation: %q", page.Body.Storage.Representation)
	}
}
