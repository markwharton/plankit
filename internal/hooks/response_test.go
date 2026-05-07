package hooks

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
)

func TestWritePermissionDecision_deny(t *testing.T) {
	var buf bytes.Buffer
	if err := WritePermissionDecision(&buf, PermissionDeny, "docs/plans/ files are immutable"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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
	if err := WritePermissionDecision(&buf, PermissionAsk, `main is "protected"`); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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

func TestWritePostToolUse_systemMessageOnly(t *testing.T) {
	var buf bytes.Buffer
	if err := WritePostToolUse(&buf, "Plan committed", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result struct {
		SystemMessage      string           `json:"systemMessage"`
		HookSpecificOutput *json.RawMessage `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v (output: %q)", err, buf.String())
	}
	if result.SystemMessage != "Plan committed" {
		t.Errorf("systemMessage = %q, want %q", result.SystemMessage, "Plan committed")
	}
	if result.HookSpecificOutput != nil {
		t.Errorf("hookSpecificOutput should be absent when additionalContext is empty")
	}
}

func TestWritePostToolUse_withAdditionalContext(t *testing.T) {
	var buf bytes.Buffer
	if err := WritePostToolUse(&buf, "Plan ready", "The plan has been approved"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result struct {
		SystemMessage      string `json:"systemMessage"`
		HookSpecificOutput struct {
			HookEventName     string `json:"hookEventName"`
			AdditionalContext string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v (output: %q)", err, buf.String())
	}
	if result.SystemMessage != "Plan ready" {
		t.Errorf("systemMessage = %q, want %q", result.SystemMessage, "Plan ready")
	}
	if result.HookSpecificOutput.HookEventName != "PostToolUse" {
		t.Errorf("hookEventName = %q, want PostToolUse", result.HookSpecificOutput.HookEventName)
	}
	if result.HookSpecificOutput.AdditionalContext != "The plan has been approved" {
		t.Errorf("additionalContext = %q, want %q", result.HookSpecificOutput.AdditionalContext, "The plan has been approved")
	}
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestWritePostToolUse_writeError(t *testing.T) {
	err := WritePostToolUse(failWriter{}, "msg", "")
	if err == nil {
		t.Fatal("expected error from failing writer")
	}
	if err.Error() != "write failed" {
		t.Errorf("error = %q, want %q", err.Error(), "write failed")
	}
}

func TestWritePermissionDecision_writeError(t *testing.T) {
	err := WritePermissionDecision(failWriter{}, PermissionDeny, "reason")
	if err == nil {
		t.Fatal("expected error from failing writer")
	}
	if err.Error() != "write failed" {
		t.Errorf("error = %q, want %q", err.Error(), "write failed")
	}
}
