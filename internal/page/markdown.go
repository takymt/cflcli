package page

import (
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
)

var (
	codeSpanRE       = regexp.MustCompile("`([^`]+)`")
	imageRE          = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	linkRE           = regexp.MustCompile(`\[([^\]]+)\]\((https?://[^)]+)\)`)
	autolinkRE       = regexp.MustCompile(`\bhttps?://[^\s<]+`)
	strongItalicRE   = regexp.MustCompile(`\*\*\*([^*]+)\*\*\*`)
	strongAsterisk   = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	strongUnderscore = regexp.MustCompile(`__([^_]+)__`)
	italicAsterisk   = regexp.MustCompile(`\*([^*]+)\*`)
	italicUnderscore = regexp.MustCompile(`_([^_]+)_`)
	strikeRE         = regexp.MustCompile(`~~([^~]+)~~`)
	inlineCommentRE  = regexp.MustCompile(`<!--.*?-->`)
	tableSepRE       = regexp.MustCompile(`^\|?\s*:?-{3,}:?\s*(\|\s*:?-{3,}:?\s*)+\|?$`)
	emojiCodeRE      = regexp.MustCompile(`:([a-z][a-z0-9_-]*):`)
	colorSpanRE      = regexp.MustCompile(`<span style="color:\s*rgb\(\s*\d{1,3}\s*,\s*\d{1,3}\s*,\s*\d{1,3}\s*\);\s*">([^<]*)</span>`)
	textAlignParaRE  = regexp.MustCompile(`^<p style="text-align:\s*(left|center|right)\s*;">(.*)</p>$`)
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
		if trimmed == "" || isInlineCommentLine(trimmed) {
			i++
			continue
		}

		if out, next, ok := convertFencedCode(lines, i); ok {
			parts = append(parts, out)
			i = next
			continue
		}
		if out, next, ok := convertDetails(lines, i); ok {
			parts = append(parts, out)
			i = next
			continue
		}
		if out, next, ok := convertTable(lines, i); ok {
			parts = append(parts, out)
			i = next
			continue
		}
		if out, next, ok := convertTaskList(lines, i); ok {
			parts = append(parts, out)
			i = next
			continue
		}
		if out, next, ok := convertUnorderedList(lines, i, leadingSpaces(lines[i])); ok {
			parts = append(parts, out)
			i = next
			continue
		}
		if out, next, ok := convertOrderedList(lines, i, leadingSpaces(lines[i])); ok {
			parts = append(parts, out)
			i = next
			continue
		}
		if out, next, ok := convertBlockquote(lines, i); ok {
			parts = append(parts, out)
			i = next
			continue
		}
		if out, ok := convertHorizontalRule(trimmed); ok {
			parts = append(parts, out)
			i++
			continue
		}
		if out, ok := convertHeading(trimmed); ok {
			parts = append(parts, out)
			i++
			continue
		}
		if out, ok := convertTextAlignParagraph(trimmed); ok {
			parts = append(parts, out)
			i++
			continue
		}

		var paragraphLines []string
		for i < len(lines) {
			current := strings.TrimSpace(lines[i])
			if current == "" || isBlockStart(lines, i) {
				break
			}
			paragraphLines = append(paragraphLines, current)
			i++
		}
		var rendered []string
		for _, line := range paragraphLines {
			rendered = append(rendered, convertInline(line))
		}
		parts = append(parts, "<p>"+strings.Join(rendered, "<br />")+"</p>")
	}

	return strings.Join(parts, "\n"), nil
}

func convertHeading(line string) (string, bool) {
	level := 0
	for level < len(line) && line[level] == '#' {
		level++
	}
	if level < 1 || level > 6 || len(line) <= level || line[level] != ' ' {
		return "", false
	}
	text := strings.TrimSpace(line[level+1-1:])
	return fmt.Sprintf("<h%d>%s</h%d>", level, convertInline(text), level), true
}

