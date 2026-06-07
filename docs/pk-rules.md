# pk rules

Report the always-on context footprint of the project's `.claude/rules/` and `CLAUDE.md`. `pk rules` only reports; it writes no files.

## Usage

```bash
pk rules                          # print the footprint report to stderr
pk rules --lint                   # scan rules for hidden/Trojan-source characters
pk rules --lint --strict          # also run plankit's house-style checks
pk rules --project-dir /path      # specify project directory
```

## How it works

1. Resolves the project root (walks up to the nearest `.git`; falls back to the directory if none).
2. Reads every `.md` file under `.claude/rules/`, sorted by filename.
3. Classifies each rule's provenance from its `pk_sha256` marker, exactly like `pk status`: `[managed]` (pk-shipped, body hash matches), `[modified]` (pk-shipped, edited), or `[local]` (no marker, user-authored).
4. Reads the optional `kind:` frontmatter key (`craft` or `conduct`); absent values render as `unclassified`.
5. Estimates each file's context cost using a calibrated characters-per-token ratio and sums it with `CLAUDE.md`, which Claude Code also loads every session. The ratio is model-specific, measured against a named model by `evals/calibrate`; figures are labelled `(estimated)` and gain `(estimated, calibrated against <model>)` once the calibration has been run. Plain `chars/4` runs ~25% low for this markdown.
6. Prints the report to stderr: a totals line, one aligned row per source (`CLAUDE.md` and each rule, tagged with its provenance and `kind`), and a provenance tally. `--lint` runs the scan instead.

## Flags

- **--lint** — Scan the rule files for hidden/Trojan-source characters (control and Unicode-format characters, bare CR, invalid UTF-8) instead of the footprint report. Exits non-zero if any are found.
- **--strict** — With `--lint`, also run plankit's house-style checks: em dashes, trailing whitespace, and hard-wrapped bullets. Requires `--lint`.
- **--project-dir** — Starting directory for git root resolution (default: current directory).

## Exit code

- **0** — report printed, or `--lint` found nothing.
- **1** — `--lint` found issues, `--strict` was passed without `--lint`, or an error occurred.

## Details

### What it reports, and what it does not

The report is a context-cost and governance view: how much always-on budget the rule set spends, and where each rule sits on the provenance/`kind` axes. It is a quick read, not a deep analysis — and it writes nothing, like every other pk command.

Reviewing the rule set *as a system* — overlap, gaps, drift, unstated precedence, whether each rule sits at the right altitude — is the job of the `/review-rules` skill, which reads the source rules directly. `pk rules` deliberately does not produce a paste-able document for that; the skill's analysis is more than a concatenation can give.

### The `kind` frontmatter convention

Managed rules carry an optional `kind:` frontmatter key recording the craft-vs-conduct split:

- **`craft`** — developer-voiced standards for the work (e.g. `development-standards`, `git-discipline`). Claude inherits them the way a teammate inherits house style.
- **`conduct`** — Claude-voiced rules about the agent's own behavior (e.g. `model-behavior`).

`pk rules` surfaces `kind` in the report but never writes or enforces it. Whether a rule is *correctly* classified, and whether craft and conduct are kept separate, is a semantic judgment left to `/review-rules`.

### What `--lint` does and does not check

The base scan is objective and universal: it catches hidden characters that could smuggle instructions past a human reviewer in files an AI reads every session (the "Trojan Source" class, CVE-2021-42574). It is the same policy `internal/safety` applies to the files pk ships downstream.

`--strict` adds plankit's own writing conventions, which are house style rather than universal, so they are opt-in. It does **not** check subjective or semantic qualities: alphabetical ordering (rule bullets are intentionally ordered by importance), craft-vs-conduct separation, or whether `kind` is correctly assigned. Those are left to the LLM review.

### Related commands

- `pk status` — report plankit configuration state, including which rules are pristine or modified.
