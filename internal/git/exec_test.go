package git

import (
	"os"
	"testing"
)

func TestExec_noDir(t *testing.T) {
	// Run from cwd — git version always works.
	out, err := Exec("", "version")
	if err != nil {
		t.Fatalf("Exec('', 'version') error: %v", err)
	}
	if out == "" {
		t.Fatal("Exec('', 'version') returned empty output")
	}
}

func TestExec_withDir(t *testing.T) {
	dir := t.TempDir()
	// Init a repo so rev-parse works.
	if _, err := Exec(dir, "init"); err != nil {
		t.Fatalf("git init error: %v", err)
	}
	out, err := Exec(dir, "rev-parse", "--is-inside-work-tree")
	if err != nil {
		t.Fatalf("Exec(dir, rev-parse) error: %v", err)
	}
	if out != "true" {
		t.Errorf("Exec(dir, rev-parse) = %q, want 'true'", out)
	}
}

func TestExec_trimOutput(t *testing.T) {
	// git version output normally ends with newline — Exec should trim it.
	out, err := Exec("", "version")
	if err != nil {
		t.Fatal(err)
	}
	if out == "" {
		t.Fatal("expected non-empty output")
	}
	if out[len(out)-1] == '\n' {
		t.Errorf("output should be trimmed, got %q", out)
	}
}

func TestExec_invalidDir(t *testing.T) {
	_, err := Exec("/nonexistent-dir-"+os.DevNull, "status")
	if err == nil {
		t.Error("expected error for invalid directory")
	}
}
