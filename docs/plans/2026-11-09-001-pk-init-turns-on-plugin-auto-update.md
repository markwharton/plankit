# pk init turns on plugin auto-update

## Context

A repository that enables the plankit plugin holds its own project-scope
install, and that install stays at the version it was created with until
someone updates it. The hooks run the pk bundled in that install, so a
stale install runs old guard and preserve rules with nothing on screen to
say so. A session lost the record of an approved plan this way: its hooks
were two releases behind the pk on the PATH.

Claude Code has a per-marketplace auto-update setting. It is off by default
for a marketplace added by URL, and it reads the setting first from the
`autoUpdate` key on the marketplace's `extraKnownMarketplaces` entry in a
settings file. Tested in this repository: with `"autoUpdate": true` on the
plankit entry in `.claude/settings.json`, a session refreshed the
marketplace and moved the user-scope install and this repository's
project-scope install from 1.7.2 to 1.9.0, a few minutes after its first
message. That line is already committed here.

`pk init` already writes the marketplace entry, so it writes this key too.
Auto-update is on by default because a stale plugin fails without a sign,
while an unwanted update is visible and one line turns it off. A bad release
then reaches every configured repository together, which is one incident
with one fix.

## Design

The plugin entries are one table, `pluginEntries()` in
`internal/repo/settings.go`. Each row is a key path and a value, and the key
is the identity: a row that is absent is missing, a row present with another
value is the developer's choice and is left as it is. `pk init` writes the
missing rows, and `pk status` notes them with the JSON to merge in.

The table becomes three rows, none containing another's key:

| Key path | Value |
|---|---|
| `extraKnownMarketplaces.plankit.source` | `{"source": "url", "url": "https://plankit.com/marketplace.json"}` |
| `extraKnownMarketplaces.plankit.autoUpdate` | `true` |
| `enabledPlugins.plankit@plankit` | `true` |

The marketplace row moves down one level, from the whole `plankit` object to
its `source`. With the row at the object, adding `autoUpdate` inside it
would make the object differ from the row's value, and init would report its
own entry as "present with another value".

What follows from the table, with no new logic:

- A new repository gets all three. The file `pk init` creates matches this
  repository's `.claude/settings.json` byte for byte.
- A repository configured by an earlier release lacks only `autoUpdate`.
  `pk status` notes it and prints the one key to merge in.
- `"autoUpdate": false` is present with another value. `pk status` says
  nothing, and `pk init` notes it and leaves it. This is how a team turns
  auto-update off.
- A developer who pointed the `plankit` marketplace at another source keeps
  that source. If that entry has no `autoUpdate` key, init adds
  `"autoUpdate": true` to it.

## DRY boundaries

- **The table is the only place the entries are named.** No second list, no
  `autoUpdate` constant elsewhere, no special case in init or status.
- **One classifier.** `pluginEntryState` sorts rows into missing and
  differing for both commands. It is not changed.
- **One writer.** `addPluginEntries` over `jsonsplice.Insert` writes the
  file for init and renders the note's JSON for status. `Insert` already
  creates objects missing on the way, at any depth. It is not changed.
- **One note per command.** Status keeps its single "merge in" note and
  init keeps its single "present with another value" note. Only the status
  note's wording changes, because "the entries that load the plugin" is
  false when the one missing key is `autoUpdate`. New wording:
  `<file> lacks plugin entries that pk init writes; merge in:`.

## Changes

1. `internal/repo/settings.go`: the three-row table; the comment on
   `settingsFile` says three entries and names what each is for.
2. `internal/repo/repo.go`: the status note's wording.
3. `internal/repo/repo_test.go`:
   - `TestInitSplicesTheDeveloperSettings`: expected bytes gain
     `"autoUpdate": true` as the marketplace object's last key.
   - `TestInitLeavesPresentEntriesAlone`: the developer's file carries
     another source and `"autoUpdate": false`; nothing is written and both
     differences are noted, under the new key paths.
   - New: a developer's marketplace entry with no `autoUpdate` gains the key
     and keeps its source.
   - `TestStatusPluginEntriesNote`: a file lacking only `autoUpdate` gets a
     note showing only that key; `"autoUpdate": false` gets no note; the
     wording assertion follows the new text.
4. `internal/jsonsplice/jsonsplice_test.go`: one case inserting a key two
   missing levels down, the depth status renders.
5. Pages, edited in `skills/` and compiled with `make docs`:
   - `skills/init/SKILL.md`: the plugin-entries bullet names `autoUpdate`,
     what it turns on, and that `false` is left as it is.
   - `skills/overview/SKILL.md`: the footprint paragraph names the key. The
     Updating section says a configured repository updates its own install
     and the new version loads in the next session or after
     `/reload-plugins`; the three manual steps stay for a repository without
     the key.
   - `skills/status/SKILL.md`: reread; its sentence about a missing plugin
     entry still holds.
6. `README.md`: the `pk init` paragraph says the run turns on auto-update.
   The Install section's sentence about auto-update says when Claude Code
   runs it, corrected to match what was tested.
7. `docs/notes/v1.10.0.md`: the release note, required for a minor. Read
   `docs/notes/README.md` and two earlier notes first. Two paragraphs:
   `pk init` turns on plugin auto-update and `pk status` prints the key for
   a repository configured earlier; the preserve hook reports an approved
   plan whose path is outside the plans directory. The version comes from
   `pk changelog --dry-run` at the time.

Reread `docs/architecture.md` and each changed command's `Summary` and
`description` before committing.

## Commits

1. `feat(init): turn on plugin auto-update in .claude/settings.json`, with
   the status note, tests, pages and README.
2. `docs: release notes for v1.10.0`.

Nothing is pushed.

## Not in this plan

Rewording the brief's first sentence and the cloud snapshot sentence on the
overview page. Both are separate changes awaiting a decision.

## Verification

- `make test` before the change and after it.
- `make docs` twice: the second run leaves no diff.
- `claude plugin validate . --strict` passes.
- Smoke test with `./bin/pk` in a scratch repository:
  - `pk init`, then compare the created `.claude/settings.json` with this
    repository's file: identical.
  - `pk status`: no note.
  - Remove the `autoUpdate` line and commit: `pk status` prints the note
    with only that key.
  - Set `"autoUpdate": false` and commit: `pk status` prints no note.
  - Must fail: break the file's JSON; `pk status` exits 2 and names the
    file.
- In this repository, `./bin/pk status` prints no note.
- `pk changelog --dry-run` reports v1.10.0.
