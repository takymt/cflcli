# Markdown Notation Catalog for Confluence CLI

This document provides a concise catalog of Markdown notations for Confluence CLI.

## Headings

```md
# Heading 1
## Heading 2
### Heading 3
#### Heading 4
##### Heading 5
###### Heading 6
```

# Heading 1
## Heading 2
### Heading 3
#### Heading 4
##### Heading 5
###### Heading 6

### Unordered Lists

```md
- Hello!
- Hola!
  - Bonjour!
    - Hi!
```

- Hello!
- Hola!
  - Bonjour!
    - Hi!

### Ordered Lists

```md
1. First
2. Second
   1. Second-First
```

1. First
2. Second
   1. Second-First

### Links

```md
[Anchor text](https://developer.atlassian.com/cloud/confluence/)
```

[Anchor text](https://developer.atlassian.com/cloud/confluence/)

### Images and Caption Text

```md
![alt-text](https://developer.atlassian.com/favicon.ico)
*Caption*
_Caption (underscore)_
```

![alt-text](https://developer.atlassian.com/favicon.ico)
*Caption*

### Task Lists

```md
- [ ] Task 1
- [x] Task 2
```

- [ ] Task 1
- [x] Task 2

### Blockquotes (including nesting)

```md
> Quote
```

> Quote

### Horizontal Rule

```md
---
```

---

### Inline Text Styles

```md
- *italic*
- _italic_
- **bold**
- __bold__
- ~~strikethrough~~
- Inline `code`
- \*escaped literal asterisks\*
```

- *italic*
- _italic_
- **bold**
- __bold__
- ~~strikethrough~~
- Inline `code`
- \*escaped literal asterisks\*

### Underline (raw HTML fallback)

```md
<u>underline via raw html</u>
```

<u>underline via raw html</u>

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

### Table

```md
| Head | Head | Head |
| ---- | ---- | ---- |
| Text | Text | Text |
| Text | Text | Text |
```

| Head | Head | Head |
| ---- | ---- | ---- |
| Text | Text | Text |
| Text | Text | Text |

### Fenced Code Block

~~~md
```javascript
const codeBlock = "this is code block";
```
~~~

```javascript
const codeBlock = "this is code block";
```

### Expand (Details-like block)

Wrap content with `<details>` to create an expand/collapse block.
Use `<summary>` to define the block title.

```md
<details><summary>title</summary>
- Collapsed body line 1
- Collapsed body line 2
</details>
```

<details>
<summary>title</summary>
<ul>
<li>Collapsed body line 1</li>
<li>Collapsed body line 2</li>
</ul>
</details>

### Admonition-like Blocks

[GitHub Markdown alerts](https://docs.github.com/en/get-started/writing-on-github/getting-started-with-writing-and-formatting-on-github/basic-writing-and-formatting-syntax#alerts) style.

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

### Inline Comments

```md
<!-- TODO: add details about this section -->
```

<!-- TODO: add details about this section -->

Comments written in this format are not rendered on the published page.
Multi-line comment handling is currently out of scope.
