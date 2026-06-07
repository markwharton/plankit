package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRewriteRegion(t *testing.T) {
	// No start marker: unchanged, not found.
	if got, found, err := rewriteRegion("plain readme\n", 100); err != nil || found || got != "plain readme\n" {
		t.Errorf("no marker: got=%q found=%v err=%v", got, found, err)
	}
	// Start without end: error.
	if _, _, err := rewriteRegion("x\n"+markerStart+"\ny\n", 100); err == nil {
		t.Error("start without end should error")
	}
	// Well-formed region: replaced, found, surrounding text preserved.
	in := "intro\n" + markerStart + "\nstale\n" + markerEnd + "\nmore\n"
	got, found, err := rewriteRegion(in, 1234)
	if err != nil || !found {
		t.Fatalf("found=%v err=%v", found, err)
	}
	if strings.Contains(got, "stale") || !strings.Contains(got, "≈1,234 tokens") {
		t.Errorf("region not regenerated:\n%s", got)
	}
	if !strings.HasPrefix(got, "intro\n") || !strings.HasSuffix(got, "more\n") {
		t.Errorf("surrounding content not preserved:\n%s", got)
	}
}

func TestCollectAndRunRewritesReadme(t *testing.T) {
	repo := t.TempDir()
	write(t, filepath.Join(repo, "internal", "setup", "template", "CLAUDE.md"), "# CLAUDE\n\nrules\n")
	write(t, filepath.Join(repo, "internal", "setup", "rules", "git-discipline.md"), "# Git\n\n- commit\n")
	write(t, filepath.Join(repo, "internal", "setup", "skills", "ship", "SKILL.md"), "# Ship\n\non demand\n")
	readme := "# plankit\n\n" + markerStart + "\nold\n" + markerEnd + "\n\ntail\n"
	write(t, filepath.Join(repo, "README.md"), readme)

	always, skills, err := collect(repo)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if len(always) != 2 || len(skills) != 1 {
		t.Fatalf("always=%d skills=%d, want 2 and 1", len(always), len(skills))
	}

	if err := run(repo); err != nil {
		t.Fatalf("run: %v", err)
	}
	got := readFile(t, filepath.Join(repo, "README.md"))
	if strings.Contains(got, "old") || !strings.Contains(got, "Always-on rules footprint:") {
		t.Errorf("README not refreshed:\n%s", got)
	}
	// Idempotent.
	if err := run(repo); err != nil {
		t.Fatalf("second run: %v", err)
	}
	if got2 := readFile(t, filepath.Join(repo, "README.md")); got2 != got {
		t.Errorf("not idempotent")
	}
}

func TestCollectMissingSourceErrors(t *testing.T) {
	if _, _, err := collect(t.TempDir()); err == nil {
		t.Error("collect should error when internal/setup is absent")
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
