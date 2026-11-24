package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// The tests build on one baseline: a definition file and two
// description files in testdata/baseline that use every shape and every
// field, and pass every check. Each break changes the baseline in one
// way and names the findings that must follow, by check and subject.

type obj = map[string]any

func loadBaseline(t *testing.T, name string) obj {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "baseline", name))
	if err != nil {
		t.Fatal(err)
	}
	var o obj
	if err := json.Unmarshal(data, &o); err != nil {
		t.Fatal(err)
	}
	return o
}

// named returns the place or element of that name in a file.
func named(f obj, list, name string) obj {
	for _, x := range f[list].([]any) {
		if o := x.(obj); o["name"] == name {
			return o
		}
	}
	panic("the baseline has no " + list + " named " + name)
}

func element(f obj, name string) obj { return named(f, "elements", name) }
func place(f obj, name string) obj   { return named(f, "places", name) }

// writeTo returns the element's write to a place.
func writeTo(e obj, placeName string) obj {
	for _, x := range e["writes"].([]any) {
		if w := x.(obj); w["place"] == placeName {
			return w
		}
	}
	panic("no write to " + placeName)
}

// checkOf returns a write's check of one kind.
func checkOf(w obj, kind string) obj {
	for _, x := range w["before"].([]any) {
		if c := x.(obj); c["check"] == kind {
			return c
		}
	}
	panic("no check " + kind)
}

// without returns a list with the items matching drop taken out.
func without(list any, drop func(obj) bool) []any {
	var out []any
	for _, x := range list.([]any) {
		if !drop(x.(obj)) {
			out = append(out, x)
		}
	}
	return out
}

func add(list any, items ...any) []any {
	l, _ := list.([]any)
	return append(append([]any{}, l...), items...)
}

