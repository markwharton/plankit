# pk rules

Report the always-on context cost of `.claude/rules/` and `CLAUDE.md`, or lint the rule files. It writes no file.

## Usage

```bash
pk rules                          # the footprint report, on stderr
pk rules --lint                   # scan for hidden characters
pk rules --lint --strict          # and plankit's house style
pk rules --project-dir /path      # start the git-root search there
```

## How it works

1. Reads every `.md` under `.claude/rules/`, recursively, as Claude Code does.
2. Marks each file `[managed]` (its `pk_sha256` matches), `[modified]` (it does not), or `[local]` (no marker), as `pk status` does.
3. Reads the `kind:` frontmatter key (`craft`, `conduct`, `docs`); absent, `unclassified`.
4. Estimates tokens per file with a characters-per-token ratio measured against a named model by `evals/calibrate`; the report says `(estimated, calibrated against <model>)` once calibration has run. `chars/4` runs about 25% low on this markdown; `go run ./evals/calibrate` in the plankit repository reproduces the ratio.
5. Lists a rule with a `paths:` key under `Conditional`: Claude Code loads it only when a matching file is read, so it is outside the always-on total.
6. Prints the always-on total and its rows (`CLAUDE.md` first), the conditional group, and a provenance tally.

```
Always-on context: 3 files, ~9 KB, 2,723 tokens (estimated, calibrated against claude-fable-5)
  CLAUDE.md                         544 B    186 tokens
  .claude/rules/plankit/conduct.md  ~4 KB  1,236 tokens  [managed] conduct
  .claude/rules/plankit/craft.md    ~4 KB  1,301 tokens  [managed] craft
Conditional (loads on matching files): 1 files, 78 B, 27 tokens (estimated, calibrated against claude-fable-5)
  .claude/rules/scoped.md            78 B     27 tokens  [local] unclassified
Provenance: 2 managed (pristine), 0 modified, 1 user-authored.
```

## Flags

- **--lint**: scan the rule files for control and Unicode format characters, bare CR and invalid UTF-8 (the Trojan Source class, CVE-2021-42574), the policy `internal/safety` applies to the files pk ships. Exits 1 on a finding.
- **--strict**: with `--lint`, also fail on em dashes, trailing whitespace and hard-wrapped bullets. See [The em-dash check](design.md#the-em-dash-check).
- **--project-dir `<dir>`**: where the search for the repository root starts.

## Exit code

`0`: report printed, or `--lint` found nothing. `1`: `--lint` found something, `--strict` without `--lint`, or an error.

## Limits

- `kind` is reported, not checked; whether a rule is classified correctly is a review question (`/review-rules` in the plankit repository).
- The lint checks characters and house style, not meaning: overlap, gaps and precedence between rules are not scanned.
