package hooks

import (
	"encoding/json"
	"io"
)

// WritePostToolUse writes a PostToolUse hook response with a systemMessage
// (shown to the user) and an optional additionalContext (injected into
// Claude's next turn). When additionalContext is empty, only the
// systemMessage is emitted; hookSpecificOutput is omitted entirely.
//
// The Claude Code hook schema requires hookEventName whenever
// hookSpecificOutput is present, so this helper sets it to "PostToolUse"
// automatically — callers don't have to remember.
func WritePostToolUse(w io.Writer, systemMessage, additionalContext string) {
	resp := struct {
		SystemMessage      string `json:"systemMessage,omitempty"`
		HookSpecificOutput *struct {
			HookEventName     string `json:"hookEventName"`
			AdditionalContext string `json:"additionalContext,omitempty"`
		} `json:"hookSpecificOutput,omitempty"`
	}{
		SystemMessage: systemMessage,
	}
	if additionalContext != "" {
		resp.HookSpecificOutput = &struct {
			HookEventName     string `json:"hookEventName"`
			AdditionalContext string `json:"additionalContext,omitempty"`
		}{
			HookEventName:     "PostToolUse",
			AdditionalContext: additionalContext,
		}
	}
	if data, err := json.Marshal(resp); err == nil {
		w.Write(data)
	}
}

// PermissionDecision values accepted by the Claude Code hook schema inside
// hookSpecificOutput for PreToolUse events.
const (
	PermissionAllow = "allow"
	PermissionDeny  = "deny"
	PermissionAsk   = "ask"
)

// WritePermissionDecision writes a PreToolUse permission decision using the
// rich hookSpecificOutput schema. decision must be one of "allow", "deny",
// "ask", or "defer". The schema requires hookEventName whenever
// hookSpecificOutput is present.
func WritePermissionDecision(w io.Writer, decision, reason string) {
	resp := struct {
		HookSpecificOutput struct {
			HookEventName            string `json:"hookEventName"`
			PermissionDecision       string `json:"permissionDecision"`
			PermissionDecisionReason string `json:"permissionDecisionReason"`
		} `json:"hookSpecificOutput"`
	}{}
	resp.HookSpecificOutput.HookEventName = "PreToolUse"
	resp.HookSpecificOutput.PermissionDecision = decision
	resp.HookSpecificOutput.PermissionDecisionReason = reason
	if data, err := json.Marshal(resp); err == nil {
		w.Write(data)
	}
}
