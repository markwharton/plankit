# Fix Windows plan preservation in pk preserve

## Context

A Windows user on the latest `pk` reports plan preservation silently does nothing, even
though `pk preserve` is configured correctly (`auto` mode, PostToolUse → `pk preserve`).
A recent fix (`dd38207 fix: normalize backslash paths in plan preservation on Windows`) was
supposed to handle this, so something is still wrong.

We captured the **real** Windows ExitPlanMode PostToolUse payload (the shape was undocumented
in any public Claude Code docs). Ground truth:

```json
{
  "tool_name": "ExitPlanMode",
  "tool_input": {},
  "tool_response": {
    "plan": "# Mock Plan ...",
    "isAgent": false,
    "filePath": "C:\\Users\\jethro.wharton\\.claude\\plans\\make-a-mock-plan-tidy-candle.md"
  }
}
```

So definitively: `tool_response` is a **JSON object**, the plan path is in **`filePath`** (not
`planPath`/`plan_path`), with standard JSON-escaped backslashes, and `tool_input` is empty.

### Confirmed root cause

`extractPlanPath` (preserve.go:298) normalizes by running `strings.ReplaceAll(text, "\\", "/")`
on the **serialized** tool-response text, conflating JSON-escape backslashes with path separators:

- For a JSON **object** `tool_response`, `ToolResponseString()` (hooks/input.go:58) returns the
  **raw bytes**, which still contain `C:\\Users\\…` (two literal backslashes per separator).
  `ReplaceAll("\\","/")` turns each `\\` into `//`, producing
  `C://Users//…//.claude//plans//…md`. The regex (preserve.go:296) requires single-slash
  `.claude/plans/`, so it **never matches** → silent no-op. This is the reported failure.
- A JSON **string** `tool_response` works only by luck: `ToolResponseString()` unescapes it first
  (`C:\\Users` → `C:\Users`), so one ReplaceAll yields single slashes and the regex matches.

The existing object-form test (`preserve_test.go:1132`) uses a Unix path (single slashes), so it
passes and masked this Windows case.

## Decisions (from the user)

- Diagnose from the **real captured payload** (done), then fix the confirmed cause.
- **Keep fail-loud**: no mtime-based fallback (the one removed in `0c77097` stays removed).

## Fix

Make plan-path extraction **structure-aware** so JSON escaping is handled by `encoding/json`,
not by `ReplaceAll` on serialized bytes. Operate on the structured `json.RawMessage`, not the
flattened `ToolResponseString()`.

In `internal/preserve/preserve.go`:

1. Change `extractPlanPath` to take `tool_response` as `json.RawMessage` (pass `input.ToolResponse`
   at the call site, preserve.go:109, instead of `input.ToolResponseString()`):

   ```go
   func extractPlanPath(toolResponse json.RawMessage, homeDir func() (string, error)) string {
       if len(toolResponse) == 0 {
           return ""
       }
       // Newer harness: tool_response is an object carrying the plan file path.
       var obj struct {
           FilePath string `json:"filePath"`
       }
       if json.Unmarshal(toolResponse, &obj) == nil && obj.FilePath != "" {
           if p := matchPlanPath(obj.FilePath, homeDir); p != "" {
               return p
           }
       }
       // Legacy/string form: tool_response is a string that may contain the path.
       var s string
       if json.Unmarshal(toolResponse, &s) == nil {
           return matchPlanPath(s, homeDir)
       }
       return ""
   }
   ```

2. Extract the existing normalize-and-regex logic into a `matchPlanPath(text string, homeDir …)`
   helper (the current body of `extractPlanPath`: `ReplaceAll("\\","/")`, `~/` expansion,
   `planPathRegex.FindString`). Because `obj.FilePath` is already JSON-unescaped
   (`C:\Users\…` single backslashes), a single `ReplaceAll` yields correct single slashes and the
   regex matches. The `planPathRegex` guard stays, so only real `.claude/plans/*.md` paths preserve.

Why this is correct and minimal:
- Reads the authoritative `filePath` field the harness actually sends; robust against the raw-bytes
  double-slash bug at the root.
- The legacy string branch keeps older/string-form payloads working (no regression).
- Object without `filePath`, or any non-path text → returns `""` → existing fail-loud
  "tool_response did not contain a .claude/plans/*.md path" reason. No mtime fallback.

Forward-compatibility (no crash on harness schema drift): because both branches use
`json.Unmarshal` into a typed target (a struct, then a string) and swallow the errors, a future
harness change — `filePath` renamed, removed, extra fields added, or even a wrong-typed value —
cannot panic. A missing/renamed field zero-fills `obj.FilePath` to `""`; a malformed or
unexpected shape errors the unmarshal, which we ignore. Worst case is a graceful `""` →
fail-loud "no plan found", never a crash. `matchPlanPath` only ever receives a valid string.
- Downstream is unaffected: the returned forward-slash path (`C:/Users/…`) works with
  `os.Stat`/`os.ReadFile` on Windows (Go accepts forward slashes), and `Run` still reads the plan
  body from that file as today. `input.CWD` handling is unchanged.

### Critical files

- `internal/preserve/preserve.go` — `extractPlanPath` (298–309) → new signature + `matchPlanPath`
  helper; call site at line 109; `planPathRegex` (296) and `Run` flow (108–116) unchanged otherwise.
- `internal/preserve/preserve_test.go` — point the existing `TestExtractPlanPath` /
  `TestExtractPlanPath_windows` string cases at `matchPlanPath`; add a new `TestExtractPlanPath`
  covering the `json.RawMessage` dispatch (see Verification).
- `docs/pk-preserve.md` — the hook-protocol "Input" line currently says tool_response "includes the
  plan path"; note it may be an object with a `filePath` field (docs tight-loop).

## Verification

- **Automated:** `make test` (race) and `make lint`. New table cases for `extractPlanPath(json.RawMessage,…)`:
  - Object + Windows `filePath`: `{"filePath":"C:\\Users\\jethro\\.claude\\plans\\my-plan.md","plan":"# X"}`
    → `C:/Users/jethro/.claude/plans/my-plan.md` (the regression case).
  - Object + Unix `filePath` → forward-slash path unchanged.
  - String form `"Plan saved to /Users/mark/.claude/plans/p.md"` → still matches.
  - Object without `filePath` (e.g. `{"plan":"# X"}`) and non-path string → `""` (fail-loud).
  Keep the `matchPlanPath` string/Windows tests green.
- **Smoke (the real proof):** rebuild `pk` (`make build`), then feed Jethro's **captured** payload
  straight in: `pk preserve --dry-run < exitplan-payload.json` must now report the found plan
  (previously: "no plan found"). Then on Windows, approve a real plan and confirm `pk preserve`
  writes `docs/plans/<date>-NNN-<slug>.md` and commits it. Negative case: a payload with no plan
  path still no-ops cleanly (dry-run prints the "no path" reason, exit 0).
