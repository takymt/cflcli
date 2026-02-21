package attachment

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveMarkdownImageAssets_RelativeLocalPathConvertsToAttachment(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	bodyFile := filepath.Join(root, "docs", "page.md")
	imagePath := mustWritePNGImage(t, filepath.Join(root, "docs", "img", "logo.png"), 640, 480)
	storage := `<ac:image><ri:url ri:value="./img/logo.png" /></ac:image>`

	gotStorage, gotAssets, err := ResolveMarkdownImageAssets(storage, bodyFile, "")
	if err != nil {
		t.Fatalf("ResolveMarkdownImageAssets: %v", err)
	}

	if !strings.Contains(gotStorage, `<ri:attachment ri:filename="logo.png" />`) {
		t.Fatalf("storage missing attachment reference: %q", gotStorage)
	}
	if !strings.Contains(gotStorage, `ac:original-width="640"`) {
		t.Fatalf("storage missing original width: %q", gotStorage)
	}
	if !strings.Contains(gotStorage, `ac:original-height="480"`) {
		t.Fatalf("storage missing original height: %q", gotStorage)
	}
	assertAssetsEqual(t, gotAssets, []Asset{{
		Filename:   "logo.png",
		SourcePath: imagePath,
	}})
}

func TestResolveMarkdownImageAssets_RelativeLocalSVGAddsOriginalDimensions(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	bodyFile := filepath.Join(root, "docs", "page.md")
	imagePath := mustWriteFile(t, filepath.Join(root, "docs", "img", "diagram.svg"), `<svg width="604" height="425" viewBox="0 0 604 425" xmlns="http://www.w3.org/2000/svg"></svg>`)
	storage := `<ac:image><ri:url ri:value="./img/diagram.svg" /></ac:image>`

	gotStorage, gotAssets, err := ResolveMarkdownImageAssets(storage, bodyFile, "")
	if err != nil {
		t.Fatalf("ResolveMarkdownImageAssets: %v", err)
	}

	if !strings.Contains(gotStorage, `<ri:attachment ri:filename="diagram.svg" />`) {
		t.Fatalf("storage missing attachment reference: %q", gotStorage)
	}
	if !strings.Contains(gotStorage, `ac:original-width="604"`) {
		t.Fatalf("storage missing original width: %q", gotStorage)
	}
	if !strings.Contains(gotStorage, `ac:original-height="425"`) {
		t.Fatalf("storage missing original height: %q", gotStorage)
	}
	assertAssetsEqual(t, gotAssets, []Asset{{
		Filename:   "diagram.svg",
		SourcePath: imagePath,
	}})
}

func TestResolveMarkdownImageAssets_RootPrefixedPathResolvesFromAssetsRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	bodyFile := filepath.Join(root, "docs", "page.md")
	assetsRoot := filepath.Join(root, "assets")
	imagePath := mustWriteFile(t, filepath.Join(root, "assets", "images", "root.png"), "ROOT")
	storage := `<ac:image><ri:url ri:value="/images/root.png" /></ac:image>`

	gotStorage, gotAssets, err := ResolveMarkdownImageAssets(storage, bodyFile, assetsRoot)
	if err != nil {
		t.Fatalf("ResolveMarkdownImageAssets: %v", err)
	}

	if !strings.Contains(gotStorage, `<ri:attachment ri:filename="root.png" />`) {
		t.Fatalf("storage missing attachment reference: %q", gotStorage)
	}
	assertAssetsEqual(t, gotAssets, []Asset{{
		Filename:   "root.png",
		SourcePath: imagePath,
	}})
}

func TestResolveMarkdownImageAssets_RemoteURLUnchanged(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	bodyFile := filepath.Join(root, "docs", "page.md")
	storage := `<ac:image><ri:url ri:value="https://example.com/logo.png" /></ac:image>`

	gotStorage, gotAssets, err := ResolveMarkdownImageAssets(storage, bodyFile, "")
	if err != nil {
		t.Fatalf("ResolveMarkdownImageAssets: %v", err)
	}

	if !strings.Contains(gotStorage, `<ri:url ri:value="https://example.com/logo.png" />`) {
		t.Fatalf("storage must keep remote url: %q", gotStorage)
	}
	if len(gotAssets) != 0 {
		t.Fatalf("got assets=%+v want empty", gotAssets)
	}
}

func TestResolveMarkdownImageAssets_MissingLocalFileReturnsError(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	bodyFile := filepath.Join(root, "docs", "page.md")
	storage := `<ac:image><ri:url ri:value="./img/missing.png" /></ac:image>`

	_, _, err := ResolveMarkdownImageAssets(storage, bodyFile, "")
	if err == nil || !strings.Contains(err.Error(), "resolve local image") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveMarkdownImageAssets_DuplicateFilenameReturnsError(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	bodyFile := filepath.Join(root, "docs", "page.md")
	assetsRoot := filepath.Join(root, "assets")
	_ = mustWriteFile(t, filepath.Join(root, "docs", "img", "logo.png"), "A")
	_ = mustWriteFile(t, filepath.Join(root, "assets", "sub", "logo.png"), "B")

	storage := strings.Join([]string{
		`<ac:image><ri:url ri:value="./img/logo.png" /></ac:image>`,
		`<ac:image><ri:url ri:value="/sub/logo.png" /></ac:image>`,
	}, "")

	_, _, err := ResolveMarkdownImageAssets(storage, bodyFile, assetsRoot)
	if err == nil || !strings.Contains(err.Error(), "duplicate image filename") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func mustWriteFile(t *testing.T, path, content string) string {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func mustWritePNGImage(t *testing.T, path string, width, height int) string {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer func() { _ = file.Close() }()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	if err := png.Encode(file, img); err != nil {
		t.Fatalf("png.Encode: %v", err)
	}
	return path
}

func assertAssetsEqual(t *testing.T, got, want []Asset) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("asset count=%d want %d (%+v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i].Filename != want[i].Filename {
			t.Fatalf("asset[%d].Filename=%q want %q", i, got[i].Filename, want[i].Filename)
		}
		if filepath.Clean(got[i].SourcePath) != filepath.Clean(want[i].SourcePath) {
			t.Fatalf("asset[%d].SourcePath=%q want %q", i, got[i].SourcePath, want[i].SourcePath)
		}
	}
}
