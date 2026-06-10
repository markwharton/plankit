package msg

import (
	"bytes"
	"testing"
)

func TestForms(t *testing.T) {
	tests := []struct {
		name  string
		write func(w *bytes.Buffer)
		want  string
	}{
		{"Errorf", func(w *bytes.Buffer) { Errorf(w, "not a git repository") }, "Error: not a git repository\n"},
		{"Errorf with args", func(w *bytes.Buffer) { Errorf(w, "invalid %s mode %q", "guard", "loud") }, "Error: invalid guard mode \"loud\"\n"},
		{"Warnf", func(w *bytes.Buffer) { Warnf(w, "pk is not in your PATH") }, "Warning: pk is not in your PATH\n"},
		{"Notef", func(w *bytes.Buffer) { Notef(w, "update available: %s", "v1.0.0") }, "Note: update available: v1.0.0\n"},
		{"Hookf", func(w *bytes.Buffer) { Hookf(w, "guard", "failed to read input: %v", "EOF") }, "pk guard: failed to read input: EOF\n"},
		{"Section", func(w *bytes.Buffer) { Section(w, "Skills") }, "Skills:\n"},
		{"Section with context", func(w *bytes.Buffer) { Section(w, "Settings (.claude/settings.json)") }, "Settings (.claude/settings.json):\n"},
		{"Itemf", func(w *bytes.Buffer) { Itemf(w, "Clean working tree") }, "  Clean working tree\n"},
		{"Hintf", func(w *bytes.Buffer) { Hintf(w, "To anchor at v0.0.0: pk setup --baseline") }, "  To anchor at v0.0.0: pk setup --baseline\n"},
		{"Or", func(w *bytes.Buffer) { Or(w, "git tag v0.0.0 && git push origin v0.0.0") }, "  or: git tag v0.0.0 && git push origin v0.0.0\n"},
		{"Banner", func(w *bytes.Buffer) { Banner(w, "Release v1.2.3") }, "=== Release v1.2.3 ===\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			tt.write(&buf)
			if got := buf.String(); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// TestLiteralPercent guards the prefix-concatenation implementation: a literal
// % in a caller's format string must not be misinterpreted.
func TestLiteralPercent(t *testing.T) {
	var buf bytes.Buffer
	Errorf(&buf, "coverage below 80%%")
	if got, want := buf.String(), "Error: coverage below 80%\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
