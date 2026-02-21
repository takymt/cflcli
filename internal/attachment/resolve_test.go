package attachment

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveMarkdownImageAssets(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name                string
		setup               func(t *testing.T, root string) (bodyFile string, assetsRoot string, storage string, wantAssets []Asset)
		wantErrSub          string
		wantStorageContains []string
	}

	testCases := []testCase{
		{
			name: "relative local path converts to attachment",
			setup: func(t *testing.T, root string) (string, string, string, []Asset) {
				t.Helper()

				bodyDir := filepath.Join(root, "docs")
				if err := os.MkdirAll(filepath.Join(bodyDir, "img"), 0o755); err != nil {
					t.Fatalf("MkdirAll: %v", err)
				}
				bodyFile := filepath.Join(bodyDir, "page.md")
				imagePath := filepath.Join(bodyDir, "img", "logo.png")
				if err := os.WriteFile(imagePath, []byte("PNG"), 0o600); err != nil {
					t.Fatalf("WriteFile(image): %v", err)
				}

				return bodyFile, "", `<ac:image><ri:url ri:value="./img/logo.png" /></ac:image>`, []Asset{{
					Filename:   "logo.png",
					SourcePath: imagePath,
				}}
			},
			wantStorageContains: []string{`<ri:attachment ri:filename="logo.png" />`},
		},
		{
			name: "root-prefixed path resolves from assets root",
			setup: func(t *testing.T, root string) (string, string, string, []Asset) {
				t.Helper()

				bodyDir := filepath.Join(root, "docs")
				assetsRoot := filepath.Join(root, "assets")
				if err := os.MkdirAll(filepath.Join(assetsRoot, "images"), 0o755); err != nil {
					t.Fatalf("MkdirAll: %v", err)
				}
				bodyFile := filepath.Join(bodyDir, "page.md")
				imagePath := filepath.Join(assetsRoot, "images", "root.png")
				if err := os.WriteFile(imagePath, []byte("ROOT"), 0o600); err != nil {
					t.Fatalf("WriteFile(image): %v", err)
				}

				return bodyFile, assetsRoot, `<ac:image><ri:url ri:value="/images/root.png" /></ac:image>`, []Asset{{
					Filename:   "root.png",
					SourcePath: imagePath,
				}}
			},
			wantStorageContains: []string{`<ri:attachment ri:filename="root.png" />`},
		},
		{
			name: "remote url is unchanged",
			setup: func(t *testing.T, root string) (string, string, string, []Asset) {
				t.Helper()
				bodyFile := filepath.Join(root, "docs", "page.md")
				return bodyFile, "", `<ac:image><ri:url ri:value="https://example.com/logo.png" /></ac:image>`, nil
			},
			wantStorageContains: []string{`<ri:url ri:value="https://example.com/logo.png" />`},
		},
		{
			name: "missing local file returns error",
			setup: func(t *testing.T, root string) (string, string, string, []Asset) {
				t.Helper()
				bodyFile := filepath.Join(root, "docs", "page.md")
				return bodyFile, "", `<ac:image><ri:url ri:value="./img/missing.png" /></ac:image>`, nil
			},
			wantErrSub: "resolve local image",
		},
		{
			name: "duplicate filename returns error",
			setup: func(t *testing.T, root string) (string, string, string, []Asset) {
				t.Helper()

				bodyDir := filepath.Join(root, "docs")
				assetsRoot := filepath.Join(root, "assets")
				if err := os.MkdirAll(filepath.Join(bodyDir, "img"), 0o755); err != nil {
					t.Fatalf("MkdirAll: %v", err)
				}
				if err := os.MkdirAll(filepath.Join(assetsRoot, "sub"), 0o755); err != nil {
					t.Fatalf("MkdirAll: %v", err)
				}
				if err := os.WriteFile(filepath.Join(bodyDir, "img", "logo.png"), []byte("A"), 0o600); err != nil {
					t.Fatalf("WriteFile(local): %v", err)
				}
				if err := os.WriteFile(filepath.Join(assetsRoot, "sub", "logo.png"), []byte("B"), 0o600); err != nil {
					t.Fatalf("WriteFile(root): %v", err)
				}

				storage := strings.Join([]string{
					`<ac:image><ri:url ri:value="./img/logo.png" /></ac:image>`,
					`<ac:image><ri:url ri:value="/sub/logo.png" /></ac:image>`,
				}, "")
				return filepath.Join(bodyDir, "page.md"), assetsRoot, storage, nil
			},
			wantErrSub: "duplicate image filename",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			bodyFile, assetsRoot, storage, wantAssets := tc.setup(t, root)

			gotStorage, gotAssets, err := ResolveMarkdownImageAssets(storage, bodyFile, assetsRoot)
			if tc.wantErrSub != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErrSub) {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveMarkdownImageAssets: %v", err)
			}

			for _, token := range tc.wantStorageContains {
				if !strings.Contains(gotStorage, token) {
					t.Fatalf("storage missing %q in %q", token, gotStorage)
				}
			}
			if len(gotAssets) != len(wantAssets) {
				t.Fatalf("asset count=%d want %d (%+v)", len(gotAssets), len(wantAssets), gotAssets)
			}
			for i := range wantAssets {
				if gotAssets[i].Filename != wantAssets[i].Filename {
					t.Fatalf("asset[%d].Filename=%q want %q", i, gotAssets[i].Filename, wantAssets[i].Filename)
				}
				if filepath.Clean(gotAssets[i].SourcePath) != filepath.Clean(wantAssets[i].SourcePath) {
					t.Fatalf("asset[%d].SourcePath=%q want %q", i, gotAssets[i].SourcePath, wantAssets[i].SourcePath)
				}
			}
		})
	}
}
