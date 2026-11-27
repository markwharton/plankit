package main

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// designs is every design under lab/: a JSON file with "parts" and "runs".
func designs(t *testing.T) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir("..", func(path string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if e.IsDir() && e.Name() == "hand-test" {
			return fs.SkipDir // records of the language's earlier form, which the engine no longer reads
		}
		if e.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}
		var top map[string]json.RawMessage
		if data, err := os.ReadFile(path); err == nil && json.Unmarshal(data, &top) == nil && top["parts"] != nil && top["runs"] != nil {
			out = append(out, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) < 16 {
		t.Fatalf("found %d designs under lab/, expected at least 16", len(out))
	}
	return out
}

func load(t *testing.T, path string) *Obj {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := Decode(data)
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return doc
}

// plain is a document as encoding/json reads it, with what the text form does not keep made the same:
// a part listed in "parts" carries its key as its name, an empty list at the top is left out, and a
// list of one in "at", "tells" or "says" is that one, which is how the engine reads it.
func plain(t *testing.T, doc *Obj) any {
	t.Helper()
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	var v map[string]any
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatal(err)
	}
	for k, p := range v["parts"].(map[string]any) {
		part := p.(map[string]any)
		if _, isAction := part["do"]; !isAction {
			part["name"] = k
		}
	}
	for _, k := range []string{"places", "facts", "expects", "rules"} {
		if l, ok := v[k].([]any); ok && len(l) == 0 {
			delete(v, k)
		}
	}
	single(v)
	return v
}

func single(x any) {
	switch o := x.(type) {
	case map[string]any:
		for k, e := range o {
			if l, ok := e.([]any); ok && len(l) == 1 && (k == "at" || k == "tells" || k == "says") {
				o[k] = l[0]
			}
			single(o[k])
		}
	case []any:
		for _, e := range o {
			single(e)
		}
	}
}

func TestEveryDesignGoesThroughBothFormsAndBack(t *testing.T) {
	for _, path := range designs(t) {
		doc := load(t, path)
		for _, form := range []Form{A, B} {
			other := B
			if form == B {
				other = A
			}
			text, err := Print(form, doc)
			if err != nil {
				t.Fatalf("%s, form %s: %v", path, form, err)
			}
			back, err := Read(form, path+".txt", text)
			if err != nil {
				t.Fatalf("%s, form %s: %v", path, form, err)
			}
			if !reflect.DeepEqual(plain(t, doc), plain(t, back)) {
				t.Errorf("%s, form %s: read back, the design differs", path, form)
				continue
			}
			again, err := Print(form, back)
			if err != nil || again != text {
				t.Errorf("%s, form %s: printed again, the text differs (%v)", path, form, err)
			}
			crossed, err := Print(other, back)
			if err != nil {
				t.Fatalf("%s, form %s: %v", path, other, err)
			}
			if there, err := Read(other, path+".txt", crossed); err != nil || !reflect.DeepEqual(plain(t, doc), plain(t, there)) {
				t.Errorf("%s: form %s printed as %s and read back differs (%v)", path, form, other, err)
			}
		}
	}
}

func TestWhatTheTextRefuses(t *testing.T) {
	cases := []struct {
		form Form
		text string
		want string
	}{
		{A, "SYSTEM x\nRUNS r\n\nFACT WHEN it rains\n", `d.txt:4: a name was expected after FACT, and WHEN is a keyword`},
		{B, "system x\nruns [r]\n\nfact [it rains\n", `d.txt:4: a name opened with "[" is not closed`},
		{A, "SYSTEM x\nRUNS r\n        PLACE p ONE OUTSIDE\n", `d.txt:3: this line is indented under nothing`},
		{A, "SYSTEM x\nRUNS r\nPLACE p ONE\tOUTSIDE\n", `d.txt:3: a tab, where only spaces may be`},
		{A, "SYSTEM x\nRUNS r\nSEQ r\n    a WHEN (b AND c\n", `d.txt:4: a bracket opened in the condition is not closed`},
		{A, "SYSTEM x\nRUNS r\nSEQ r\n    a WHEN b AND\n", `d.txt:4: the condition ends where a fact was expected`},
		{A, "SYSTEM x\nRUNS r\nEXPECT e\n    FROM p\n", `d.txt:4: an expectation does not take FROM`},
		{B, "system x\nruns [r]\nplace [p] one outside\n    hold\n", `d.txt:4: hold names a file and a test`},
	}
	for _, c := range cases {
		_, err := Read(c.form, "d.txt", c.text)
		if err == nil || err.Error() != c.want {
			t.Errorf("%q, form %s:\n got %v\nwant %s", c.text, c.form, err, c.want)
		}
	}
}

func TestANameThatFormACannotHoldIsRefusedWhenPrinted(t *testing.T) {
	doc, err := Decode([]byte(`{"system": "x", "places": [], "facts": [{"name": "WHEN it rains"}], "parts": {"r": {"do": "r"}}, "runs": ["r"]}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Print(A, doc); err == nil || err.Error() != `the name "WHEN it rains" holds the keyword WHEN: form A cannot tell it from the keyword; write it in lower case` {
		t.Errorf("got %v", err)
	}
	if _, err := Print(B, doc); err != nil {
		t.Errorf("form B can hold the name: %v", err)
	}
}

func TestEveryKeywordIsReadAsOneInBothForms(t *testing.T) {
	for _, w := range keywords {
		for _, form := range []Form{A, B} {
			toks, err := lex(form, form.kw(w))
			if err != nil || len(toks) != 1 || toks[0].kind != tKeyword || toks[0].text != w {
				t.Errorf("form %s: %q lexes as %v (%v)", form, form.kw(w), toks, err)
			}
		}
	}
}
