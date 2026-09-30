# pk init carries the plugin-loading keys in .claude/settings.json

## Context

Anthropic's documented practice for a repository whose contributors use a
plugin is to set two keys in that repository's `.claude/settings.json`
(plugins/org: "set `extraKnownMarketplaces` and `enabledPlugins` in that
repository's `.claude/settings.json`"):

```json
"extraKnownMarketplaces": { "plankit": { "source": { "source": "url", "url": "https://plankit.com/marketplace.json" } } },
"enabledPlugins": { "plankit@plankit": true }
```

A teammate who trusts the folder gets the marketplace added. The plugin is
not fetched, because its source is external (archive); the plugin errors tab
shows `Plugin "plankit" is enabled in project settings but isn't installed
here`, and one `/plugin install plankit@plankit` finishes it. Project settings
win over user settings key by key, so a project `true` overrides a teammate's
user `false`. Cloud sessions never show the trust dialog and are out of scope.

Nothing in plankit writes or documents these keys today. This repository has
no committed `.claude/settings.json` either; the plugin-first rewrite removed
it because the plugin's `hooks/hooks.json` wires what that file used to. It
is the first consumer.

Two ways were weighed: a step in the init skill where the session edits the
file, and pk init doing it. The skill route was rejected: it turns "run pk
init and relay" into a conditional procedure, has a model hand-edit the
developer's JSON in the file where the 2026-04-06 clobber happened, and puts
a write-settings.json instruction in a shipped skill. The decision is that
pk init writes the keys: they are part of the per-repo footprint, and init
is the one run that configures a repository.

## What exists to build on

- `internal/changelog/versionfile.go:26` `spliceJSONVersion`: token-walks the
  root object with `json.Decoder`, records `InputOffset` spans, replaces only
  the bytes of the root `version` value. Replace-only, root-only, string
  values only, errors when the key is absent. The technique carries over.
- `internal/changelog/changelog_test.go:282`
  `TestVersionFileSplicePreservesFormatting`: the one byte-preservation test,
  the shape to copy.
- `internal/repo/repo.go:46` the run-twice refusal; `:109` the commit through
  `git.CommitPaths(dir, message, paths...)`, already variadic; `:215`
  `runStatus` with the readiness-note slot at `:282-287`.
- `docs/design.md` "A file format": "Write it whole if pk owns the file;
  splice it if the developer does." `.claude/settings.json` is the
  developer's file.

## Changes

### 1. A splice primitive, shared

Lift the offset-walking core of `spliceJSONVersion` into a small package
(`internal/jsonsplice` or similar) with two operations: replace the value of
a key at a path of depth one or two, and insert a key into an object that
lacks it, copying the indentation of its siblings and placing it before the
closing brace. `spliceJSONVersion` becomes a caller, so the changelog test
proves the primitive too. Values are encoded with `SetEscapeHTML(false)`. An
absent file is treated as `{}` and goes through the same insert, so there is
no separate write-whole path; a file that does not parse is refused with
exit 2 naming the file, before anything is written. The primitive is bytes
in, bytes out: it does no file I/O, and a `Has(path)` query built on the
same key-locating walk serves both init's no-op check and status's note.

### 2. Init wiring, `internal/repo/repo.go`

A step labelled `.claude/settings.json` after `.pk.json`; both paths go to
`git.CommitPaths`. `--no-commit` leaves both uncommitted. `--dry-run`
previews it like the other steps. Precondition: if `.claude/settings.json`
exists with uncommitted changes, refuse (exit 2, "commit or stash it first")
so init's commit does not sweep in unrelated edits. The check is
`git.Clean` given the path: `Clean(dir, paths ...string)` scopes the one
porcelain call, and existing callers pass nothing and keep the whole-tree
meaning. If both keys are already present with the right values, the step
is a no-op and the file is not touched or committed.

Key names are the identity: `plankit` under the marketplaces and
`plankit@plankit` under the plugins. A key that is present with a different
value is the developer's choice; init leaves it and says so, and never
replaces it. The file path, the marketplace name and URL, and the plugin id
are constants declared once in this package, next to `configureSubject`,
and every reader (init, status, tests) uses them.

### 3. Status note

In the existing readiness slot, through `msg.Notef` under the same `--quiet`
gate as the other notes: when `.pk.json` is present and
`.claude/settings.json` lacks either entry, one stderr note naming the file
and printing the missing entries, rendered from the same constants init
writes. The note is the one place the exact entries are shown to a
developer, so no page carries a hand-copied JSON block. No new field in the
`state` struct: readiness notes are stderr-only today and this one matches.
This covers repositories that `pk init` already configured; init keeps
refusing to run twice.

### Tests

- Splice: odd whitespace and key order preserved byte-for-byte outside the
  inserted region; insert into an empty object; insert into an object with
  existing entries (comma placement); replace an existing entry; nested
  insert under a present root key; malformed input refused with no write.
- Init: absent settings file; existing file with a `hooks` block and
  `permissions.allow`, diff shows only the added keys; already-present keys
  give no rewrite and no second path in the commit; a present key with a
  different value is left and reported; dirty settings file is refused with
  a clean tree; `--dry-run` lists the step; `--format json` `created` names
  it.
- Status: note present when keys are missing, absent when present, dropped
  with `--quiet`.

### Pages

- `skills/init/SKILL.md`: add the settings file to the created list with one
  sentence saying what the two keys do for a teammate; `--no-commit` now
  leaves two files; the refusal-on-dirty-settings sentence.
- `skills/overview/SKILL.md` footprint: `.claude/settings.json` carries the
  marketplace and the plugin enabled, so a teammate who trusts the folder is
  one install away. Prose names the two keys; no JSON block, since the
  status note prints the entries from code.
- `skills/status/SKILL.md`: the new readiness note in the list of notes.
- `README.md`: the after-init paragraph adds the settings file to what one
  run writes. The `Bash(pk:*)` allow block stays as it is; permissions
  remain the developer's and pk never writes them.
- This is a `feat`, so the release is a minor and needs
  `docs/notes/<version>.md`.

## Boundaries, so the change adds no second copy

- **One JSON walker.** The primitive replaces the body of
  `spliceJSONVersion`; the changelog keeps its `updateVersionFile` wrapper
  and its `readFile`/`writeFile` seams in `trailer.go`. The new package has
  no file I/O and no read/write wrappers of its own; init and status read
  the file with `os.ReadFile` as `repo.go` already does for stat.
- **One encoder for inserted values.** The no-escape encoder lives in the
  primitive. `config.Write` still uses `MarshalIndent` and escapes `&`;
  that is a latent issue for hook commands in `.pk.json` and is not this
  plan's, but if it is fixed it moves onto this encoder rather than growing
  a second one.
- **One presence check.** `Has` in the primitive answers "is this key here
  with this value" for both init's no-op and status's note. Neither command
  parses the file on its own.
- **One set of constants**, in `internal/repo` beside `configureSubject`:
  the settings path, marketplace name, marketplace URL, plugin id. The
  README's install command already states the URL for humans; nothing else
  repeats it.
- **One clean check.** `git.Clean` gains variadic paths; no `CleanPaths`
  sibling.
- **One byte-preservation test.** `TestVersionFileSplicePreservesFormatting`
  moves to the primitive's package as its formatting test; the changelog
  keeps a single end-to-end version-stamp test.
- **Existing test helpers.** Init and status tests use `scratch`, `mustGit`,
  and `run` from `internal/repo/repo_test.go`.

## Lessons from the v0.x settings code

The pre-rewrite `pk setup` parsed settings.json into Go structures and
re-serialised the file. Five of the seven fixes on it are one bug class,
things the round trip dropped: hooks replaced (`4319731`), unknown hook
categories and a stamped `"timeout": 0` (`97ce733`), unknown fields on hook
objects (`8b2d05d`), alphabetical key order (`1813b68`, OrderedObject), HTML
escaping of `&` `<` `>` even inside a custom MarshalJSON (`f4fd4d5`).
OrderedObject was that approach's ceiling and still re-indented the tree.
A byte splice has none of these failure modes; the old code stays deleted.

What carries over:

- Never decode a developer value into a typed struct; read offsets and raw
  bytes only.
- Encode inserted values with `SetEscapeHTML(false)`.
- Absent and malformed are different: absent is written whole, malformed is
  refused with exit 2 naming the file, nothing written.
- Write only when the bytes change, compared against the bytes read. No
  backup file: git holds the before state, and `69ce680` shows a backup
  scheme destroying the original on the second run.
- No injected filesystem. Test through real repositories in `t.TempDir()`
  as the suite does now.

## Verification

- `make test` before and after each change.
- Smoke, scratch repository: write a `.claude/settings.json` with unusual
  spacing, a `hooks` block, and `permissions.allow`; `pk init --dry-run`
  lists the step; `pk init` commits two files; `git show --stat` names both;
  `diff` of the settings file shows only the two added regions.
- Smoke, second scratch repository configured before the change: `pk status`
  prints the note; `pk status --quiet` does not.
- Must fail: a settings file containing `{` alone; `pk init` exits 2 naming
  the file and the tree stays clean with no `.pk.json`.
- `claude plugin validate . --strict` and `make docs` leave the tree as
  expected.

## Out of scope

- Cloud sessions.
- Permissions: pk never writes `permissions.allow`.
- Relaxing the run-twice refusal or a new command for configured
  repositories; the status note and the overview cover them.
