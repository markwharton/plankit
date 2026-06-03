# Separate git-discipline's two audiences: keep craft, move conduct to model-behavior

## Context

A session pushed a commit to `origin/develop` without being asked, folding a push into a commit step the user had requested, and justified it by citing `git-discipline.md`'s "push when you're confident."

Investigating the misread surfaced the real defect, named by the user: `git-discipline.md` is the **developer's git discipline** (craft for the history artifact), but over time it quietly absorbed **agent-conduct** bullets too. Reading developer-voiced craft ("push when you're confident" = *the developer* pushes when confident), the model appropriated it as self-authorization.

The repo already solved this exact problem once: `development-standards.md` (craft of the work) and `model-behavior.md` (conduct of the agent) were originally one rule, separated when the mixed audience caused confusion. `git-discipline.md` is a second **craft** file (sibling to development-standards) that needs the same treatment: its conduct content belongs in the existing conduct file.

**Chosen design (confirmed):** keep `git-discipline.md` as the developer's git craft; move its agent-conduct content into `model-behavior.md`, and add the rule that was missing entirely — Claude carries out the developer's git decisions and never originates a commit/push/release. This keeps all agent conduct in one file, matching the established craft-vs-conduct axis, with no new shipped file.

## Audience allocation

Bullet-by-bullet, `git-discipline.md` is almost entirely developer craft and stays. The single agent-conduct bullet is the "unexpected state" one, which moves:

- **Moves to model-behavior** — current `git-discipline.md` line 14: *"When git state is unexpected, stop and investigate ... report what you see to the user; and wait for explicit instructions."* A developer practicing git craft doesn't tell themselves to "report to the user" — this is written for the agent.
- **New in model-behavior** — the authority rule never written down anywhere in the project: Claude executes git decisions, doesn't originate them, acts on the exact instruction each time.
- **Stays as craft** — everything else: don't push until happy, separate decisions, never force push, the soft-reset procedure, verify push state before rewriting, commit with purpose, message weight, conventional commits, commit before risk.

Both files are pk-managed and shipped, so each edit lands in the embedded source (`internal/setup/rules/`) **and** the local copy (`.claude/rules/`), with the local copy's `pk_sha256` recomputed.

## Change 1 — `model-behavior.md` (both copies): add a Git Conduct section

Insert a new `## Git Conduct` section after `## Scope Discipline` (it is scope/authority applied to git) and before `## Read Before Writing`. Second-person voice to match the file; "the developer" per CLAUDE.md's role terminology:

> ## Git Conduct
>
> - **Carry out the developer's git decisions; don't originate them.** Commit, push, and release are the developer's calls; their discipline lives in `git-discipline.md`. Do the exact action asked, each time: commit only what you were told to commit, and push only when the developer says to. A request to commit is never a request to push, and approval to push once doesn't carry forward to the next push.
> - **On unexpected git state, stop and defer to the developer.** If a command reports diverged branches, "local is behind remote", or anything you didn't anticipate, don't reflexively run `git pull`, `git pull --rebase`, `git merge`, or `git reset` to "fix" it; these can replay or duplicate commits irreversibly. Diagnose with `git log --oneline --graph HEAD origin/<branch>`, report what you see to the developer, and wait for explicit instructions.

Update the frontmatter `description:` to include the new section, in section order:

> `description: Honesty, scope discipline, git conduct, reading before writing, and testing`

(Currently: `Honesty, scope discipline, read before writing, and testing`.)

The second bullet preserves all the technical content of the moved git-discipline line 14 (the dangerous commands, the diagnostic). The first bullet's closing clause keeps the "approval doesn't carry forward" idea the user valued — now correctly placed as a Claude-facing conduct rule rather than reverse-engineered from developer-voiced craft. No em dashes.

## Change 2 — `git-discipline.md` (both copies): remove the moved bullet

Delete the line-14 bullet ("When git state is unexpected, stop and investigate ...") — it now lives in `model-behavior.md`. The remaining bullets are all developer craft; "you" reads consistently as the developer once the conduct is gone. The frontmatter `description:` is unchanged.

Recurrence is prevented by the model-behavior authority rule: even if a future session reads "push when you're confident" as craft, the conduct file states plainly that Claude doesn't decide pushes on its own.

### Optional (flagged for your call): reciprocal pointer

The model-behavior section references `git-discipline.md`. A reciprocal one-line note under git-discipline's `# Git Discipline` heading ("The developer's discipline for git; Claude's git conduct lives in `model-behavior.md`") would make the split self-documenting and resist re-tangling. It adds prose to a currently bullets-only file, so it's **opt-in** — included only if you want it. Default: omit.

