package jsonsplice

import (
	"strings"
	"testing"
)

// The developer's file, with spacing no encoder would produce. Every
// byte outside the edited region must come back as it went in.
const odd = "{\n    \"name\":   \"demo\",\n    \"hooks\":  {\"PreToolUse\": [{\"matcher\": \"Bash\", \"extra\": true}]},\n    \"permissions\": {\n        \"allow\": [\"Bash(pk:*)\"]\n    },\n    \"version\": \"0.0.0\"\n}\n"

func TestGet(t *testing.T) {
	raw, ok, err := Get([]byte(odd), "permissions", "allow")
	if err != nil || !ok || string(raw) != `["Bash(pk:*)"]` {
		t.Fatalf("got %q ok=%v err=%v", raw, ok, err)
	}
	if _, ok, err := Get([]byte(odd), "permissions", "deny"); err != nil || ok {
		t.Fatalf("absent key: ok=%v err=%v", ok, err)
	}
	if _, ok, err := Get([]byte(odd), "missing", "child"); err != nil || ok {
		t.Fatalf("missing parent: ok=%v err=%v", ok, err)
	}
	if _, _, err := Get([]byte(odd), "name", "child"); err == nil || !strings.Contains(err.Error(), "name: expected a JSON object") {
		t.Fatalf("a scalar parent must be named: %v", err)
	}
}

func TestReplaceKeepsEveryOtherByte(t *testing.T) {
	got, err := Replace([]byte(odd), "1.2.3", "version")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(odd, `"0.0.0"`, `"1.2.3"`, 1)
	if string(got) != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
	if _, err := Replace([]byte(odd), "x", "absent"); err == nil || !strings.Contains(err.Error(), "absent: not present") {
		t.Fatalf("replace of an absent key: %v", err)
	}
}

func TestInsertIntoMultilineObject(t *testing.T) {
	got, err := Insert([]byte(odd), map[string]bool{"plankit@plankit": true}, "enabledPlugins")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(odd, "\"version\": \"0.0.0\"\n", "\"version\": \"0.0.0\",\n    \"enabledPlugins\": {\n        \"plankit@plankit\": true\n    }\n", 1)
	if string(got) != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
	// One level down, the sibling's indentation and separator are copied.
	got, err = Insert([]byte(odd), []string{"Bash(rm:*)"}, "permissions", "deny")
	if err != nil {
		t.Fatal(err)
	}
	want = strings.Replace(odd, "\"allow\": [\"Bash(pk:*)\"]\n", "\"allow\": [\"Bash(pk:*)\"],\n        \"deny\": [\n            \"Bash(rm:*)\"\n        ]\n", 1)
	if string(got) != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
}

func TestInsertIntoInlineObject(t *testing.T) {
	got, err := Insert([]byte(odd), true, "hooks", "Stop")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(odd, `"extra": true}]}`, `"extra": true}],"Stop": true}`, 1)
	if string(got) != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
	spaced := `{ "a": 1 }`
	got, err = Insert([]byte(spaced), "x", "b")
	if err != nil || string(got) != `{ "a": 1, "b": "x" }` {
		t.Fatalf("got %q err=%v", got, err)
	}
	tight := `{"a":1}`
	got, err = Insert([]byte(tight), 2, "b")
	if err != nil || string(got) != `{"a":1,"b":2}` {
		t.Fatalf("got %q err=%v", got, err)
	}
}

func TestInsertIntoEmptyObject(t *testing.T) {
	got, err := Insert([]byte("{}"), map[string]bool{"k": true}, "top")
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"top\": {\n    \"k\": true\n  }\n}"
	if string(got) != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
	// A nested empty object opens out at its line's indentation, and the
	// document's own unit (a tab here) is used.
	doc := "{\n\t\"outer\": {}\n}\n"
	got, err = Insert([]byte(doc), 1, "outer", "k")
	if err != nil {
		t.Fatal(err)
	}
	want = "{\n\t\"outer\": {\n\t\t\"k\": 1\n\t}\n}\n"
	if string(got) != want {
		t.Fatalf("got\n%q\nwant\n%q", got, want)
	}
}

func TestInsertCreatesMissingParents(t *testing.T) {
	got, err := Insert([]byte("{}"), true, "enabledPlugins", "plankit@plankit")
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"enabledPlugins\": {\n    \"plankit@plankit\": true\n  }\n}"
	if string(got) != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
}

func TestInsertRefusesPresentKey(t *testing.T) {
	if _, err := Insert([]byte(odd), 1, "version"); err == nil || !strings.Contains(err.Error(), "version: already present") {
		t.Fatalf("got %v", err)
	}
}

func TestValuesAreNotHTMLEscaped(t *testing.T) {
	got, err := Insert([]byte("{}"), "a && b <c>", "cmd")
	if err != nil || !strings.Contains(string(got), `"a && b <c>"`) {
		t.Fatalf("got %q err=%v", got, err)
	}
}

func TestMalformedDocumentIsAnError(t *testing.T) {
	for _, doc := range []string{"", "{", "[]", "null", `{"a": }`} {
		if _, _, err := Get([]byte(doc), "a"); err == nil {
			t.Errorf("%q: no error", doc)
		}
		if _, err := Insert([]byte(doc), 1, "b"); err == nil {
			t.Errorf("%q: insert did not fail", doc)
		}
	}
}
