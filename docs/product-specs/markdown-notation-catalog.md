# Markdown Notation Catalog for Confluence Sync

## Purpose

Define a clear, implementation-oriented catalog of Markdown notations that should be recognized for Confluence page sync.

This document is based on observed content from Confluence page `1146968` in space `TEST`.

## Scope

This catalog describes user-facing notation and expected rendering intent.
It does not prescribe parser internals or API payload shape.

## Canonical Notations

### Headings

```md
# Heading 1
## Heading 2
### Heading 3
#### Heading 4
```

Expected intent: preserve heading levels.

### Unordered Lists (including nesting)

```md
- Hello!
- Hola!
  - Bonjour!
  - Hi!
```

Expected intent: preserve nested list hierarchy.

### Ordered Lists

```md
1. First
2. Second
```

Expected intent: render as ordered list.

### Links

```md
[Anchor text](https://developer.atlassian.com/cloud/confluence/)
```

Expected intent: render as clickable inline link.

### Images and Caption Text

```md
![alt-text](https://developer.atlassian.com/favicon.ico)
*Caption*
_Caption (underscore)_
```

Expected intent: render image with alt text and caption-like inline text formatting.

### Task Lists

```md
- [ ] Task 1
- [x] Task 2
- [x] This is not intended as a task item if escaped or plain text
```

Expected intent:
- checked and unchecked task items should map to Confluence tasks
- plain text that only looks similar should stay a normal list item

### Blockquotes (including nesting)

```md
> Quote
>> Nested quote
```

Expected intent: preserve quote nesting.

### Horizontal Rule

```md
---
```

Expected intent: render a thematic break.

### Inline Text Styles

```md
*italic*
_italic_
**bold**
__bold__
~~strikethrough~~
Inline `code`
\*escaped literal asterisks\*
```

Expected intent: preserve inline emphasis, code spans, and escaping.

### Underline (raw HTML fallback)

```md
<u>underline via raw html</u>
```

Expected intent: when supported, preserve raw HTML underline semantics.

### URL Autolink / Link Card Candidate

```md
https://zenn.dev/zenn/articles/markdown-guide
```

Expected intent: keep as a valid link, optionally rendered as inline card by Confluence rules.

### Emoji

```md
:smile: :thumbsup:
```

Expected intent: map to supported Confluence emoji where possible.

### Fenced Code Block

~~~md
```javascript
const codeBlock = "this is code block";
```
~~~

Expected intent: render as Confluence code block with language metadata.

### Expand (Details-like block)

```md
<details>
<summary>Collapsed title</summary>

Collapsed body line 1
Collapsed body line 2
</details>
```

Expected intent: map to Confluence expand/collapse structure.

### Admonition-like Blocks

```md
> [!NOTE]
> Note text

> [!TIP]
> Tip text

> [!IMPORTANT]
> Information text

> [!WARNING]
> Warning text

> [!CAUTION]
> Error text
```

Expected intent: map to Confluence informational panels (info, tip, note, warning) when possible.

## Non-Goals

- Exact one-to-one macro parity for every Confluence storage extension.
- Perfect round-trip between Markdown and storage format.
- Defining behavior for unsupported third-party Markdown extensions not listed above.

## Acceptance Criteria

- The CLI documentation and tests can reference this catalog as the canonical user-visible notation list.
- Each listed notation has at least one test case in implementation test suites.
- Unsupported notation must fail gracefully or degrade to readable plain text.
