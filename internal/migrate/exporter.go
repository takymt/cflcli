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

// ExportWarning represents a non-fatal issue encountered during export.
type ExportWarning struct {
	Message string
}

// ExportResult represents migrate export output metadata.
type ExportResult struct {
	SpaceID        string
	SpaceKey       string
	OutDir         string
	AttachmentsDir string
	Pages          []ExportedPage
	Warnings       []ExportWarning
}

type exportConfig struct {
	spaceID        string
	spaceKey       string
	rootPageID     string
	outDir         string
	attachmentsDir string
}

type exportExecutionPlan struct {
	cfg       *exportConfig
	pageFiles map[string]string
	ordered   []client.Page
}

type renderedExportPage struct {
	page        ExportedPage
	relPath     string
	content     []byte
	attachments []string
}

// Export executes migrate export orchestration using Confluence API + filesystem output.
func Export(cli *client.Client, req *ExportRequest) (*ExportResult, error) {
	if cli == nil {
		return nil, fmt.Errorf("client is required")
	}
	cfg, err := normalizeExportConfig(req)
	if err != nil {
		return nil, err
	}
	plan, err := buildExportExecutionPlan(cli, cfg)
	if err != nil {
		return nil, err
	}
	return executeExportPlan(cli, plan)
}

func normalizeExportConfig(req *ExportRequest) (*exportConfig, error) {
	if req == nil {
		return nil, fmt.Errorf("export request is required")
	}

	cfg := &exportConfig{
		spaceID:    strings.TrimSpace(req.SpaceID),
		spaceKey:   strings.TrimSpace(req.SpaceKey),
		rootPageID: strings.TrimSpace(req.RootPageID),
		outDir:     strings.TrimSpace(req.OutDir),
	}
	if cfg.spaceID == "" {
		return nil, fmt.Errorf("space id is required")
	}
	if cfg.spaceKey == "" {
		return nil, fmt.Errorf("space key is required")
	}
	if cfg.outDir == "" {
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
	cfg.attachmentsDir = attachmentsDir

	return cfg, nil
}

func buildExportExecutionPlan(cli *client.Client, cfg *exportConfig) (*exportExecutionPlan, error) {
	pagesByID, err := listAllSpacePages(cli, cfg.spaceID)
	if err != nil {
		return nil, err
	}
	if err := ensureRootPageInSpacePageSet(cli, pagesByID, cfg.spaceID, cfg.rootPageID); err != nil {
		return nil, err
	}

	folderResolver := newFolderResolver(cli)
	selectedPages, err := selectPages(pagesByID, cfg.rootPageID, folderResolver)
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

	return &exportExecutionPlan{
		cfg:       cfg,
		pageFiles: pageFileMap,
		ordered:   orderedPagesForExport(selectedPages, pageFileMap),
	}, nil
}

func ensureRootPageInSpacePageSet(cli *client.Client, pagesByID map[string]client.Page, spaceID, rootPageID string) error {
	rootPageID = strings.TrimSpace(rootPageID)
	if rootPageID == "" {
		return nil
	}
	if _, ok := pagesByID[rootPageID]; ok {
		return nil
	}

	root, err := cli.GetPage(rootPageID)
	if err != nil {
		return err
	}
	if root.SpaceID != "" && root.SpaceID != spaceID {
		return fmt.Errorf("root page %q does not belong to space %q", rootPageID, spaceID)
	}

	pagesByID[root.ID] = client.Page{
		ID:         root.ID,
		Title:      root.Title,
		Status:     root.Status,
		SpaceID:    root.SpaceID,
		ParentID:   root.ParentID,
		ParentType: root.ParentType,
	}
	return nil
}

func executeExportPlan(cli *client.Client, plan *exportExecutionPlan) (*ExportResult, error) {
	if plan == nil {
		return nil, fmt.Errorf("export plan is required")
	}
	if err := os.MkdirAll(plan.cfg.outDir, 0o755); err != nil {
		return nil, fmt.Errorf("create output directory: %w", err)
	}

	exportedPages := make([]ExportedPage, 0, len(plan.ordered))
	warnings := make([]ExportWarning, 0)
	for _, page := range plan.ordered {
		rendered, renderWarnings, err := exportAndWritePage(cli, plan.cfg, page.ID, plan.pageFiles[page.ID])
		if err != nil {
			return nil, err
		}
		exportedPages = append(exportedPages, rendered.page)
		warnings = append(warnings, renderWarnings...)
	}

	return &ExportResult{
		SpaceID:        plan.cfg.spaceID,
		SpaceKey:       plan.cfg.spaceKey,
		OutDir:         plan.cfg.outDir,
		AttachmentsDir: filepath.ToSlash(plan.cfg.attachmentsDir),
		Pages:          exportedPages,
		Warnings:       warnings,
	}, nil
}

func exportAndWritePage(cli *client.Client, cfg *exportConfig, pageID, exportRelPath string) (*renderedExportPage, []ExportWarning, error) {
	detail, err := cli.GetPage(pageID)
	if err != nil {
		return nil, nil, err
	}

	rendered, err := renderExportPage(detail, exportRelPath, cfg.attachmentsDir, cfg.spaceKey)
	if err != nil {
		return nil, nil, err
	}

	downloadWarnings, err := downloadAttachments(cli, cfg.outDir, cfg.attachmentsDir, pageID, rendered.attachments)
	if err != nil {
		return nil, nil, err
	}
	if err := writeRenderedExportPage(cfg.outDir, rendered); err != nil {
		return nil, nil, err
	}

	return rendered, downloadWarnings, nil
}

func renderExportPage(detail *client.PageDetail, exportRelPath, attachmentsDir, spaceKey string) (*renderedExportPage, error) {
	if detail == nil {
		return nil, fmt.Errorf("page detail is required")
	}

	pageDir := filepath.Dir(exportRelPath)
	markdown, attachments, err := StorageToMarkdown(detail.Body.Storage.Value, func(filename string) string {
		return relativeAttachmentPathForMarkdown(pageDir, attachmentsDir, detail.ID, filename)
	})
	if err != nil {
		return nil, fmt.Errorf("convert page %q body to markdown: %w", detail.ID, err)
	}

	frontMatter := buildFrontMatter(detail.ID, detail.Title, detail.ParentID, spaceKey)
	return &renderedExportPage{
		page: ExportedPage{
			ID:       detail.ID,
			Title:    detail.Title,
			ParentID: strings.TrimSpace(detail.ParentID),
			File:     filepath.ToSlash(exportRelPath),
		},
		relPath:     exportRelPath,
		content:     []byte(frontMatter + markdown),
		attachments: attachments,
	}, nil
}

func relativeAttachmentPathForMarkdown(pageDir, attachmentsDir, pageID, filename string) string {
	attachmentPath := filepath.Join(attachmentsDir, pageID, safeAttachmentFilename(filename))
	relativePath, err := filepath.Rel(pageDir, attachmentPath)
	if err != nil {
		return filepath.ToSlash(attachmentPath)
	}
	return filepath.ToSlash(relativePath)
}

func writeRenderedExportPage(outDir string, rendered *renderedExportPage) error {
	if rendered == nil {
		return fmt.Errorf("rendered export page is required")
	}
	exportPath := filepath.Join(outDir, rendered.relPath)
	if err := os.MkdirAll(filepath.Dir(exportPath), 0o755); err != nil {
		return fmt.Errorf("create export directory for %q: %w", exportPath, err)
	}
	if err := os.WriteFile(exportPath, rendered.content, 0o600); err != nil {
		return fmt.Errorf("write exported markdown %q: %w", exportPath, err)
	}
	return nil
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
		pagesByID:        pagesByID,
		folders:          folders,
		pageContainerDir: map[string]string{},
		pageAncestors:    map[string][]string{},
		pageVisit:        map[string]struct{}{},
		folderDirMemo:    map[string]string{},
		folderAncestors:  map[string][]string{},
		folderVisit:      map[string]struct{}{},
	}

	hasChildren, err := builder.buildHasChildrenMap()
	if err != nil {
		return nil, err
	}

	pageFileByID := map[string]string{}
	for id, page := range pagesByID {
		containerDir, err := builder.pageContainer(id, hasChildren)
		if err != nil {
			return nil, err
		}
		name := sanitizePageFileBaseName(page.Title)
		if hasChildren[id] {
			pageFileByID[id] = filepath.Join(containerDir, name, "_index.md")
			continue
		}
		pageFileByID[id] = filepath.Join(containerDir, name+".md")
	}

	return pageFileByID, nil
}

