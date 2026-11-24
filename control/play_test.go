package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

// playOn plays one run of a definition with one break, on one of its
// paths.
func playOn(t *testing.T, def obj, run string, brk Break, path int) Trace {
	t.Helper()
	m := modelOf(t, def, false)
	paths := m.paths(run)
	if path >= len(paths) {
		t.Fatalf("%s has %d paths, and the case plays path %d", run, len(paths), path+1)
	}
	return play(m, run, brk, paths[path])
}

func stepOf(t *testing.T, tr Trace, element string) StepTrace {
	t.Helper()
	for _, s := range tr.Steps {
		if s.Element == element {
			return s
		}
	}
	t.Fatalf("the trace has no step %s: %+v", element, tr.Steps)
	return StepTrace{}
}

func stateIn(list []Seen, place string) string {
	for _, s := range list {
		if s.Place == place {
			return s.State
		}
	}
	return ""
}

func outcomesOf(tr Trace) string {
	var out []string
	for _, s := range tr.Steps {
		out = append(out, s.Element+" "+s.Outcome)
	}
	return strings.Join(out, ", ")
}

func expect(t *testing.T, rule, what string, got, want any) {
	t.Helper()
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("%s: %s is %v, want %v", rule, what, got, want)
	}
}

// Each rule of view.md's table, on the baseline library: intake reads a
// request, shelves the book by add or update, and notifies by log and
// mail; read reports its refusal to parsed, and so does intake.
var playCases = []struct {
	rule   string
	change func(d obj)
	run    string
	brk    Break
	path   int
	want   func(t *testing.T, rule string, tr Trace)
}{
	{"start", nil, "intake", Break{}, 0, func(t *testing.T, rule string, tr Trace) {
		expect(t, rule, "request at the start", stateIn(tr.Start, "request"), "given")
		expect(t, rule, "shelf at the start", stateIn(tr.Start, "shelf"), "present")
		expect(t, rule, "parsed at the start", stateIn(tr.Start, "parsed"), "empty")
	}},
	{"broken", nil, "intake", Break{Place: "parsed"}, 0, func(t *testing.T, rule string, tr Trace) {
		expect(t, rule, "parsed at the start", stateIn(tr.Start, "parsed"), "empty")
		expect(t, rule, "parsed as read writes it", stateIn(stepOf(t, tr, "read").Writes, "parsed"), "invalid")
		given := playOn(t, loadBaseline(t, "library.json"), "intake", Break{Place: "request"}, 0)
		expect(t, rule, "a broken request at the start", stateIn(given.Start, "request"), "invalid")
	}},
	{"join", nil, "intake", Break{Place: "shelf"}, 0, func(t *testing.T, rule string, tr Trace) {
		expect(t, rule, "the steps", outcomesOf(tr), "read done, add refused, log not-run, mail not-run")
	}},
	{"include", func(d obj) {
		delete(element(d, "add"), "rejects")
		element(d, "notify")["rejects"] = obj{"reason": "a notice cannot be sent", "to": "outbox"}
	}, "intake", Break{Place: "parsed"}, 0, func(t *testing.T, rule string, tr Trace) {
		expect(t, rule, "log and mail", stepOf(t, tr, "log").Outcome+" "+stepOf(t, tr, "mail").Outcome, "refused refused")
		expect(t, rule, "the refusals", len(tr.Refusals), 2)
	}},
	{"or", nil, "intake", Break{}, 1, func(t *testing.T, rule string, tr Trace) {
		expect(t, rule, "the path", tr.Path, []Pick{{Or: "shelve", Child: "update"}})
		expect(t, rule, "the steps", outcomesOf(tr), "read done, update done, log done, mail done")
	}},
	{"refuse", nil, "intake", Break{Place: "request"}, 0, func(t *testing.T, rule string, tr Trace) {
		expect(t, rule, "read", stepOf(t, tr, "read").Outcome, "refused")
		expect(t, rule, "what read wrote", len(stepOf(t, tr, "read").Writes), 0)
	}},
	{"climb", nil, "intake", Break{Place: "shelf"}, 0, func(t *testing.T, rule string, tr Trace) {
		if len(tr.Refusals) != 1 {
			t.Fatalf("%s: %d refusals, want 1", rule, len(tr.Refusals))
		}
		c := tr.Refusals[0]
		expect(t, rule, "the climb", c.From+" "+strings.Join(c.Through, " ")+" to "+c.To, "add shelve intake to parsed")
	}},
	{"report", nil, "intake", Break{Place: "request"}, 0, func(t *testing.T, rule string, tr Trace) {
		expect(t, rule, "the steps", outcomesOf(tr), "read refused, add done, log done, mail done")
		expect(t, rule, "parsed", stateIn(stepOf(t, tr, "add").Reads, "parsed"), "refusal")
	}},
	{"lost", func(d obj) { delete(element(d, "shelve"), "rejects") }, "intake", Break{Place: "shelf"}, 0, func(t *testing.T, rule string, tr Trace) {
		if len(tr.Refusals) != 1 {
			t.Fatalf("%s: %d refusals, want 1", rule, len(tr.Refusals))
		}
		expect(t, rule, "where the refusal stops", tr.Refusals[0].LostAt, "shelve")
		expect(t, rule, "the steps", outcomesOf(tr), "read done, add refused, log done, mail done")
	}},
	{"pass-on", func(d obj) { delete(element(d, "read"), "rejects") }, "intake", Break{Place: "request"}, 0, func(t *testing.T, rule string, tr Trace) {
		expect(t, rule, "read", stepOf(t, tr, "read").Outcome, "passed-on")
		expect(t, rule, "parsed as read writes it", stateIn(stepOf(t, tr, "read").Writes, "parsed"), "invalid")
		expect(t, rule, "add", stepOf(t, tr, "add").Outcome, "refused")
	}},
}

