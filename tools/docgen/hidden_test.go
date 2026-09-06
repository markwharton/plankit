package main

import (
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
