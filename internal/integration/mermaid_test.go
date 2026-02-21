//go:build integration

package integration

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/takymt/cflcli/internal/mermaid"
)

func TestMermaidRendererSmoke(t *testing.T) {
	if os.Getenv("CFL_IT_ENABLE_MERMAID") != "1" {
		t.Skip("set CFL_IT_ENABLE_MERMAID=1 to run mermaid renderer integration test")
	}

	renderer, err := mermaid.NewRenderer()
	if err != nil {
		t.Fatalf("mermaid.NewRenderer: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := renderer.Close(); closeErr != nil {
			t.Fatalf("renderer.Close: %v", closeErr)
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	svg, err := renderer.Render(ctx, strings.Join([]string{
		"flowchart TD",
		"  Start --> Check{is it working?}",
		"  Check -->|yes| End",
		"  Check -->|no| Fix",
		"  Fix --> End",
	}, "\n"))
	if err != nil {
		t.Fatalf("renderer.Render: %v", err)
	}

	output := string(svg)
	if !strings.Contains(output, "<svg") {
		t.Fatalf("rendered output must contain <svg: %q", output)
	}
	if strings.TrimSpace(output) == "" {
		t.Fatalf("rendered output is empty")
	}
}
