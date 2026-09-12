# pk help document, and a man renderer for the IR

## Context

`pk help` prints one page at a time. There is no way to read plankit's
documentation as a single document, or to hand it to anything that
expects one file. `pk help document > plankit.md` should produce the
whole set, in the order the index already defines.

The second reason is structural. `skills/<topic>/SKILL.md` compiles to
an IR (`internal/help/ir.go`) that two surfaces read: the terminal
renderer and the site. A roff writer is a third reader, and an
exhaustive one, so it turns "the IR carries enough structure" from an
assumption into something the test suite asserts. It adds no
dependency: roff is text.

The outcome: `pk help document` for the whole set, `--format man` for
roff, and `pk help document --format man > pk.1` for anyone who wants
a man page. Installing one is not in scope.

## The document

A reserved pseudo-topic, intercepted in `help.run` before the `Topic`
lookup (`internal/help/help.go:25`), assembled from `Topics()` in
display order (`internal/help/embed.go:65`, the same `rank` the site
mirrors).

Frontmatter is stripped and pages are separated by a `---` rule with
exactly one blank line either side. Each page already carries its own
H1, so no headings are invented. A single topic keeps today's
guarantee of the exact authored bytes; `document` is a reader's
document, and the help page says so.

`document` must shadow no topic. A test asserts no embedded topic
carries that name, since `TestCommandsAndSkillsAreOneToOne` would not
catch it.

## The format flag

`--format` is declared per command (`cli.FormatFlag`, `cli.go:293`) but
validated centrally against `text|json` (`context.go:72`), so `man`
cannot be added at the edge. Adding it to the central switch would
make `pk status --format man` parse, which is wrong.

Instead, the allowed set becomes part of the declaration, which is how
this frame already works: commands are data.

- `cli.Command` gains `Formats []string`. Empty means `{"text","json"}`,
  so every existing command is unchanged.
- `parse` passes `cmd.Formats` into `resolve` (`cli.go:199`, the only
  caller), which validates against it and names the set in the error.
- `help` declares `cli.FormatFlag` with `Formats: []string{"text","man"}`.

`internal/cli/cli_test.go:132` pins the mechanism, not the words, so it
keeps passing; the usage line for help changes because `FlagBlock`
derives it from the declaration.

## The roff writer

New `internal/help/roff.go`, `RenderMan(d *Doc, title string) string`.

It is a parallel walker, not a reuse of `renderBlock`: that one is
welded to width-wrapping and the `codes` table (`engine.go:75`), and
roff fills text itself. Mapping: headings to `.SH` and `.SS`,
paragraphs to `.PP`, code blocks to `.nf`/`.fi` in `.RS`, lists to
`.IP`, quotes to `.RS`/`.RE`, rules to `.PP`; spans to `\fB`, `\fI`,
`\fR`. Highlight `Run` kinds collapse to plain text.

Two things that break naive writers, so they are the first tests:

- Escaping: a leading `.` or `'` must be escaped (`\&.`), backslashes
  doubled, and hyphens written `\-`.
- Exhaustive switches over `Block.Type` and `Span.Type` with a
  `default` that returns an error, so a new IR node fails the suite
  rather than vanishing.

The `.TH` line carries the page name, section 1, and the version from
`internal/version`. No volatile date: the date comes from a
package-level `now` var, stubbed in tests the way `internal/changelog`
and `internal/preserve` already do.

## Files

- `internal/cli/cli.go`, `internal/cli/context.go`: the `Formats` field
  and the validation it drives.
- `internal/help/help.go`: the flag declaration, the `document`
  intercept, the format dispatch.
- `internal/help/roff.go`: new.
- `internal/help/roff_test.go`: new. `internal/help/help_test.go` and
  `internal/cli/cli_test.go`: extended.
- `skills/help/SKILL.md`: prose for `document` and `--format man`. The
  `## Flags` region appends itself, because `setRegion` adds the
  heading and markers when a page lacks them and the body is non-empty
  (`tools/docgen/regions.go:113`).

## Tests

- Roff unit tests in the shape of `engine_test.go`: `Doc` literals
  built with `para`/`txt`, compared as exact strings. One per node
  type, plus the escaping cases.
- Every topic renders as roff, alongside `TestEveryEmbeddedTopicRenders`
  (`help_test.go:81`), iterating `Topics()` and `Topic()`.
- `pk help document` contains every topic's H1, carries no `name:`
  frontmatter line, and separates pages with one rule.
- `document` is not an embedded topic name.
- `pk help --format yaml` is a usage error naming text and man;
  `pk status --format man` is a usage error; `pk status --format json`
  still works.

## Verification

- `make test` before and after each step, then `make build` and
  `make docs`, confirming `make docs` appends the Flags region to
  `skills/help/SKILL.md` and leaves the tree otherwise clean.
- Smoke, with output read:
  - `pk help document | head -40`, and `grep -c '^name:'` returns 0.
  - `pk help document --format man > /tmp/pk.1`, then `head -20`.
    Where groff exists, `man -l /tmp/pk.1` for a look. Not a test
    dependency.
  - `pk help ship --format man` renders one page.
  - Must fail: `pk status --format man` exits 1 as a usage error, and
    `pk help --format yaml` names text and man.

## Not in this change

- Installing a man page: no manpath write, no homebrew formula change,
  no packaging. `pk help document --format man > pk.1` is the whole
  delivery.
- `--format man` on any command other than help.
