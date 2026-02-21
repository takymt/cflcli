package cmd

import (
	"fmt"
	stdhtml "html"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/takymt/cflcli/internal/body"
	"github.com/takymt/cflcli/internal/client"
	"github.com/takymt/cflcli/internal/config"
)

type pageBodyInput struct {
	StorageBody         string
	FrontMatterTitle    string
	FrontMatterParentID string
	LocalImageAssets    []pageLocalImageAsset
}

const pageBodyFormatValues = "markdown, storage"

var storageImageURLPattern = regexp.MustCompile(`(?s)<ac:image([^>]*)><ri:url ri:value="([^"]*)" /></ac:image>`)

type pageLocalImageAsset struct {
	Filename   string
	SourcePath string
}

func newPageCmd() *cobra.Command {
	pageCmd := &cobra.Command{
		Use:   "page",
		Short: "Manage Confluence pages",
	}

	pageCmd.AddCommand(newPageListCmd())
	pageCmd.AddCommand(newPageGetCmd())
	pageCmd.AddCommand(newPageCreateCmd())
	pageCmd.AddCommand(newPageUpdateCmd())
	pageCmd.AddCommand(newPageDeleteCmd())

	return pageCmd
}

func resolveProfile(cfg *config.Config) (*config.Profile, error) {
	if profileFlag != "" {
		profile := cfg.FindProfile(profileFlag)
		if profile == nil {
			return nil, fmt.Errorf("profile %q not found", profileFlag)
		}
		return profile, nil
	}

	profile := cfg.CurrentProfile()
	if profile == nil {
		return nil, fmt.Errorf("no current profile; run 'cfl config init' or 'cfl use <name>'")
	}
	return profile, nil
}

func resolvePageSpaceID(spaceID, spaceKey string, profile *config.Profile, cli *client.Client) (string, error) {
	spaceID = strings.TrimSpace(spaceID)
	spaceKey = strings.TrimSpace(spaceKey)
	if spaceID != "" && spaceKey != "" {
		return "", fmt.Errorf("--space-id and --space-key are mutually exclusive; specify only one")
	}
	if spaceID != "" {
		return spaceID, nil
	}
	if spaceKey == "" {
		spaceKey = strings.TrimSpace(profile.SpaceKey)
	}
	if spaceKey == "" {
		return "", fmt.Errorf("--space-id or --space-key is required; or configure space_key in profile")
	}

	return cli.ResolveSpaceIDByKey(spaceKey)
}

func normalizePageBodyFormat(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return body.FormatMarkdown, nil
	}
	format, err := body.NormalizeFormat(value)
	if err != nil {
		return "", fmt.Errorf("invalid --body-format %q; allowed values: %s", value, pageBodyFormatValues)
	}
	return format, nil
}

func loadPageStorageBody(path, format, assetsRoot string) (*pageBodyInput, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read body file: %w", err)
	}
	normalized, err := normalizePageBodyFormat(format)
	if err != nil {
		return nil, err
	}

	frontMatterTitle := ""
	frontMatterParentID := ""
	if normalized == body.FormatMarkdown {
		parsedTitle, parsedParentID, bodyContent, parseErr := parseMarkdownFrontMatter(content)
		if parseErr != nil {
			return nil, parseErr
		}
		frontMatterTitle = parsedTitle
		frontMatterParentID = parsedParentID
		content = bodyContent
	}

	localImageAssets := []pageLocalImageAsset{}
	storage, err := body.ToStorage(content, normalized)
	if err != nil {
		return nil, err
	}
	if normalized == body.FormatMarkdown {
		storage, localImageAssets, err = resolvePageLocalImageAssets(storage, path, assetsRoot)
		if err != nil {
			return nil, err
		}
	}

	return &pageBodyInput{
		StorageBody:         storage,
		FrontMatterTitle:    frontMatterTitle,
		FrontMatterParentID: frontMatterParentID,
		LocalImageAssets:    localImageAssets,
	}, nil
}

func resolvePageTitle(flagTitle, frontMatterTitle string) string {
	flagTitle = strings.TrimSpace(flagTitle)
	if flagTitle != "" {
		return flagTitle
	}
	return strings.TrimSpace(frontMatterTitle)
}

func validatePageTitleSources(flagTitle, frontMatterTitle string) error {
	if strings.TrimSpace(flagTitle) != "" && strings.TrimSpace(frontMatterTitle) != "" {
		return fmt.Errorf("--title and frontmatter title are mutually exclusive; specify only one")
	}
	return nil
}

func resolvePageParentID(flagParentID, frontMatterParentID string) string {
	flagParentID = strings.TrimSpace(flagParentID)
	if flagParentID != "" {
		return flagParentID
	}
	return strings.TrimSpace(frontMatterParentID)
}

func validatePageParentIDSources(flagParentID, frontMatterParentID string) error {
	if strings.TrimSpace(flagParentID) != "" && strings.TrimSpace(frontMatterParentID) != "" {
		return fmt.Errorf("--parent-id and frontmatter parent-id are mutually exclusive; specify only one")
	}
	return nil
}

