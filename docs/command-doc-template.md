# Command doc template

All `docs/pk-<command>.md` files follow a single template so readers can scan any command doc and know where to find what they need. The order mirrors Unix man page convention: synopsis, description, options, configuration, environment, notes.

## Canonical section order

1. `# pk <command>` — title
2. One-line summary — plain paragraph under the title (no heading)
3. `## Usage`
4. `## How it works`
5. `## Flags`
6. `## Configuration`
7. `## Hook protocol`
8. `## Environment`
9. `## Details` — command-specific deep dives, as `###` subsections

**Omit or keep — never rename.** If a command has no flags, drop the `## Flags` heading entirely. Don't write "No flags." Always use `## Configuration` — never `## .pk.json`.

## Section rules

### Title + summary

Title is `# pk <command>`. One-line summary follows as a plain paragraph — no "Description" heading.

### Usage

Bash code block showing common invocations. For hook-only commands (guard, protect), use prose to explain that the command runs as a hook and isn't called directly.

### How it works

Numbered list of steps in present tense. Describe what the command does end-to-end.

### Flags

Bullet list. Each entry: **--flag** — description. Omit the section if the command takes no flags.

### Configuration

Show the `.pk.json` schema in a JSON code block, then describe each field. Omit the section if the command reads no config.

### Hook protocol

Use this exact shape:

- **Input:** `<hook type>` JSON on stdin (what fields are read)
- **Output:** What the command writes to stdout, and when
- **Exit code:** Always 0 (or describe non-zero behavior if applicable)

Omit for commands that aren't hooks.

### Environment

Bullet list of environment variables the command reads. Each entry: **VAR_NAME** — what it's used for. Omit if none.

### Details

Container for command-specific deep dives that don't fit the standard sections. Each deep dive is an `###` subsection. Examples: workflow tables, edge-case behavior, related-concept explanations, error recovery notes. Place subsections in the order they're referenced from earlier sections.

Omit the `## Details` heading if the command has no deep-dive content.

## Why this order

Unix man pages use NAME → SYNOPSIS → DESCRIPTION → OPTIONS → CONFIG → ENVIRONMENT → NOTES. Developers reading `pk` docs already know this pattern from years of `man <anything>`. The order also matches the reader's progressive curiosity:

1. What is this? — title, summary
2. How do I run it? — usage
3. What does it do? — how it works
4. What knobs are there? — flags, configuration
5. How does it integrate? — hook protocol, environment
6. What else should I know? — details

## Template

Copy this skeleton when creating a new command doc:

````markdown
# pk <command>

<one-line summary>.

## Usage

```bash
pk <command>                 # common case
pk <command> --flag          # flag example
```

## How it works

1. Step one.
2. Step two.
3. Step three.

## Flags

- **--flag** — description.

## Configuration

Add a `<command>` key to `.pk.json`:

```json
{
  "<command>": {
    "field": "value"
  }
}
```

## Hook protocol

- **Input:** PreToolUse JSON on stdin.
- **Output:** `{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"..."}}` on stdout to block. No output to allow. `hookEventName` is required by the Claude Code hook schema whenever `hookSpecificOutput` is present.
- **Exit code:** Always 0.

## Environment

- **CLAUDE_PROJECT_DIR** — Used to resolve the project root.

## Details

### Topic one

Deep-dive content.
````
