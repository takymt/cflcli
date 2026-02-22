package migrate

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/takymt/cflcli/internal/client"
)

// DefaultAttachmentsDir is the default attachments directory used by migrate export.
const DefaultAttachmentsDir = "attachments/_migrate"

// ExportRequest is the input for migrate export orchestration.
type ExportRequest struct {
	SpaceID        string
	SpaceKey       string
	RootPageID     string
	OutDir         string
	AttachmentsDir string
}

// ExportedPage represents a written markdown page artifact.
type ExportedPage struct {
	ID       string
	Title    string
	ParentID string
	File     string
}

// ExportResult represents migrate export output metadata.
type ExportResult struct {
	SpaceID        string
	SpaceKey       string
	OutDir         string
	AttachmentsDir string
	Pages          []ExportedPage
}

// Export executes migrate export orchestration using Confluence API + filesystem output.
func Export(cli *client.Client, req *ExportRequest) (*ExportResult, error) {
	if cli == nil {
		return nil, fmt.Errorf("client is required")
	}
	if req == nil {
		return nil, fmt.Errorf("export request is required")
	}

	spaceID := strings.TrimSpace(req.SpaceID)
	if spaceID == "" {
		return nil, fmt.Errorf("space id is required")
	}

	spaceKey := strings.TrimSpace(req.SpaceKey)
	if spaceKey == "" {
		return nil, fmt.Errorf("space key is required")
	}

	outDir := strings.TrimSpace(req.OutDir)
	if outDir == "" {
		return nil, fmt.Errorf("out directory is required")
	}

	attachmentsDir := strings.TrimSpace(req.AttachmentsDir)
	if attachmentsDir == "" {
		attachmentsDir = DefaultAttachmentsDir
	}
	attachmentsDir = filepath.Clean(attachmentsDir)
	if filepath.IsAbs(attachmentsDir) {
		return nil, fmt.Errorf("--attachments-dir must be a relative path")
	}
	if attachmentsDir == ".." || strings.HasPrefix(attachmentsDir, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("--attachments-dir must stay within --out")
	}

	pagesByID, err := listAllSpacePages(cli, spaceID)
	if err != nil {
		return nil, err
	}

	rootPageID := strings.TrimSpace(req.RootPageID)
	if rootPageID != "" {
		if _, ok := pagesByID[rootPageID]; !ok {
			root, rootErr := cli.GetPage(rootPageID)
			if rootErr != nil {
				return nil, rootErr
			}
			if root.SpaceID != "" && root.SpaceID != spaceID {
				return nil, fmt.Errorf("root page %q does not belong to space %q", rootPageID, spaceID)
			}
			pagesByID[root.ID] = client.Page{
				ID:       root.ID,
				Title:    root.Title,
				Status:   root.Status,
				SpaceID:  root.SpaceID,
				ParentID: root.ParentID,
			}
		}
	}

	selectedPages := selectPages(pagesByID, rootPageID)
	if len(selectedPages) == 0 {
		return nil, fmt.Errorf("no pages found to export")
	}

	orderedPages := orderPages(selectedPages, rootPageID)
	pageFileMap := buildPageFileMap(orderedPages)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("create output directory: %w", err)
	}

	exportedPages := make([]ExportedPage, 0, len(orderedPages))
	for _, page := range orderedPages {
		detail, err := cli.GetPage(page.ID)
		if err != nil {
			return nil, err
		}

		exportRelPath := pageFileMap[page.ID]
		pageDir := filepath.Dir(exportRelPath)

		markdown, attachments, err := StorageToMarkdown(detail.Body.Storage.Value, func(filename string) string {
			attachmentPath := filepath.Join(attachmentsDir, page.ID, safeAttachmentFilename(filename))
			relativePath, relErr := filepath.Rel(pageDir, attachmentPath)
			if relErr != nil {
				return filepath.ToSlash(attachmentPath)
			}
			return filepath.ToSlash(relativePath)
		})
		if err != nil {
			return nil, fmt.Errorf("convert page %q body to markdown: %w", page.ID, err)
		}

		if err := downloadAttachments(cli, outDir, attachmentsDir, page.ID, attachments); err != nil {
			return nil, err
		}

		frontMatter := buildFrontMatter(detail.ID, detail.Title, detail.ParentID, spaceKey)
		content := frontMatter + markdown

		exportPath := filepath.Join(outDir, exportRelPath)
		if err := os.MkdirAll(filepath.Dir(exportPath), 0o755); err != nil {
			return nil, fmt.Errorf("create export directory for %q: %w", exportPath, err)
		}
		if err := os.WriteFile(exportPath, []byte(content), 0o600); err != nil {
			return nil, fmt.Errorf("write exported markdown %q: %w", exportPath, err)
		}

		exportedPages = append(exportedPages, ExportedPage{
			ID:       detail.ID,
			Title:    detail.Title,
			ParentID: strings.TrimSpace(detail.ParentID),
			File:     filepath.ToSlash(exportRelPath),
		})
	}

	return &ExportResult{
		SpaceID:        spaceID,
		SpaceKey:       spaceKey,
		OutDir:         outDir,
		AttachmentsDir: filepath.ToSlash(attachmentsDir),
		Pages:          exportedPages,
	}, nil
}

