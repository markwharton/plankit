---
name: help
description: Show documentation for pk and its commands
---

# pk help

Renders plankit's documentation in the terminal. The same pages ship as
skills in the plankit plugin, compiled from one source, so the
typeahead entry, `pk help <topic>`, and the file in the repository
never disagree.

## Usage

```bash
pk help
pk help <topic>
```

Without a topic, prints the index. Output adapts to the reader: a
terminal gets styled, wrapped text; a pipe gets the raw authored
bytes, so tools and Claude read exactly the source document. `--plain`
forces undecorated output, and `NO_COLOR` is respected.
