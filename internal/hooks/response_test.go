package hooks

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestWriteBlockDecision(t *testing.T) {
	var buf bytes.Buffer
	WriteBlockDecision(&buf, "Branch is protected")

	// Verify it's valid JSON.
	var result map[string]string
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v (output: %q)", err, buf.String())
	}
	if result["decision"] != "block" {
		t.Errorf("decision = %q, want 'block'", result["decision"])
	}
	if result["reason"] != "Branch is protected" {
		t.Errorf("reason = %q, want 'Branch is protected'", result["reason"])
	}
}

func TestWriteBlockDecision_escapesQuotes(t *testing.T) {
	var buf bytes.Buffer
	WriteBlockDecision(&buf, `Branch "main" is protected`)

	var result map[string]string
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v (output: %q)", err, buf.String())
	}
	if result["reason"] != `Branch "main" is protected` {
		t.Errorf("reason = %q, want quotes preserved", result["reason"])
	}
}
