package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanHidden(t *testing.T) {
	clean := []byte("---\nname: x\ndescription: y\n---\n\n# X\n\n\tindented code is fine\n")
	if bad := scanHidden(clean); len(bad) != 0 {
		t.Fatalf("clean file flagged: %v", bad)
	}

	cases := []struct {
		name string
		data string
		want string
	}{
		{"zero width space", "a\u200bb", "U+200B"},
		{"rtl override", "safe \u202edangerous", "U+202E"},
		{"ansi escape", "x\x1b[1mbold", "U+001B"},
		{"carriage return", "line one\r\nline two", "U+000D"},
		{"bom", "\ufeffcontent", "U+FEFF"},
	}
	for _, tc := range cases {
		bad := scanHidden([]byte(tc.data))
		if len(bad) == 0 {
			t.Errorf("%s: not flagged", tc.name)
			continue
		}
		if !strings.Contains(bad[0], tc.want) {
			t.Errorf("%s: finding %q missing %s", tc.name, bad[0], tc.want)
		}
	}

	multi := scanHidden([]byte("a\u200bb\nc\u202ed"))
	if len(multi) != 2 || !strings.Contains(multi[1], "line 2") {
		t.Errorf("want two findings with line numbers, got %v", multi)
	}
}

// TestCompileAllLabelsItsOutput checks the generated directory carries
// its own do-not-edit README, so a reader landing there is told where
// the source is.
func TestCompileAllLabelsItsOutput(t *testing.T) {
	skills := t.TempDir()
	os.MkdirAll(filepath.Join(skills, "demo"), 0o755)
	os.WriteFile(filepath.Join(skills, "demo", "SKILL.md"), []byte("---\nname: demo\ndescription: d\n---\n\n# demo\n\nBody.\n"), 0o644)
	out := t.TempDir()
	if err := compileAll(skills, out); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(out, "README.md"))
	if err != nil || !strings.Contains(string(b), "Do not edit") || !strings.Contains(string(b), "skills/<topic>/SKILL.md") {
		t.Fatalf("generated marker missing or incomplete: %v %q", err, b)
	}
}
