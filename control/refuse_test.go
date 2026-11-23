package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// refusals reads a file given as an object and returns every reason it
// is refused.
func refusals(t *testing.T, o obj, description bool) []string {
	t.Helper()
	data, err := json.Marshal(o)
	if err != nil {
		t.Fatal(err)
	}
	f, problems := parseFile("file.json", data, description)
	if len(problems) > 0 {
		return problems
	}
	_, problems = wellFormed(f)
	return problems
}

// Each change makes a file the checker refuses, and the reason names
// what is wrong. fields.md lists the first group; checks.md, "When a
// file is refused", lists the second.
func TestAFileTheFormatDoesNotAllowIsRefused(t *testing.T) {
	src := obj{"file": "lib/x.code", "function": "x"}
	cases := []struct {
		name        string
		description bool
		change      func(f obj)
		reason      string
	}{
		// fields.md
		{"a key the format does not list", false, func(f obj) { place(f, "shelf")["colour"] = "red" }, `"colour"`},
		{"a key in another case", false, func(f obj) { place(f, "shelf")["Keyed"] = true }, `spelled "keyed"`},
		{"a top-level key missing", false, func(f obj) { delete(f, "unsaid") }, `no "unsaid"`},
		{"a name missing", false, func(f obj) { delete(place(f, "shelf"), "name") }, `no "name"`},
		{"a value of the wrong type", false, func(f obj) { place(f, "shelf")["keyed"] = "yes" }, "not true or false"},
		{"a field written as false", false, func(f obj) { place(f, "parsed")["keyed"] = false }, "left out"},
		{"an empty list below the top level", false, func(f obj) { element(f, "audit")["reads"] = []any{} }, "empty list"},
		{"a value the table does not give", false, func(f obj) { writeTo(element(f, "add"), "shelf")["mode"] = "append" }, `"append"`},
		{"a keyed place with no onExisting", false, func(f obj) { delete(place(f, "shelf"), "onExisting") }, `no "onExisting"`},
		{"onExisting on a place that is not keyed", false, func(f obj) { place(f, "parsed")["onExisting"] = "replace" }, "not keyed"},
		{"both belongsTo and outside", false, func(f obj) { place(f, "parsed")["outside"] = true }, "exactly one"},
		{"neither belongsTo nor outside", false, func(f obj) { delete(place(f, "parsed"), "belongsTo") }, "exactly one"},
		{"a hold on a place whose store replaces", false, func(f obj) {
			place(f, "shelf")["hold"] = obj{"file": "a", "test": "b"}
		}, `only when its "onExisting" is "refuse"`},
		{"a source in a definition file", false, func(f obj) { element(f, "audit")["source"] = src }, "a definition file does not have it"},
		{"a hold in a description file", true, func(f obj) {
			element(f, "read")["rejects"].(obj)["hold"] = obj{"file": "a", "test": "b"}
		}, "a description file does not have it"},
		{"belongsTo in a description file", true, func(f obj) { place(f, "shelf")["belongsTo"] = "shelve" }, "a description file does not have it"},
		{"invalid in a description file", true, func(f obj) { place(f, "shelf")["invalid"] = "x" }, "a description file does not have it"},
		{"outside in a description file", true, func(f obj) { place(f, "request")["outside"] = true }, "a description file does not have it"},
		{"lasting in a description file", true, func(f obj) { place(f, "shelf")["lasting"] = true }, "a description file does not have it"},
		{"a rejects with no to", false, func(f obj) { delete(element(f, "audit")["rejects"].(obj), "to") }, `no "to"`},
		{"an ask on a check that only refuses", false, func(f obj) {
			checkOf(writeTo(element(f, "add"), "shelf"), "repeated")["then"] = "ask"
		}, `"ask"`},
		{"otherwise on an element that is not an or", false, func(f obj) { element(f, "notify")["otherwise"] = "log" }, "not an or"},
		{"a join of one child", false, func(f obj) {
			f["elements"] = add(f["elements"], obj{"name": "lone", "join": []any{"audit"}})
		}, "at least two"},

		// checks.md
		{"two places with one name", false, func(f obj) {
			f["places"] = add(f["places"], obj{"name": "shelf", "outside": true, "invalid": "x"})
		}, `two places are named "shelf"`},
		{"two elements with one name", false, func(f obj) {
			f["elements"] = add(f["elements"], obj{"name": "audit", "writes": []any{obj{"place": "summary", "mode": "write"}}})
		}, `two elements are named "audit"`},
		{"a place named parent", false, func(f obj) {
			f["places"] = add(f["places"], obj{"name": "parent", "outside": true, "invalid": "x"})
		}, `named "parent"`},
		{"two entries in unsaid with the same what", false, func(f obj) {
			f["unsaid"] = add(f["unsaid"], obj{"what": "a book is lent to one reader at a time"})
		}, `the same "what"`},
		{"system names no element", false, func(f obj) { f["system"] = "bookshop" }, `"system" names "bookshop"`},
		{"describes names no element", true, func(f obj) { f["describes"] = "bookshop" }, `"describes" names "bookshop"`},
		{"belongsTo names no element", false, func(f obj) { place(f, "summary")["belongsTo"] = "nobody" }, `belongs to "nobody"`},
		{"a child that is not listed", false, func(f obj) { element(f, "notify")["include"] = []any{"log", "post"} }, `the child "post"`},
		{"a read of a place that is not listed", false, func(f obj) {
			element(f, "audit")["reads"] = []any{obj{"place": "attic"}}
		}, `reads "attic"`},
		{"a to that is not a place", false, func(f obj) { element(f, "audit")["rejects"].(obj)["to"] = "nowhere" }, `is "nowhere"`},
		{"more than one shape", false, func(f obj) { element(f, "notify")["reads"] = []any{obj{"place": "parsed"}} }, "more than one shape"},
		{"the child of two elements", false, func(f obj) { element(f, "notify")["include"] = []any{"log", "mail", "read"} }, "child of two elements"},
		{"an element below itself", false, func(f obj) { element(f, "notify")["include"] = []any{"log", "mail", "library"} }, `"library" is below itself`},
		{"a leaf with no reads and no writes", false, func(f obj) {
			delete(element(f, "mail"), "reads")
			delete(element(f, "mail"), "writes")
		}, "no reads and no writes"},
		{"an element that is not below the top element", false, func(f obj) {
			f["elements"] = add(f["elements"], obj{"name": "stray", "writes": []any{obj{"place": "summary", "mode": "write"}}})
		}, `"stray" is not the top element and is not below it`},
		{"an element that is not below the described one", true, func(f obj) {
			f["elements"] = add(f["elements"], obj{"name": "stray", "source": src, "reads": []any{obj{"place": "parsed", "source": src}}})
		}, `"stray" is not the described element and is not below it`},
		{"steps on an element that is not the top", false, func(f obj) {
			n := element(f, "notify")
			n["steps"] = n["include"]
			delete(n, "include")
		}, `"steps" and is not the top element`},
		{"a top element with another shape", false, func(f obj) {
			top := element(f, "library")
			top["include"] = top["steps"]
			delete(top, "steps")
		}, `has no "steps"`},
		{"steps in a description file", true, func(f obj) {
			n := element(f, "notify")
			n["steps"] = n["include"]
			delete(n, "include")
		}, "only the top element of a definition file"},
		{"create for a place that is not keyed", false, func(f obj) { writeTo(element(f, "mail"), "outbox")["mode"] = "create" }, "not keyed"},
		{"write for a keyed place", false, func(f obj) { writeTo(element(f, "log"), "journal")["mode"] = "write" }, "is keyed"},
		{"a keyed write with no key", false, func(f obj) { delete(writeTo(element(f, "log"), "journal"), "key") }, `no "key"`},
		{"a to that the element does not write", false, func(f obj) { element(f, "audit")["rejects"].(obj)["to"] = "outbox" }, "neither it nor an element below it writes"},
		{"the top element passing to a parent", false, func(f obj) {
			element(f, "library")["rejects"] = obj{"reason": "x", "to": "parent"}
		}, "has no parent"},
		{"a place read twice", false, func(f obj) {
			a := element(f, "audit")
			a["reads"] = add(a["reads"], obj{"place": "shelf"})
		}, `reads "shelf" twice`},
		{"a place written twice", false, func(f obj) {
			a := element(f, "audit")
			a["writes"] = add(a["writes"], obj{"place": "summary", "mode": "write"})
		}, `writes "summary" twice`},
		{"two checks of one kind", false, func(f obj) {
			w := writeTo(element(f, "add"), "shelf")
			w["before"] = add(w["before"], obj{"check": "repeated", "then": "refuse"})
		}, `two "repeated" checks`},
		{"two branches with the same child", false, func(f obj) {
			s := element(f, "shelve")
			s["or"] = add(s["or"], obj{"when": "the book is lost", "do": "add"})
		}, "two branches with the same child"},
	}
	for _, tc := range cases {
		name := "library.json"
		if tc.description {
			name = "intake.json"
		}
		f := loadBaseline(t, name)
		tc.change(f)
		problems := refusals(t, f, tc.description)
		if len(problems) == 0 {
			t.Errorf("%s: the file is read, and it should be refused", tc.name)
			continue
		}
		if !strings.Contains(strings.Join(problems, "\n"), tc.reason) {
			t.Errorf("%s: no reason names %q:\n  %s", tc.name, tc.reason, strings.Join(problems, "\n  "))
		}
	}
}

