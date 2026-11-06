# Review fixes, phase 5: docgen reads the main module, and leftovers

## Context

A code review of the repository at v1.8.3 is being fixed in phases.
Phase 4 gave each message and check in the main module one home.
Phase 5 does the same for `tools/docgen`, the build-time module that
compiles the skills into `pk help`'s data and builds the site, and
then clears the leftovers the review listed.

docgen is a separate module so the pk binary carries no third-party
code, but it already imports pk's own packages (`cli`, `config`,
`registry`, `help`). The comment that justifies its copies, "the
schema is declared on both sides", was written before that was true.
The repository's own rule is that a new surface is a new reader of an
existing source, never a new copy. The copies:

- The five IR types and the schema number are declared in
  `tools/docgen/main.go` and again in `internal/help/ir.go`.
- "Is this page a command page" is decided four ways: `isCommandDoc`
  in `help`, `isCommandTitle` in docgen, a byte comparison in the site
  builder, and a regular expression in `cmd/pk/main_test.go`. The
  index order is written twice (`rank` in `help` and in the site
  builder).
- `notes.go` parses and compares versions by hand, and finds the
  previous tag by building a fake release note to reuse its own
  parser. `version.ParseSemver` and `version.Latest` exist.
- The latest tag, the tag list, and the repository's web address each
  have a docgen version beside the main module's (`changelog.LatestTag`,
  `git.Tags`, `git.ParseRepoURL`).
- Frontmatter is read three ways: for a skill, for a release note, and
  to strip it for `pk help document`.

The outcome: seven local commits on `develop`, the suite green before
and after each. Nothing is pushed and no release is cut.

## 1. The IR is the help package's

`tools/docgen/main.go`: the `doc`, `meta`, `block`, `span`, and `run`
types go; the compiler builds `help.Doc`, `help.Block`, `help.Span`,
and `help.Run`, and stamps `help.SchemaVersion` where it wrote `1`.
The field order and JSON tags are the same, so the compiled output is
byte for byte what it is today.

Commit: `refactor(docgen): the IR is the help package's, not a copy`

## 2. One rule for a command page, one index order

`internal/help/embed.go` exports the two things it already decides:
`IsCommandDoc(d *Doc)` (the opening heading is "pk <name>") and
`Rank(m Meta)` (the overview, then documents, then commands).

- docgen's compiler asks `help.IsCommandDoc` for its heading check;
  `isCommandTitle` goes.
- The site builder compiles each skill once and asks the same
  question, in place of comparing bytes, and orders its pages with
  `help.Rank`; its own `rank` goes.
- `cmd/pk/main_test.go`, the test that holds commands and skills one to
  one, reads the embedded page's `Meta.Command` in place of its
  regular expression.

Commit: `refactor: one rule for what a command page is, and one index order`

## 3. Versions, tags, and the repository URL come from the main module

- `tools/docgen/notes.go`: a note's version is a `version.Semver`,
  parsed by `version.ParseSemver` under the release-tag rule
  (`vMAJOR.MINOR.PATCH`, nothing after it); notes sort by
  `Semver.Compare`. `previousTag` is `version.Latest` over the tags
  below the note's version; the fake-note trick goes. The tag list is
  `git.Tags`.
- `tools/docgen/site.go`: the front page's latest release is
  `changelog.LatestTag`, the highest semver tag, where it ran
  `git describe`. At a release the two agree.
- `internal/git/git.go` gains `RepoURL(dir)`: origin's URL through
  `ParseRepoURL`, "" without an origin. `pk changelog` and docgen both
  call it. docgen keeps a link only when the address is a web one, so a
  local-path origin gives no link there; `repoWebURL` goes.

Behaviour that changes: the notes page links an origin on any web
host, as `CHANGELOG.md` does, where it linked GitHub only.

Commit: `refactor(docgen): versions, tags, and the repository URL come from the main module`

## 4. One frontmatter reader