func convertHorizontalRule(line string) (string, bool) {
	if line == "---" || line == "***" || line == "___" {
		return "<hr />", true
	}
	return "", false
}

func convertFencedCode(lines []string, start int) (string, int, bool) {
	trimmed := strings.TrimSpace(lines[start])
	if !strings.HasPrefix(trimmed, "```") {
		return "", start, false
	}
	lang := strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
	var block []string
	i := start + 1
	for i < len(lines) {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "```") {
			i++
			break
		}
		block = append(block, lines[i])
		i++
	}
	var languageParam string
	if lang != "" {
		languageParam = `<ac:parameter ac:name="language">` + html.EscapeString(lang) + `</ac:parameter>`
	}
	return `<ac:structured-macro ac:name="code">` + languageParam + `<ac:plain-text-body><![CDATA[` +
		strings.Join(block, "\n") +
		`]]></ac:plain-text-body></ac:structured-macro>`, i, true
}

func convertTaskList(lines []string, start int) (string, int, bool) {
	indent := leadingSpaces(lines[start])
	trimmed := strings.TrimSpace(lines[start])
	status, text, ok := parseTaskItem(trimmed)
	if !ok {
		return "", start, false
	}

	var tasks []string
	i := start
	taskID := 1
	for i < len(lines) {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			break
		}
		if leadingSpaces(line) != indent {
			break
		}
		st, body, itemOK := parseTaskItem(strings.TrimSpace(line))
		if !itemOK {
			break
		}
		tasks = append(tasks, `<ac:task><ac:task-id>`+strconv.Itoa(taskID)+`</ac:task-id><ac:task-status>`+st+`</ac:task-status><ac:task-body>`+convertInline(body)+`</ac:task-body></ac:task>`)
		taskID++
		i++
	}

	if len(tasks) == 0 {
		tasks = append(tasks, `<ac:task><ac:task-id>1</ac:task-id><ac:task-status>`+status+`</ac:task-status><ac:task-body>`+convertInline(text)+`</ac:task-body></ac:task>`)
		i = start + 1
	}
	return "<ac:task-list>" + strings.Join(tasks, "") + "</ac:task-list>", i, true
}

func convertUnorderedList(lines []string, start int, baseIndent int) (string, int, bool) {
	type item struct {
		text   string
		nested []string
	}

	trimmed := strings.TrimSpace(lines[start])
	if !isUnorderedItem(trimmed) || leadingSpaces(lines[start]) != baseIndent || isTaskItem(trimmed) {
		return "", start, false
	}

	var items []item
	i := start
	for i < len(lines) {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			break
		}
		indent := leadingSpaces(line)
		t := strings.TrimSpace(line)
		if indent < baseIndent || !isUnorderedItem(t) || isTaskItem(t) {
			break
		}
		if indent > baseIndent {
			if len(items) == 0 {
				break
			}
			if nested, next, ok := convertUnorderedList(lines, i, indent); ok {
				items[len(items)-1].nested = append(items[len(items)-1].nested, nested)
				i = next
				continue
			}
			if nested, next, ok := convertOrderedList(lines, i, indent); ok {
				items[len(items)-1].nested = append(items[len(items)-1].nested, nested)
				i = next
				continue
			}
			break
		}
		items = append(items, item{text: strings.TrimSpace(t[2:])})
		i++
	}
	var rendered []string
	for _, it := range items {
		rendered = append(rendered, "<li><p>"+convertInline(it.text)+"</p>"+strings.Join(it.nested, "")+"</li>")
	}
	return "<ul>" + strings.Join(rendered, "") + "</ul>", i, len(items) > 0
}

