package setup

import (
	"fmt"
	"io/fs"
	"testing"
	"unicode"
	"unicode/utf8"
)

// embeddedTexts returns every managed text asset that pk setup ships into
// downstream repositories, keyed by display path. These files are read by AI
// agents in every Claude Code session, so they must carry no hidden or control
// characters that could smuggle instructions past a human reviewer (the
// "Trojan Source" class, CVE-2021-42574). The scan covers exactly what ships:
// the embedded bytes, not the working-tree source.
func embeddedTexts(t *testing.T) map[string]string {
	t.Helper()
	texts := map[string]string{
		"internal/setup/template/install-pk.sh": installScriptTemplate,
	}
	for _, fsys := range []fs.FS{templateFS, rulesFS, skillsFS} {
		err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			b, err := fs.ReadFile(fsys, path)
			if err != nil {
				return err
			}
			texts["internal/setup/"+path] = string(b)
			return nil
		})
		if err != nil {
			t.Fatalf("walk embedded FS: %v", err)
		}
	}
	return texts
}

// scanHidden reports hidden/control-character violations in s as
// "line:col detail" strings. It allows tab and newline, and CR only as part of
// a CRLF line ending; it rejects bare CR, all other control characters,
// Unicode format characters (Cf: zero-width, bidi overrides, word joiners),
// and invalid UTF-8. Pairing this with the repo's .gitattributes (eol=lf) keeps
// the CR rule from ever false-failing a contributor on Windows.
func scanHidden(s string) []string {
	var violations []string
	line, col := 1, 0
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		col++
		switch {
		case r == '\n':
			line++
			col = 0
		case r == '\t':
			// allowed
		case r == '\r':
			// allowed only immediately before '\n' (a normal CRLF line ending)
			if !(i+size < len(s) && s[i+size] == '\n') {
				violations = append(violations, fmt.Sprintf("%d:%d bare CR (U+000D)", line, col))
			}
		case r == utf8.RuneError && size == 1:
			violations = append(violations, fmt.Sprintf("%d:%d invalid UTF-8 byte 0x%02X", line, col, s[i]))
		case unicode.IsControl(r):
			violations = append(violations, fmt.Sprintf("%d:%d control U+%04X", line, col, r))
		case unicode.In(r, unicode.Cf):
			violations = append(violations, fmt.Sprintf("%d:%d hidden/format U+%04X", line, col, r))
		}
		i += size
	}
	return violations
}

// TestEmbeddedManagedFilesHaveNoHiddenCharacters is the build-time guard
// against a Trojan-source contribution to a file pk distributes downstream.
func TestEmbeddedManagedFilesHaveNoHiddenCharacters(t *testing.T) {
	texts := embeddedTexts(t)
	if len(texts) == 0 {
		t.Fatal("no embedded managed files found to scan")
	}
	for path, content := range texts {
		if v := scanHidden(content); len(v) > 0 {
			t.Errorf("%s contains hidden/control characters: %v", path, v)
		}
	}
}

// TestScanHidden locks the policy independently of the embedded fixtures, so
// the guard keeps catching the dangerous cases even if every managed file is
// clean. The hidden runes are built from code points (not pasted) so this
// source file stays pure ASCII.
func TestScanHidden(t *testing.T) {
	var (
		zwsp = string(rune(0x200B)) // zero-width space
		bidi = string(rune(0x202E)) // right-to-left override
		bom  = string(rune(0xFEFF)) // byte order mark / zero-width no-break space
	)
	tests := []struct {
		name    string
		input   string
		wantBad bool
	}{
		{"plain text", "hello world\n", false},
		{"tabs and lf", "a\tb\nc\n", false},
		{"crlf line endings", "a\r\nb\r\n", false},
		{"no trailing newline", "no newline", false},
		{"ansi escape", "click \x1b[2J here", true},
		{"zero-width space", "legit" + zwsp + "word", true},
		{"bidi override", "amount" + bidi + "000", true},
		{"byte order mark", bom + "title", true},
		{"bare cr", "old\rmac", true},
		{"del control", "x\x7fy", true},
		{"invalid utf-8", "bad\xffbyte", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scanHidden(tt.input)
			if (len(got) > 0) != tt.wantBad {
				t.Errorf("scanHidden(%q) = %v, wantBad=%v", tt.input, got, tt.wantBad)
			}
		})
	}
}
