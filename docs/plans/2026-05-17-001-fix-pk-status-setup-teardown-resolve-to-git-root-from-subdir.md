# Fix: pk status, setup, teardown resolve to git root from subdirectories

## Context

Running `pk status` or `pk setup` from a subdirectory fails to find the project's plankit configuration at the repository root. Worse, `pk setup` creates files in the wrong directory. This was already fixed for `pk changelog` and `pk release` in commit 77a57fa, but the same pattern wasn't applied to the remaining commands.

## Approach

Add a stat-based `git.RepoRoot()` function that walks up to the directory containing `.git` and returns it. Add a `resolveProjectDir()` helper in `main.go` that resolves the `--project-dir` flag to an absolute path then walks up to the git root. Replace `resolveDir` with `resolveProjectDir` at the three call sites (setup, status, teardown).

The `--project-dir` flag stays (portable for non-Claude environments) but now means "start here and resolve up to the git root" rather than "use this exact directory."

## Changes

### 1. `internal/git/isrepo.go` — refactor `IsRepo` into `RepoRoot`

`IsRepo` already walks up looking for `.git`. Rename the implementation to `RepoRoot` (returns path + bool), then rewrite `IsRepo` as a wrapper:

```go
func RepoRoot(stat func(string) (os.FileInfo, error), dir string) (string, bool) {
    abs, err := filepath.Abs(dir)
    if err != nil {
        return "", false
    }
    for {
        if _, err := stat(filepath.Join(abs, ".git")); err == nil {
            return abs, true
        }
        parent := filepath.Dir(abs)
        if parent == abs {
            return "", false
        }
        abs = parent
    }
}

func IsRepo(stat func(string) (os.FileInfo, error), dir string) bool {
    _, ok := RepoRoot(stat, dir)
    return ok
}
```

### 2. `internal/git/isrepo_test.go` — add test for `RepoRoot()` return value

Existing `IsRepo` tests continue to pass (behavior unchanged). Add test that `RepoRoot` returns the correct ancestor path.

### 3. `cmd/pk/main.go` — add `resolveProjectDir()`, change 3 call sites

```go
func resolveProjectDir(dir string) string {
    abs := resolveDir(dir)
    if root, ok := pkgit.RepoRoot(os.Stat, abs); ok {
        return root
    }
    return abs
}
```

Import `pkgit "github.com/markwharton/plankit/internal/git"`.

Change:
- `runSetup` line 158: `resolveDir` → `resolveProjectDir`
- `runStatus` line 241: `resolveDir` → `resolveProjectDir`
- `runTeardown` line 262: `resolveDir` → `resolveProjectDir`

No changes to `Run()` functions, Config structs, or mode-inference logic. The mode-inference block in `runSetup` (lines 163-185) already reads settings from `dir` — now `dir` is the git root, so it finds the correct settings.json.

### 4. `CLAUDE.md` — add design rule

Under "Design", add:

- **All commands resolve to the git repository root.** A pk command can be invoked from any subdirectory; it walks up to find `.git` and operates there. Commands don't require being at the root, but they effectively run as if they are. Non-git fallback: when no `.git` exists up the tree, the command uses the provided directory as-is.

Update "Repo Checks" to note that `git.RepoRoot` is the standard for directory resolution across all commands.

### 5. Tests

- `internal/git/isrepo_test.go`: unit tests for `RepoRoot`
- Smoke: `cd internal/status && ../../dist/pk status` reports configured

## Non-git fallback

When no `.git` exists up the tree, `RepoRoot` returns `("", false)` and `resolveProjectDir` falls back to the absolute path. All existing non-git behavior is preserved (`--allow-non-git` for setup, "not a git repository" reporting for status).

## Files to modify

- `internal/git/isrepo.go`
- `internal/git/isrepo_test.go`
- `cmd/pk/main.go`
- `CLAUDE.md`

## Verification

1. `make test` — all existing tests pass
2. `make build && dist/pk status` from root — still works
3. `cd internal/status && ../../dist/pk status` — now reports configured
4. `make lint` — clean
