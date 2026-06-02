# Plan: Stop `/ship` from honoring (or provoking) a guessed version number

## Context

At the end of a session, Claude pre-filled the input box with `ship v0.4.0`, but
`pk changelog --dry-run` would actually compute `v0.3.1`. Claude guessed a *minor*
bump and tacked a version onto the suggestion without running the authoritative
computation.

Two facts make this guess harmful:

1. **The version is computed, never chosen.** `pk changelog` derives the next
   version from commit types (`internal/changelog/changelog.go` `detectBump`/
   `resolveBump`): breaking → major, `feat` → minor, else patch. To *intend* a
   larger bump you use `pk changelog --bump minor|major` — you never type a target
   version. So a guessed `v0.4.0` is meaningless as input.
2. **`/ship` takes no version argument at all.** Its only argument is `auto`
   (`argument-hint: [auto]`). Neither `pk changelog` nor `pk release` accepts an
   explicit version (`cmd/pk/main.go`).

The `ship v0.4.0` string is model-generated; plankit can't patch what Claude Code
pre-fills, but it governs model behavior through its **managed rules and skills**.
The worry to fix: a user copies the guessed suggestion, runs `ship v0.4.0`, and
either gets a surprise or Claude reacts as if something broke and interrogates them.

Scope chosen: **both** a defensive skill fix and a preventive rule.

## Approach

Two small edits to shipped managed files (so every downstream project benefits).
Each managed file is edited in **both** the embedded source (`internal/setup/...`)
and the local copy (`.claude/...`), and the local copy's `pk_sha256` is recomputed
(per CLAUDE.md "Updating pk-managed files").

### 1. Defensive — harden the `/ship` skill

Files:
- `internal/setup/skills/ship/SKILL.md`
- `.claude/skills/ship/SKILL.md` (recompute `pk_sha256`)

Add a bullet to the **## Rules** section making a typed version harmless:

> - **The only argument is `auto`; `/ship` takes no version.** The version is
>   computed by `pk changelog` and shown in its dry-run. Ignore any version-like
>   argument (e.g. `ship v0.4.0`): run the flow normally and let the dry-run reveal
>   the real version. If a typed version differs from the computed one, note it in
>   one calm line (the version is derived from commit types; for a deliberately
>   larger bump use `pk changelog --bump minor|major`) and proceed with the
>   computed version. Don't treat the mismatch as an error or interrogate the user.

(Single long bullet, matching the rules line-length convention.)

### 2. Preventive — stop the guessed suggestion at the source

Files:
- `internal/setup/rules/plankit-tooling.md`
- `.claude/rules/plankit-tooling.md` (recompute `pk_sha256`)

Extend the existing **Skills** bullet under "## Three Layers" as a peer clause
(not a new section — avoids over-documentation):

> **Skills:** User-invoked workflows (`/conventions`, `/preserve`, `/ship`). Each
> has its own instructions. Execute them only when the user asks. When suggesting
> `/ship` as a next step, write it bare — never append a version (`/ship`, not
> `/ship v0.4.0`). `/ship` takes no version; the version is computed by
> `pk changelog` and revealed by its dry-run.

This rule is loaded every session, so it is in context when the model composes an
end-of-session suggestion.

### SHA recompute (each changed managed file)

```bash
sed -n '/^---$/,/^---$/!p' internal/setup/skills/ship/SKILL.md | shasum -a 256
sed -n '/^---$/,/^---$/!p' internal/setup/rules/plankit-tooling.md | shasum -a 256
```

Put each result in the `pk_sha256:` line of the corresponding `.claude/` copy.
The sed pattern excludes the frontmatter block, matching Go's body-hash byte-for-byte.

### Docs check

Grep for any doc that enumerates `/ship` arguments (`docs/`, `README.md`). The
change clarifies that there is *no* version argument rather than adding one, so an
update is only needed if a doc currently implies a version can be passed. Update if
found; otherwise no doc change.

## Out of scope (by design, not omission)

- **Not** adding a version argument to `/ship`, `pk changelog`, or `pk release`.
  The version is derived from commit history; `--bump` already covers an
  intentional larger bump. Accepting a typed version would contradict the design.
- No Go code changes — this is entirely about managed rule/skill text.

## Verification

Automated:
- `make build && make test` — includes `internal/setup` tests (managed-file
  hash / embed-safety checks) and confirms the recomputed `pk_sha256` is consistent.
- Re-run the two `sed | shasum` commands and confirm each matches the `pk_sha256`
  written into the `.claude/` copy.

Smoke (in a session, in a repo where dry-run computes a patch bump):
- **Negative/surprise case:** invoke `/ship v0.4.0`. Confirm Claude does *not*
  honor `v0.4.0`, runs `pk changelog --dry-run`, presents the real computed version,
  and notes the mismatch in a single calm line rather than asking a barrage of
  questions.
- **Happy path:** `/ship` (bare) and `/ship auto` still behave exactly as before.
- **Preventive:** confirm the new Skills bullet renders in `.claude/rules/plankit-tooling.md`
  and reads naturally as end-of-session guidance.