func convertOrderedList(lines []string, start int, baseIndent int) (string, int, bool) {
	type item struct {
		text   string
		nested []string
	}

	trimmed := strings.TrimSpace(lines[start])
	ok, _ := orderedListItem(trimmed)
	if !ok || leadingSpaces(lines[start]) != baseIndent {
		return "", start, false
	}

	var items []item
	i := start
	for i < len(lines) {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			break
		}
		indent := leadingSpaces(line)
		t := strings.TrimSpace(line)
		isOrdered, value := orderedListItem(t)
		if indent < baseIndent || !isOrdered {
			break
		}
		if indent > baseIndent {
			if len(items) == 0 {
				break
			}
			if nested, next, nestedOK := convertUnorderedList(lines, i, indent); nestedOK {
				items[len(items)-1].nested = append(items[len(items)-1].nested, nested)
				i = next
				continue
			}
			if nested, next, nestedOK := convertOrderedList(lines, i, indent); nestedOK {
				items[len(items)-1].nested = append(items[len(items)-1].nested, nested)
				i = next
				continue
			}
			break
		}
		items = append(items, item{text: value})
		i++
	}
	var rendered []string
	for _, it := range items {
		rendered = append(rendered, "<li><p>"+convertInline(it.text)+"</p>"+strings.Join(it.nested, "")+"</li>")
	}
	return "<ol>" + strings.Join(rendered, "") + "</ol>", i, len(items) > 0
}

func convertTable(lines []string, start int) (string, int, bool) {
	if start+1 >= len(lines) {
		return "", start, false
	}
	headerLine := strings.TrimSpace(lines[start])
	separatorLine := strings.TrimSpace(lines[start+1])
	if !looksLikeTableRow(headerLine) || !tableSepRE.MatchString(separatorLine) {
		return "", start, false
	}

	headerCells := parseTableCells(headerLine)
	if len(headerCells) == 0 {
		return "", start, false
	}

	i := start + 2
	var rows [][]string
	for i < len(lines) {
		rowLine := strings.TrimSpace(lines[i])
		if !looksLikeTableRow(rowLine) || rowLine == "" {
			break
		}
		rows = append(rows, parseTableCells(rowLine))
		i++
	}

	var b strings.Builder
	b.WriteString("<table><tbody>")
	b.WriteString("<tr>")
	for _, cell := range headerCells {
		b.WriteString("<th>")
		b.WriteString(convertInline(cell))
		b.WriteString("</th>")
	}
	b.WriteString("</tr>")
	for _, row := range rows {
		b.WriteString("<tr>")
		for _, cell := range row {
			b.WriteString("<td>")
			b.WriteString(convertInline(cell))
			b.WriteString("</td>")
		}
		b.WriteString("</tr>")
	}
	b.WriteString("</tbody></table>")
	return b.String(), i, true
}

func convertBlockquote(lines []string, start int) (string, int, bool) {
	depth, content, ok := parseQuoteLine(lines[start])
	if !ok {
		return "", start, false
	}
	var entries []struct {
		depth   int
		content string
	}
	entries = append(entries, struct {
		depth   int
		content string
	}{depth: depth, content: content})

	i := start + 1
	for i < len(lines) {
		d, c, quoteOK := parseQuoteLine(lines[i])
		if !quoteOK {
			break
		}
		entries = append(entries, struct {
			depth   int
			content string
		}{depth: d, content: c})
		i++
	}

	currentDepth := 0
	var b strings.Builder
	for _, entry := range entries {
		for currentDepth < entry.depth {
			b.WriteString("<blockquote>")
			currentDepth++
		}
		for currentDepth > entry.depth {
			b.WriteString("</blockquote>")
			currentDepth--
		}
		b.WriteString("<p>")
		b.WriteString(convertInline(entry.content))
		b.WriteString("</p>")
	}
	for currentDepth > 0 {
		b.WriteString("</blockquote>")
		currentDepth--
	}
	return b.String(), i, true
}

