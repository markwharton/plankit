package msg

import (
	"bytes"
	"testing"
)

func TestPrefixes(t *testing.T) {
	var b bytes.Buffer
	Errorf(&b, "e%d", 1)
	Warnf(&b, "w")
	Notef(&b, "n")
	Hintf(&b, "h")
	want := "Error: e1\nWarning: w\nNote: n\nHint: h\n"
	if b.String() != want {
		t.Fatalf("got %q want %q", b.String(), want)
	}
}
