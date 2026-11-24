package main

import (
	"encoding/json"
	"strings"
)

// FaultRef points at one fault of the sweep: Runs[Run].Faults[Fault].
type FaultRef struct {
	Run   int    `json:"run"`
	Fault int    `json:"fault"`
	Place string `json:"place"`
}

// FirstRefuser is the leaf that refuses an input before the rule's own
// element reads it.
type FirstRefuser struct {
	Place string `json:"place"`
	By    string `json:"by"`
}

// TestEntry is a test to write for a rule that names no hold: view.md,
// "Tests from the gaps".
type TestEntry struct {
	Kind          string         `json:"kind"`
	Subject       string         `json:"subject"`
	Element       string         `json:"element,omitempty"`
	Place         string         `json:"place,omitempty"`
	Run           string         `json:"run,omitempty"`
	Words         string         `json:"words"`
	Input         string         `json:"input,omitempty"`
	Inputs        []FaultRef     `json:"inputs"`
	FirstRefusers []FirstRefuser `json:"firstRefusers"`
	Asserts       string         `json:"asserts,omitempty"`
	Climb         *Climb         `json:"climb,omitempty"`
	NotRun        []string       `json:"notRun"`
	Break         string         `json:"break,omitempty"`
	Baseline      string         `json:"baseline,omitempty"`
	TestFile      string         `json:"testFile,omitempty"`
	Hold          string         `json:"hold,omitempty"`
	Unwritten     bool           `json:"unwritten"`
}

// runOf is the step of the system an element is under, or "" for the
// top element.
func (m *model) runOf(name string) string {
	for at := name; at != ""; at = m.parent[at] {
		if m.parent[at] == m.f.Top {
			return at
		}
	}
	return ""
}

// testEntries makes an entry for each rule that names no hold, in the
// order of rules(m), from the sweep of the same model.
func testEntries(m *model, runs []RunView, root string) []TestEntry {
	out := []TestEntry{}
	for _, r := range rules(m) {
		if missing(root, r.Ev, "hold", "test", false) == "" {
			continue
		}
		e := TestEntry{Kind: r.Kind, Subject: r.Subject.Text, Element: r.Subject.Element, Place: r.Subject.Place, Inputs: []FaultRef{}, FirstRefusers: []FirstRefuser{}, NotRun: []string{}}
		switch r.Kind {
		case "unsaid":
			e.Words, e.Unwritten = r.Unsaid.What, true
		case "store":
			e.Run = m.runOf(r.Place.BelongsTo)
			e.Words = "the store refuses to create a record at a key that already has one, and keeps the old record"
			e.Input = "a create in " + r.Place.Name + " at a key that already has a record"
			e.Asserts = "the create fails, and the record at the key is the one that was there"
			e.Break = "let the create replace the record at the key. The test fails."
			e.TestFile = holdFile(m, e.Run)
		case "rejects":
			e.Run = m.runOf(r.Elem.Name)
			e.Words = r.Refusal.Reason
			refusing(m, runs, r.Elem, &e)
			e.Break = "remove the refusal of " + r.Elem.Name + " from the code, so the run goes on. The test fails."
			e.TestFile = holdFile(m, e.Run)
		case "declines":
			e.Run = m.runOf(r.Elem.Name)
			e.Words = r.Refusal.Reason
			e.Input = "the person says no"
			forced(m, e.Run, r.Elem.Name, Break{Declines: r.Elem.Name}, &e)
			e.Break = "go on with the write when the person says no. The test fails."
			e.TestFile = holdFile(m, e.Run)
		case "check":
			e.Run = m.runOf(r.Elem.Name)
			e.Words = r.Check.Kind + " before writing " + r.Write.Place + ", then " + r.Check.Then
			e.Input = checkInputs[r.Check.Kind]
			brk := Break{Refuses: r.Elem.Name}
			if r.Check.Then == "ask" {
				brk = Break{Declines: r.Elem.Name}
			}
			forced(m, e.Run, r.Elem.Name, brk, &e)
			e.Break = "write without the " + r.Check.Kind + " check. The test fails."
			e.TestFile = holdFile(m, e.Run)
		}
		if !e.Unwritten {
			e.Baseline = "the code as it is: the test passes."
		}
		if e.TestFile != "" {
			file, _ := json.Marshal(e.TestFile)
			e.Hold = `"hold": { "file": ` + string(file) + `, "test": "" }`
		}
		out = append(out, e)
	}
	return out
}

