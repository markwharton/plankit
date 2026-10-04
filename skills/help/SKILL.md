---
name: help
description: Show documentation for pk and its commands
argument-hint: <topic>
---

# pk help

Renders plankit's documentation in a terminal. The same pages ship as
skills in the plugin, compiled from one source.

## Usage

```bash
pk help
pk help <topic>
pk help document
pk help document > plankit.md
pk help document --format man > pk.1
man ./pk.1
```

Without a topic, prints the index. At a terminal you get styled,
wrapped text; redirect or pipe it and you get the page's markdown.
`NO_COLOR` is respected, and `CLICOLOR_FORCE` keeps the styled page
through a pipe, for a pager like `less -R`.

`document` prints every page in one go, in the order the index lists
them: for reading straight through, or for keeping as a file.

`--format man` writes a man page instead, for one topic or for the
whole set. Read it back with `man ./pk.1`. The `./` matters, or man
searches its manpath for a page of that name and finds none.

## Universal flags

<!-- generated: universal-flags -->
Every command accepts these:

```
  --plain
        Undecorated output: no color, no wrapping
  --project-dir <value>
        Project directory (default: PK_PROJECT_DIR, else the current directory)
  --quiet
        Suppress notes and hints (errors still print)
```
<!-- /generated: universal-flags -->

## Flags

<!-- generated: flags -->
```
  --format <value>
        Output format: text or man (default text)
```
<!-- /generated: flags -->
