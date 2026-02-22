package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/spf13/cobra"
	"github.com/takymt/cflcli/internal/client"
	"github.com/takymt/cflcli/internal/config"
	migrateconv "github.com/takymt/cflcli/internal/migrate"
)

type migrateExportOptions struct {
	SpaceID        string
	SpaceKey       string
	RootPageID     string
	Out            string
	AttachmentsDir string
}

type migrateExportResult struct {
	SpaceID        string                `json:"space_id"`
	SpaceKey       string                `json:"space_key"`
	Out            string                `json:"out"`
	AttachmentsDir string                `json:"attachments_dir"`
	Pages          []migrateExportedPage `json:"pages"`
}

type migrateExportedPage struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	ParentID string `json:"parent_id,omitempty"`
	File     string `json:"file"`
}

const defaultMigrateAttachmentsDir = "attachments/_migrate"

func newMigrateExportCmd() *cobra.Command {
	opts := &migrateExportOptions{}

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export Confluence pages as markdown",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runMigrateExport(cmd.OutOrStdout(), opts)
		},
	}

	cmd.Flags().StringVar(&opts.SpaceID, "space-id", "", "space id (numeric)")
	cmd.Flags().StringVar(&opts.SpaceKey, "space-key", "", "space key (mutually exclusive with --space-id)")
	cmd.Flags().StringVar(&opts.RootPageID, "root-page-id", "", "root page id to export as subtree")
	cmd.Flags().StringVar(&opts.Out, "out", ".", "output directory")
	cmd.Flags().StringVar(&opts.AttachmentsDir, "attachments-dir", defaultMigrateAttachmentsDir, "attachments directory under --out")

	return cmd
}

func runMigrateExport(out io.Writer, opts *migrateExportOptions) error {
	cfg, err := loadConfig("")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	return RunMigrateExportWithConfig(out, opts, cfg)
}

