# Template extensibility + `pk release`

## Context

Two related problems surfaced from the `--dry` vs `--dry-run` inconsistency:

1. **Templates lack insertion points.** `templates/base.md` tells developers to "extend with project-specific sections" but provides no clear place to do it. A convention like "CLI flags use --kebab-case" had nowhere obvious to live.

2. **Mixed release modes.** `scripts/release.sh` (shell) alongside `pk changelog` (Go) created two languages, two flag conventions, and redundant cross-compilation (CI already builds all platforms). A generic `pk release` subcommand eliminates the script.

---

## Part 1: Template extensibility

### `templates/base.md` — add Project Conventions section at end

After the existing Commits subsection (line 95), add:

```markdown

## Project Conventions

Project-specific rules that supplement the standards above. Replace these examples with your own.

- (Example: CLI flags use `--kebab-case`; shell scripts match the same flag names)
- (Example: All HTTP handlers return `Result<T>` with explicit error codes)
- (Example: Release via `pk changelog` then `pk release`)
```

Simple flat bullet list — no subsections. Developers replace examples with their own conventions. The parenthetical "(Example: ...)" format makes it unmistakable these are placeholders.

### Extension templates — add reference line

Add a second line to the header of `go.md`, `typescript.md`, and `azure.md`:

```
These can be added as sections in your CLAUDE.md or folded into Project Conventions.
```

### plankit's own `CLAUDE.md` — add convention

Add to the existing Conventions section:

```markdown
- CLI flags use `--kebab-case` (e.g., `--dry-run`, `--project-dir`)
```

### Files modified
- `templates/base.md`
- `templates/go.md`
- `templates/typescript.md`
- `templates/azure.md`
- `CLAUDE.md`

---

## Part 2: `pk release` subcommand

### Design principle: fully generic

`pk release` is language-agnostic, just like `pk changelog`. The core is git operations (validate tag, check branch state, push). Project-specific validation (Go tests, npm build, Python lint) lives in `hooks.preRelease` configured per-project in `.changelog.json`:

| Project type | `hooks.preRelease` example |
|---|---|
| Go | `"go test -race ./..."` |
| Node/npm | `"npm test && npm run build"` |
| Python | `"pytest && mypy ."` |
| No tests | (omit — pk release just validates and pushes) |

### What it does

1. Find version tag at HEAD (error if none: "run `pk changelog` first")
2. Validate semver format
3. Pre-flight: clean working tree, on expected branch, not behind remote
4. Run `hooks.preRelease` from `.changelog.json` if configured
5. Push branch + tag to origin (skip if `--dry-run`)

### Flags

```
pk release [--dry-run] [--branch main]
```

- `--dry-run` — validate without pushing (default: false)
- `--branch` — expected branch name (default: "main")

### Config sharing

The `Hooks` struct in `internal/changelog/changelog.go` gets a new field:

```go
type Hooks struct {
    PostVersion string `json:"postVersion,omitempty"`
    PreCommit   string `json:"preCommit,omitempty"`
    PreRelease  string `json:"preRelease,omitempty"`  // new
}
```

Export `ChangelogConfig` and `LoadConfig` from the changelog package so the release package can read `.changelog.json` without duplicating the parser. (Currently `loadConfig` is unexported.)

### New files

- `internal/release/release.go` — `Config` struct (same DI pattern: `Stderr`, `GitExec`, `RunScript`), `DefaultConfig()`, `Run(cfg Config) int`
- `internal/release/release_test.go` — tests using stub functions like `changelog_test.go`

### `cmd/pk/main.go` changes

Add to switch:
```go
case "release":
    runRelease(os.Args[2:])
```

Add `runRelease`:
```go
func runRelease(args []string) {
    fs := flag.NewFlagSet("release", flag.ExitOnError)
    dryRun := fs.Bool("dry-run", false, "Validate without pushing")
    branch := fs.String("branch", "main", "Expected branch for release")
    fs.Parse(args)
    // ...
    os.Exit(release.Run(cfg))
}
```

Update `printUsage()` to include `pk release`.

### plankit's `.changelog.json`

Add preRelease hook:
```json
{
  "types": [...],
  "hooks": {
    "preRelease": "go test -race ./..."
  }
}
```

### Delete

- `scripts/release.sh` — fully replaced by `pk release` + `hooks.preRelease`
- `scripts/` directory (empty after deletion)
- `Makefile` targets `release` and `release-dry` — rewrite to call `pk release`

### Update

- `CONTRIBUTING.md` — single unified release workflow:
  ```bash
  pk changelog              # scan, write, commit, tag
  pk release --dry-run      # validate (optional)
  pk release                # push branch + tag
  ```
- `CLAUDE.md` — add `pk release` to Quick Commands and Architecture sections
- `docs/changelog.md` — rename to `docs/pk-changelog.md` (avoids ambiguity with actual changelogs, groups command docs)
- `docs/pk-release.md` — new doc covering `pk release`, pre-flight checks, `hooks.preRelease`, and the unified workflow (`pk changelog` then `pk release`)

---

## Implementation order

1. Template changes (Part 1) — no code dependencies
2. Add `PreRelease` field to `Hooks` struct, export `ChangelogConfig`/`LoadConfig`
3. Create `internal/release/` with implementation and tests
4. Wire up in `cmd/pk/main.go`
5. Update `.changelog.json` with preRelease hook
6. Delete `scripts/release.sh` and `scripts/` directory
7. Rename `docs/changelog.md` to `docs/pk-changelog.md`; create `docs/pk-release.md`
8. Update `Makefile`, `CONTRIBUTING.md`, `CLAUDE.md`

## Verification

```bash
make test                         # all existing tests pass
pk release --dry-run              # validates pre-flight (will fail if no tag at HEAD — expected)
pk changelog --dry-run            # confirm changelog still works
grep -r 'release\.sh' .          # zero references remain
```