func convertDetails(lines []string, start int) (string, int, bool) {
	trimmed := strings.TrimSpace(lines[start])
	if !strings.HasPrefix(trimmed, "<details><summary>") || !strings.Contains(trimmed, "</summary>") {
		return "", start, false
	}
	title := strings.TrimPrefix(trimmed, "<details><summary>")
	title = strings.SplitN(title, "</summary>", 2)[0]

	var bodyLines []string
	i := start + 1
	for i < len(lines) {
		if strings.TrimSpace(lines[i]) == "</details>" {
			i++
			break
		}
		bodyLines = append(bodyLines, lines[i])
		i++
	}
	body, _ := ConvertMarkdownToStorage(strings.Join(bodyLines, "\n"))
	return `<ac:structured-macro ac:name="expand"><ac:parameter ac:name="expanded">false</ac:parameter><ac:parameter ac:name="title">` +
		convertInline(title) + `</ac:parameter><ac:rich-text-body>` + body + `</ac:rich-text-body></ac:structured-macro>`, i, true
}

func convertTextAlignParagraph(line string) (string, bool) {
	sub := textAlignParaRE.FindStringSubmatch(line)
	if len(sub) != 3 {
		return "", false
	}
	align := sub[1]
	content := convertInline(sub[2])
	return `<p style="text-align: ` + align + `;">` + content + `</p>`, true
}

func convertInline(text string) string {
	text = inlineCommentRE.ReplaceAllString(text, "")
	text = strings.TrimSpace(text)

	placeholders := make(map[string]string)
	stash := func(value string) string {
		key := "@@INLINE" + strconv.Itoa(len(placeholders)) + "@@"
		placeholders[key] = value
		return key
	}

	text = codeSpanRE.ReplaceAllStringFunc(text, func(m string) string {
		value := strings.TrimSuffix(strings.TrimPrefix(m, "`"), "`")
		return stash("<code>" + html.EscapeString(value) + "</code>")
	})
	text = colorSpanRE.ReplaceAllStringFunc(text, func(m string) string {
		sub := colorSpanRE.FindStringSubmatch(m)
		if len(sub) != 2 {
			return m
		}
		start := strings.Index(m, ">")
		end := strings.LastIndex(m, "</span>")
		if start < 0 || end < 0 || end <= start {
			return m
		}
		openTag := m[:start+1]
		return stash(openTag + html.EscapeString(sub[1]) + "</span>")
	})
	text = emojiCodeRE.ReplaceAllStringFunc(text, func(m string) string {
		sub := emojiCodeRE.FindStringSubmatch(m)
		if len(sub) != 2 {
			return m
		}
		return stash(`<ac:emoticon ac:name="` + sub[1] + `" />`)
	})
	text = imageRE.ReplaceAllStringFunc(text, func(m string) string {
		sub := imageRE.FindStringSubmatch(m)
		if len(sub) != 3 {
			return m
		}
		return stash(`<ac:image ac:alt="` + html.EscapeString(sub[1]) + `"><ri:url ri:value="` + html.EscapeString(sub[2]) + `" /></ac:image>`)
	})
	text = linkRE.ReplaceAllStringFunc(text, func(m string) string {
		sub := linkRE.FindStringSubmatch(m)
		if len(sub) != 3 {
			return m
		}
		return stash(`<a href="` + html.EscapeString(sub[2]) + `">` + html.EscapeString(sub[1]) + `</a>`)
	})
	text = autolinkRE.ReplaceAllStringFunc(text, func(m string) string {
		return stash(`<a href="` + html.EscapeString(m) + `">` + html.EscapeString(m) + `</a>`)
	})

	escaped := html.EscapeString(text)
	escaped = preserveAllowedRawInlineHTML(escaped)
	escaped = strongItalicRE.ReplaceAllString(escaped, "<strong><em>$1</em></strong>")
	escaped = strongAsterisk.ReplaceAllString(escaped, "<strong>$1</strong>")
	escaped = strongUnderscore.ReplaceAllString(escaped, "<strong>$1</strong>")
	escaped = italicAsterisk.ReplaceAllString(escaped, "<em>$1</em>")
	escaped = italicUnderscore.ReplaceAllString(escaped, "<em>$1</em>")
	escaped = strikeRE.ReplaceAllString(escaped, `<span style="text-decoration: line-through;">$1</span>`)

	for key, value := range placeholders {
		escaped = strings.ReplaceAll(escaped, key, value)
	}
	return escaped
}