// What JSON itself does not rule out is refused too: a key given twice,
// and anything after the object.
func TestAFileThatIsNotOneObjectIsRefused(t *testing.T) {
	for text, reason := range map[string]string{
		`{"system": "a", "system": "b", "places": [], "elements": [], "unsaid": []}`: "given twice",
		`{"system": "a", "places": [], "elements": [], "unsaid": []} {}`:             "after the object's end",
		`{"system": "a", "places": [], "elements": [], "unsaid": []}]`:               "after the object's end",
		`[]`:              "not one JSON object",
		`{"system": "a",`: "not valid JSON",
		`{"places": [], "elements": [], "unsaid": []}`:                    `no "system"`,
		`{"describes": "a", "places": [], "elements": [], "unsaid": []}`:  "a definition file was expected",
		`{"system": 3, "places": [], "elements": [], "unsaid": []}`:       "not a string",
		`{"system": "a", "places": {}, "elements": [], "unsaid": []}`:     "not a list",
		`{"system": "a", "places": [null], "elements": [], "unsaid": []}`: "not an object",
	} {
		_, problems := parseFile("file.json", []byte(text), false)
		if !strings.Contains(strings.Join(problems, "\n"), reason) {
			t.Errorf("%s:\n  want a reason naming %q, got %q", text, reason, problems)
		}
	}
}

