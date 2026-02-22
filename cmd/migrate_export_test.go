package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunMigrateExportWithConfig_RequiresOut(t *testing.T) {
	err := RunMigrateExportWithConfig(&bytes.Buffer{}, &migrateExportOptions{
		SpaceKey: "WORK",
	}, newPageListConfig("example.atlassian.net", "WORK"))
	if err == nil || !strings.Contains(err.Error(), "--out is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}
