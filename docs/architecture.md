# Architecture

pk is one Go binary in three layers, distinguished by how much of Claude Code each one depends on. This record states the layers, the file boundary that keeps them apart, and how a second AI coding tool would be added.

## Three layers

- **Git workflow, no coupling**: `pk init`, `pk changelog`, `pk release`, `pk status`, `pk pin`, `pk version`. They read git and `.pk.json` and write git artifacts: commits, tags, `CHANGELOG.md`. They run with any AI tool or none.
- **Rules and skills, format coupling**: markdown that the tool loads as context (`.claude/rules/`) or runs on request (`.claude/skills/`). The content is tool-independent; the paths and frontmatter are Claude Code's.
- **Hooks, protocol coupling**: `pk guard`, `pk protect`, `pk preserve`, the settings that wire them, and `install-pk.sh`. They exist because Claude Code offers PreToolUse, PostToolUse and SessionStart hooks; no other tool currently intercepts a tool call before it runs.

Rules describe; hooks enforce. In a tool without hooks, the first two layers apply and nothing backstops a rule the model ignores.

## The file boundary

`internal/setup/` keeps provider code in one file:

```
internal/setup/
├── baseline.go           git tag baseline
├── claude.go             Claude Code: hooks, settings merge, permission, bootstrap script
├── managed.go            hash-tracked file management
├── pin.go                version pinning
├── ruleset.go            GitHub ruleset template
├── setup.go              Config, Run(), OrderedObject
├── walk.go               rule-file walking
├── rules/                shipped rules
├── skills/               shipped skills, Claude Code format
└── template/             CLAUDE.md template, install-pk.sh
```

`Run()` in `setup.go` calls the provider-independent files and `claude.go`.

## Adding a provider

Not built. When a second tool needs support: copy `claude.go` to `<provider>.go`; adapt hook types, settings paths and file formats; wire its steps into `Run()` behind a flag or detection. `baseline.go`, `managed.go`, `pin.go` and `setup.go` do not change. The cost is one file per provider and a skills directory in its format. Reopen when a tool ships pre-tool interception; without it the provider gets rules and skills only.

Other design decisions: [docs/design.md](design.md).