func TestEachPlayRuleFailsOnItsBreak(t *testing.T) {
	for _, c := range playCases {
		t.Run(c.rule, func(t *testing.T) {
			def := loadBaseline(t, "library.json")
			if c.change != nil {
				c.change(def)
			}
			c.want(t, c.rule, playOn(t, def, c.run, c.brk, c.path))
		})
	}
}

// A rule with no case is not proven: the rules in view.md's table and
// the cases are the same list.
func TestEveryPlayRuleHasABreak(t *testing.T) {
	var cases []string
	for _, c := range playCases {
		cases = append(cases, c.rule)
	}
	page := firstColumn(t, page(t, "view.md"), "The rules:")
	if strings.Join(page, " ") != strings.Join(cases, " ") {
		t.Errorf("view.md, the rules:\n page  %q\n cases %q", page, cases)
	}
}

// A verdict is decided from the trace and the places the run reads,
// and from nothing else.
func TestVerdictsComeFromTraces(t *testing.T) {
	m := modelOf(t, loadBaseline(t, "library.json"), false)
	for _, c := range []struct {
		name string
		tr   Trace
		want string
	}{
		{"a refusal that reaches a place", Trace{Run: "intake", Break: Break{Place: "shelf"}, Refusals: []Climb{{From: "add", To: "parsed"}}}, "refused"},
		{"a refusal that stops", Trace{Run: "intake", Break: Break{Place: "shelf"}, Refusals: []Climb{{From: "add", LostAt: "shelve"}}}, "lost"},
		{"the first of two refusals", Trace{Run: "intake", Break: Break{Place: "parsed"}, Refusals: []Climb{{From: "log", LostAt: "notify"}, {From: "mail", To: "parsed"}}}, "lost"},
		{"a refusal in the broken place", Trace{Run: "intake", Break: Break{Place: "parsed"}, Refusals: []Climb{{From: "read", To: "parsed"}}}, "refused"},
		{"a place the run does not read", Trace{Run: "intake", Break: Break{Place: "journal"}}, "output"},
		{"a leaf that passed it on", Trace{Run: "intake", Break: Break{Place: "parsed"}, Steps: []StepTrace{{Element: "add", Outcome: "passed-on"}}}, "passed"},
		{"no leaf that met it", Trace{Run: "intake", Break: Break{Place: "parsed"}, Steps: []StepTrace{{Element: "add", Outcome: "done"}}}, "unread"},
	} {
		if got := m.verdict(c.tr); got != c.want {
			t.Errorf("%s: %s, want %s", c.name, got, c.want)
		}
	}
}

// The states, outcomes and verdicts on view.md are the ones play uses.
func TestTheSweepPageNamesEveryStateAndVerdict(t *testing.T) {
	text := page(t, "view.md")
	for _, tb := range []struct {
		marker string
		want   []string
	}{
		{"A place holds one of these states:", playStates},
		{"Each leaf of a trace ends in one of these outcomes:", outcomes},
		{"these verdicts that applies:", verdicts},
	} {
		if got := firstColumn(t, text, tb.marker); strings.Join(got, " ") != strings.Join(tb.want, " ") {
			t.Errorf("view.md, the table after %q:\n page    %q\n checker %q", tb.marker, got, tb.want)
		}
	}
}

