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
pk help document --format man > pk.1
```

Without a topic, prints the index. A terminal gets styled, wrapped
text; a pipe gets the raw authored bytes, so what Claude reads is the
skill file. `--plain` forces undecorated output. `NO_COLOR` is
respected.

`document` is every page in index order, read as one document. It
takes the same fork: rendered at a terminal, and through a pipe the
pages as markdown with their frontmatter removed. That makes it a
reader's document; a single topic is still the authored bytes.

`--format man` writes roff for `man(1)`, a page for a topic or one for
the whole document, and a flag outranks the terminal probe. Nothing is
installed: redirect it where you want it.

## Universal flags

<!-- generated: universal-flags -->
Every command accepts these:

```
  --project-dir <value>
        Project directory (default: PK_PROJECT_DIR, else the current directory)
  --plain
        Undecorated output: no color, no wrapping
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