// evidence writes, under root, a file for every hold and source the
// files name, each containing the names given for it.
func evidence(t *testing.T, root string, files ...obj) {
	t.Helper()
	names := map[string][]string{}
	var walk func(v any)
	walk = func(v any) {
		switch x := v.(type) {
		case obj:
			for k, val := range x {
				if ev, ok := val.(obj); ok && (k == "hold" || k == "source") {
					file, _ := ev["file"].(string)
					name, _ := ev["test"].(string)
					if k == "source" {
						name, _ = ev["function"].(string)
					}
					names[file] = append(names[file], name)
					continue
				}
				walk(val)
			}
		case []any:
			for _, item := range x {
				walk(item)
			}
		}
	}
	for _, f := range files {
		walk(f)
	}
	for file, list := range names {
		path := filepath.Join(root, filepath.FromSlash(file))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(strings.Join(list, "\n")+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// modelOf reads a file given as an object and fails the test when the
// checker refuses it.
func modelOf(t *testing.T, o obj, description bool) *model {
	t.Helper()
	data, err := json.Marshal(o)
	if err != nil {
		t.Fatal(err)
	}
	f, problems := parseFile("file.json", data, description)
	if len(problems) == 0 {
		var m *model
		if m, problems = wellFormed(f); len(problems) == 0 {
			return m
		}
	}
	t.Fatalf("the file is refused, and the test needs it read:\n  %s", strings.Join(problems, "\n  "))
	return nil
}

// lines returns findings as "check  subject", sorted.
func lines(c *collector) []string {
	var out []string
	for _, f := range c.list {
		out = append(out, f.Check+"  "+f.Subject)
	}
	sort.Strings(out)
	return out
}

func same(t *testing.T, name string, got, want []string) {
	t.Helper()
	sort.Strings(want)
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("%s:\n got  %q\n want %q", name, got, want)
	}
}

// writeFiles writes named JSON files into dir and returns their paths.
func writeFiles(t *testing.T, dir string, files map[string]obj) {
	t.Helper()
	for name, o := range files {
		data, err := json.MarshalIndent(o, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// The baseline passes every check: a whole run, with every hold and
// source in place, has no finding.
func TestBaselinePassesEveryCheck(t *testing.T) {
	def, intake, audit := loadBaseline(t, "library.json"), loadBaseline(t, "intake.json"), loadBaseline(t, "audit.json")
	dir := t.TempDir()
	writeFiles(t, dir, map[string]obj{"library.json": def, "intake.json": intake, "audit.json": audit})
	evidence(t, dir, def, intake, audit)
	run, refused := checkFiles(filepath.Join(dir, "library.json"),
		[]string{filepath.Join(dir, "intake.json"), filepath.Join(dir, "audit.json")}, dir)
	if len(refused) > 0 {
		t.Fatalf("the baseline is refused:\n  %s", strings.Join(refused, "\n  "))
	}
	for _, f := range run.findings {
		t.Errorf("the baseline has a finding: %s  %s: %s", f.Check, f.Subject, f.Message)
	}
}

type soundBreak struct {
	name   string
	change func(def obj)
	want   []string
}

var newWrite = func(placeName, mode string) obj { return obj{"place": placeName, "mode": mode} }

// soundBreaks breaks the baseline's design one way at a time. Each row
// of the table of checks in checks.md has at least one break here.
var soundBreaks = []soundBreak{
	{"a leaf writes a place that belongs to another element", func(d obj) {
		place(d, "parsed")["belongsTo"] = "mail"
	}, []string{"one-owner  read writes parsed"}},
	{"a leaf writes a place from outside the system", func(d obj) {
		audit := element(d, "audit")
		audit["writes"] = add(audit["writes"], newWrite("request", "write"))
	}, []string{"one-owner  audit writes request"}},
	{"two leaves under a join write one place", func(d obj) {
		place(d, "journal")["belongsTo"] = "intake"
		read := element(d, "read")
		read["writes"] = add(read["writes"], obj{"place": "journal", "mode": "create", "key": "assigned"})
	}, []string{"one-owner  place journal"}},
	{"two leaves under an include write one place", func(d obj) {
		place(d, "journal")["belongsTo"] = "notify"
		mail := element(d, "mail")
		mail["writes"] = add(mail["writes"], obj{"place": "journal", "mode": "create", "key": "assigned"})
	}, []string{"one-owner  place journal", "include-independent  element notify"}},
	{"a leaf writes a place it reads", func(d obj) {
		audit := element(d, "audit")
		audit["reads"] = add(audit["reads"], obj{"place": "summary"})
	}, []string{"reads-for-reference  audit writes summary"}},
	{"a create into a store that replaces has no exists check", func(d obj) {
		a := element(d, "add")
		a["reads"] = without(a["reads"], func(r obj) bool { return r["place"] == "shelf" })
		w := writeTo(a, "shelf")
		w["before"] = without(w["before"], func(c obj) bool { return c["check"] == "exists" })
	}, []string{"create-never-replaces  add writes shelf"}},
	{"an exists check without reading the place", func(d obj) {
		a := element(d, "add")
		a["reads"] = without(a["reads"], func(r obj) bool { return r["place"] == "shelf" })
	}, []string{"create-never-replaces  add writes shelf, check exists"}},
	{"a replace with no belongs check", func(d obj) {
		delete(writeTo(element(d, "update"), "shelf"), "before")
	}, []string{"replace-checks-owner  update writes shelf"}},
	{"a key from content with no repeated check", func(d obj) {
		w := writeTo(element(d, "add"), "shelf")
		w["before"] = without(w["before"], func(c obj) bool { return c["check"] == "repeated" })
	}, []string{"key-from-content  add writes shelf"}},
	{"a leaf that reads has no rejects", func(d obj) {
		delete(element(d, "audit"), "rejects")
	}, []string{"rejects-declared  audit rejects"}},
	{"a refusal passed to a parent with no rejects", func(d obj) {
		delete(element(d, "notify"), "rejects")
	}, []string{"rejects-declared  notify rejects"}},
	{"a leaf that asks has no declines", func(d obj) {
		delete(element(d, "add"), "declines")
	}, []string{"declines-declared  add declines"}},
	{"an answer passed to a parent with no declines", func(d obj) {
		delete(element(d, "shelve"), "declines")
	}, []string{"declines-declared  shelve declines"}},
	{"a place with no invalid", func(d obj) {
		delete(place(d, "summary"), "invalid")
	}, []string{"reject-values  place summary"}},
	{"an included child reads what its sibling writes", func(d obj) {
		mail := element(d, "mail")
		mail["reads"] = add(mail["reads"], obj{"place": "journal"})
	}, []string{"include-independent  element notify"}},
	{"a child of a join reads what only a later child writes", func(d obj) {
		read := element(d, "read")
		read["reads"] = add(read["reads"], obj{"place": "journal"})
	}, []string{"join-ordered  element intake"}},
	{"an or with no otherwise", func(d obj) {
		delete(element(d, "shelve"), "otherwise")
		d["elements"] = without(d["elements"], func(e obj) bool { return e["name"] == "update" })
	}, []string{"or-conditions  element shelve"}},
	{"two branches with the same condition", func(d obj) {
		shelve := element(d, "shelve")
		shelve["or"] = add(shelve["or"], obj{"when": "the book is new", "do": "skip"})
		d["elements"] = add(d["elements"], obj{"name": "skip", "reads": []any{obj{"place": "parsed"}},
			"rejects": obj{"reason": "the request cannot be read", "to": "parent"}})
	}, []string{"or-conditions  element shelve"}},
}

func TestEverySoundnessCheckFailsOnItsBreak(t *testing.T) {
	c := &collector{}
	sound(modelOf(t, loadBaseline(t, "library.json"), false), c)
	same(t, "the baseline", lines(c), nil)
	for _, b := range soundBreaks {
		def := loadBaseline(t, "library.json")
		b.change(def)
		c := &collector{}
		sound(modelOf(t, def, false), c)
		same(t, b.name, lines(c), b.want)
	}
}

// A lasting place holds what earlier runs left. A part may read it
// before a later part writes it, parts that run in order may each write
// it, and a part may read it and write it. Independent parts still may
// not both write it.
func TestALastingPlaceIsReadFirstAndWrittenInOrder(t *testing.T) {
	journalCreate := obj{"place": "journal", "mode": "create", "key": "assigned"}
	cases := []struct {
		name   string
		change func(d obj)
		want   []string
	}{
		{"two parts of a join write it", func(d obj) {
			place(d, "journal")["belongsTo"] = "intake"
			read := element(d, "read")
			read["writes"] = add(read["writes"], journalCreate)
		}, nil},
		{"a part reads it before a later part writes it", func(d obj) {
			read := element(d, "read")
			read["reads"] = add(read["reads"], obj{"place": "journal"})
		}, nil},
		{"a part reads it and writes it", func(d obj) {
			place(d, "summary")["lasting"] = true
			audit := element(d, "audit")
			audit["reads"] = add(audit["reads"], obj{"place": "summary"})
		}, nil},
		{"two independent parts write it", func(d obj) {
			place(d, "journal")["belongsTo"] = "notify"
			mail := element(d, "mail")
			mail["writes"] = add(mail["writes"], journalCreate)
		}, []string{"one-owner  place journal", "include-independent  element notify"}},
	}
	for _, tc := range cases {
		def := loadBaseline(t, "library.json")
		place(def, "journal")["lasting"] = true
		tc.change(def)
		c := &collector{}
		sound(modelOf(t, def, false), c)
		same(t, tc.name, lines(c), tc.want)
	}
}

// A check with no break is not proven: this fails when one of the
// checks of the first question has none.
func TestEverySoundnessCheckHasABreak(t *testing.T) {
	broken := map[string]bool{}
	for _, b := range soundBreaks {
		for _, w := range b.want {
			check, _, _ := strings.Cut(w, "  ")
			broken[check] = true
		}
	}
	for _, id := range soundChecks {
		if !broken[id] {
			t.Errorf("no break makes %s report", id)
		}
	}
}

// Every finding is reported, not only the first: several breaks at once
// give the findings of each.
func TestEveryFindingIsReported(t *testing.T) {
	def := loadBaseline(t, "library.json")
	var want []string
	for _, i := range []int{4, 7, 9, 13} {
		soundBreaks[i].change(def)
		want = append(want, soundBreaks[i].want...)
	}
	c := &collector{}
	sound(modelOf(t, def, false), c)
	same(t, "four breaks together", lines(c), want)
}

// Cases with the same subject share one finding, and its message names
// them all.
func TestCasesWithOneSubjectShareAFinding(t *testing.T) {
	def := loadBaseline(t, "library.json")
	delete(element(def, "notify"), "rejects")
	c := &collector{}
	sound(modelOf(t, def, false), c)
	if len(c.list) != 1 || !strings.Contains(c.list[0].Message, "log passes") || !strings.Contains(c.list[0].Message, "mail passes") {
		t.Errorf("want one finding naming log and mail, got %+v", c.list)
	}
}

func TestHeldReportsARuleWithoutItsHold(t *testing.T) {
	root := t.TempDir()
	base := loadBaseline(t, "library.json")
	evidence(t, root, base)
	run := func(def obj) []string {
		c := &collector{}
		held(modelOf(t, def, false), root, c)
		return lines(c)
	}
	same(t, "the baseline", run(base), nil)

	cases := []struct {
		name   string
		change func(d obj)
		want   string
	}{
		{"a check", func(d obj) { delete(checkOf(writeTo(element(d, "add"), "shelf"), "exists"), "hold") }, "held  add writes shelf, check exists"},
		{"a rejects", func(d obj) { delete(element(d, "read")["rejects"].(obj), "hold") }, "held  read rejects"},
		{"a declines", func(d obj) { delete(element(d, "add")["declines"].(obj), "hold") }, "held  add declines"},
		{"a place whose store refuses", func(d obj) { delete(place(d, "journal"), "hold") }, "held  place journal"},
		{"an entry in unsaid", func(d obj) { delete(d["unsaid"].([]any)[0].(obj), "hold") }, `held  "a book is lent to one reader at a time"`},
		{"a hold whose file does not exist", func(d obj) {
			element(d, "read")["rejects"].(obj)["hold"] = obj{"file": "lib/missing.test", "test": "refuses a request it cannot read"}
		}, "held  read rejects"},
		{"a hold whose file does not contain the test", func(d obj) {
			element(d, "read")["rejects"].(obj)["hold"] = obj{"file": "lib/library.test", "test": "a test nobody wrote"}
		}, "held  read rejects"},
	}
	for _, tc := range cases {
		def := loadBaseline(t, "library.json")
		tc.change(def)
		same(t, tc.name, run(def), []string{tc.want})
	}
}

func TestFollowsReportsEachDifference(t *testing.T) {
	run := func(def, desc obj) []string {
		c := &collector{}
		follows(modelOf(t, def, false), modelOf(t, desc, true), 0, c)
		return lines(c)
	}
	same(t, "the baseline, intake", run(loadBaseline(t, "library.json"), loadBaseline(t, "intake.json")), nil)
	same(t, "the baseline, audit", run(loadBaseline(t, "library.json"), loadBaseline(t, "audit.json")), nil)

	src := obj{"file": "lib/x.code", "function": "x"}
	cases := []struct {
		name   string
		change func(def, desc obj)
		want   []string
	}{
		{"an element only in the definition file", func(_, s obj) {
			s["elements"] = without(s["elements"], func(e obj) bool { return e["name"] == "mail" })
			element(s, "notify")["include"] = []any{"log"}
			s["places"] = without(s["places"], func(p obj) bool { return p["name"] == "outbox" })
		}, []string{"follows  element mail", "follows  element notify"}},
		{"an element only in the description file, with one below it", func(_, s obj) {
			element(s, "notify")["include"] = []any{"log", "mail", "extra"}
			s["elements"] = add(s["elements"],
				obj{"name": "extra", "source": src, "include": []any{"deeper"}},
				obj{"name": "deeper", "source": src, "reads": []any{obj{"place": "parsed", "source": src}}})
		}, []string{"follows  element extra", "follows  element notify"}},
		{"a leaf in one file and a join in the other", func(_, s obj) {
			mail := element(s, "mail")
			delete(mail, "reads")
			delete(mail, "writes")
			mail["join"] = []any{"one", "two"}
			for _, n := range []string{"one", "two"} {
				s["elements"] = add(s["elements"], obj{"name": n, "source": src, "reads": []any{obj{"place": "parsed", "source": src}}})
			}
		}, []string{"follows  element mail", "follows  element one", "follows  element two"}},
		{"the children of a join in another order", func(_, s obj) {
			element(s, "intake")["join"] = []any{"shelve", "read", "notify"}
		}, []string{"follows  element intake"}},
		{"another otherwise", func(d, _ obj) {
			shelve := element(d, "shelve")
			shelve["or"], shelve["otherwise"] = []any{obj{"when": "the book is old", "do": "update"}}, "add"
		}, []string{"follows  element shelve"}},
		{"a branch worded differently is no difference", func(_, s obj) {
			element(s, "shelve")["or"].([]any)[0].(obj)["when"] = "nobody has this title yet"
		}, nil},
		{"a read only in the definition file", func(_, s obj) {
			a := element(s, "add")
			a["reads"] = without(a["reads"], func(r obj) bool { return r["place"] == "shelf" })
		}, []string{"follows  add reads shelf"}},
		{"a read only in the description file", func(_, s obj) {
			mail := element(s, "mail")
			mail["reads"] = add(mail["reads"], obj{"place": "shelf", "source": src})
		}, []string{"follows  mail reads shelf"}},
		{"a write only in the definition file", func(_, s obj) {
			delete(element(s, "mail"), "writes")
			s["places"] = without(s["places"], func(p obj) bool { return p["name"] == "outbox" })
		}, []string{"follows  mail writes outbox"}},
		{"a write only in the description file, with its checks", func(_, s obj) {
			mail := element(s, "mail")
			mail["writes"] = add(mail["writes"], obj{"place": "shelf", "mode": "create", "key": "content", "source": src,
				"before": []any{obj{"check": "repeated", "then": "refuse", "source": src}}})
		}, []string{"follows  mail writes shelf"}},
		{"another mode", func(_, s obj) {
			w := writeTo(element(s, "update"), "shelf")
			w["mode"] = "create"
		}, []string{"follows  update writes shelf"}},
		{"another key", func(_, s obj) { writeTo(element(s, "log"), "journal")["key"] = "content" }, []string{"follows  log writes journal"}},
		{"records left out", func(_, s obj) { delete(writeTo(element(s, "update"), "shelf"), "records") }, []string{"follows  update writes shelf"}},
		{"a check only in the definition file", func(_, s obj) {
			w := writeTo(element(s, "add"), "shelf")
			w["before"] = without(w["before"], func(c obj) bool { return c["check"] == "repeated" })
		}, []string{"follows  add writes shelf, check repeated"}},
		{"a check only in the description file", func(_, s obj) {
			w := writeTo(element(s, "log"), "journal")
			w["before"] = []any{obj{"check": "exists", "then": "refuse", "source": src}}
		}, []string{"follows  log writes journal, check exists"}},
		{"another then", func(_, s obj) {
			checkOf(writeTo(element(s, "add"), "shelf"), "exists")["then"] = "refuse"
		}, []string{"follows  add writes shelf, check exists"}},
		{"a rejects only in the definition file", func(_, s obj) { delete(element(s, "mail"), "rejects") }, []string{"follows  mail rejects"}},
		{"a rejects that goes somewhere else", func(_, s obj) {
			element(s, "read")["rejects"].(obj)["to"] = "parent"
		}, []string{"follows  read rejects"}},
		{"a declines only in the description file", func(_, s obj) {
			element(s, "update")["declines"] = obj{"reason": "the person said no", "source": src}
		}, []string{"follows  update declines"}},
		{"a declines whose to is left out", func(_, s obj) {
			delete(element(s, "add")["declines"].(obj), "to")
		}, []string{"follows  add declines"}},
		{"a place the definition file does not have", func(_, s obj) {
			s["places"] = add(s["places"], obj{"name": "cache", "source": src})
			read := element(s, "read")
			read["reads"] = add(read["reads"], obj{"place": "cache", "source": src})
		}, []string{"follows  place cache", "follows  read reads cache"}},
		{"a place keyed in one file only", func(_, s obj) {
			p := place(s, "outbox")
			p["keyed"], p["onExisting"] = true, "replace"
			w := writeTo(element(s, "mail"), "outbox")
			w["mode"], w["key"] = "create", "assigned"
		}, []string{"follows  place outbox", "follows  mail writes outbox"}},
		{"a store that does something else", func(_, s obj) { place(s, "shelf")["onExisting"] = "refuse" }, []string{"follows  place shelf"}},
		{"describes an element the definition file does not have", func(d, _ obj) {
			element(d, "intake")["name"] = "taking in"
			element(d, "library")["steps"] = []any{"taking in", "audit"}
		}, []string{"follows  element intake"}},
		{"describes an element that is not a child of the top element", func(_, s obj) {
			s["describes"] = "notify"
			s["elements"] = without(s["elements"], func(e obj) bool {
				return e["name"] != "notify" && e["name"] != "log" && e["name"] != "mail"
			})
		}, []string{"follows  element notify"}},
	}
	for _, tc := range cases {
		def, desc := loadBaseline(t, "library.json"), loadBaseline(t, "intake.json")
		tc.change(def, desc)
		same(t, tc.name, run(def, desc), tc.want)
	}
}

func TestAChildWithNoDescriptionFileIsAFinding(t *testing.T) {
	c := &collector{}
	undescribed(modelOf(t, loadBaseline(t, "library.json"), false), map[string]bool{"intake": true}, 1, c)
	same(t, "audit has none", lines(c), []string{"follows  element audit"})
}

func TestSourceReportsAStatementWithoutItsSource(t *testing.T) {
	root := t.TempDir()
	base := loadBaseline(t, "intake.json")
	evidence(t, root, base)
	run := func(desc obj) []string {
		c := &collector{}
		sourced(modelOf(t, desc, true), nil, root, 0, c)
		return lines(c)
	}
	same(t, "the baseline", run(base), nil)

	cases := []struct {
		name   string
		change func(s obj)
		want   []string
	}{
		{"a place", func(s obj) { delete(place(s, "shelf"), "source") }, []string{"source  place shelf"}},
		{"an element", func(s obj) { delete(element(s, "notify"), "source") }, []string{"source  element notify"}},
		{"a read", func(s obj) { delete(element(s, "read")["reads"].([]any)[0].(obj), "source") }, []string{"source  read reads request"}},
		{"a write", func(s obj) { delete(writeTo(element(s, "mail"), "outbox"), "source") }, []string{"source  mail writes outbox"}},
		{"a check", func(s obj) { delete(checkOf(writeTo(element(s, "add"), "shelf"), "repeated"), "source") }, []string{"source  add writes shelf, check repeated"}},
		{"a rejects", func(s obj) { delete(element(s, "log")["rejects"].(obj), "source") }, []string{"source  log rejects"}},
		{"a declines", func(s obj) { delete(element(s, "add")["declines"].(obj), "source") }, []string{"source  add declines"}},
		{"a branch", func(s obj) { delete(element(s, "shelve")["or"].([]any)[0].(obj), "source") }, []string{"source  shelve branch to add"}},
		{"an entry in unsaid", func(s obj) { s["unsaid"] = []any{obj{"what": "titles are trimmed"}} },
			[]string{`source  "titles are trimmed"`, `unsaid  "titles are trimmed"`}},
		{"a source whose file does not exist", func(s obj) {
			element(s, "mail")["source"] = obj{"file": "lib/missing.code", "function": "mail"}
		}, []string{"source  element mail"}},
		{"a name that is only part of a longer word", func(s obj) {
			element(s, "mail")["source"] = obj{"file": "lib/mail.code", "function": "ai"}
		}, []string{"source  element mail"}},
	}
	for _, tc := range cases {
		desc := loadBaseline(t, "intake.json")
		tc.change(desc)
		same(t, tc.name, run(desc), tc.want)
	}
}

func TestAWholeWordIsNotPartOfALongerOne(t *testing.T) {
	for text, want := range map[string]bool{
		"disk.write(name)": true, "write": true, "function write() {": true,
		"rewrite(name)": false, "writeFile(name)": false, "re_write": false, "write2": false, "": false,
	} {
		if got := hasWord(text, "write"); got != want {
			t.Errorf("hasWord(%q, \"write\") = %v, want %v", text, got, want)
		}
	}
}

func TestAnEntryInADescriptionsUnsaidIsAFinding(t *testing.T) {
	root := t.TempDir()
	desc := loadBaseline(t, "intake.json")
	desc["unsaid"] = []any{obj{"what": "titles are trimmed", "source": obj{"file": "lib/add.code", "function": "add"}}}
	evidence(t, root, desc)
	c := &collector{}
	sourced(modelOf(t, desc, true), nil, root, 0, c)
	same(t, "one entry", lines(c), []string{`unsaid  "titles are trimmed"`})

	// The same words in the definition file's unsaid: the code does what
	// the design requires, and the entry is no finding.
	c = &collector{}
	sourced(modelOf(t, desc, true), map[string]bool{"titles are trimmed": true}, root, 0, c)
	same(t, "an entry the design also states", lines(c), nil)
	c = &collector{}
	sourced(modelOf(t, desc, true), map[string]bool{"Titles are trimmed.": true}, root, 0, c)
	same(t, "other words", lines(c), []string{`unsaid  "titles are trimmed"`})
}
