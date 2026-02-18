package client

import (
	"context"
	"testing"

	"github.com/takymt/cflcli/internal/config"
)

func TestNew_Validation(t *testing.T) {
	profile := &config.Profile{Domain: "example.atlassian.net", User: "user@example.com"}
	if _, err := New(context.Background(), profile, ""); err == nil {
		t.Fatalf("expected error for missing token")
	}
}
