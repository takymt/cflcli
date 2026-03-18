package page

import (
	"testing"
)

func TestParseMarkdownFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		wantFM    Frontmatter
		wantBody  string
		wantError bool
	}{
		{
			name: "valid frontmatter",
			input: "---\n" +
				"title: Hello\n" +
				"space-key: TEST\n" +
				"page-id: 200\n" +
				"parent-id: 300\n" +
				"draft: true\n" +
				"---\n" +
				"# hello\n",
			wantFM: Frontmatter{
				Title:    "Hello",
				SpaceKey: "TEST",
				PageID:   "200",
				ParentID: "300",
				Draft:    true,
			},
			wantBody: "# hello\n",
		},
		{
			name: "missing draft defaults false",
			input: "---\n" +
				"title: Hello\n" +
				"space-key: TEST\n" +
				"page-id: 200\n" +
				"parent-id: 300\n" +
				"---\n" +
				"# hello\n",
			wantFM: Frontmatter{
				Title:    "Hello",
				SpaceKey: "TEST",
				PageID:   "200",
				ParentID: "300",
				Draft:    false,
			},
			wantBody: "# hello\n",
		},
		{
			name:      "missing frontmatter",
			input:     "# hello\n",
			wantError: true,
		},
		{
			name: "missing required key",
			input: "---\n" +
				"title: Hello\n" +
				"space-key: TEST\n" +
				"page-id: 200\n" +
				"---\n",
			wantError: true,
		},
		{
			name: "unknown key",
			input: "---\n" +
				"title: Hello\n" +
				"space-key: TEST\n" +
				"page-id: 200\n" +
				"parent-id: 300\n" +
				"author: hello\n" +
				"---\n",
			wantError: true,
		},
		{
			name: "missing title",
			input: "---\n" +
				"space-key: TEST\n" +
				"page-id: 200\n" +
				"parent-id: 300\n" +
				"---\n",
			wantError: true,
		},
		{
			name: "malformed yaml",
			input: "---\n" +
				"title: Hello\n" +
				"space-key: [TEST\n" +
				"page-id: 200\n" +
				"parent-id: 300\n" +
				"---\n",
			wantError: true,
		},
		{
			name: "invalid space key",
			input: "---\n" +
				"title: Hello\n" +
				"space-key: invalid key\n" +
				"page-id: 200\n" +
				"parent-id: 300\n" +
				"---\n",
			wantError: true,
		},
		{
			name: "invalid draft type",
			input: "---\n" +
				"title: Hello\n" +
				"space-key: TEST\n" +
				"page-id: 200\n" +
				"parent-id: 300\n" +
				"draft: nope\n" +
				"---\n",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotFM, gotBody, err := ParseMarkdownFile([]byte(tt.input))
			if tt.wantError {
				if err == nil {
					t.Fatal("ParseMarkdownFile() error = nil, want non-nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("ParseMarkdownFile() error = %v", err)
			}

			if gotFM != tt.wantFM {
				t.Fatalf("ParseMarkdownFile() frontmatter = %#v, want %#v", gotFM, tt.wantFM)
			}

			if gotBody != tt.wantBody {
				t.Fatalf("ParseMarkdownFile() body = %q, want %q", gotBody, tt.wantBody)
			}
		})
	}
}

func TestFormatMarkdownFile(t *testing.T) {
	t.Parallel()

	got := string(FormatMarkdownFile(Frontmatter{
		Title:    "Hello",
		SpaceKey: "TEST",
		PageID:   "200",
		ParentID: "300",
		Draft:    false,
	}, ""))

	want := "---\ntitle: Hello\nspace-key: TEST\npage-id: 200\nparent-id: 300\ndraft: false\n---\n"
	if got != want {
		t.Fatalf("FormatMarkdownFile() = %q, want %q", got, want)
	}
}