`internal/help` exports `SplitFrontmatter(raw)`: the fields of the
leading `---` block as flat `key: value` lines, and the body after it,
with an error for a block that is missing, unterminated, or holds a
line that is not `key: value`. Each caller keeps what is its own:

- `help.stripFrontmatter`, for `pk help document`, takes the body and
  returns the page unchanged when there is no block.
- docgen's skill reader checks its allowed keys (`name`,
  `description`, `argument-hint`).
- docgen's `parseNote` checks its three required keys.

Behaviour that changes: a release note with a frontmatter line that is
not `key: value` fails the site build, where the line was skipped.

Commit: `refactor: one frontmatter reader`

## 5. Dead code in the site builder

- `tools/docgen/site.go`: the sidebar list is built once; the first
  build, discarded a few lines later, goes.
- `tools/docgen/demo.go`: the demo's `run` returned an error that was
  always nil and always ignored; it returns the step.

Commit: `refactor(docgen): dead code in the site builder is removed`

## 6. Stale comments and leftovers

- `cmd/pk/main.go`: the comment about a `commands` function that moved
  to `internal/registry` goes.
- `internal/repo/repo.go`: the `StatusCmd` comment sits on `StatusCmd`,
  not on the function above it.
- The scaffolding vocabulary ("layer 0", "layer 1", "layer 6") in
  `internal/cli/cli.go`, the package comments of `changelog` and
  `release`, and the `Makefile` is replaced by what each line means
  now. The frame's fallback for a command list without `help` stays;
  its comment says when that happens.
- `Makefile`: `.PHONY` no longer names a `vet` target that does not
  exist.
- `internal/help/engine.go` and `internal/version/version.go`: the
  local `min` functions go; the language has had one since Go 1.21 and
  the module is on 1.22.
- `internal/changelog/changelog.go`: the inner `since` that shadows the
  outer one is renamed.
- Import blocks that mix the standard library with pk's packages
  (`internal/cli/cli.go`, `tools/docgen/main.go`, two test files) are
  grouped.

Commit: `chore: stale comments, an unused target name, and shadowed names are cleared`

## 7. One protect entry in the plugin's hooks

`hooks/hooks.json`: the two identical PreToolUse entries for `Edit`
and `Write` become one with the matcher `Edit|Write`, the form the
guard entry already uses for `Bash|PowerShell`.

Commit: `refactor(plugin): one protect entry for Edit and Write`

## Where each fact lives after this phase

- The IR and its schema number: `internal/help/ir.go`.
- What a command page is, and the index order: `internal/help/embed.go`.
- Parsing and comparing versions: `internal/version`. The latest
  release tag: `changelog.LatestTag`. The tag list: `git.Tags`. The
  repository's web address: `git.RepoURL`.
- Reading frontmatter: `help.SplitFrontmatter`.
- Check: a search of `tools/docgen` for `strconv.Atoi`, `git describe`,
  and `type doc struct` shows no hits.

## Verification

- `make test` before the first change and after each of the seven,
  with the output shown.
- The compiled help data is the proof for steps 1, 2, and 4: `make
  docs` leaves no diff after each.
- The site is the proof for steps 1 to 5: before the phase,
  `make site-preview` is run with a fixed `VERSION` and its output
  copied aside; after each step the same build is compared with it,
  and the comparison shows no differences.
- The case that must fail: docgen run over a copy of the skills in
  which one command page's heading is wrong exits non-zero and names
  the heading it wanted.
- After step 7, `claude plugin validate . --strict` passes, and the
  test that matches `hooks.json` against the registered commands
  passes.
- `gofmt -l cmd internal tools` prints nothing.

## The last phase, with its own plan

- Phase 6, tokens: named mode values, the help style table, narration
  through `msg`, guard's reason prefixes, the list joiners, the site's
  hard-coded facts, and the stylesheet. Decision there: which sizes and
  radii the stylesheet keeps.
