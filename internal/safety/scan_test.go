package safety

import "testing"

// TestScanHidden locks the policy independently of any caller's fixtures, so the
// guard keeps catching the dangerous cases even when every scanned file is clean.
// The hidden runes are built from code points (not pasted) so this source file
// stays pure ASCII.
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
			got := ScanHidden(tt.input)
			if (len(got) > 0) != tt.wantBad {
				t.Errorf("ScanHidden(%q) = %v, wantBad=%v", tt.input, got, tt.wantBad)
			}
		})
	}
}
