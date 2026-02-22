package migrate

import (
	"errors"
	"fmt"
	"net/http"
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
				ID:         root.ID,
				Title:      root.Title,
				Status:     root.Status,
				SpaceID:    root.SpaceID,
				ParentID:   root.ParentID,
				ParentType: root.ParentType,
			}
		}
	}

	folderResolver := newFolderResolver(cli)
	selectedPages, err := selectPages(pagesByID, rootPageID, folderResolver)
	if err != nil {
		return nil, err
	}
	if len(selectedPages) == 0 {
		return nil, fmt.Errorf("no pages found to export")
	}

	pageFileMap, err := buildPageFileMap(selectedPages, folderResolver)
	if err != nil {
		return nil, err
	}
	orderedPages := orderedPagesForExport(selectedPages, pageFileMap)

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
	pagesByID, err := listAllSpacePagesWithListFn(func(cursor string) (*client.PageListResult, error) {
		return cli.ListPagesBySpace(spaceID, 250, cursor, "all", []string{"current"}, "")
	})
	if err == nil {
		return pagesByID, nil
	}

	var httpErr *client.HTTPError
	if !errors.As(err, &httpErr) || (httpErr.StatusCode != http.StatusBadRequest && httpErr.StatusCode != http.StatusNotFound) {
		return nil, err
	}

	return listAllSpacePagesWithListFn(func(cursor string) (*client.PageListResult, error) {
		return cli.ListPages(spaceID, 250, cursor, []string{"current"}, "")
	})
}

