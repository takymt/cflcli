# Page MVP Observable Test Cases

## Purpose

This document expresses `page-mvp` behavior as transformations:

`(InputState, Command) -> Output`

Each case treats the CLI as a function over observable state.

## Output Shape

Each output is described in the same shape:

- `exit`: process exit result
- `stdout`: user-visible output
- `local`: resulting local file state
- `remote`: resulting remote page state

## `cfl page new`

### FP-NEW-001 Explicit parent

```text
(
  InputState{
    local.files = {},
    remote.space[100] exists,
    remote.page[200] exists,
    remote.children[200] does not contain title "guide"
  },
  Command("cfl page new guide.md --space-id 100 --parent-id 200")
)
-> Output{
  exit = success,
  stdout contains page URL,
  local.files["guide.md"] = MarkdownFile{
    frontmatter.space-id = 100,
    frontmatter.page-id = <new page id>,
    frontmatter.parent-id = 200,
    body = ""
  },
  remote.children[200] contains Page{
    title = "guide"
  }
}
```

### FP-NEW-002 Resolved root parent

```text
(
  InputState{
    local.files = {},
    remote.space[100].root = 300,
    remote.children[300] does not contain title "guide"
  },
  Command("cfl page new guide.md --space-id 100")
)
-> Output{
  exit = success,
  stdout contains page URL,
  local.files["guide.md"] = MarkdownFile{
    frontmatter.space-id = 100,
    frontmatter.page-id = <new page id>,
    frontmatter.parent-id = 300,
    body = ""
  },
  remote.children[300] contains Page{
    title = "guide"
  }
}
```

### FP-NEW-003 Existing local file

```text
(
  InputState{
    local.files["guide.md"] = <existing file>
  },
  Command("cfl page new guide.md --space-id 100 --parent-id 200")
)
-> Output{
  exit = failure(1),
  stdout contains clear error,
  local.files["guide.md"] unchanged,
  remote unchanged
}
```

### FP-NEW-004 Duplicate sibling title

```text
(
  InputState{
    local.files = {},
    remote.children[200] contains Page{ title = "guide" }
  },
  Command("cfl page new guide.md --space-id 100 --parent-id 200")
)
-> Output{
  exit = failure(1),
  stdout contains clear error,
  local.files does not contain "guide.md",
  remote.children[200] has no additional page titled "guide"
}
```

### FP-NEW-005 Basename-derived title

```text
(
  InputState{
    local.files = {},
    remote.children[200] does not contain title "architecture-overview"
  },
  Command("cfl page new architecture-overview.md --space-id 100 --parent-id 200")
)
-> Output{
  exit = success,
  local.files["architecture-overview.md"] exists,
  remote contains Page{
    title = "architecture-overview"
  }
}
```

## `cfl page sync`

### FP-SYNC-001 Valid file

```text
(
  InputState{
    local.files["guide.md"] = MarkdownFile{
      frontmatter = { space-id: 100, page-id: 400, parent-id: 200 },
      body = "<non-empty markdown>"
    },
    remote.pages[400] exists
  },
  Command("cfl page sync guide.md")
)
-> Output{
  exit = success,
  stdout contains page URL,
  local.files["guide.md"] unchanged,
  remote.pages[400] = Page{
    title = "guide",
    body = convert(markdown)
  }
}
```

### FP-SYNC-002 Empty body

```text
(
  InputState{
    local.files["guide.md"] = MarkdownFile{
      frontmatter = { space-id: 100, page-id: 400, parent-id: 200 },
      body = ""
    },
    remote.pages[400] exists
  },
  Command("cfl page sync guide.md")
)
-> Output{
  exit = success,
  stdout contains page URL,
  remote.pages[400] = Page{
    title = "guide",
    body = ""
  }
}
```

### FP-SYNC-003 Missing frontmatter

