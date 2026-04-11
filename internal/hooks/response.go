package hooks

import (
	"encoding/json"
	"io"
)

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