// RunMigrateExportWithConfig runs migrate export with a provided config.
func RunMigrateExportWithConfig(out io.Writer, opts *migrateExportOptions, cfg *config.Config) error {
	if opts == nil {
		return fmt.Errorf("options are required")
	}

	outDir := strings.TrimSpace(opts.Out)
	if outDir == "" {
		outDir = "."
	}
	attachmentsDir := strings.TrimSpace(opts.AttachmentsDir)
	if attachmentsDir == "" {
		attachmentsDir = defaultMigrateAttachmentsDir
	}
	attachmentsDir = filepath.Clean(attachmentsDir)
	if filepath.IsAbs(attachmentsDir) {
		return fmt.Errorf("--attachments-dir must be a relative path")
	}
	if attachmentsDir == ".." || strings.HasPrefix(attachmentsDir, ".."+string(filepath.Separator)) {
		return fmt.Errorf("--attachments-dir must stay within --out")
	}

	runtime, err := newPageRuntime(cfg)
	if err != nil {
		return err
	}

	spaceID, err := runtime.resolveSpaceID(opts.SpaceID, opts.SpaceKey)
	if err != nil {
		return err
	}
	spaceKey, err := resolveMigrateExportSpaceKey(runtime, opts, spaceID)
	if err != nil {
		return err
	}

	pagesByID, err := listAllSpacePages(runtime.Client, spaceID)
	if err != nil {
		return err
	}

	rootPageID := strings.TrimSpace(opts.RootPageID)
	if rootPageID != "" {
		if _, ok := pagesByID[rootPageID]; !ok {
			root, rootErr := runtime.Client.GetPage(rootPageID)
			if rootErr != nil {
				return rootErr
			}
			if root.SpaceID != "" && root.SpaceID != spaceID {
				return fmt.Errorf("root page %q does not belong to space %q", rootPageID, spaceID)
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

	selectedPages := selectMigratePages(pagesByID, rootPageID)
	if len(selectedPages) == 0 {
		return fmt.Errorf("no pages found to export")
	}

	orderedPages := orderMigratePages(selectedPages, rootPageID)
	pageFileMap := buildMigratePageFileMap(orderedPages)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	exportedPages := make([]migrateExportedPage, 0, len(orderedPages))
	for _, page := range orderedPages {
		detail, err := runtime.Client.GetPage(page.ID)
		if err != nil {
			return err
		}
		exportRelPath := pageFileMap[page.ID]
		pageDir := filepath.Dir(exportRelPath)

		markdown, attachments, err := migrateconv.StorageToMarkdown(detail.Body.Storage.Value, func(filename string) string {
			attachmentPath := filepath.Join(attachmentsDir, page.ID, safeAttachmentFilename(filename))
			relativePath, relErr := filepath.Rel(pageDir, attachmentPath)
			if relErr != nil {
				return filepath.ToSlash(attachmentPath)
			}
			return filepath.ToSlash(relativePath)
		})
		if err != nil {
			return fmt.Errorf("convert page %q body to markdown: %w", page.ID, err)
		}

		if err := downloadMigrateAttachments(runtime.Client, outDir, attachmentsDir, page.ID, attachments); err != nil {
			return err
		}

		frontMatter := buildMigrateFrontMatter(detail.ID, detail.Title, detail.ParentID, spaceKey)
		content := frontMatter + markdown

		exportPath := filepath.Join(outDir, exportRelPath)
		if err := os.MkdirAll(filepath.Dir(exportPath), 0o755); err != nil {
			return fmt.Errorf("create export directory for %q: %w", exportPath, err)
		}
		if err := os.WriteFile(exportPath, []byte(content), 0o600); err != nil {
			return fmt.Errorf("write exported markdown %q: %w", exportPath, err)
		}

		exportedPages = append(exportedPages, migrateExportedPage{
			ID:       detail.ID,
			Title:    detail.Title,
			ParentID: strings.TrimSpace(detail.ParentID),
			File:     filepath.ToSlash(exportRelPath),
		})
	}

	result := migrateExportResult{
		SpaceID:        spaceID,
		SpaceKey:       spaceKey,
		Out:            outDir,
		AttachmentsDir: filepath.ToSlash(attachmentsDir),
		Pages:          exportedPages,
	}

	switch outputFlag {
	case "table":
		_, err := fmt.Fprintf(out, "Exported %d pages to %q.\n", len(exportedPages), outDir)
		return err
	case "json":
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	default:
		return fmt.Errorf("unsupported output format: %s", outputFlag)
	}
}

func resolveMigrateExportSpaceKey(runtime *pageRuntime, opts *migrateExportOptions, resolvedSpaceID string) (string, error) {
	if explicit := strings.TrimSpace(opts.SpaceKey); explicit != "" {
		return explicit, nil
	}
	if profileSpaceKey := strings.TrimSpace(runtime.Profile.SpaceKey); profileSpaceKey != "" && strings.TrimSpace(opts.SpaceID) == "" {
		return profileSpaceKey, nil
	}
	return runtime.Client.ResolveSpaceKeyByID(resolvedSpaceID)
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

func selectMigratePages(pagesByID map[string]client.Page, rootPageID string) map[string]client.Page {
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

func orderMigratePages(pagesByID map[string]client.Page, rootPageID string) []client.Page {
	children := map[string][]client.Page{}
	for _, page := range pagesByID {
		parentID := strings.TrimSpace(page.ParentID)
		children[parentID] = append(children[parentID], page)
	}
	for parentID := range children {
		sortMigratePages(children[parentID])
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
		sortMigratePages(roots)
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
	sortMigratePages(leftovers)
	for _, page := range leftovers {
		visit(page)
	}

	return ordered
}

func sortMigratePages(pages []client.Page) {
	sort.SliceStable(pages, func(i, j int) bool {
		leftTitle := strings.ToLower(strings.TrimSpace(pages[i].Title))
		rightTitle := strings.ToLower(strings.TrimSpace(pages[j].Title))
		if leftTitle == rightTitle {
			return pages[i].ID < pages[j].ID
		}
		return leftTitle < rightTitle
	})
}

func downloadMigrateAttachments(cli *client.Client, outDir, attachmentsDir, pageID string, filenames []string) error {
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

func buildMigrateFrontMatter(pageID, title, parentID, spaceKey string) string {
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

func buildMigratePageFileMap(orderedPages []client.Page) map[string]string {
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