func preserveAllowedRawInlineHTML(escaped string) string {
	replacer := strings.NewReplacer(
		"&lt;sub&gt;", "<sub>",
		"&lt;/sub&gt;", "</sub>",
		"&lt;sup&gt;", "<sup>",
		"&lt;/sup&gt;", "</sup>",
		"&lt;ins&gt;", "<ins>",
		"&lt;/ins&gt;", "</ins>",
		"&lt;u&gt;", "<u>",
		"&lt;/u&gt;", "</u>",
	)
	return replacer.Replace(escaped)
}

func parseHeading(line string) (level int, text string, ok bool) {
	level = 0
	for level < len(line) && line[level] == '#' {
		level++
	}
	if level < 1 || level > 6 || len(line) <= level || line[level] != ' ' {
		return 0, "", false
	}
	return level, strings.TrimSpace(line[level+1-1:]), true
}

func isBlockStart(lines []string, index int) bool {
	line := strings.TrimSpace(lines[index])
	if line == "" || isInlineCommentLine(line) {
		return true
	}
	if _, _, ok := parseHeading(line); ok {
		return true
	}
	if _, ok := convertHorizontalRule(line); ok {
		return true
	}
	if strings.HasPrefix(line, "```") ||
		strings.HasPrefix(line, "<details><summary>") ||
		textAlignParaRE.MatchString(line) ||
		isTaskItem(line) ||
		isUnorderedItem(line) {
		return true
	}
	if ok, _ := orderedListItem(line); ok {
		return true
	}
	if _, _, ok := parseQuoteLine(lines[index]); ok {
		return true
	}
	if index+1 < len(lines) && looksLikeTableRow(line) && tableSepRE.MatchString(strings.TrimSpace(lines[index+1])) {
		return true
	}
	return false
}

func isInlineCommentLine(line string) bool {
	return strings.HasPrefix(line, "<!--") && strings.HasSuffix(line, "-->")
}

func isUnorderedItem(line string) bool {
	return strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ")
}

func isTaskItem(line string) bool {
	_, _, ok := parseTaskItem(line)
	return ok
}

func parseTaskItem(line string) (status string, text string, ok bool) {
	if len(line) < 6 || !strings.HasPrefix(line, "- [") {
		return "", "", false
	}
	if line[4] != ']' || line[5] != ' ' {
		return "", "", false
	}
	switch line[3] {
	case ' ':
		return "incomplete", strings.TrimSpace(line[6:]), true
	case 'x', 'X':
		return "complete", strings.TrimSpace(line[6:]), true
	default:
		return "", "", false
	}
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
	return true, strings.TrimSpace(line[idx+2:])
}

func leadingSpaces(line string) int {
	count := 0
	for _, r := range line {
		if r != ' ' {
			break
		}
		count++
	}
	return count
}

func looksLikeTableRow(line string) bool {
	return strings.HasPrefix(line, "|") && strings.Contains(strings.TrimPrefix(line, "|"), "|")
}

func parseTableCells(line string) []string {
	trimmed := strings.Trim(line, "|")
	parts := strings.Split(trimmed, "|")
	var out []string
	for _, part := range parts {
		out = append(out, strings.TrimSpace(part))
	}
	return out
}

func parseQuoteLine(line string) (depth int, content string, ok bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, ">") {
		return 0, "", false
	}
	for strings.HasPrefix(trimmed, ">") {
		depth++
		trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, ">"))
	}
	return depth, trimmed, true
}
