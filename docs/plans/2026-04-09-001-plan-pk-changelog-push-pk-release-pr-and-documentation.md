# Plan: pk changelog --push, pk release --pr, and documentation

## Context

Plankit currently supports two release flows: legacy (push from current branch) and merge (merge dev → main, push). Two gaps emerged from a real-world analysis of how plankit would work with CI/CD platforms like Azure Static Web Apps:

1. **`pk changelog` doesn't push.** After `pk changelog` on main (feature branch → PR → merge flow), the tag and commit stay local. There's no built-in way to push them.
2. **`pk release` can't create PRs.** For preview-environment workflows (Azure SWA, Netlify, Vercel), developers want a PR — not a direct merge — so the platform creates a preview site for review before merging to production.
3. **Squash merge silently breaks release tags.** This isn't documented anywhere. Builders need to know.

## Implementation

### Phase 1: Extract `parseRepoURL` to shared package

Move `parseRepoURL` from `internal/changelog/changelog.go:595-607` to `internal/git/url.go` as exported `ParseRepoURL`. Move tests from `internal/changelog/changelog_test.go` to `internal/git/url_test.go`. Update changelog.go to call `pkgit.ParseRepoURL`.

This is a pure refactor — no behavior change. Needed because `pk release --pr` needs the compare URL fallback.

**Files:**
- Create `internal/git/url.go`
- Create `internal/git/url_test.go`
- Edit `internal/changelog/changelog.go` — delete local function, use `pkgit.ParseRepoURL`
- Edit `internal/changelog/changelog_test.go` — remove `TestParseRepoURL`

### Phase 2: `pk changelog --push`

Add `Push bool` to `changelog.Config`. After the tag step (`changelog.go:307`), conditionally push:

```go
if cfg.Push {
    if _, err := cfg.GitExec("", "push", "origin", "HEAD", nextTag); err != nil {
        fmt.Fprintf(cfg.Stderr, "Error: git push failed: %v\n", err)
        return 1
    }
    fmt.Fprintf(cfg.Stderr, "Pushed to origin\n")
}
```

No interaction with `--dry-run` needed — dry run returns before the push code runs.

Add `--push` flag in `cmd/pk/main.go:runChangelog`.

**Files:**
- Edit `internal/changelog/changelog.go` — add Push field, push logic
- Edit `internal/changelog/changelog_test.go` — add TestRun_push, TestRun_pushFailure, TestRun_pushDryRun
- Edit `cmd/pk/main.go` — add --push flag

**Tests:**
- Happy path: Push=true → verify "push origin HEAD v0.1.0" in git calls
- Push failure: push returns error → exit code 1, "git push failed" in stderr
- Dry run: Push=true, DryRun=true → no push called

### Phase 3: `pk release --pr`

Add two fields to `release.Config`:

```go
PR     bool
ExecGh func(args ...string) (string, error)
```

`DefaultConfig` wires `ExecGh` to `exec.Command("gh", args...).CombinedOutput()`.

New flow variables after loading config:

```go
needsMerge := releaseBranch != "" && sourceBranch != releaseBranch && !cfg.PR
needsPR := cfg.PR && releaseBranch != "" && sourceBranch != releaseBranch
```

Validation: `--pr` without `release.branch` → error, exit 1.

The PR flow block goes after the preRelease hook and before the existing dry-run exit. It handles its own dry-run path:

1. **Dry run:** print "Would push [branch] [tag]" and "Would create PR: dev → main", return 0
2. **Push:** `git push origin [sourceBranch] [tag]` (same single command as merge flow)
3. **Create PR:** `gh pr create --base [releaseBranch] --head [sourceBranch] --title "Release [tag]" --fill`
4. **Fallback:** if gh fails, print compare URL via `pkgit.ParseRepoURL` + `/compare/main...dev`
5. gh failure is **not fatal** — exit 0 with the compare URL. The branch is pushed; PR can be created manually.

Add `--pr` flag in `cmd/pk/main.go:runRelease`.

**Files:**
- Edit `internal/release/release.go` — add PR/ExecGh fields, PR flow
- Edit `internal/release/release_test.go` — add PR flow tests
- Edit `cmd/pk/main.go` — add --pr flag

**Tests:**
- Happy path: PR=true, ExecGh succeeds → "PR created" with URL, no merge
- gh fails: ExecGh returns error → compare URL printed, exit 0
- No release.branch: PR=true, no config → exit 1 with clear error
- Dry run: PR=true, DryRun=true → no push, no gh, "Would push" + "Would create PR"
- No tag: PR without tag → push branch only, title uses branch name
- Push fails: → exit 1, "git push failed"

### Phase 4: Documentation

**`docs/pk-changelog.md`** — Add `--push` to usage, how-it-works, and flags sections.

**`docs/pk-release.md`** — Add PR flow section, `--pr` flag, three-workflow table:

| Flow | Config | Command | What happens |
|------|--------|---------|--------------|
| Legacy | no `release.branch` | `pk release` | Push current branch + tag |
| Merge | `release.branch` set | `pk release` | Merge to release branch, push both |
| PR | `release.branch` set | `pk release --pr` | Push source branch, create PR |

**`docs/anti-patterns.md`** — Add "Squash Merge and Release Tags" section:
- The pattern: squash merge creates new commit, orphans the tag
- Diagram showing merge commit vs squash
- The fix: tag on the branch that receives the final commit, use merge or rebase (not squash) for release PRs

**`docs/resources.md`** — Update release flow examples to show all three workflows including `pk changelog --push` and `pk release --pr`.

**`cmd/pk/main.go`** — Update usage strings for both commands.

## Verification

1. `make build` — compiles
2. `make test` — all tests pass including new ones
3. `make lint` — go vet clean
4. Manual: `pk changelog --push --dry-run` — previews without pushing
5. Manual: `pk release --pr --dry-run` — shows "Would push" and "Would create PR"
6. Manual: verify `pk changelog --push` actually pushes (on a test branch)
7. Manual: verify `pk release --pr` creates a PR or prints compare URL

## Commit plan

Three commits:
1. `refactor: extract parseRepoURL to internal/git package`
2. `feat: add --push flag to pk changelog and --pr flag to pk release`
3. `docs: document release workflows, squash merge warning, new flags`