type pathBuilder struct {
	pagesByID map[string]client.Page
	folders   *folderResolver

	pageContainerDir map[string]string
	pageAncestors    map[string][]string
	pageVisit        map[string]struct{}

	folderDirMemo   map[string]string
	folderAncestors map[string][]string
	folderVisit     map[string]struct{}
}

func (b *pathBuilder) buildHasChildrenMap() (map[string]bool, error) {
	hasChildren := map[string]bool{}
	for id := range b.pagesByID {
		hasChildren[id] = false
	}

	for id := range b.pagesByID {
		ancestors, err := b.pageAncestorIDs(id)
		if err != nil {
			return nil, err
		}
		for _, ancestor := range ancestors {
			if ancestor == id {
				continue
			}
			if _, ok := hasChildren[ancestor]; ok {
				hasChildren[ancestor] = true
			}
		}
	}
	return hasChildren, nil
}

func (b *pathBuilder) pageAncestorIDs(pageID string) ([]string, error) {
	pageID = strings.TrimSpace(pageID)
	if pageID == "" {
		return nil, nil
	}
	if ancestors, ok := b.pageAncestors[pageID]; ok {
		copied := make([]string, len(ancestors))
		copy(copied, ancestors)
		return copied, nil
	}
	if _, visiting := b.pageVisit[pageID]; visiting {
		return []string{pageID}, nil
	}

	page, ok := b.pagesByID[pageID]
	if !ok {
		return nil, fmt.Errorf("page %q not found while building ancestors", pageID)
	}

	b.pageVisit[pageID] = struct{}{}
	defer delete(b.pageVisit, pageID)

	ancestors := []string{pageID}
	parentID := strings.TrimSpace(page.ParentID)
	parentType := strings.ToLower(strings.TrimSpace(page.ParentType))

	switch parentType {
	case "page":
		if _, ok := b.pagesByID[parentID]; ok {
			parentAncestors, err := b.pageAncestorIDs(parentID)
			if err != nil {
				return nil, err
			}
			ancestors = append(ancestors, parentAncestors...)
		}
	case "folder":
		folderAncestors, err := b.folderAncestorPageIDs(parentID)
		if err != nil {
			return nil, err
		}
		ancestors = append(ancestors, folderAncestors...)
	default:
		if _, ok := b.pagesByID[parentID]; ok {
			parentAncestors, err := b.pageAncestorIDs(parentID)
			if err != nil {
				return nil, err
			}
			ancestors = append(ancestors, parentAncestors...)
		}
	}

	b.pageAncestors[pageID] = ancestors
	copied := make([]string, len(ancestors))
	copy(copied, ancestors)
	return copied, nil
}

