package setup

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func rulesetConfig() Config {
	return Config{
		ReadFile:  os.ReadFile,
		WriteFile: os.WriteFile,
		MkdirAll:  os.MkdirAll,
	}
}

func TestWriteRuleset_writesWhenAbsent(t *testing.T) {
	dir := t.TempDir()

	wrote, err := WriteRuleset(rulesetConfig(), dir)
	if err != nil {
		t.Fatalf("WriteRuleset() error = %v", err)
	}
	if !wrote {
		t.Error("WriteRuleset() = false, want true for a fresh project")
	}

	got, err := os.ReadFile(filepath.Join(dir, RulesetPath))
	if err != nil {
		t.Fatalf("ruleset not written: %v", err)
	}
	if !bytes.Equal(got, rulesetTemplate) {
		t.Error("written ruleset differs from the embedded template")
	}
}

func TestWriteRuleset_leavesExistingCopyAlone(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".github"), 0755); err != nil {
		t.Fatalf("mkdir .github: %v", err)
	}
	custom := []byte(`{"name":"my-own-policy"}`)
	if err := os.WriteFile(filepath.Join(dir, RulesetPath), custom, 0644); err != nil {
		t.Fatalf("seed ruleset: %v", err)
	}

	wrote, err := WriteRuleset(rulesetConfig(), dir)
	if err != nil {
		t.Fatalf("WriteRuleset() error = %v", err)
	}
	if wrote {
		t.Error("WriteRuleset() = true, want false when the project has its own copy")
	}

	got, err := os.ReadFile(filepath.Join(dir, RulesetPath))
	if err != nil {
		t.Fatalf("read ruleset: %v", err)
	}
	if !bytes.Equal(got, custom) {
		t.Errorf("overwrote the project's ruleset: got %s", got)
	}
}

func TestWriteRuleset_mkdirFailure(t *testing.T) {
	cfg := rulesetConfig()
	cfg.MkdirAll = func(string, os.FileMode) error { return fmt.Errorf("disk full") }

	wrote, err := WriteRuleset(cfg, t.TempDir())
	if err == nil {
		t.Fatal("WriteRuleset() succeeded, want a mkdir error")
	}
	if wrote {
		t.Error("WriteRuleset() = true despite the failure")
	}
	if !strings.Contains(err.Error(), "failed to create .github directory") {
		t.Errorf("error = %q, want it to name the failed step", err)
	}
}

func TestWriteRuleset_writeFailure(t *testing.T) {
	cfg := rulesetConfig()
	cfg.WriteFile = func(string, []byte, os.FileMode) error { return fmt.Errorf("read-only fs") }

	wrote, err := WriteRuleset(cfg, t.TempDir())
	if err == nil {
		t.Fatal("WriteRuleset() succeeded, want a write error")
	}
	if wrote {
		t.Error("WriteRuleset() = true despite the failure")
	}
	if !strings.Contains(err.Error(), "failed to write") {
		t.Errorf("error = %q, want it to name the failed step", err)
	}
}