func listAllSpacePages(cli *client.Client, spaceID string) (map[string]client.Page, error) {
	pagesByID := map[string]client.Page{}

	cursor := ""
	for {
		result, err := cli.ListPages(spaceID, 250, cursor, []string{"current"}, "")
		if err != nil {
			return nil, err
		}
		for _, page := range result.Results {
			pagesByID[page.ID] = page
		}

		nextCursor := extractNextCursor(result.Links.Next)
		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	return pagesByID, nil
}

func extractNextCursor(next string) string {
	next = strings.TrimSpace(next)
	if next == "" {
		return ""
	}
	if !strings.Contains(next, "cursor=") {
		return next
	}

	u, err := url.Parse(next)
	if err == nil {
		if cursor := strings.TrimSpace(u.Query().Get("cursor")); cursor != "" {
			return cursor
		}
	}
	if strings.HasPrefix(next, "/") {
		u, err = url.Parse("https://invalid.local" + next)
	} else {
		u, err = url.Parse("https://invalid.local/?" + strings.TrimPrefix(next, "?"))
	}
	if err != nil {
		return next
	}
	if cursor := strings.TrimSpace(u.Query().Get("cursor")); cursor != "" {
		return cursor
	}

	return next
}

func selectPages(pagesByID map[string]client.Page, rootPageID string) map[string]client.Page {
	selected := map[string]client.Page{}

	rootPageID = strings.TrimSpace(rootPageID)
	if rootPageID == "" {
		for id, page := range pagesByID {
			selected[id] = page
		}
		return selected
	}

	queue := []string{rootPageID}
	for len(queue) > 0 {
		pageID := queue[0]
		queue = queue[1:]

		page, ok := pagesByID[pageID]
		if !ok {
			continue
		}
		if _, exists := selected[pageID]; exists {
			continue
		}

		selected[pageID] = page
		for _, candidate := range pagesByID {
			if strings.TrimSpace(candidate.ParentID) == pageID {
				queue = append(queue, candidate.ID)
			}
		}
	}

	return selected
}

func orderPages(pagesByID map[string]client.Page, rootPageID string) []client.Page {
	children := map[string][]client.Page{}
	for _, page := range pagesByID {
		parentID := strings.TrimSpace(page.ParentID)
		children[parentID] = append(children[parentID], page)
	}
	for parentID := range children {
		sortPages(children[parentID])
	}

	var roots []client.Page
	if strings.TrimSpace(rootPageID) != "" {
		if root, ok := pagesByID[rootPageID]; ok {
			roots = append(roots, root)
		}
	} else {
		for _, page := range pagesByID {
			parentID := strings.TrimSpace(page.ParentID)
			if parentID == "" {
				roots = append(roots, page)
				continue
			}
			if _, ok := pagesByID[parentID]; !ok {
				roots = append(roots, page)
			}
		}
		sortPages(roots)
	}

	ordered := []client.Page{}
	visited := map[string]struct{}{}
	var visit func(client.Page)
	visit = func(page client.Page) {
		if _, ok := visited[page.ID]; ok {
			return
		}
		visited[page.ID] = struct{}{}
		ordered = append(ordered, page)
		for _, child := range children[page.ID] {
			visit(child)
		}
	}

	for _, root := range roots {
		visit(root)
	}

	var leftovers []client.Page
	for _, page := range pagesByID {
		if _, ok := visited[page.ID]; ok {
			continue
		}
		leftovers = append(leftovers, page)
	}
	sortPages(leftovers)
	for _, page := range leftovers {
		visit(page)
	}

	return ordered
}

func sortPages(pages []client.Page) {
	sort.SliceStable(pages, func(i, j int) bool {
		leftTitle := strings.ToLower(strings.TrimSpace(pages[i].Title))
		rightTitle := strings.ToLower(strings.TrimSpace(pages[j].Title))
		if leftTitle == rightTitle {
			return pages[i].ID < pages[j].ID
		}
		return leftTitle < rightTitle
	})
}

func downloadAttachments(cli *client.Client, outDir, attachmentsDir, pageID string, filenames []string) error {
	for _, filename := range filenames {
		content, err := cli.DownloadPageAttachmentByFilename(pageID, filename)
		if err != nil {
			return fmt.Errorf("download attachment %q for page %q: %w", filename, pageID, err)
		}

		safeName := safeAttachmentFilename(filename)
		targetPath := filepath.Join(outDir, attachmentsDir, pageID, safeName)
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return fmt.Errorf("create attachment directory for %q: %w", targetPath, err)
		}
		if err := os.WriteFile(targetPath, content, 0o600); err != nil {
			return fmt.Errorf("write attachment %q: %w", targetPath, err)
		}
	}
	return nil
}

