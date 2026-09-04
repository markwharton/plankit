# plankit

Plan-driven development for Claude Code.

This branch is the plugin-first rewrite, built in layers. plankit ships
as a Claude Code plugin: the skills are the documentation, the hooks
call the pk binary, and `pk help` renders the same pages in the
terminal. The pk runtime is the smallest useful kernel with the help
engine built in.

A configured repository carries exactly two things: `.pk.json`
(committed repo policy) and `docs/plans/` (preserved plans).

The v1 implementation and its documentation live in git history on
`main`.
