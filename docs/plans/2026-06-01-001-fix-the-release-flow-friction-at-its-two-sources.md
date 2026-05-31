# Fix the release-flow friction at its two sources

## Context

Two recurring failures, both rooted in the release flow, surfaced across sessions:

1. **Sessions try to push during a release.** A session reasons from the foundational rule
   "commit and push are separate decisions" and expects to `git push` after `pk release`,
   or looks for a `pk release --push` flag. We confirmed `--push` should **not** be added:
   `pk release` pushes the **tag** as part of the same step, and the tag must travel with
   the merge that anchors it — a deferred push would orphan the tag and leave the release
   branch ahead of origin. So `pk release` is deliberately atomic, and the rule text needs
   to say so where a session will actually read it. (`plankit-tooling.md` already scopes
   `--push` and states release publishes atomically; the one remaining contradiction is in
   `git-discipline.md`, which states the universal principle with no exception.)

2. **A new project shipped a release on `develop` instead of `main`, silently.** Root cause:
   the `new-plankit-project` skill creates the `main` + `develop` topology but never writes
   a `.pk.json`. With no `release.branch`, `pk release` falls back to trunk flow and tags
   the current branch (`develop`). `config.Load` treats "no file" and "file with no
   release.branch" identically (`internal/config/config.go:65`), and the trunk-flow notice
   (`internal/release/release.go:207`) is a soft mid-preflight line that scrolls past. The
   topology is known at init, so the bootstrap skill should write the matching `.pk.json`
   then — not defer it to a `/conventions` run that may never happen. This follows
   plankit's "if a feature needs a hand-written config, redesign so setup writes it."

These are two independent changes (one shipped managed rule, one local-only skill) and get
two separate commits on `develop`.

---

## Part 1 — Separate the release *decision* from the *--push flag* (managed rules)

One conceptual change across two rule files, with a separation of concerns so the fact
"you don't push for a release" lives once per altitude instead of being repeated. The
existing "Commit and push are separate decisions" bullet stays exactly as written. This is
a single commit.

### 1a — `git-discipline.md` owns the *decision* (triad, one bullet)

