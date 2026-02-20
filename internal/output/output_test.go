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
	}

	t.Run("with status", func(t *testing.T) {
		var out bytes.Buffer
		if err := WritePagesTable(&out, pages, true); err != nil {
			t.Fatalf("WritePagesTable: %v", err)
		}

		raw := out.String()
		if !strings.Contains(raw, "STATUS") || !strings.Contains(raw, "current") {
			t.Fatalf("missing status values: %q", raw)
		}
	})
}