// Every reason is reported, not only the first.
func TestEveryReasonForRefusingIsReported(t *testing.T) {
	f := loadBaseline(t, "library.json")
	place(f, "shelf")["colour"] = "red"
	place(f, "parsed")["keyed"] = false
	delete(element(f, "audit")["rejects"].(obj), "to")
	problems := strings.Join(refusals(t, f, false), "\n")
	for _, want := range []string{`"colour"`, "left out", `no "to"`} {
		if !strings.Contains(problems, want) {
			t.Errorf("the reasons do not name %q:\n%s", want, problems)
		}
	}
}

func TestAnAcceptancesFileIsReadStrictly(t *testing.T) {
	for text, reason := range map[string]string{
		`{"accepted": [{"finding": "held  a rejects", "reason": "by hand", "who": "me"}]}`: `"who"`,
		`{"accepted": [{"finding": "held  a rejects"}]}`:                                   `no "reason"`,
		`{"accepted": [{"finding": "held  a rejects", "reason": ""}]}`:                     `"reason" is empty`,
		`{"accepted": [{"finding": "nonsense  a rejects", "reason": "x"}]}`:                "does not begin with the name of a check",
		`{"accepted": [{"finding": "held a rejects", "reason": "x"}]}`:                     "does not begin with the name of a check",
		`{"accepted": [{"finding": "stale  held  a rejects", "reason": "x"}]}`:             "does not begin with the name of a check",
		`{"accepted": []} 1`: "after the object's end",
		`{}`:                 `no "accepted"`,
	} {
		_, problems := parseAcceptances([]byte(text), checkOrder)
		if !strings.Contains(strings.Join(problems, "\n"), reason) {
			t.Errorf("%s:\n  want a reason naming %q, got %q", text, reason, problems)
		}
	}
	if _, problems := parseAcceptances([]byte(`{"accepted": [{"finding": "held  a rejects", "reason": "by hand"}]}`), checkOrder); len(problems) > 0 {
		t.Errorf("a good acceptances file is refused: %q", problems)
	}
}
