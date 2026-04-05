# Plan: Expand plankit setup with universal CLAUDE.md, /init skill, and skill templates

## Context

plankit's value extends beyond plan preservation — guidelines and testing discipline prevent the nightmare sessions builders experience with unconstrained AI-assisted development. But the current onboarding has gaps:

- Templates require manual modification (base.md has placeholder Project Conventions)
- Non-dev builders don't know what to put in CLAUDE.md
- There are no skill examples for builders to learn from
- `pk setup` doesn't touch CLAUDE.md at all

This plan adds three things: a universal CLAUDE.md that works as-is (copied by `pk setup`), a `/init` skill that bootstraps project-specific conventions, and skill templates as reference examples.

## Changes

### 1. SHA-based managed file protection

All files installed by `pk setup` (skills and CLAUDE.md) use a SHA256 marker to detect user modifications:

```markdown
<!-- pk:sha256:abc123def... -->
```

The marker is the first line of the installed file. The SHA covers the file content excluding the marker line itself.

**On install (no existing file):** Write content with SHA marker as first line.

**On re-setup (file exists with marker):**
- Strip marker line, SHA the remaining content
- SHA matches marker → file is pristine → overwrite with new version + new SHA
- SHA doesn't match → user modified it → skip with stderr warning

**On re-setup (file exists, no marker):** Not pk-managed → skip.

**`--force` flag:** `pk setup --force` overwrites all managed files regardless of marker or modification state. Handles migration from pre-marker installs and "give me pk's version back" scenarios.

This gives us: automatic updates when pk ships improved skills, protection for user-edited files, and a clear signal in the file about what pk manages.

**Implementation in `internal/setup/setup.go`:**
- `managedFileSHA(content string) string` — compute SHA256 of content
- `managedMarker(sha string) string` — return `<!-- pk:sha256:... -->`
- `writeManaged(path, content string, stderr io.Writer, force bool) error` — write with marker
- `shouldUpdate(path, newContent string, force bool) (bool, string)` — check marker, return (should update, reason)
- `Run()` signature adds `force bool` parameter, threaded from `--force` CLI flag in `cmd/pk/main.go`

### 2. Universal CLAUDE.md template (embedded in binary)

**Create `internal/setup/template/CLAUDE.md`**

Derived from `templates/base.md`, condensed with consistent voice. Changes from base.md:
- Merge Honesty + Transparency → "Honesty and Transparency"
- Merge No Shortcuts + Stay Focused + Complete Work → "Scope Discipline"
- Condense Read Before Writing (4 bullets → 2)
- Condense Data-First (3 bullets → 1), Fail Fast (2 → 1), Security (4 → 2), Debugging (3 → 1)
- Replace Commits with Git Discipline (principle-level git guidance)
- Remove Project Conventions placeholder, "Base Template" subtitle, "Copy this" instruction, emojis line

Every section follows the same rhythm: **bold principle → concise context**.

The full template content:

```markdown
# CLAUDE.md

## Model Behavior

### Honesty and Transparency

- **If you don't know, say so.** Never assume or guess — accuracy matters more than speed.
- **Ask, don't assume.** When in doubt about any decision, ask the user rather than making assumptions.
- **No surprises.** Disclose decisions and tradeoffs upfront. The user should never discover shortcuts after the fact.
- **When uncertain, say so.** Explain what you are doing and why.

### Scope Discipline

- **Only do what was asked.** A bug fix does not need surrounding code cleaned up. A simple feature does not need extra configurability.
- **Never take shortcuts without asking.** This includes: placeholder logic, approximations, skipping validation, omitting features for an "initial version", or using mock data instead of real integrations.
- **If you see something worth improving, mention it** — do not act on it without permission.
- When tempted to cut corners or expand scope:
  1. **STOP** — Do not proceed.
  2. **ASK** — Explain the tradeoffs.
  3. **WAIT** — Get explicit approval.
- **Finish what you start.** If you cannot complete something, explain why and what remains.

### Read Before Writing

- **Understand existing code before changing it.** Follow established conventions in the codebase.
- **Check before creating.** Existing files, existing patterns, coupled code that must be updated together.

### Testing Discipline

- **Test at the start of each session** and report the status.
- **Test before and after changes.** If tests fail after your changes, you know the cause.
- **Run tests yourself.** This closes the loop — no copy-paste back and forth.

## Development Standards

### Data-First, Model-First

- **Preserve the structure you were given.** When data has hierarchy, parse and maintain it. Let the data model drive the code, not the other way around.
- **Never flatten structured data into flat lists then reconstruct with heuristics** — the context is already lost.

### Fail Fast, No Silent Fallbacks

- **Surface errors clearly.** When something is missing or wrong, fail with a clear message. Never silently fall back to made-up defaults.

### All-or-Nothing Consistency

- **Update every related location together.** No partial renames, no half-updated contracts. If you change it in one place, change it everywhere.

### Two-Pass Code Generation

- **First pass:** Generate working code — DRY violations are acceptable.
- **Second pass:** Refactor — add abstractions, remove magic numbers, create shared components.
- **Resist premature abstraction.** Three similar lines is better than the wrong abstraction. ([The Wrong Abstraction](https://sandimetz.com/blog/2016/1/20/the-wrong-abstraction))

### Security

- **Never commit secrets** to version control. Use `.env` files in `.gitignore`.
- **Guard against** command injection, XSS, SQL injection, and other OWASP top 10 vulnerabilities.

### Debugging

- **Diagnostic scripts over rebuild cycles.** When debugging, create a minimal script that tests the specific issue. If you are about to do your second full rebuild, stop and write a diagnostic script instead.

### Git Discipline

- **Don't push your work until you're happy with it.** Locally, you have full freedom — amend, reorder, combine. Once pushed, history is shared and rewriting creates problems downstream.
- **Never force push.** If a pushed commit needs fixing, make a new commit.
- **Commit with purpose.** Each commit is one logical change. Follow Conventional Commits to make history scannable.
- **Never include BREAKING CHANGE** in commit messages unless there is an actual breaking change.
- **Commit before risk.** Before refactoring or trying an uncertain approach — commit what works. Git is your safety net, but only if you've saved your progress.
```

**Modify `internal/setup/setup.go`**

- Add embed: `//go:embed template/CLAUDE.md` as `embed.FS` — same pattern as skills, read via `fs.ReadFile`
- Use `writeManaged()` to install CLAUDE.md with SHA marker
- On re-setup: `shouldUpdate()` determines whether to overwrite or skip

**Modify `internal/setup/setup_test.go`**

- `TestRun_createsClaudeMD` — fresh project, verify file created with SHA marker + expected content
- `TestRun_skipsModifiedClaudeMD` — user-edited CLAUDE.md (marker present, SHA mismatch) preserved unchanged
- `TestRun_updatesUnmodifiedClaudeMD` — pristine CLAUDE.md (marker present, SHA matches) gets updated
- `TestRun_skipsUnmanagedClaudeMD` — CLAUDE.md without marker left alone
- `TestManagedFileProtection` — unit tests for SHA computation, marker generation, shouldUpdate logic
- Update `TestRun_freshProject` to verify CLAUDE.md creation
- Fix skill list to include all skills: `{"changelog", "init", "preserve", "release", "review"}`

Skills also use the same managed file protection — existing skill installation in `Run()` switches from raw `os.WriteFile` to `writeManaged()`.

### 3. /init bootstrap skill

**Create `internal/setup/skills/init/SKILL.md`**

A skill that Claude uses to interrogate the project and generate a Project Conventions section. When invoked:

1. Read existing CLAUDE.md
2. Explore the project (language, framework, build system, test runner, directory structure)
3. Propose a `## Project Conventions` section with discovered conventions
4. Show the user and ask for confirmation before appending to CLAUDE.md

Key constraints in the skill wording:
- Append only — never modify Model Behavior or Development Standards
- If Project Conventions already exists, update it
- Show proposed changes before writing

Installed by `pk setup` automatically (same embed pattern as other skills).

### 4. Skill templates as reference examples

**Create `templates/skills/`** with example skills builders can copy and adapt:

- `smoke-test.md` — run project tests, report results (demonstrates wrapping shell commands)
- `validate.md` — check project structure and configuration (demonstrates using Claude's intelligence)

These are NOT installed by `pk setup` — they're reference material like the existing CLAUDE.md templates.

### 5. Documentation architecture

**Principle:** Every command gets its own `docs/pk-<command>.md`. This is the single source of truth for that command's flags, behavior, and examples. The tight loop is: **code → tests → command doc**.

Higher-level docs (README, getting-started, methodology) link to command docs for details. They only change when concepts change, not when flags change.

**Create `docs/pk-setup.md`** — flags, behavior, SHA markers, what gets created, `--force`

**Create `docs/pk-preserve.md`**, `docs/pk-protect.md`, `docs/pk-version.md` — complete the set

**Update `README.md`** — Commands section becomes a link table:

```markdown
## Commands

| Command | Description |
|---------|-------------|
| `pk setup` | Configure project hooks, skills, and CLAUDE.md — [details](docs/pk-setup.md) |
| `pk changelog` | Generate changelog, commit, and tag — [details](docs/pk-changelog.md) |
| `pk release` | Validate and push release — [details](docs/pk-release.md) |
| `pk preserve` | Preserve approved plan — [details](docs/pk-preserve.md) |
| `pk protect` | Block edits to docs/plans/ — [details](docs/pk-protect.md) |
| `pk version` | Print version and check for updates — [details](docs/pk-version.md) |
```

**Update `README.md`** — "What it does" adds CLAUDE.md creation, Templates section mentions skill templates

**Update `docs/getting-started.md`** — Update directory tree, replace "Copy a CLAUDE.md template" with `/init` guidance, link to `pk-setup.md` for details

**Update `docs/methodology.md`** — Update Guidelines section references. Add a new "Plan review" section covering:
- Plans are drafts, not commitments — they improve through iteration
- Push back on specifics, not the whole plan. Name what's wrong and why.
- Stay calm during review. Frustration during plan review is normal but counterproductive — the plan is a conversation, not a confrontation
- Take the time needed. Context is not the constraint it used to be — invest in getting the plan right rather than rushing to approve

**Update `CLAUDE.md` (plankit's own)** — Architecture section adds new paths. Add documentation convention: "The tight loop is: code → tests → command doc. All three are about the same thing. Higher-level docs (README, getting-started, methodology) link to command docs and only change when concepts change."

### 6. Existing templates — no restructuring

Keep `templates/base.md`, `go.md`, `typescript.md`, `azure.md` where they are. They remain examples of what has been used and works — reference material, not prescriptive. No directory moves, no churn.

`base.md` and the embedded template share most content. `base.md` includes Project Conventions examples as a reference. If they drift, `base.md` can be deprecated later.

## File changes

| Action | File |
|--------|------|
| Create | `internal/setup/template/CLAUDE.md` |
| Create | `internal/setup/skills/init/SKILL.md` |
| Create | `templates/skills/smoke-test.md` |
| Create | `templates/skills/validate.md` |
| Create | `docs/pk-setup.md` |
| Create | `docs/pk-preserve.md` |
| Create | `docs/pk-protect.md` |
| Create | `docs/pk-version.md` |
| Modify | `internal/setup/setup.go` |
| Modify | `internal/setup/setup_test.go` |
| Modify | `cmd/pk/main.go` |
| Modify | `README.md` |
| Modify | `docs/getting-started.md` |
| Modify | `docs/methodology.md` |
| Modify | `CLAUDE.md` |

## Implementation order

1. **Phase 1**: Managed file protection — SHA helpers + `writeManaged`/`shouldUpdate` in setup.go, unit tests → `make test`
2. **Phase 2**: Universal CLAUDE.md — create embedded template, integrate with managed writes, tests → `make test`
3. **Phase 3**: Retrofit existing skills — switch from raw `os.WriteFile` to `writeManaged()`, update tests → `make test`
4. **Phase 4**: /init skill — create skill, update test assertions → `make test`
5. **Phase 5**: Skill templates in `templates/skills/`
6. **Phase 6**: Command docs (`docs/pk-setup.md`, `pk-preserve.md`, `pk-protect.md`, `pk-version.md`)
7. **Phase 7**: Update README, getting-started, methodology, CLAUDE.md (links + concepts only)

## Verification

```bash
make test                    # All tests pass including new ones
pk setup --project-dir /tmp/test-project  # Creates CLAUDE.md + settings + skills

# Verify managed file markers
head -1 /tmp/test-project/CLAUDE.md                        # <!-- pk:sha256:... -->
head -1 /tmp/test-project/.claude/skills/changelog/SKILL.md # <!-- pk:sha256:... -->

# Verify update behavior
pk setup --project-dir /tmp/test-project  # Re-run: updates pristine files, skips modified ones

# Verify skip on modification
echo "# custom" >> /tmp/test-project/.claude/skills/review/SKILL.md
pk setup --project-dir /tmp/test-project  # Skips review skill with warning

# Verify force overwrite
pk setup --force --project-dir /tmp/test-project  # Overwrites all managed files
```
