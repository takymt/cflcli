package migrate

import (
	stdhtml "html"
	"strconv"
	"strings"

	xhtml "golang.org/x/net/html"
)

func convertHTMLToMarkdownSyntax(input string) string {
	if strings.TrimSpace(input) == "" {
		return input
	}

	doc, err := xhtml.Parse(strings.NewReader(`<div data-cfl-root="1">` + input + `</div>`))
	if err != nil {
		return input
	}
	root := findElementNode(doc, "div", "data-cfl-root", "1")
	if root == nil {
		return input
	}

	var b strings.Builder
	for node := root.FirstChild; node != nil; node = node.NextSibling {
		b.WriteString(renderMarkdownNode(node))
	}
	return normalizeMarkdownWhitespace(b.String())
}

func renderMarkdownNode(node *xhtml.Node) string {
	if node == nil {
		return ""
	}

	switch node.Type {
	case xhtml.TextNode:
		return node.Data
	case xhtml.CommentNode:
		return "<!--" + node.Data + "-->"
	case xhtml.DocumentNode:
		return renderMarkdownChildren(node)
	case xhtml.ElementNode:
		tag := strings.ToLower(strings.TrimSpace(node.Data))
		switch tag {
		case "h1", "h2", "h3", "h4", "h5", "h6":
			level, _ := strconv.Atoi(strings.TrimPrefix(tag, "h"))
			if level < 1 || level > 6 {
				level = 1
			}
			text := strings.TrimSpace(renderInlineChildren(node))
			if text == "" {
				return ""
			}
			return "\n" + strings.Repeat("#", level) + " " + text + "\n\n"
		case "p":
			content := strings.TrimSpace(renderMarkdownChildren(node))
			if content == "" {
				return ""
			}
			return content + "\n\n"
		case "br":
			return "\n"
		case "hr":
			return "\n---\n\n"
		case "strong", "b":
			content := strings.TrimSpace(renderInlineChildren(node))
			if content == "" {
				return ""
			}
			return "**" + content + "**"
		case "em", "i":
			content := strings.TrimSpace(renderInlineChildren(node))
			if content == "" {
				return ""
			}
			return "*" + content + "*"
		case "code":
			content := strings.TrimSpace(renderInlineChildren(node))
			if content == "" {
				return ""
			}
			return "`" + strings.ReplaceAll(content, "`", "\\`") + "`"
		case "span":
			if isStrikeSpan(node) {
				content := strings.TrimSpace(renderInlineChildren(node))
				if content == "" {
					return ""
				}
				return "~~" + content + "~~"
			}
			return renderMarkdownChildren(node)
		case "a":
			href := strings.TrimSpace(getAttr(node, "href"))
			label := strings.TrimSpace(renderInlineChildren(node))
			if href == "" {
				return label
			}
			if label == "" {
				label = href
			}
			return "[" + label + "](" + href + ")"
		case "blockquote":
			content := strings.TrimSpace(renderMarkdownChildren(node))
			if content == "" {
				return ""
			}
			return "\n" + prefixMarkdownLines(content, "> ") + "\n\n"
		case "ul":
			return renderMarkdownList(node, false, 0)
		case "ol":
			return renderMarkdownList(node, true, 0)
		case "li":
			return strings.TrimSpace(renderMarkdownChildren(node))
		case "u":
			return renderRawNode(node)
		case "ac:task-list":
			return renderTaskList(node)
		case "ac:emoticon":
			name := strings.TrimSpace(getAttr(node, "ac:name"))
			if name == "" {
				return ""
			}
			return ":" + name + ":"
		case "ac:link":
			return renderACLink(node)
		case "ac:plain-text-link-body":
			return renderInlineChildren(node)
		case "ac:adf-content":
			// Keep plain text content if parser surfaces this outside an unsupported fallback.
			return renderMarkdownChildren(node)
		default:
			if strings.Contains(tag, ":") {
				return renderRawNode(node)
			}
			return renderMarkdownChildren(node)
		}
	default:
		return ""
	}
}

func renderMarkdownChildren(node *xhtml.Node) string {
	var b strings.Builder
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		b.WriteString(renderMarkdownNode(child))
	}
	return b.String()
}

func renderInlineChildren(node *xhtml.Node) string {
	return collapseInlineWhitespace(renderMarkdownChildren(node))
}

func renderMarkdownList(listNode *xhtml.Node, ordered bool, depth int) string {
	items := make([]string, 0)
	index := 1
	for child := listNode.FirstChild; child != nil; child = child.NextSibling {
		if child.Type != xhtml.ElementNode || !strings.EqualFold(child.Data, "li") {
			continue
		}
		item := renderMarkdownListItem(child, ordered, depth, index)
		if strings.TrimSpace(item) != "" {
			items = append(items, item)
			index++
		}
	}
	if len(items) == 0 {
		return ""
	}
	return "\n" + strings.Join(items, "\n") + "\n\n"
}