func parseMarkdownFrontMatter(content []byte) (string, string, []byte, error) {
	text := strings.ReplaceAll(string(content), "\r\n", "\n")
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return "", "", content, nil
	}

	closingIndex := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			closingIndex = i
			break
		}
	}
	if closingIndex == -1 {
		return "", "", nil, fmt.Errorf("invalid frontmatter: missing closing ---")
	}

	frontMatterLines := lines[1:closingIndex]
	title := ""
	parentID := ""
	for _, line := range frontMatterLines {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		raw := strings.TrimSpace(value)
		parsed := parseFrontMatterValue(raw)
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "title":
			if title == "" {
				title = parsed
			}
		case "parent-id", "parent_id", "parentid":
			if parentID == "" {
				parentID = parsed
			}
		}
	}

	bodyLines := lines[closingIndex+1:]
	bodyText := strings.Join(bodyLines, "\n")
	return strings.TrimSpace(title), strings.TrimSpace(parentID), []byte(bodyText), nil
}

func parseFrontMatterValue(raw string) string {
	switch {
	case strings.HasPrefix(raw, "\"") && strings.HasSuffix(raw, "\"") && len(raw) >= 2:
		unquoted, err := strconv.Unquote(raw)
		if err != nil {
			return raw[1 : len(raw)-1]
		}
		return unquoted
	case strings.HasPrefix(raw, "'") && strings.HasSuffix(raw, "'") && len(raw) >= 2:
		return raw[1 : len(raw)-1]
	default:
		return raw
	}
}

func resolvePageLocalImageAssets(storage, bodyFilePath, assetsRoot string) (string, []pageLocalImageAsset, error) {
	baseDir := filepath.Dir(bodyFilePath)
	if strings.TrimSpace(baseDir) == "" {
		baseDir = "."
	}
	baseDirAbs, err := filepath.Abs(baseDir)
	if err != nil {
		return "", nil, fmt.Errorf("resolve body file directory: %w", err)
	}

	assetsRootAbs, err := resolveAssetsRootPath(assetsRoot, baseDirAbs)
	if err != nil {
		return "", nil, err
	}

	localImageAssets := make([]pageLocalImageAsset, 0)
	seenByFilename := make(map[string]string)
	var resolveErr error

	resolvedStorage := storageImageURLPattern.ReplaceAllStringFunc(storage, func(match string) string {
		if resolveErr != nil {
			return match
		}

		submatches := storageImageURLPattern.FindStringSubmatch(match)
		if len(submatches) != 3 {
			return match
		}

		attrs := submatches[1]
		rawSource := stdhtml.UnescapeString(submatches[2])
		source := strings.TrimSpace(rawSource)
		if source == "" || isRemoteImageSource(source) {
			return match
		}

		resolvedPath, err := resolveLocalImagePath(source, baseDirAbs, assetsRootAbs)
		if err != nil {
			resolveErr = err
			return match
		}

		info, err := os.Stat(resolvedPath)
		if err != nil {
			resolveErr = fmt.Errorf("resolve local image %q: %w", source, err)
			return match
		}
		if info.IsDir() {
			resolveErr = fmt.Errorf("resolve local image %q: %q is a directory", source, resolvedPath)
			return match
		}

		filename := filepath.Base(resolvedPath)
		key := strings.ToLower(filename)
		if prev, ok := seenByFilename[key]; ok && filepath.Clean(prev) != filepath.Clean(resolvedPath) {
			resolveErr = fmt.Errorf("duplicate image filename %q from %q and %q; rename one file", filename, prev, resolvedPath)
			return match
		}
		if _, ok := seenByFilename[key]; !ok {
			seenByFilename[key] = resolvedPath
			localImageAssets = append(localImageAssets, pageLocalImageAsset{
				Filename:   filename,
				SourcePath: resolvedPath,
			})
		}

		return `<ac:image` + attrs + `><ri:attachment ri:filename="` +
			stdhtml.EscapeString(filename) +
			`" /></ac:image>`
	})

	if resolveErr != nil {
		return "", nil, resolveErr
	}

	return resolvedStorage, localImageAssets, nil
}

func resolveAssetsRootPath(rawRoot, fallback string) (string, error) {
	root := strings.TrimSpace(rawRoot)
	if root == "" {
		root = fallback
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve --assets-root: %w", err)
	}
	return filepath.Clean(rootAbs), nil
}

func isRemoteImageSource(source string) bool {
	lower := strings.ToLower(strings.TrimSpace(source))
	return strings.HasPrefix(lower, "http://") ||
		strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "data:") ||
		strings.HasPrefix(lower, "//")
}

func resolveLocalImagePath(source, baseDirAbs, assetsRootAbs string) (string, error) {
	if strings.HasPrefix(source, "/") {
		rel := filepath.Clean(filepath.FromSlash(strings.TrimLeft(source, "/")))
		if rel == "." || rel == "" {
			return "", fmt.Errorf("invalid root-based image path %q", source)
		}
		resolved := filepath.Clean(filepath.Join(assetsRootAbs, rel))
		if !isPathWithinRoot(resolved, assetsRootAbs) {
			return "", fmt.Errorf("image path %q escapes --assets-root %q", source, assetsRootAbs)
		}
		return resolved, nil
	}

	normalized := filepath.FromSlash(source)
	if filepath.IsAbs(normalized) {
		return filepath.Clean(normalized), nil
	}
	return filepath.Clean(filepath.Join(baseDirAbs, normalized)), nil
}

func isPathWithinRoot(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func uploadPageLocalImageAssets(cli *client.Client, pageID string, assets []pageLocalImageAsset) error {
	for _, asset := range assets {
		if err := cli.UpsertPageAttachment(pageID, asset.Filename, asset.SourcePath); err != nil {
			return fmt.Errorf("upload image %q: %w", asset.Filename, err)
		}
	}
	return nil
}
