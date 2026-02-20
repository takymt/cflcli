package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/takymt/cflcli/internal/body"
	"github.com/takymt/cflcli/internal/client"
	"github.com/takymt/cflcli/internal/config"
)

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

func resolvePageListSpaceID(opts *PageListOptions, profile *config.Profile, cli *client.Client) (string, error) {
	return resolvePageSpaceID(opts.SpaceID, opts.SpaceKey, profile, cli)
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

func loadPageStorageBody(path, format string) (*pageBodyInput, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read body file: %w", err)
	}
	normalized, err := normalizePageBodyFormat(format)
	if err != nil {
		return nil, err
	}

	frontMatterTitle := ""
	if normalized == body.FormatMarkdown {
		parsedTitle, bodyContent, parseErr := parseMarkdownFrontMatter(content)
		if parseErr != nil {
			return nil, parseErr
		}
		frontMatterTitle = parsedTitle
		content = bodyContent
	}

	storage, err := body.ToStorage(content, normalized)
	if err != nil {
		return nil, err
	}
	return &pageBodyInput{
		StorageBody:      storage,
		FrontMatterTitle: frontMatterTitle,
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

func parseMarkdownFrontMatter(content []byte) (string, []byte, error) {
	text := strings.ReplaceAll(string(content), "\r\n", "\n")
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return "", content, nil
	}

	closingIndex := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			closingIndex = i
			break
		}
	}
	if closingIndex == -1 {
		return "", nil, fmt.Errorf("invalid frontmatter: missing closing ---")
	}

	frontMatterLines := lines[1:closingIndex]
	title := ""
	for _, line := range frontMatterLines {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		if strings.ToLower(strings.TrimSpace(key)) != "title" {
			continue
		}
		raw := strings.TrimSpace(value)
		switch {
		case strings.HasPrefix(raw, "\"") && strings.HasSuffix(raw, "\"") && len(raw) >= 2:
			unquoted, err := strconv.Unquote(raw)
			if err != nil {
				title = raw[1 : len(raw)-1]
			} else {
				title = unquoted
			}
		case strings.HasPrefix(raw, "'") && strings.HasSuffix(raw, "'") && len(raw) >= 2:
			title = raw[1 : len(raw)-1]
		default:
			title = raw
		}
		break
	}

	bodyLines := lines[closingIndex+1:]
	bodyText := strings.Join(bodyLines, "\n")
	return strings.TrimSpace(title), []byte(bodyText), nil
}
