# pk setup

Install pk into a project: hooks, skills, rules, `CLAUDE.md`, the bootstrap script, and the modes in `.pk.json`.

## Usage

```bash
pk setup                              # install or refresh; modes default to block, block, manual
pk setup --guard ask                  # branch policy: block | ask | off
pk setup --push-guard ask             # push policy:   block | ask | off
pk setup --preserve auto              # preserve:      auto | manual | off
pk setup --force                      # overwrite modified managed skills and rules
pk setup --allow-non-git              # proceed outside a git working tree
pk setup --baseline                   # tag v0.0.0 on HEAD when no semver tag exists
pk setup --baseline --at <ref>        # tag <ref> instead of HEAD
pk setup --baseline --push            # and push the tag (HEAD too, without --at)
pk setup --project-dir /path/to/dir   # start the git-root search there
```

## How it works

1. Writes the hooks into `.claude/settings.json`: `pk guard` (PreToolUse on Bash and PowerShell), `pk protect` (PreToolUse on Edit and Write), `pk preserve` (PostToolUse on ExitPlanMode), and `.claude/install-pk.sh` (SessionStart); adds the `Bash(pk:*)` permission. Hook lines are bare; the modes live in `.pk.json`. Hooks that are not pk's are kept.
2. Writes `guard.mode`, `guard.push` and `preserve.mode` into `.pk.json`, field-merged with the keys already there. Each mode is resolved as: the flag, then the existing `.pk.json` value, then a value migrated from an old flag-bearing hook line, then the default (`block`, `block`, `manual`).
3. Writes `CLAUDE.md` when there is none; updates it when it is pk-managed and unmodified; leaves it when modified or unmanaged.
4. Installs `craft.md`, `conduct.md` and `docs.md` under `.claude/rules/plankit/`, a directory a project's own rules never collide with; Claude Code discovers rules recursively.
5. Installs `/pk-configure`, `/preserve` and `/ship` under `.claude/skills/`.
6. Writes `.claude/install-pk.sh`, pinned to the running pk version. Skipped on dev builds.
7. Warns when `pk` is not on PATH.
8. With no valid semver tag in the repository, prints the hint `pk setup --baseline --push` with its git equivalent.

Restart Claude Code afterwards; hooks are read when a session starts. Re-run `pk setup` after each pk upgrade and commit the result on its own.

## Flags

- **--guard `<mode>`**: writes `guard.mode`.
- **--push-guard `<mode>`**: writes `guard.push`.
- **--preserve `<mode>`**: writes `preserve.mode`.
- **--force**: overwrite managed skills and rules whose content was modified. `CLAUDE.md` is never overwritten.
- **--allow-non-git**: install outside a git working tree. Rules, skills and `pk protect` work there; `pk guard` and `pk preserve` do nothing; `pk changelog` and `pk release` exit 1.
- **--project-dir `<dir>`**: where the search for the repository root starts. Default: the current directory.
- **--baseline**: after the install, tag `v0.0.0` on HEAD when no tag parses as semver. With an existing semver tag, prints which tag was found and does nothing.
- **--at `<ref>`**: with `--baseline`, tag `<ref>` instead of HEAD.
- **--push**: with `--baseline`, push the tag; without `--at`, push HEAD as well so the tagged commit is on a branch on origin. Without `--push` the tag stays local and pk prints the push command.

The modes and their values: [.pk.json](pk-json.md#guard).

## Configuration

`pk setup` writes `guard.mode`, `guard.push` and `preserve.mode`. Every other key in `.pk.json` is left as it is. See [.pk.json](pk-json.md).

## Managed files

Files pk installs carry a hash of their body: `<!-- pk:sha256:... -->` in `CLAUDE.md`, `pk_sha256` in the frontmatter of skills and rules.

- A file whose hash still matches is updated on the next `pk setup`, and removed when pk no longer ships it.
- A file whose hash does not match was edited: it is left alone, with a warning for a removed one. `--force` overwrites modified skills and rules; nothing overwrites a modified `CLAUDE.md`.
- A skill or rule without a marker is yours and is never touched.
- `.claude/install-pk.sh` carries no marker and is always rewritten.

`pk status` reports each managed file as pristine or modified.

## Baseline

`pk changelog` reads commits since the latest semver tag, so a repository needs one before its first release. `pk init` creates it for a new repository; `pk setup --baseline` creates it for an existing one:

| Situation | Command | Effect |
|---|---|---|
| No tag; history is not release notes | `pk setup --baseline` | `v0.0.0` on HEAD; the first release covers what follows |
| No tag; history belongs in the first release | `pk setup --baseline --at $(git rev-list --max-parents=0 HEAD)` | `v0.0.0` on the first commit; the first release lists everything |
| A semver tag exists | nothing | the step is a no-op; the next release is computed from that tag |

Add `--push` to publish the tag in the same run.

## Bootstrap

`.claude/install-pk.sh` runs at session start. On a machine with `pk` on PATH it exits at once. In a cloud sandbox (Claude Code on the web) it downloads the pinned `pk` from GitHub Releases into `$HOME/.local/bin`, then runs `git fetch --tags`, since sandboxes clone only the working branch. With no `pk` on PATH on a local machine it prints a warning with the install command. `pk version` reports when the pinned version differs from the running one.