func (b *pathBuilder) folderAncestorPageIDs(folderID string) ([]string, error) {
	folderID = strings.TrimSpace(folderID)
	if folderID == "" {
		return nil, nil
	}
	if ancestors, ok := b.folderAncestors[folderID]; ok {
		copied := make([]string, len(ancestors))
		copy(copied, ancestors)
		return copied, nil
	}
	if _, visiting := b.folderVisit[folderID]; visiting {
		return nil, nil
	}

	b.folderVisit[folderID] = struct{}{}
	defer delete(b.folderVisit, folderID)

	folder, err := b.folders.Get(folderID)
	if err != nil {
		return nil, fmt.Errorf("resolve folder %q: %w", folderID, err)
	}

	parentID := strings.TrimSpace(folder.ParentID)
	parentType := strings.ToLower(strings.TrimSpace(folder.ParentType))
	var ancestors []string

	switch parentType {
	case "page":
		if _, ok := b.pagesByID[parentID]; ok {
			ancestors, err = b.pageAncestorIDs(parentID)
			if err != nil {
				return nil, err
			}
		}
	case "folder":
		ancestors, err = b.folderAncestorPageIDs(parentID)
		if err != nil {
			return nil, err
		}
	}

	b.folderAncestors[folderID] = ancestors
	copied := make([]string, len(ancestors))
	copy(copied, ancestors)
	return copied, nil
}

