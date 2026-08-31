---
description: Documentation says the fact - the four sentence kinds, one home per fact, a rule names its guard, numbers come from commands
kind: docs
paths: ["*.md", "**/*.md"]
---

# Documentation

Docs say the fact. Every sentence in a document is one of four things: a fact about this project a reader can check, a command with what it prints, a decision with its reason, or a limit. A sentence that is none of these is deleted, not restyled: introductions, transitions, restatements, encouragement, and explanations of concepts the reader can look up.

- **Fact.** Present tense, naming the file, command, or option. Twenty-five words or fewer.
- **Command.** What to run and what it prints. Every number in a document is reproducible by a command the same document names.
- **Decision.** The choice, the reason, and what would reopen it.
- **Limit.** What it does not do, stated as plainly as what it does.

**One home per fact.** A fact appears in one place; every other place links to it. Test: search for the sentence; a second copy becomes a link. The one permitted copy is a summary in a file that leaves the repository alone.

**A rule names its guard.** A convention states the invariant, the mechanism, and the check that catches a violation. A rule with no check is given one or says so plainly.

**Structure is the project's own.** This rule governs sentences, not which documentation files exist.