// checkInputs is the case each check before a write is for, from the
// question fields.md gives it.
var checkInputs = map[string]string{
	"exists":   "a record already at the key",
	"repeated": "the same key twice among the records of one run",
	"belongs":  "a record at the key that belongs to something else",
}

// refusing fills a rejects entry from the sweep: the faults whose
// refusal leaves from the element or climbs through it, and what a play
// with that refusal asserts. When no fault reaches it, the entry names
// the leaf that refuses each of its inputs first, and a leaf's own
// refusal is forced to say what the test asserts.
func refusing(m *model, runs []RunView, el *Element, e *TestEntry) {
	var first *Trace
	for ri, rv := range runs {
		for fi, f := range rv.Faults {
			t := rv.Traces[f.Trace]
			if len(t.Refusals) == 0 || !contains(t.Refusals[0].reached(), el.Name) {
				continue
			}
			if first == nil {
				first = &rv.Traces[f.Trace]
			}
			if !hasInput(e.Inputs, f.Place) {
				e.Inputs = append(e.Inputs, FaultRef{Run: ri, Fault: fi, Place: f.Place})
			}
		}
	}
	if first != nil {
		asserted(*first, e)
		return
	}
	if el.Shape != "leaf" {
		return
	}
	for _, rd := range el.Reads {
		for _, rv := range runs {
			if rv.Run != e.Run {
				continue
			}
			for _, f := range rv.Faults {
				if f.Place == rd.Place && f.RefusedAt != "" && f.RefusedAt != el.Name && !hasRefuser(e.FirstRefusers, rd.Place) {
					e.FirstRefusers = append(e.FirstRefusers, FirstRefuser{Place: rd.Place, By: f.RefusedAt})
				}
			}
		}
	}
	e.Input = "what makes the refusal happen: " + el.Rejects.Reason
	forced(m, e.Run, el.Name, Break{Refuses: el.Name}, e)
}

// forced plays the run with a refusal or decline forced at a leaf, on
// the first path that reaches the leaf, and takes what the test asserts
// from that trace.
func forced(m *model, run, leaf string, brk Break, e *TestEntry) {
	for _, path := range m.paths(run) {
		t := play(m, run, brk, path)
		for _, s := range t.Steps {
			if s.Element == leaf && s.Outcome == "refused" {
				asserted(t, e)
				return
			}
		}
	}
}

// asserted states what the test asserts from a trace: the leaf the run
// fails at, the refusal's climb, and the leaves that do not run.
func asserted(t Trace, e *TestEntry) {
	c := t.Refusals[0]
	e.Climb = &c
	for _, s := range t.Steps {
		if s.Outcome == "not-run" {
			e.NotRun = append(e.NotRun, s.Element)
		}
	}
	what, at, own := "the refusal", "fails at", "refusal"
	if c.Kind == "declines" {
		what, at, own = "the person's no", "stops at", "declines"
	}
	end := "is reported nowhere, because " + c.LostAt + " declares no " + own + " of its own"
	if c.LostAt == "" && c.To != "" {
		end = "reaches " + c.To
	} else if c.LostAt == "" {
		end = "is not reported"
	}
	e.Asserts = "the run " + at + " " + c.From + " and " + what + " " + end
	switch len(e.NotRun) {
	case 0:
	case 1:
		e.Asserts += ", and " + e.NotRun[0] + " does not run"
	default:
		e.Asserts += ", and " + strings.Join(e.NotRun, ", ") + " do not run"
	}
}

// holdFile is the file the holds of a run's rules name most, the first
// in the order of rules(m) on a tie. For "", the run of a place that
// belongs to the top element, it counts the rules of the whole file.
func holdFile(m *model, run string) string {
	count, best := map[string]int{}, ""
	for _, r := range rules(m) {
		if r.Ev == nil || r.Ev.File == "" {
			continue
		}
		owner := ""
		switch {
		case r.Elem != nil:
			owner = r.Elem.Name
		case r.Place != nil:
			owner = r.Place.BelongsTo
		}
		if run != "" && m.runOf(owner) != run {
			continue
		}
		count[r.Ev.File]++
		if best == "" || count[r.Ev.File] > count[best] {
			best = r.Ev.File
		}
	}
	return best
}

func hasInput(list []FaultRef, place string) bool {
	for _, x := range list {
		if x.Place == place {
			return true
		}
	}
	return false
}

func hasRefuser(list []FirstRefuser, place string) bool {
	for _, x := range list {
		if x.Place == place {
			return true
		}
	}
	return false
}
