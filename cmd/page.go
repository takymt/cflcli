package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/takymt/cflcli/internal/attachment"
	"github.com/takymt/cflcli/internal/body"
	"github.com/takymt/cflcli/internal/client"
	"github.com/takymt/cflcli/internal/config"
)

type pageBodyInput struct {
	StorageBody         string
	FrontMatterTitle    string
	FrontMatterParentID string
	LocalImageAssets    []attachment.Asset
}

const pageBodyFormatValues = "markdown, storage"

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

type pageRuntime struct {
	Profile *config.Profile
	Client  *client.Client
}

func newPageRuntime(cfg *config.Config) (*pageRuntime, error) {
	profile, err := resolveProfile(cfg)
	if err != nil {
		return nil, err
	}

	cli, err := client.New(context.Background(), profile, os.Getenv("CFL_API_TOKEN"))
	if err != nil {
		return nil, err
	}

	return &pageRuntime{
		Profile: profile,
		Client:  cli,
	}, nil
}

func (runtime *pageRuntime) resolveSpaceID(spaceID, spaceKey string) (string, error) {
	spaceID = strings.TrimSpace(spaceID)
	spaceKey = strings.TrimSpace(spaceKey)
	if spaceID != "" && spaceKey != "" {
		return "", fmt.Errorf("--space-id and --space-key are mutually exclusive; specify only one")
	}
	if spaceID != "" {
		return spaceID, nil
	}
	if spaceKey != "" {
		return runtime.Client.ResolveSpaceIDByKey(spaceKey)
	}

	if strings.TrimSpace(runtime.Profile.SpaceKey) == "" {
		return "", fmt.Errorf("--space-id or --space-key is required; or configure space_key in profile")
	}
	return runtime.Client.ResolveSpaceIDByKey(strings.TrimSpace(runtime.Profile.SpaceKey))
}

func resolveAssetsRoot(explicitRoot string, profile *config.Profile) string {
	if strings.TrimSpace(explicitRoot) != "" || profile == nil {
		return explicitRoot
	}
	return strings.TrimSpace(profile.ContentRoot)
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

	localImageAssets := []attachment.Asset{}
	storage, err := body.ToStorage(content, normalized)
	if err != nil {
		return nil, err
	}
	if normalized == body.FormatMarkdown {
		storage, localImageAssets, err = attachment.ResolveMarkdownImageAssets(storage, path, assetsRoot)
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

func resolveTitle(flagTitle, frontMatterTitle string) string {
	flagTitle = strings.TrimSpace(flagTitle)
	if flagTitle != "" {
		return flagTitle
	}
	return strings.TrimSpace(frontMatterTitle)
}

func validateTitleSources(flagTitle, frontMatterTitle string) error {
	if strings.TrimSpace(flagTitle) != "" && strings.TrimSpace(frontMatterTitle) != "" {
		return fmt.Errorf("--title and frontmatter title are mutually exclusive; specify only one")
	}
	return nil
}

func resolveParentID(flagParentID, frontMatterParentID string) string {
	flagParentID = strings.TrimSpace(flagParentID)
	if flagParentID != "" {
		return flagParentID
	}
	return strings.TrimSpace(frontMatterParentID)
}

func validateParentIDSources(flagParentID, frontMatterParentID string) error {
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