```text
(
  InputState{
    local.files["guide.md"] = MarkdownFileWithoutFrontmatter
  },
  Command("cfl page sync guide.md")
)
-> Output{
  exit = failure(1),
  stdout contains clear error,
  local unchanged,
  remote unchanged
}
```

### FP-SYNC-004 Missing required key

```text
(
  InputState{
    local.files["guide.md"] = MarkdownFile{
      frontmatter missing one of { space-id, page-id, parent-id }
    }
  },
  Command("cfl page sync guide.md")
)
-> Output{
  exit = failure(1),
  stdout contains validation error,
  remote unchanged
}
```

### FP-SYNC-005 Malformed frontmatter

```text
(
  InputState{
    local.files["guide.md"] = MarkdownFile{
      frontmatter = malformed YAML
    }
  },
  Command("cfl page sync guide.md")
)
-> Output{
  exit = failure(1),
  stdout contains clear error,
  remote unchanged
}
```

### FP-SYNC-006 Title follows basename

```text
(
  InputState{
    local.files["renamed-guide.md"] = MarkdownFile{
      frontmatter = { space-id: 100, page-id: 400, parent-id: 200 }
    },
    remote.pages[400].title = "old-title"
  },
  Command("cfl page sync renamed-guide.md")
)
-> Output{
  exit = success,
  stdout contains page URL,
  remote.pages[400].title = "renamed-guide"
}
```

### FP-SYNC-007 Local file is the source of truth

```text
(
  InputState{
    local.files["guide.md"] = MarkdownFile{
      frontmatter = { space-id: 100, page-id: 400, parent-id: 200 },
      body = "<local markdown>"
    },
    remote.pages[400].body = "<remote manual edits>"
  },
  Command("cfl page sync guide.md")
)
-> Output{
  exit = success,
  remote.pages[400].body = convert("<local markdown>")
}
```

### FP-SYNC-008 Supported Markdown subset

```text
(
  InputState{
    local.files["guide.md"] = MarkdownFile{
      frontmatter = valid,
      body = paragraphs + h1/h2/h3 + unordered-list + ordered-list + fenced-code-block
    }
  },
  Command("cfl page sync guide.md")
)
-> Output{
  exit = success,
  remote page renders:
    paragraphs as paragraphs,
    h1/h2/h3 at matching levels,
    unordered lists as unordered lists,
    ordered lists as ordered lists,
    fenced code blocks as code blocks
}
```

## `cfl page sync --watch`

### FP-WATCH-001 Initial sync

```text
(
  InputState{
    local.files["guide.md"] = valid sync target,
    remote.pages[400] exists
  },
  Command("cfl page sync guide.md --watch")
)
-> Output{
  exit = running,
  remote.pages[400] synced once before waiting,
  stdout may include watch startup output
}
```

### FP-WATCH-002 Single file change

```text
(
  InputState{
    watch("guide.md") is active,
    local.files["guide.md"] changes once
  },
  Event(last_change + 800ms)
)
-> Output{
  remote page matches latest local file,
  stdout prints green "."
}
```

### FP-WATCH-003 Debounced burst

```text
(
  InputState{
    watch("guide.md") is active,
    local.files["guide.md"] changes multiple times within 800ms
  },
  Event(no_more_changes_for_800ms)
)
-> Output{
  exactly one sync is observed,
  remote page matches the final local file,
  stdout prints one green "."
}
```

### FP-WATCH-004 Error then recovery

```text
(
  InputState{
    watch("guide.md") is active,
    local.files["guide.md"] first becomes invalid,
    then becomes valid again
  },
  Events(
    invalid_change + 800ms,
    valid_change + 800ms
  )
)
-> Output{
  first sync attempt prints red "!" with error,
  watch process continues running,
  later sync succeeds,
  stdout prints green "."
}
```

### FP-WATCH-005 Ignore unrelated files

```text
(
  InputState{
    watch("guide.md") is active,
    unrelated files change
  },
  Event(unrelated_changes)
)
-> Output{
  no sync is triggered,
  stdout prints no green "."
}
```