// sweepLines prints a sweep one fault a line, for the fixed lists.
func sweepLines(rvs []RunView) []string {
	var out []string
	for _, rv := range rvs {
		for _, f := range rv.Faults {
			var picks []string
			for _, p := range rv.Paths[f.Path] {
				picks = append(picks, p.Or+" "+p.Child)
			}
			line := fmt.Sprintf("%s  %s [%s]: %s", rv.Run, f.Place, strings.Join(picks, ", "), f.Verdict)
			if f.RefusedAt != "" {
				line += " by " + f.RefusedAt
			}
			if f.Verdict == "refused" {
				line += map[bool]string{true: ", held", false: ", no hold"}[f.Held]
			}
			out = append(out, line)
		}
	}
	return out
}

func sweepOf(t *testing.T, path, root string) []RunView {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var o obj
	if err := json.Unmarshal(data, &o); err != nil {
		t.Fatal(err)
	}
	return sweep(modelOf(t, o, false), root)
}

func TestTheBaselineSweepIsFixed(t *testing.T) {
	def := loadBaseline(t, "library.json")
	root := t.TempDir()
	evidence(t, root, def)
	sameOrder(t, "the baseline's sweep", sweepLines(sweep(modelOf(t, def, false), root)), []string{
		"intake  request [shelve add]: refused by read, held",
		"intake  parsed [shelve add]: refused by add, held",
		"intake  shelf [shelve add]: refused by add, held",
		"intake  journal [shelve add]: output",
		"intake  outbox [shelve add]: output",
		"intake  request [shelve update]: refused by read, held",
		"intake  parsed [shelve update]: refused by update, held",
		"intake  shelf [shelve update]: refused by update, held",
		"intake  journal [shelve update]: output",
		"intake  outbox [shelve update]: output",
		"audit  shelf []: refused by audit, held",
		"audit  journal []: unread",
		"audit  summary []: output",
	})
}

// The save trial has an or inside an or: three paths for save.
func TestTheTrialsSweepAsFixed(t *testing.T) {
	sameOrder(t, "the save trial's sweep", sweepLines(sweepOf(t, "examples/save/games.json", "examples/save")), []string{
		"save  typed name [save say full]: unread",
		"save  game [save say full]: unread",
		"save  disk [save say full]: unread",
		"save  game name [save say full]: unread",
		"save  saved text [save say full]: output",
		"save  message [save say full]: output",
		"save  typed name [save saving, store save as]: refused by save as, held",
		"save  game [save saving, store save as]: unread",
		"save  disk [save saving, store save as]: unread",
		"save  game name [save saving, store save as]: unread",
		"save  saved text [save saving, store save as]: output",
		"save  message [save saving, store save as]: output",
		"save  typed name [save saving, store save in place]: unread",
		"save  game [save saving, store save in place]: unread",
		"save  disk [save saving, store save in place]: unread",
		"save  game name [save saving, store save in place]: unread",
		"save  saved text [save saving, store save in place]: output",
		"save  message [save saving, store save in place]: output",
		"engine  game []: output",
	})
}

// The example on view.md is the sweep of the example in fields.md.
func TestThePagesExampleSweepsAsThePageShows(t *testing.T) {
	_, example, found := strings.Cut(page(t, "fields.md"), "\n## An example\n")
	blocks := jsonBlock.FindAllStringSubmatch(example, -1)
	if !found || len(blocks) == 0 {
		t.Fatal("fields.md: no JSON block under \"An example\"")
	}
	var def obj
	if err := json.Unmarshal([]byte(blocks[0][1]), &def); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	evidence(t, root, def)
	var got []string
	for _, rv := range sweep(modelOf(t, def, false), root) {
		for _, f := range rv.Faults {
			got = append(got, f.Place+" "+f.Verdict+" "+f.RefusedAt)
		}
	}
	rows, ok := tableRows(page(t, "view.md"), "## An example")
	if !ok {
		t.Fatal("view.md: no table under \"An example\"")
	}
	var shown []string
	for _, r := range rows[1:] {
		shown = append(shown, strings.Trim(r[0], "`")+" "+strings.Trim(r[1], "`")+" "+strings.Trim(r[2], "`"))
	}
	sameOrder(t, "view.md, the example", got, shown)
}
