# pk rules

Aggregate the project's `.claude/rules/` into a single `RULES.md` and report the always-on context footprint.

## Usage

```bash
pk rules                          # write RULES.md, print the footprint to stderr
pk rules --dry-run                # print the footprint only, don't write RULES.md
pk rules --lint                   # scan rules for hidden/Trojan-source characters
pk rules --lint --strict          # also run plankit's house-style checks
pk rules --project-dir /path      # specify project directory
```

## How it works

1. Resolves the project root (walks up to the nearest `.git`; falls back to the directory if none).
2. Reads every `.md` file under `.claude/rules/`, sorted by filename, and strips each file's YAML frontmatter to get its body.
3. Classifies each rule's provenance from its `pk_sha256` marker, exactly like `pk status`: `[managed]` (pk-shipped, body hash matches), `[modified]` (pk-shipped, edited), or `[local]` (no marker, user-authored).
4. Reads the optional `kind:` frontmatter key (`craft` or `conduct`) and the `description:`; absent values render as `unclassified` and `(no description)`.
5. Estimates each file's context cost (~4 characters per token) and sums it with `CLAUDE.md`, which Claude Code also loads every session.
6. Writes `RULES.md` at the repository root and prints the footprint summary to stderr. `--lint` runs the scan instead and writes nothing.

## Flags

- **--lint** — Scan the rule files for hidden/Trojan-source characters (control and Unicode-format characters, bare CR, invalid UTF-8) instead of generating `RULES.md`. Exits non-zero if any are found.
- **--strict** — With `--lint`, also run plankit's house-style checks: em dashes, trailing whitespace, and hard-wrapped bullets. Requires `--lint`.
- **--dry-run** — Print the footprint summary without writing `RULES.md`.
- **--project-dir** — Starting directory for git root resolution (default: current directory).

## Exit code

- **0** — `RULES.md` generated, or `--lint` found nothing.
- **1** — `--lint` found issues, `--strict` was passed without `--lint`, or an error occurred.

## Details

### What RULES.md is for

`RULES.md` is a derived artifact, like `CHANGELOG.md`: regenerate it, don't hand-edit it. Its purpose is to be pasted into a Claude session for review of the rule set as a whole — overlap, gaps, drift, and whether each rule sits at the right altitude. To make that review possible from a single paste, the document is self-describing: it carries the pk version that shipped the managed rules, per-rule provenance, an estimated context footprint, and the `kind` classification.

It is not committed by default. Commit it if you want the snapshot in history, or add it to `.gitignore` and regenerate on demand.

### The `kind` frontmatter convention

Managed rules carry an optional `kind:` frontmatter key recording the craft-vs-conduct split:

- **`craft`** — developer-voiced standards for the work (e.g. `development-standards`, `git-discipline`). Claude inherits them the way a teammate inherits house style.
- **`conduct`** — Claude-voiced rules about the agent's own behavior (e.g. `model-behavior`).

`pk rules` surfaces `kind` but never writes or enforces it. Whether a rule is *correctly* classified, and whether craft and conduct are kept separate, is a semantic judgment left to the review session.

### What `--lint` does and does not check

The base scan is objective and universal: it catches hidden characters that could smuggle instructions past a human reviewer in files an AI reads every session (the "Trojan Source" class, CVE-2021-42574). It is the same policy `internal/safety` applies to the files pk ships downstream.

`--strict` adds plankit's own writing conventions, which are house style rather than universal, so they are opt-in. It does **not** check subjective or semantic qualities: alphabetical ordering (rule bullets are intentionally ordered by importance), craft-vs-conduct separation, or whether `kind` is correctly assigned. Those are left to the LLM review.

### Related commands

- `pk status` — report plankit configuration state, including which rules are pristine or modified.
