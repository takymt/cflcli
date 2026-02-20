package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/takymt/cflcli/internal/model"
)

func TestWritePageListJSON(t *testing.T) {
	t.Parallel()

	payload := PageListOutput{
		Request: PageListRequest{SpaceID: "SPACE-1", Limit: 10},
		Next:    "CURSOR-2",
		Results: []model.Page{
			{ID: "1", Title: "Page", Status: "current"},
		},
	}

	var out bytes.Buffer
	if err := WritePageListJSON(&out, payload); err != nil {
		t.Fatalf("WritePageListJSON: %v", err)
	}

	var got PageListOutput
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if got.Request.SpaceID != "SPACE-1" || got.Request.Limit != 10 || got.Next != "CURSOR-2" || len(got.Results) != 1 {
		t.Fatalf("unexpected output: %+v", got)
	}
}

func TestWritePagesTable(t *testing.T) {
	t.Parallel()

	pages := []model.Page{
		{ID: "1", Title: "Doc", Status: "current"},
		{ID: "2", Title: "概要", Status: "archived"},
	}

	t.Run("with status", func(t *testing.T) {
		var out bytes.Buffer
		if err := WritePagesTable(&out, pages, true); err != nil {
			t.Fatalf("WritePagesTable: %v", err)
		}

		raw := out.String()
		lines := strings.Split(strings.TrimRight(raw, "\n"), "\n")
		if len(lines) < 4 {
			t.Fatalf("unexpected table lines: %q", raw)
		}
		if !strings.Contains(lines[1], "---") {
			t.Fatalf("missing header separator: %q", raw)
		}

		row1 := lines[2]
		row2 := lines[3]
		i1 := strings.Index(row1, "current")
		i2 := strings.Index(row2, "archived")
		if i1 < 0 || i2 < 0 {
			t.Fatalf("status cell not found: %q", raw)
		}
		prefixWidth1 := stringWidth(row1[:i1])
		prefixWidth2 := stringWidth(row2[:i2])
		if prefixWidth1 != prefixWidth2 {
			t.Fatalf("status column is not aligned: %d vs %d, raw=%q", prefixWidth1, prefixWidth2, raw)
		}
	})

	t.Run("without status", func(t *testing.T) {
		var out bytes.Buffer
		if err := WritePagesTable(&out, pages, false); err != nil {
			t.Fatalf("WritePagesTable: %v", err)
		}

		raw := out.String()
		if strings.Contains(raw, "STATUS") {
			t.Fatalf("unexpected status header: %q", raw)
		}
	})
}
