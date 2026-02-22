//go:build integration

package integration

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/takymt/cflcli/internal/client"
)

func TestAttachmentDownloadSmoke(t *testing.T) {
	if os.Getenv("CFL_IT_ENABLE_ATTACHMENT_DOWNLOAD") != "1" {
		t.Skip("set CFL_IT_ENABLE_ATTACHMENT_DOWNLOAD=1 to run attachment download integration test")
	}

	profile, token := integrationProfile(t)

	pageID := strings.TrimSpace(os.Getenv("CFL_IT_ATTACHMENT_PAGE_ID"))
	filename := strings.TrimSpace(os.Getenv("CFL_IT_ATTACHMENT_FILENAME"))
	if pageID == "" || filename == "" {
		t.Skip("set CFL_IT_ATTACHMENT_PAGE_ID and CFL_IT_ATTACHMENT_FILENAME for attachment download integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cli, err := client.New(ctx, profile, token)
	if err != nil {
		t.Fatalf("client.New: %v", err)
	}

	content, err := cli.DownloadPageAttachmentByFilename(pageID, filename)
	if err != nil {
		t.Fatalf("DownloadPageAttachmentByFilename(%q,%q): %v", pageID, filename, err)
	}
	if len(content) == 0 {
		t.Fatalf("downloaded attachment %q from page %q is empty", filename, pageID)
	}
}
