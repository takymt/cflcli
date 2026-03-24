package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestProgressLinePrintln_OverwritesActiveProgressWithoutClearPadding(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	progress := &progressLine{writer: &buf, enabled: true}

	progress.Set("Rendering Mermaid...")
	progress.Println(`Warning: skipped absolute local path "/docs/images/emojis/smile.svg"`)

	got := buf.String()
	if strings.Contains(got, "\r"+strings.Repeat(" ", len("Rendering Mermaid..."))+"\rWarning:") {
		t.Fatalf("Println() output = %q, must not clear with padding before warning", got)
	}
	if !strings.Contains(got, "\rWarning: skipped absolute local path \"/docs/images/emojis/smile.svg\"") {
		t.Fatalf("Println() output = %q, want warning line", got)
	}
	if progress.active {
		t.Fatal("Println() must clear active progress state")
	}
}
