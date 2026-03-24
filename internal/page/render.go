package page

import "context"

// RenderResult is the page-sync preparation output used by the CLI.
type RenderResult struct {
	Storage           string
	AttachmentSources map[string]string
	Warnings          []string
	Generated         map[string]string
	SaveCache         func() error
}

// RenderMarkdownForSync prepares markdown for page sync by resolving relative
// references first and then rendering any mermaid blocks.
func RenderMarkdownForSync(ctx context.Context, markdownPath string, markdown string, siteBaseURL string) (RenderResult, error) {
	refs, err := resolveRelativeReferences(markdownPath, markdown, siteBaseURL)
	if err != nil {
		return RenderResult{}, err
	}

	mermaid, err := ConvertMarkdownToStorageWithMermaid(ctx, markdownPath, refs.Markdown)
	if err != nil {
		return RenderResult{}, err
	}

	return RenderResult{
		Storage:           mermaid.Storage,
		AttachmentSources: refs.AttachmentSources,
		Warnings:          refs.Warnings,
		Generated:         mermaid.Generated,
		SaveCache:         mermaid.SaveCache,
	}, nil
}
