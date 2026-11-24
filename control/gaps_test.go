package main

import (
	"fmt"
	"strings"
	"testing"
)

// noHolds takes the hold off every rule of a definition.
func noHolds(d obj) {
	var walk func(v any)
	walk = func(v any) {
		switch x := v.(type) {
		case obj:
			delete(x, "hold")
			for _, val := range x {
				walk(val)
			}
		case []any:
			for _, item := range x {
				walk(item)
			}
		}
	}
	walk(d)
}

func entriesOf(t *testing.T, def obj) []TestEntry {
	t.Helper()
	root := t.TempDir()
	evidence(t, root, def)
	m := modelOf(t, def, false)
	return testEntries(m, sweep(m, root), root)
}

func entryLines(entries []TestEntry) []string {
	var out []string
	for _, e := range entries {
		var in []string
		for _, i := range e.Inputs {
			in = append(in, i.Place)
		}
		for _, f := range e.FirstRefusers {
			in = append(in, f.Place+" first by "+f.By)
		}
		if e.Input != "" {
			in = append(in, e.Input)
		}
		line := fmt.Sprintf("%s (%s): %s | %s | %s", e.Subject, e.Kind, strings.Join(in, "; "), e.Asserts, e.TestFile)
		if e.Unwritten {
			line = fmt.Sprintf("%s (%s): unwritten", e.Subject, e.Kind)
		}
		out = append(out, line)
	}
	return out
}

// The baseline with the holds taken off six rules, one of each kind and
// one rejects of an element with children, in the order rules(m) walks
// them. intake declines with no "to", so a no is not reported.
func TestTheTestsFromTheGapsAreFixed(t *testing.T) {
	def := loadBaseline(t, "library.json")
	delete(place(def, "journal"), "hold")
	delete(checkOf(writeTo(element(def, "add"), "shelf"), "exists"), "hold")
	delete(element(def, "read")["rejects"].(obj), "hold")
	delete(element(def, "add")["declines"].(obj), "hold")
	delete(element(def, "intake")["rejects"].(obj), "hold")
	delete(def["unsaid"].([]any)[0].(obj), "hold")
	sameOrder(t, "the entries", entryLines(entriesOf(t, def)), []string{
		"place journal (store): a create in journal at a key that already has a record | the create fails, and the record at the key is the one that was there | lib/library.test",
		"intake rejects (rejects): parsed; shelf | the run fails at add and the refusal reaches parsed, and log, mail do not run | lib/library.test",
		"read rejects (rejects): request | the run fails at read and the refusal reaches parsed | lib/library.test",
		"add writes shelf, check exists (check): a record already at the key | the run stops at add and the person's no is not reported, and log, mail do not run | lib/library.test",
		"add declines (declines): the person says no | the run stops at add and the person's no is not reported, and log, mail do not run | lib/library.test",
		`"a book is lent to one reader at a time" (unsaid): unwritten`,
	})
}

func TestAnUnsaidEntryIsMarkedUnwritten(t *testing.T) {
	def := loadBaseline(t, "library.json")
	delete(def["unsaid"].([]any)[0].(obj), "hold")
	entries := entriesOf(t, def)
	if len(entries) != 1 {
		t.Fatalf("%d entries, want 1", len(entries))
	}
	e := entries[0]
	if !e.Unwritten || e.Words != "a book is lent to one reader at a time" || e.Input != "" || e.Asserts != "" || e.Break != "" || e.Baseline != "" || e.TestFile != "" || e.Hold != "" {
		t.Errorf("the entry is %+v", e)
	}
}

// held and the tests from the gaps walk the same rules: with every hold
// taken off, each finding of held has an entry and each entry a finding.
func TestHeldAndTheTestsWalkTheSameRules(t *testing.T) {
	def := loadBaseline(t, "library.json")
	noHolds(def)
	root := t.TempDir()
	m := modelOf(t, def, false)
	c := &collector{}
	held(m, root, c)
	var findings, entries []string
	for _, f := range c.list {
		findings = append(findings, f.Subject)
	}
	for _, e := range testEntries(m, sweep(m, root), root) {
		entries = append(entries, e.Subject)
	}
	if len(findings) == 0 {
		t.Fatal("held reports nothing with every hold taken off")
	}
	sameOrder(t, "held and the entries", entries, findings)
}
