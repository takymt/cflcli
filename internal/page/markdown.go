package page

import (
	"fmt"
	"html"
	"strings"
)

// ConvertMarkdownToStorage converts the supported markdown subset to Confluence storage format.
func ConvertMarkdownToStorage(markdown string) (string, error) {
	if markdown == "" {
		return "", nil
	}

	lines := strings.Split(markdown, "\n")
	var parts []string

	for i := 0; i < len(lines); {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" {
			i++
			continue
		}

		if strings.HasPrefix(trimmed, "```") {
			var block []string
			i++
			for i < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[i]), "```") {
				block = append(block, lines[i])
				i++
			}
			if i < len(lines) {
				i++
			}
			parts = append(parts, fmt.Sprintf(
				`<ac:structured-macro ac:name="code"><ac:plain-text-body><![CDATA[%s]]></ac:plain-text-body></ac:structured-macro>`,
				strings.Join(block, "\n"),
			))
			continue
		}

		if level, text, ok := parseHeading(trimmed); ok {
			parts = append(parts, fmt.Sprintf("<h%d>%s</h%d>", level, html.EscapeString(text), level))
			i++
			continue
		}

		if isUnorderedItem(trimmed) {
			var items []string
			for i < len(lines) && isUnorderedItem(strings.TrimSpace(lines[i])) {
				item := strings.TrimSpace(lines[i])[2:]
				items = append(items, "<li><p>"+html.EscapeString(item)+"</p></li>")
				i++
			}
			parts = append(parts, "<ul>"+strings.Join(items, "")+"</ul>")
			continue
		}

		if ordered, _ := orderedListItem(trimmed); ordered {
			var items []string
			for i < len(lines) {
				ok, item := orderedListItem(strings.TrimSpace(lines[i]))
				if !ok {
					break
				}
				items = append(items, "<li><p>"+html.EscapeString(item)+"</p></li>")
				i++
			}
			parts = append(parts, "<ol>"+strings.Join(items, "")+"</ol>")
			continue
		}

		var paragraph []string
		for i < len(lines) {
			current := strings.TrimSpace(lines[i])
			if current == "" || startsBlock(current) {
				break
			}
			paragraph = append(paragraph, current)
			i++
		}
		parts = append(parts, "<p>"+html.EscapeString(strings.Join(paragraph, " "))+"</p>")
	}

	return strings.Join(parts, "\n"), nil
}

func parseHeading(line string) (level int, text string, ok bool) {
	switch {
	case strings.HasPrefix(line, "### "):
		return 3, line[4:], true
	case strings.HasPrefix(line, "## "):
		return 2, line[3:], true
	case strings.HasPrefix(line, "# "):
		return 1, line[2:], true
	}
	return 0, "", false
}

func startsBlock(line string) bool {
	if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "```") || isUnorderedItem(line) {
		return true
	}
	ok, _ := orderedListItem(line)
	return ok
}

func isUnorderedItem(line string) bool {
	return strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ")
}

func orderedListItem(line string) (bool, string) {
	idx := strings.Index(line, ". ")
	if idx <= 0 {
		return false, ""
	}
	for _, r := range line[:idx] {
		if r < '0' || r > '9' {
			return false, ""
		}
	}
	return true, line[idx+2:]
}
