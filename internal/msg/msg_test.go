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

func TestCatalog(t *testing.T) {
	Register("t.b", Def{Text: "b"})
	Register("t.a", Def{Text: "a", Hint: "fix a"})
	ids := IDs()
	if len(ids) < 2 || ids[0] > ids[1] {
		t.Fatalf("ids not sorted: %v", ids)
	}
	if d, ok := Lookup("t.a"); !ok || d.Hint != "fix a" {
		t.Fatalf("lookup: %v %v", d, ok)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("duplicate Register should panic")
		}
	}()
	Register("t.a", Def{})
}
