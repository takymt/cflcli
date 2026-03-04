package page

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

var orderedListPattern = regexp.MustCompile(`^[0-9]+\.\s+`)

func ConvertMarkdown(body string) (string, error) {
	lines := strings.Split(body, "\n")
	var out strings.Builder

	for i := 0; i < len(lines); {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			i++
			continue
		}

		if strings.HasPrefix(trimmed, "```") {
			j := i + 1
			var block []string
			for ; j < len(lines); j++ {
				if strings.HasPrefix(strings.TrimSpace(lines[j]), "```") {
					break
				}
				block = append(block, lines[j])
			}
			if j == len(lines) {
				return "", fmt.Errorf("unterminated fenced code block")
			}
			out.WriteString(`<ac:structured-macro ac:name="code"><ac:plain-text-body><![CDATA[`)
			out.WriteString(strings.Join(block, "\n"))
			out.WriteString(`]]></ac:plain-text-body></ac:structured-macro>`)
			i = j + 1
			continue
		}

		if level := headingLevel(trimmed); level > 0 {
			content := strings.TrimSpace(trimmed[level:])
			out.WriteString(fmt.Sprintf("<h%d>%s</h%d>", level, html.EscapeString(content), level))
			i++
			continue
		}

		if strings.HasPrefix(trimmed, "- ") {
			out.WriteString("<ul>")
			for ; i < len(lines); i++ {
				item := strings.TrimSpace(lines[i])
				if !strings.HasPrefix(item, "- ") {
					break
				}
				out.WriteString("<li>")
				out.WriteString(html.EscapeString(strings.TrimSpace(item[2:])))
				out.WriteString("</li>")
			}
			out.WriteString("</ul>")
			continue
		}

		if orderedListPattern.MatchString(trimmed) {
			out.WriteString("<ol>")
			for ; i < len(lines); i++ {
				item := strings.TrimSpace(lines[i])
				if !orderedListPattern.MatchString(item) {
					break
				}
				content := orderedListPattern.ReplaceAllString(item, "")
				out.WriteString("<li>")
				out.WriteString(html.EscapeString(strings.TrimSpace(content)))
				out.WriteString("</li>")
			}
			out.WriteString("</ol>")
			continue
		}

		var paragraph []string
		for ; i < len(lines); i++ {
			item := strings.TrimSpace(lines[i])
			if item == "" || strings.HasPrefix(item, "#") || strings.HasPrefix(item, "- ") ||
				strings.HasPrefix(item, "```") || orderedListPattern.MatchString(item) {
				break
			}
			paragraph = append(paragraph, item)
		}

		out.WriteString("<p>")
		out.WriteString(html.EscapeString(strings.Join(paragraph, " ")))
		out.WriteString("</p>")
	}

	return out.String(), nil
}

func headingLevel(line string) int {
	switch {
	case strings.HasPrefix(line, "### "):
		return 3
	case strings.HasPrefix(line, "## "):
		return 2
	case strings.HasPrefix(line, "# "):
		return 1
	default:
		return 0
	}
}
