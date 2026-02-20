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
	if created.Title != title {
		t.Fatalf("title=%q want %q", created.Title, title)
	}
}
