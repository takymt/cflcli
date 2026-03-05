package page

import "testing"

func TestAttachmentFilenamesFromStorage(t *testing.T) {
	t.Parallel()

	storage := `<ac:image><ri:attachment ri:filename="mermaid-1.svg" /></ac:image>` +
		`<ac:image><ri:attachment ri:filename="diagram.svg" /></ac:image>` +
		`<ac:image><ri:attachment ri:filename="mermaid-1.svg" /></ac:image>`

	got := AttachmentFilenamesFromStorage(storage)
	if len(got) != 2 {
		t.Fatalf("AttachmentFilenamesFromStorage() len = %d, want 2", len(got))
	}
	if got[0] != "mermaid-1.svg" || got[1] != "diagram.svg" {
		t.Fatalf("AttachmentFilenamesFromStorage() = %#v, want [mermaid-1.svg diagram.svg]", got)
	}
}
