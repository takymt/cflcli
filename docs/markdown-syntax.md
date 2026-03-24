# Markdown Syntax Catalog for Confluence CLI

This document provides a concise catalog of Markdown syntax for Confluence CLI.

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

## Inline Styles

```md
- **This is bold text**
- _This text is italicized_
- ~~This was mistaken text~~
- **This text is _extremely_ important**
- ***All this text is important***
- This is `quoted code` text
- This is a <sub>subscript</sub> text
- This is a <sup>superscript</sup> text
- This is an <ins>underlined</ins> text
```

- **This is bold text**
- _This text is italicized_
- ~~This was mistaken text~~
- **This text is _extremely_ important**
- ***All this text is important***
- This is `quoted code` text
- This is a <sub>subscript</sub> text
- This is a <sup>superscript</sup> text
- This is an <ins>underlined</ins> text

## Unordered Lists

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

## Ordered Lists

```md
1. First
2. Second
   1. Second-First
```

1. First
2. Second
   1. Second-First

## Links

```md
[Anchor text](https://developer.atlassian.com/cloud/confluence/)
```

[Anchor text](https://developer.atlassian.com/cloud/confluence/)

## Images and Caption Text

```md
![alt-text](https://developer.atlassian.com/favicon.ico)
*Caption*
```

![alt-text](https://developer.atlassian.com/favicon.ico)
*Caption*

## Task Lists

```md
- [ ] Task 1
- [x] Task 2
```

- [ ] Task 1
- [x] Task 2

## Blockquotes (including nesting)

Nested blockquotes are unsupported (legacy in Confluence).

```md
> Quote
```

> Quote

## Horizontal Rule

```md
---
```

---

## URL Autolink / Link Card Candidate

```md
https://github.com/takymt/cflcli/blob/main/docs/markdown-syntax.md
```

https://github.com/takymt/cflcli/blob/main/docs/markdown-syntax.md

Rendered in Confluence: Inline Smart Link (resolved), or raw URL (fallback).

## Line breaks

```md
This example
Will span two lines
```

This example

Will span two lines

## Table

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

## Fenced Code Block

~~~md
```javascript
const codeBlock = "this is code block";
```
~~~

```javascript
const codeBlock = "this is code block";
```

## Expand (Details-like block)

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

## Inline Comments

```md
<!-- TODO: add details about this section -->

<!-- This is
a multi-line comment -->
```

<!-- TODO: add details about this section -->

Comments written in this format are not rendered on the published page.

## Emoji

```md
:smile:
:sad:
:information:
:warning:
```


![smile](/docs/images/emojis/smile.svg)

![sad](/docs/images/emojis/sad.svg)

![information](/docs/images/emojis/information.svg)

![warning](/docs/images/emojis/warning.svg)

## Color

```md
<span style="color: rgb(255,0,0);">red text</span>
```

<span style="color: rgb(255,0,0);">red text</span>

## Text Align

```md
<p style="text-align: left;">left aligned text</p>
<p style="text-align: center;">centered text</p>
<p style="text-align: right;">right aligned text</p>
```

<p align="left">left aligned text</p>
<p align="center">centered text</p>
<p align="right">right aligned text</p>

## Alerts

```md
:::info
information
:::

:::note
note
:::

:::success
success
:::

:::warning
warning
:::

:::error
error
:::
```

![information](/docs/images/panels/information.png)

![note](/docs/images/panels/note.png)

![success](/docs/images/panels/success.png)

![warning](/docs/images/panels/warning.png)

![error](/docs/images/panels/error.png)

## Mermaid Diagrams

This CLI supports diagrams written with [mermaid.js](https://mermaid.js.org/).

~~~md
```mermaid
graph TD
  A --> B
```
~~~

You can also set image options in the fence info:

~~~md
```mermaid width=900 align=right
graph TD
  A --> B
```
~~~

- Supported options:
  - `width=<number>`
  - `align=left|center|right`
- During `page sync`, `mermaid` fenced blocks are rendered into local SVG files and referenced as page attachments (`mermaid-1.svg`, `mermaid-2.svg`, ...).
- Rendering uses [mermaid-cli](https://github.com/mermaid-js/mermaid-cli).
- If one mermaid block exceeds 2000 characters, sync returns an error.