func buildFrontMatter(pageID, title, parentID, spaceKey string) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("page-id: " + quoteFrontMatterValue(pageID) + "\n")
	b.WriteString("title: " + quoteFrontMatterValue(title) + "\n")
	b.WriteString("parent-id: " + quoteFrontMatterValue(parentID) + "\n")
	b.WriteString("space-key: " + quoteFrontMatterValue(spaceKey) + "\n")
	b.WriteString("---\n\n")
	return b.String()
}

func quoteFrontMatterValue(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\"", "\\\"")
	return `"` + value + `"`
}

func sanitizePageFileBaseName(title string) string {
	title = strings.TrimSpace(strings.ToLower(title))
	if title == "" {
		return "page"
	}

	var b strings.Builder
	lastDash := false
	for _, r := range title {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if lastDash {
			continue
		}
		b.WriteByte('-')
		lastDash = true
	}

	normalized := strings.Trim(b.String(), "-")
	if normalized == "" {
		return "page"
	}
	return normalized
}

func safeAttachmentFilename(filename string) string {
	cleaned := filepath.Base(filepath.Clean(filename))
	if cleaned == "." || cleaned == string(filepath.Separator) {
		return "attachment"
	}
	return cleaned
}

func buildPageFileMap(orderedPages []client.Page) map[string]string {
	pageDirByID := map[string]string{}
	pageFileByID := map[string]string{}

	for _, page := range orderedPages {
		parentDir := ""
		if parent, ok := pageDirByID[strings.TrimSpace(page.ParentID)]; ok {
			parentDir = parent
		}

		pageDir := filepath.Join(parentDir, sanitizePageFileBaseName(page.Title)+"-"+page.ID)
		pageDirByID[page.ID] = pageDir
		pageFileByID[page.ID] = filepath.Join(pageDir, "index.md")
	}

	return pageFileByID
}
