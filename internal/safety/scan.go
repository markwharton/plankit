// Package safety provides hidden-character scanning for managed text assets.
//
// Files that plankit ships into downstream repositories, and the rule files an
// AI agent reads every session, must carry no hidden or control characters that
// could smuggle instructions past a human reviewer (the "Trojan Source" class,
// CVE-2021-42574). ScanHidden is the shared policy: it is used both by the
// build-time guard over embedded assets and by `pk rules --lint` over a
// project's installed rules.
package safety

import (
	"fmt"
	"unicode"
	"unicode/utf8"
)

// ScanHidden reports hidden/control-character violations in s as
// "line:col detail" strings. It allows tab and newline, and CR only as part of
// a CRLF line ending; it rejects bare CR, all other control characters, Unicode
// format characters (Cf: zero-width, bidi overrides, word joiners), and invalid
// UTF-8. Pairing this with a repo's .gitattributes (eol=lf) keeps the CR rule
// from ever false-failing a contributor on Windows.
func ScanHidden(s string) []string {
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
