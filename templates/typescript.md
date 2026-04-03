# CLAUDE.md — TypeScript Extensions

TypeScript/Node/Bun guidelines to add to your project's CLAUDE.md alongside the base template.
These can be added as sections in your CLAUDE.md or folded into Project Conventions.

## Package Management

- State which package manager the project uses (npm, pnpm, or bun) — don't mix.
- Always commit lock files when `package.json` changes.
- Bump `package.json` version ranges when code uses newer dependency features.

## TypeScript Configuration

- Strict mode enabled. No unused locals or parameters.
- ESM throughout: `"type": "module"` in `package.json`.
- Never use `Omit`, `Pick`, or mapped types for exported types — they're not emitted in `.d.ts`.
- Use explicit `interface` declarations for discoverability.

## Result Pattern for Fallible Operations

```typescript
interface Result<T> {
  ok: boolean;
  data?: T;
  error?: string;
  status?: number;
}
```

- Helpers: `ok(data)`, `err(message, status?)`.
- Unwrap at API boundary: check `result.ok`, translate to HTTP status.

## React Patterns

- React 18 Strict Mode: reset refs in `useEffect`, not in `useRef` initializer.
- Avoid over-engineering: don't wrap trivial ops in `useCallback`, don't track previous values manually, don't store derived state.

## Testing

- Run `npm test` (or equivalent) at session start to establish baseline.
- Co-located tests: `*.test.ts` alongside source files.
- CI matrix: test against Node.js 20 and 22.

## Git Rules

- Never amend commits — always create new commits.
- Never force push.
- No version bumps in feature commits — bumps happen at publish time.