func (b *pathBuilder) pageContainer(pageID string, hasChildren map[string]bool) (string, error) {
	pageID = strings.TrimSpace(pageID)
	if pageID == "" {
		return "", nil
	}
	if dir, ok := b.pageContainerDir[pageID]; ok {
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

	parentDir := ""
	parentID := strings.TrimSpace(page.ParentID)
	parentType := strings.ToLower(strings.TrimSpace(page.ParentType))

	switch parentType {
	case "page":
		if _, ok := b.pagesByID[parentID]; ok {
			parentContainer, err := b.pageContainer(parentID, hasChildren)
			if err != nil {
				return "", err
			}
			parentPage := b.pagesByID[parentID]
			if hasChildren[parentID] {
				parentDir = filepath.Join(parentContainer, sanitizePageFileBaseName(parentPage.Title))
			} else {
				parentDir = parentContainer
			}
		}
	case "folder":
		var err error
		parentDir, err = b.folderDir(parentID, hasChildren)
		if err != nil {
			return "", err
		}
	default:
		if _, ok := b.pagesByID[parentID]; ok {
			parentContainer, err := b.pageContainer(parentID, hasChildren)
			if err != nil {
				return "", err
			}
			parentPage := b.pagesByID[parentID]
			if hasChildren[parentID] {
				parentDir = filepath.Join(parentContainer, sanitizePageFileBaseName(parentPage.Title))
			} else {
				parentDir = parentContainer
			}
		}
	}

	b.pageContainerDir[pageID] = parentDir
	return parentDir, nil
}

func (b *pathBuilder) folderDir(folderID string, hasChildren map[string]bool) (string, error) {
	folderID = strings.TrimSpace(folderID)
	if folderID == "" {
		return "", nil
	}
	if dir, ok := b.folderDirMemo[folderID]; ok {
		return dir, nil
	}
	if _, visiting := b.folderVisit[folderID]; visiting {
		return "", nil
	}

	b.folderVisit[folderID] = struct{}{}
	defer delete(b.folderVisit, folderID)

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
			parentContainer, err := b.pageContainer(parentID, hasChildren)
			if err != nil {
				return "", err
			}
			parentPage := b.pagesByID[parentID]
			if hasChildren[parentID] {
				parentDir = filepath.Join(parentContainer, sanitizePageFileBaseName(parentPage.Title))
			} else {
				parentDir = parentContainer
			}
		}
	case "folder":
		parentDir, err = b.folderDir(parentID, hasChildren)
		if err != nil {
			return "", err
		}
	}

	dir := filepath.Join(parentDir, sanitizePageFileBaseName(folder.Title))
	b.folderDirMemo[folderID] = dir
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

func downloadAttachments(cli *client.Client, outDir, attachmentsDir, pageID string, filenames []string) ([]ExportWarning, error) {
	warnings := make([]ExportWarning, 0)
	for _, filename := range filenames {
		content, err := cli.DownloadPageAttachmentByFilename(pageID, filename)
		if err != nil {
			warning, classifyErr := classifyAttachmentDownloadError(pageID, filename, err)
			if classifyErr != nil {
				return nil, classifyErr
			}
			if warning != nil {
				warnings = append(warnings, *warning)
				continue
			}
		}

		safeName := safeAttachmentFilename(filename)
		targetPath := filepath.Join(outDir, attachmentsDir, pageID, safeName)
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return nil, fmt.Errorf("create attachment directory for %q: %w", targetPath, err)
		}
		if err := os.WriteFile(targetPath, content, 0o600); err != nil {
			return nil, fmt.Errorf("write attachment %q: %w", targetPath, err)
		}
	}
	return warnings, nil
}

func classifyAttachmentDownloadError(pageID, filename string, err error) (*ExportWarning, error) {
	if err == nil {
		return nil, nil
	}

	var httpErr *client.HTTPError
	if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
		return &ExportWarning{
			Message: fmt.Sprintf(`download attachment %q for page %q skipped: %s`, filename, pageID, httpErr.Status),
		}, nil
	}

	return nil, fmt.Errorf("download attachment %q for page %q: %w", filename, pageID, err)
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
