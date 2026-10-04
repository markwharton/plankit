# Privacy

plankit is a Claude Code plugin and a command, pk, that run on your
machine. This page says what they read, what they write, and what
leaves the machine.

## What pk reads

The repository it runs in: its working tree, its git history (commit
hashes, subjects, bodies, and trailers, never author names or emails),
its tags, and its remotes. Two files it owns or adds to:
`.pk.json`, the policy, and `.claude/settings.json`, the plugin
entries. When Claude Code runs a hook, pk also reads the hook's
payload, which names the session's working directory and the tool
call being made.

## What pk writes

Only into the repository: `.pk.json`, the two plugin entries in
`.claude/settings.json`, preserved plans under the plans directory,
`CHANGELOG.md`, the version fields the policy names, commits, and
tags. Nothing is written anywhere else on the machine, and nothing is
stored by plankit.

## What leaves the machine

Nothing, except what git pushes to your own remote when you ask for a
release, or ask `pk init` or `pk preserve` to push. pk contacts no
service, sends no telemetry, and reads no credentials. The plugin's
five hooks and fourteen skills run pk locally; the skills are text.

Installing or updating the plugin is Claude Code's download from
GitHub, through the marketplace file plankit.com serves. This site
sets no cookies and runs no analytics.

## Contact

Questions go to the repository's issues,
https://github.com/markwharton/plankit/issues.