Fold release into the existing bullet as the third peer decision, rather than adding a
separate sibling bullet. A separate bullet risks the short-circuit problem (a reader
recognizes the familiar "commit and push are separate" and never registers a third decision
below it); the triad headline makes the whole model unmissable in one read. This is an
*extension*, not the *exception* framing that would dilute the rule. It carries the decision
and the "why", but **not** the `--push` flag mechanic (that is plankit-tooling's job).

Replace:
```
- **Commit and push are separate decisions.** Commit when the work is ready; push when you're confident.
```
With:
```
- **Commit, push, and release are separate decisions.** Commit when the work is ready; push when you're confident; release when it's time to publish. Release is not an ordinary push: `pk release` tags, merges, and pushes as one atomic action, because the tag must travel with the merge that anchors it, so never push by hand to publish a release.
```
This cures the original misfire (a session reasoning "push is separate, so I push after
release") by making release its own decision, not a push.

Cross-reference check: `plankit-tooling.md` bullet 1 ("because commit and push are separate
decisions (git-discipline)") still reads correctly against the triad headline — no break.

Files: `internal/setup/rules/git-discipline.md` (embedded, no `pk_sha256`) and
`.claude/rules/git-discipline.md` (active copy, carries `pk_sha256`).

### 1b — `plankit-tooling.md` owns the *--push flag mechanic*, trimmed

In the "Flag Conventions" section, keep the closed-list bullet and the one explicit release
bullet; remove the duplicated rationale and the release re-mention; fix the em dashes that
slipped back in after `91632ac`.

- **Bullet 1 (closed list):** keep `No other pk command takes --push` (required by the
  `plankit-development` rule). Only fix the em dash: change `means "publish what I just produced, fully" — pushing any refs` to use a colon (`fully": pushing any refs`).
- **Bullet 3 (release):** keep the flag fact, drop the trailing rationale now owned by
  git-discipline. New text:
  ```
  - **`pk release` has no `--push`; it publishes atomically.** It fast-forward merges into the release branch, tags, and pushes in one step; the only flag is `--dry-run` (preview). Passing `--push` errors.
  ```
  (Removed: "It won't be added: a local-only release would leave main merged and tagged locally but unpushed, diverging from origin/main and orphaning the tag." — that "why" now lives in git-discipline's "the tag must travel with the merge that anchors it.")
- **Bullet 4 (don't infer flags):** drop the `(pk release)` re-mention and fix the em dash.
  Keep the concrete `pk setup --baseline` example. New text:
  ```
  - **Don't infer a pk flag from another command; check `pk <cmd> --help`.** Each command's `--help` is the authoritative, always-current flag list and can't drift from the binary the way a copied list can. Flag conventions aren't universal: a flag on one command, like `--push` on `pk setup --baseline`, may not exist on another, so when unsure run `pk <cmd> --help` rather than assuming.
  ```

Files: `internal/setup/rules/plankit-tooling.md` (embedded, no `pk_sha256`) and
`.claude/rules/plankit-tooling.md` (active copy, carries `pk_sha256`).

### `pk_sha256` recompute (both rule files, per CLAUDE.md)
For each of `git-discipline.md` and `plankit-tooling.md`, after editing both copies:
```bash
sed -n '/^---$/,/^---$/!p' internal/setup/rules/<file>.md | shasum -a 256
```
Write each result into the matching active copy's `pk_sha256:` line. Bodies must be
byte-identical between embedded and active copy.

Net effect: git-discipline gains one concept bullet; plankit-tooling gets shorter (one
rationale sentence and one re-mention removed) and loses two em dashes. The "release has no
--push" fact now appears once explicitly (plankit-tooling bullet 3) plus the closed-list
exclusion (bullet 1), with the decision/why in git-discipline — no fourth restatement.

### `pk_sha256` recompute (per CLAUDE.md "Updating pk-managed files")
After editing both files:
```bash
sed -n '/^---$/,/^---$/!p' internal/setup/rules/git-discipline.md | shasum -a 256
```
Write the result into the `pk_sha256:` line of `.claude/rules/git-discipline.md`. The two
bodies must be byte-identical.

### Out of scope for Part 1 (audited clean)
`plankit-tooling.md` already scopes `--push` and states `pk release` publishes atomically
("orphaning the tag" rationale present). The `/ship` skill already says "Never run
`git push` directly." All `docs/*.md` and README describe release as atomic. No other edit.

---

## Part 2 — Write a default `.pk.json` at project creation (local-only skill)

Make `new-plankit-project` write the `.pk.json` that matches the `main` + `develop`
topology it creates, so `pk release` uses merge flow from day one.

### File
- `.claude/skills/new-plankit-project/SKILL.md` (local-only; no embedded source, no `pk_sha256`)

### Edits
1. **Both command templates** (public, lines 31-48; private, lines 52-69): after
   `pk setup --baseline` and before `git add -A`, add a step that writes `.pk.json` so it
   rides in the `chore: pk setup` commit:
   ```bash
   cat > .pk.json <<'JSON'
   {
     "guard": {
       "branches": ["main"]
     },
     "release": {
       "branch": "main"
     }
   }
   JSON
   ```
   Keys alphabetical (`guard` before `release`) and values matching the topology, exactly
   as the `/conventions` skill would write them — so a later `/conventions` run merges
   cleanly rather than conflicting.

2. **Design notes:** revise note 80 ("No `/conventions` afterwards…") to draw the line
   precisely: branch-topology config (`release.branch`, `guard.branches`) is written at init
   because the script establishes those branches; only CLAUDE.md *project conventions* (which
   need codebase shape) wait for a separate `/conventions` run. Add a short note explaining
   that without this file `pk release` would silently trunk-flow `develop`.

3. **`description` frontmatter (line 3):** add `.pk.json release config` to the enumerated
   list of what the skill sets up (keep the existing style; this is the "update every
   enumeration" pass).

---

## Part 3 — `pk setup` reminds the user to run `/conventions` (shipped behavior)

The bootstrap skill (Part 2) only helps projects created through it. A `pk setup` reminder
helps **every** project: when no `.pk.json` exists, point the user at `/conventions` so the
release/guard branches get configured before the first release. This reinforces guidance
already in `docs/adoption.md` and `docs/pk-setup.md`.

### File
- `internal/setup/setup.go` — completion block, near line 333 (after the baseline/version-tag
  tip, before "Restart Claude Code to apply changes." at line 346).

### Edit
Add a conditional reminder, gated on `.pk.json` **absence** (so a configured project
re-running `pk setup` on an upgrade is not nagged). Reuse `cfg.Stat` on
`filepath.Join(projectDir, ".pk.json")`. Message in the existing short-tip style, e.g.:
```
No .pk.json found. Run /conventions in Claude Code to set release and guard branches.
  Without it, pk release uses trunk flow (tags the current branch, no merge to a release branch).
```
It is a reminder, not a warning: a project can legitimately opt out of `.pk.json`.

### Tests & docs
- `internal/setup/setup_test.go`: assert the reminder appears when `.pk.json` is absent and
  is **omitted** when it is present (inject `Stat` accordingly). Follows the existing
  Config-mock test pattern.
- `docs/pk-setup.md`: it already tells the user to run `/conventions` after setup; add a line
  noting `pk setup` now prints this reminder when `.pk.json` is missing. Keep it tight.

### Residual gap (flagged, not in this plan unless you want it)
Parts 2 and 3 cover new projects and any project that re-runs `pk setup`. A repo that never
re-runs setup and never configures `.pk.json` would still silently trunk-flow at release
time. If you want defense-in-depth at the release itself, a separate follow-up could make
`pk release` distinguish "no `.pk.json` file at all" from "file present, no release.branch"
and warn (or, in ask-mode, prompt) when trunk-releasing a non-trunk branch like `develop`.
Left out here to keep scope at the two sources you identified; raise it if you want it.

---

## Verification

**Part 1** (for **both** `git-discipline.md` and `plankit-tooling.md`)
1. `sed -n '/^---$/,/^---$/!p' internal/setup/rules/<file>.md | shasum -a 256` equals the `pk_sha256:` now in the matching `.claude/rules/<file>.md`.
2. `diff <(sed -n '/^---$/,/^---$/!p' internal/setup/rules/<file>.md) <(sed -n '/^---$/,/^---$/!p' .claude/rules/<file>.md)` prints nothing.
3. `grep -n "—" .claude/rules/plankit-tooling.md internal/setup/rules/plankit-tooling.md` finds no em dashes in the Flag Conventions bullets.
4. `make test` (includes `embed_safety_test.go`, which scans the edited embedded files) and `make lint` pass.

**Part 2** (smoke test the emitted script in a throwaway dir)
1. Run the new `cat > .pk.json` heredoc in a temp dir; confirm it parses: `python3 -m json.tool .pk.json` (or `jq . .pk.json`) succeeds and shows `release.branch == "main"`, `guard.branches == ["main"]`.
2. In a temp git repo on a `develop` branch with that `.pk.json` present, `pk release --dry-run` reports the merge flow ("Would merge develop into main"), **not** "Trunk flow (no release.branch in .pk.json)".
3. Negative case: with the `.pk.json` removed, `pk release --dry-run` falls back to the trunk-flow line — confirming the file is what flips the behavior.
4. Read back the rendered skill: both templates emit valid shell; the `.pk.json` step sits after `pk setup --baseline` and before the setup commit.

**Part 3**
1. `make test` passes, including the new setup test (reminder shown when `.pk.json` absent, omitted when present).
2. Smoke: `make build`, then `pk setup` (or a dry equivalent) in a temp git repo with **no** `.pk.json` prints the `/conventions` reminder; with a `.pk.json` present it does not.

## Commits (separate, on `develop`)
- Part 1: `docs: make release the third separate decision, trim --push repetition`
- Part 2: `fix: write default .pk.json at project init so pk release uses merge flow`
- Part 3: `feat: remind to run /conventions when pk setup finds no .pk.json`

Commit only; pushing stays the user's separate decision.
