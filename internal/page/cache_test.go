package page

import "testing"

func TestMermaidCacheRoundTrip(t *testing.T) {
	t.Parallel()

	path := t.TempDir() + "/mermaid-cache.json"
	want := mermaidCache{
		Entries: map[string]mermaidCacheEntry{
			"mermaid-1.svg": {
				Source: "source-hash",
				File:   "file-hash",
			},
		},
	}

	if err := saveMermaidCache(path, want); err != nil {
		t.Fatalf("saveMermaidCache() error = %v", err)
	}

	got, err := loadMermaidCache(path)
	if err != nil {
		t.Fatalf("loadMermaidCache() error = %v", err)
	}
	if len(got.Entries) != 1 {
		t.Fatalf("len(got.Entries) = %d, want 1", len(got.Entries))
	}
	entry, ok := got.Entries["mermaid-1.svg"]
	if !ok {
		t.Fatal("got.Entries missing mermaid-1.svg")
	}
	if entry.Source != want.Entries["mermaid-1.svg"].Source {
		t.Fatalf("entry.Source = %q, want %q", entry.Source, want.Entries["mermaid-1.svg"].Source)
	}
	if entry.File != want.Entries["mermaid-1.svg"].File {
		t.Fatalf("entry.File = %q, want %q", entry.File, want.Entries["mermaid-1.svg"].File)
	}
}