func listAllSpacePagesWithListFn(listFn func(cursor string) (*client.PageListResult, error)) (map[string]client.Page, error) {
	pagesByID := map[string]client.Page{}
	cursor := ""
	for {
		result, err := listFn(cursor)
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

func selectPages(pagesByID map[string]client.Page, rootPageID string, folders *folderResolver) (map[string]client.Page, error) {
	selected := map[string]client.Page{}
	rootPageID = strings.TrimSpace(rootPageID)
	if rootPageID == "" {
		for id, page := range pagesByID {
			selected[id] = page
		}
		return selected, nil
	}

	matcher := &subtreeMatcher{
		rootPageID: rootPageID,
		pagesByID:  pagesByID,
		folders:    folders,
		pageMemo:   map[string]bool{},
		pageVisit:  map[string]struct{}{},
		foldMemo:   map[string]bool{},
		foldVisit:  map[string]struct{}{},
	}

	for id, page := range pagesByID {
		inRoot, err := matcher.pageInRoot(id)
		if err != nil {
			return nil, err
		}
		if inRoot {
			selected[id] = page
		}
	}

	return selected, nil
}

type subtreeMatcher struct {
	rootPageID string
	pagesByID  map[string]client.Page
	folders    *folderResolver
	pageMemo   map[string]bool
	pageVisit  map[string]struct{}
	foldMemo   map[string]bool
	foldVisit  map[string]struct{}
}

func (m *subtreeMatcher) pageInRoot(pageID string) (bool, error) {
	pageID = strings.TrimSpace(pageID)
	if pageID == "" {
		return false, nil
	}
	if pageID == m.rootPageID {
		return true, nil
	}
	if v, ok := m.pageMemo[pageID]; ok {
		return v, nil
	}
	if _, visiting := m.pageVisit[pageID]; visiting {
		return false, nil
	}
	page, ok := m.pagesByID[pageID]
	if !ok {
		m.pageMemo[pageID] = false
		return false, nil
	}

	m.pageVisit[pageID] = struct{}{}
	defer delete(m.pageVisit, pageID)

	parentID := strings.TrimSpace(page.ParentID)
	parentType := strings.ToLower(strings.TrimSpace(page.ParentType))

	var (
		inRoot bool
		err    error
	)
	switch parentType {
	case "page":
		inRoot, err = m.pageInRoot(parentID)
	case "folder":
		inRoot, err = m.folderInRoot(parentID)
	default:
		if _, ok := m.pagesByID[parentID]; ok {
			inRoot, err = m.pageInRoot(parentID)
		}
	}
	if err != nil {
		return false, err
	}

	m.pageMemo[pageID] = inRoot
	return inRoot, nil
}

func (m *subtreeMatcher) folderInRoot(folderID string) (bool, error) {
	folderID = strings.TrimSpace(folderID)
	if folderID == "" {
		return false, nil
	}
	if v, ok := m.foldMemo[folderID]; ok {
		return v, nil
	}
	if _, visiting := m.foldVisit[folderID]; visiting {
		return false, nil
	}

	m.foldVisit[folderID] = struct{}{}
	defer delete(m.foldVisit, folderID)

	folder, err := m.folders.Get(folderID)
	if err != nil {
		return false, fmt.Errorf("resolve folder %q: %w", folderID, err)
	}

	parentID := strings.TrimSpace(folder.ParentID)
	parentType := strings.ToLower(strings.TrimSpace(folder.ParentType))

	var inRoot bool
	switch parentType {
	case "page":
		inRoot, err = m.pageInRoot(parentID)
	case "folder":
		inRoot, err = m.folderInRoot(parentID)
	default:
		inRoot = false
	}
	if err != nil {
		return false, err
	}

	m.foldMemo[folderID] = inRoot
	return inRoot, nil
}

func buildPageFileMap(pagesByID map[string]client.Page, folders *folderResolver) (map[string]string, error) {
	builder := &pathBuilder{
		pagesByID:   pagesByID,
		folders:     folders,
		pageDirMemo: map[string]string{},
		pageVisit:   map[string]struct{}{},
		foldDirMemo: map[string]string{},
		foldVisit:   map[string]struct{}{},
	}

	pageFileByID := map[string]string{}
	for id := range pagesByID {
		dir, err := builder.pageDir(id)
		if err != nil {
			return nil, err
		}
		pageFileByID[id] = filepath.Join(dir, "index.md")
	}

	return pageFileByID, nil
}

type pathBuilder struct {
	pagesByID   map[string]client.Page
	folders     *folderResolver
	pageDirMemo map[string]string
	pageVisit   map[string]struct{}
	foldDirMemo map[string]string
	foldVisit   map[string]struct{}
}

func (b *pathBuilder) pageDir(pageID string) (string, error) {
	pageID = strings.TrimSpace(pageID)
	if pageID == "" {
		return "", nil
	}
	if dir, ok := b.pageDirMemo[pageID]; ok {
		return dir, nil
	}
	if _, visiting := b.pageVisit[pageID]; visiting {
		return "", nil
	}

	page, ok := b.pagesByID[pageID]
	if !ok {
		return "", fmt.Errorf("page %q not found while building path", pageID)
	}

	b.pageVisit[pageID] = struct{}{}
	defer delete(b.pageVisit, pageID)

	parentID := strings.TrimSpace(page.ParentID)
	parentType := strings.ToLower(strings.TrimSpace(page.ParentType))
	parentDir := ""

	switch parentType {
	case "page":
		if _, ok := b.pagesByID[parentID]; ok {
			var err error
			parentDir, err = b.pageDir(parentID)
			if err != nil {
				return "", err
			}
		}
	case "folder":
		var err error
		parentDir, err = b.folderDir(parentID)
		if err != nil {
			return "", err
		}
	default:
		if _, ok := b.pagesByID[parentID]; ok {
			var err error
			parentDir, err = b.pageDir(parentID)
			if err != nil {
				return "", err
			}
		}
	}

	dir := filepath.Join(parentDir, sanitizePageFileBaseName(page.Title)+"-"+page.ID)
	b.pageDirMemo[pageID] = dir
	return dir, nil
}

func (b *pathBuilder) folderDir(folderID string) (string, error) {
	folderID = strings.TrimSpace(folderID)
	if folderID == "" {
		return "", nil
	}
	if dir, ok := b.foldDirMemo[folderID]; ok {
		return dir, nil
	}
	if _, visiting := b.foldVisit[folderID]; visiting {
		return "", nil
	}

	b.foldVisit[folderID] = struct{}{}
	defer delete(b.foldVisit, folderID)

	folder, err := b.folders.Get(folderID)
	if err != nil {
		return "", fmt.Errorf("resolve folder %q: %w", folderID, err)
	}

	parentID := strings.TrimSpace(folder.ParentID)
	parentType := strings.ToLower(strings.TrimSpace(folder.ParentType))
	parentDir := ""

	switch parentType {
	case "page":
		if _, ok := b.pagesByID[parentID]; ok {
			parentDir, err = b.pageDir(parentID)
			if err != nil {
				return "", err
			}
		}
	case "folder":
		parentDir, err = b.folderDir(parentID)
		if err != nil {
			return "", err
		}
	}

	dir := filepath.Join(parentDir, sanitizePageFileBaseName(folder.Title)+"-"+folder.ID)
	b.foldDirMemo[folderID] = dir
	return dir, nil
}

func orderedPagesForExport(pagesByID map[string]client.Page, pageFileByID map[string]string) []client.Page {
	ordered := make([]client.Page, 0, len(pagesByID))
	for _, page := range pagesByID {
		ordered = append(ordered, page)
	}

	sort.SliceStable(ordered, func(i, j int) bool {
		leftPath := filepath.ToSlash(pageFileByID[ordered[i].ID])
		rightPath := filepath.ToSlash(pageFileByID[ordered[j].ID])
		if leftPath == rightPath {
			return ordered[i].ID < ordered[j].ID
		}
		return leftPath < rightPath
	})

	return ordered
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

type folderResolver struct {
	cli   *client.Client
	cache map[string]*client.Folder
}

func newFolderResolver(cli *client.Client) *folderResolver {
	return &folderResolver{cli: cli, cache: map[string]*client.Folder{}}
}

func (r *folderResolver) Get(folderID string) (*client.Folder, error) {
	folderID = strings.TrimSpace(folderID)
	if folderID == "" {
		return nil, fmt.Errorf("folder id is required")
	}
	if folder, ok := r.cache[folderID]; ok {
		return folder, nil
	}
	folder, err := r.cli.GetFolder(folderID)
	if err != nil {
		return nil, err
	}
	r.cache[folderID] = folder
	return folder, nil
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
