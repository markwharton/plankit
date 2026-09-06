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
```

Without a topic, prints the index. A terminal gets styled, wrapped
text. A pipe gets the raw authored bytes. `--plain` forces undecorated
output. `NO_COLOR` is respected.

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