func renderMarkdownListItem(itemNode *xhtml.Node, ordered bool, depth, index int) string {
	inlineParts := make([]string, 0)
	nestedParts := make([]string, 0)
	for child := itemNode.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == xhtml.ElementNode {
			tag := strings.ToLower(child.Data)
			if tag == "ul" {
				nestedParts = append(nestedParts, renderMarkdownListNested(child, false, depth+1))
				continue
			}
			if tag == "ol" {
				nestedParts = append(nestedParts, renderMarkdownListNested(child, true, depth+1))
				continue
			}
		}
		inlineParts = append(inlineParts, renderMarkdownNode(child))
	}

	content := strings.TrimSpace(collapseInlineWhitespace(strings.Join(inlineParts, "")))
	indent := strings.Repeat("  ", depth)
	marker := "- "
	if ordered {
		marker = strconv.Itoa(index) + ". "
	}

	var b strings.Builder
	b.WriteString(indent)
	b.WriteString(marker)
	b.WriteString(content)

	for _, nested := range nestedParts {
		nested = strings.TrimRight(nested, "\n")
		if strings.TrimSpace(nested) == "" {
			continue
		}
		b.WriteString("\n")
		b.WriteString(nested)
	}

	return strings.TrimRight(b.String(), "\n")
}

func renderMarkdownListNested(listNode *xhtml.Node, ordered bool, depth int) string {
	items := make([]string, 0)
	index := 1
	for child := listNode.FirstChild; child != nil; child = child.NextSibling {
		if child.Type != xhtml.ElementNode || !strings.EqualFold(child.Data, "li") {
			continue
		}
		item := renderMarkdownListItem(child, ordered, depth, index)
		if strings.TrimSpace(item) != "" {
			items = append(items, item)
			index++
		}
	}
	return strings.Join(items, "\n")
}

func renderTaskList(node *xhtml.Node) string {
	lines := make([]string, 0)
	for task := node.FirstChild; task != nil; task = task.NextSibling {
		if task.Type != xhtml.ElementNode || !strings.EqualFold(task.Data, "ac:task") {
			continue
		}
		status := ""
		bodyText := ""
		for child := task.FirstChild; child != nil; child = child.NextSibling {
			if child.Type != xhtml.ElementNode {
				continue
			}
			switch strings.ToLower(child.Data) {
			case "ac:task-status":
				status = strings.TrimSpace(collapseInlineWhitespace(renderMarkdownChildren(child)))
			case "ac:task-body":
				bodyText = strings.TrimSpace(collapseInlineWhitespace(renderMarkdownChildren(child)))
			}
		}
		if bodyText == "" {
			continue
		}
		checkbox := "[ ]"
		if strings.EqualFold(status, "complete") {
			checkbox = "[x]"
		}
		lines = append(lines, "- "+checkbox+" "+bodyText)
	}
	if len(lines) == 0 {
		return ""
	}
	return "\n" + strings.Join(lines, "\n") + "\n\n"
}

func renderACLink(node *xhtml.Node) string {
	var (
		url   string
		title string
	)
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type != xhtml.ElementNode {
			continue
		}
		switch strings.ToLower(child.Data) {
		case "ri:url":
			url = strings.TrimSpace(getAttr(child, "ri:value"))
		case "ri:page":
			title = strings.TrimSpace(getAttr(child, "ri:content-title"))
		case "ac:plain-text-link-body":
			label := strings.TrimSpace(renderInlineChildren(child))
			if label != "" {
				title = label
			}
		}
	}
	if url != "" {
		label := title
		if label == "" {
			label = url
		}
		return "[" + label + "](" + url + ")"
	}
	if title != "" {
		return "[" + title + "]"
	}
	return renderRawNode(node)
}

func isStrikeSpan(node *xhtml.Node) bool {
	style := strings.ToLower(strings.TrimSpace(getAttr(node, "style")))
	return strings.Contains(style, "line-through")
}

func getAttr(node *xhtml.Node, key string) string {
	for _, attr := range node.Attr {
		if strings.EqualFold(attr.Key, key) {
			return attr.Val
		}
	}
	return ""
}

func renderRawNode(node *xhtml.Node) string {
	if node == nil {
		return ""
	}
	var b strings.Builder
	if err := xhtml.Render(&b, node); err != nil {
		return ""
	}
	return b.String()
}

func prefixMarkdownLines(content, prefix string) string {
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	for i, line := range lines {
		line = strings.TrimRight(line, " \t")
		if strings.TrimSpace(line) == "" {
			lines[i] = strings.TrimSpace(prefix)
			continue
		}
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}

func collapseInlineWhitespace(value string) string {
	fields := strings.Fields(value)
	return strings.Join(fields, " ")
}

func normalizeMarkdownWhitespace(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	lines := strings.Split(value, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	value = strings.Join(lines, "\n")
	value = strings.ReplaceAll(value, "\u00a0", " ")
	value = stdhtml.UnescapeString(value)
	return value
}

func findElementNode(node *xhtml.Node, tag, attrKey, attrValue string) *xhtml.Node {
	if node == nil {
		return nil
	}
	if node.Type == xhtml.ElementNode && strings.EqualFold(node.Data, tag) {
		if attrKey == "" {
			return node
		}
		if strings.EqualFold(getAttr(node, attrKey), attrValue) {
			return node
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if found := findElementNode(child, tag, attrKey, attrValue); found != nil {
			return found
		}
	}
	return nil
}