## Change 3 — `plankit-development.md` (local-only): document the audience split

You asked whether `plankit-tooling.md` should note the separation. It shouldn't: `plankit-tooling.md` is a *shipped* rule whose theme is the three plankit layers (commands, hooks, skills) and flag conventions, written for downstream consumers *using* plankit. The principle "managed rules separate craft from conduct, never mix" is *maintainer-side guidance for authoring plankit's own rule set* — downstream projects receive the rules, they don't restructure them. Its home is `plankit-development.md` (maintainer-only, not shipped), alongside its siblings "Evolving pk Commands", "Skill Authoring", and "Repo Checks". Putting rule-authoring meta into the shipped tooling rule would itself mix two audiences, the exact failure this documents.

`plankit-development.md` is local-only and not pk-managed (no `pk_sha256`, no embedded copy), so this is a single-file edit with no hash recompute. Add a new section (one long single-line bullet per the rules convention, no em dashes):

> ## Rule Authoring
>
> - **Managed rules split by audience: developer craft vs agent conduct, and the two never mix in one file.** Craft files state standards for the work (`development-standards.md` for code, `git-discipline.md` for git history); they are developer-voiced and Claude inherits them the way a teammate inherits house style. Conduct files state how the agent behaves (`model-behavior.md`); they are Claude-voiced. The two blur easily: a developer-voiced line like "push when you're confident" gets misread as the agent's own license to push. When adding or editing a bullet, decide whose discipline it is and put it in that file; if a craft file has accumulated an agent-conduct bullet, move it. This is why `model-behavior.md` was split out from `development-standards.md`, and why git agent-conduct (don't originate a push; on unexpected state, defer to the developer) lives in `model-behavior.md`, not `git-discipline.md`.

## Recompute `pk_sha256` (both edited local copies)

After editing each embedded source, recompute its body hash and write it into the matching local copy, per CLAUDE.md's "Updating pk-managed files":

```bash
sed -n '/^---$/,/^---$/!p' internal/setup/rules/model-behavior.md | shasum -a 256
sed -n '/^---$/,/^---$/!p' internal/setup/rules/git-discipline.md | shasum -a 256
```

Replace the `pk_sha256:` line in `.claude/rules/model-behavior.md` (currently `b6787f...`) and `.claude/rules/git-discipline.md` (currently `fdac93...`) with the respective outputs. Avoids running `pk setup`, which would touch other managed files.

## What does NOT change

- **No new/removed rule file**, so the hardcoded name slice at `setup_test.go:98` stays as-is, and the `//go:embed rules/*.md` glob + `embed_safety_test.go` walk need no change (confirmed via wiring map).
- **`plankit-tooling.md` is deliberately untouched** (see Change 3): it is the shipped consumer-facing layer/flag guide, not the home for maintainer rule-authoring guidance.
- **`development-standards.md`** is untouched.
- Reference docs (`pk-json.md`, `error-reference.md`, `environment-variables.md`) are unaffected: no config key, error, or env var changes.

## Verification

1. `make test` — re-scans both edited embedded files via `embed_safety_test.go` (catches any stray em dash/control char) and confirms `setup_test.go` still finds `pk_sha256` in every managed rule.
2. Grep for stale references to the moved content before declaring done (development-standards "grep before done"): search README, `docs/`, and CLAUDE.md for any text that enumerates git-discipline's bullets or quotes the moved "unexpected state" rule by location. The CLAUDE.md list of rule *file names* is unchanged (no file added/removed), so it needs no edit; verify nothing describes the per-file *contents* that just shifted.
3. Hash match: a fresh `sed ... | shasum` on each embedded source equals the `pk_sha256:` written into its local copy.
4. `git diff` shows exactly five files changed: `model-behavior.md` and `git-discipline.md` each in `internal/setup/rules/` and `.claude/rules/` (four), plus `.claude/rules/plankit-development.md` (one, local-only). The shipped section text is identical across each file's two copies; only the shipped local copies carry a changed `pk_sha256` (plankit-development has none).
5. Read-through: model-behavior's new section is second-person and em-dash-free; git-discipline no longer contains the "report ... to the user" bullet and reads as pure developer craft.

## Commit

One commit on `develop`, e.g. `refactor: move git agent-conduct from git-discipline to model-behavior`. Commit only — do not push. The push is the developer's decision, made explicitly; this whole plan exists because that line was crossed.
