package hooks

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestWritePermissionDecision_deny(t *testing.T) {
	var buf bytes.Buffer
	WritePermissionDecision(&buf, PermissionDeny, "docs/plans/ files are immutable")

	var result struct {
		HookSpecificOutput struct {
			HookEventName            string `json:"hookEventName"`
			PermissionDecision       string `json:"permissionDecision"`
			PermissionDecisionReason string `json:"permissionDecisionReason"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v (output: %q)", err, buf.String())
	}
	if got := result.HookSpecificOutput.HookEventName; got != "PreToolUse" {
		t.Errorf("hookEventName = %q, want PreToolUse", got)
	}
	if got := result.HookSpecificOutput.PermissionDecision; got != "deny" {
		t.Errorf("permissionDecision = %q, want deny", got)
	}
	if got := result.HookSpecificOutput.PermissionDecisionReason; got != "docs/plans/ files are immutable" {
		t.Errorf("permissionDecisionReason = %q, want full reason", got)
	}
}

func TestWritePermissionDecision_ask(t *testing.T) {
	var buf bytes.Buffer
	WritePermissionDecision(&buf, PermissionAsk, `main is "protected"`)

	var result struct {
		HookSpecificOutput struct {
			HookEventName            string `json:"hookEventName"`
			PermissionDecision       string `json:"permissionDecision"`
			PermissionDecisionReason string `json:"permissionDecisionReason"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v (output: %q)", err, buf.String())
	}
	if got := result.HookSpecificOutput.PermissionDecision; got != "ask" {
		t.Errorf("permissionDecision = %q, want ask", got)
	}
	if got := result.HookSpecificOutput.PermissionDecisionReason; got != `main is "protected"` {
		t.Errorf("permissionDecisionReason = %q, want quotes preserved", got)
	}
}
