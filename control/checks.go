package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Finding is one line the checker reports: a check, the subject it is
// about, and a message. checks.md fixes the check's name and the
// subject; the message is the checker's wording.
type Finding struct {
	Check   string
	Subject string
	// Element and Place are what the subject names, when it names them.
	Element string
	Place   string
	Message string
	// file is the file the finding is accepted beside: -1 for the
	// definition file, otherwise the index of a description file.
	file int
	// order places the finding among others of its check: the index of
	// its description file, or after them all.
	order    int
	Accepted bool
	Reason   string
}

// soundChecks are the checks of the first question, in the order of
// their table in checks.md.
var soundChecks = []string{
	"one-owner", "reads-for-reference", "create-never-replaces", "replace-checks-owner",
	"key-from-content", "rejects-declared", "declines-declared", "reject-values",
	"include-independent", "join-ordered", "or-conditions",
}

// checkOrder is every check, in the order findings are printed.
var checkOrder = append(append([]string{}, soundChecks...), "held", "follows", "source", "unsaid", "stale")

// Subject is what a finding is about: the text checks.md gives, and the
// element, place, check and unsaid entry that text names, set where the
// subject is made so that no reader has to parse the text back.
type Subject struct {
	Text    string
	Element string
	Place   string
	Check   string
	Unsaid  string
}

// The subjects, as checks.md gives them.
func sPlace(p string) Subject   { return Subject{Text: "place " + p, Place: p} }
func sElement(e string) Subject { return Subject{Text: "element " + e, Element: e} }
func sRead(e, p string) Subject { return Subject{Text: e + " reads " + p, Element: e, Place: p} }
func sWrite(e, p string) Subject {
	return Subject{Text: e + " writes " + p, Element: e, Place: p}
}
func sCheck(e, p, k string) Subject {
	s := sWrite(e, p)
	s.Text += ", check " + k
	s.Check = k
	return s
}
func sRejects(e string) Subject  { return Subject{Text: e + " rejects", Element: e} }
func sDeclines(e string) Subject { return Subject{Text: e + " declines", Element: e} }
func sBranch(e, c string) Subject {
	return Subject{Text: e + " branch to " + c, Element: e}
}
func sUnsaid(what string) Subject { return Subject{Text: `"` + what + `"`, Unsaid: what} }

// collector gathers findings, one for each check, file and subject.
// Several cases with the same subject share one finding, and its
// message names them all.
type collector struct {
	list  []*Finding
	index map[string]*Finding
}

func (c *collector) add(check string, file, order int, s Subject, format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	key := fmt.Sprintf("%s\x00%d\x00%d\x00%s", check, file, order, s.Text)
	if c.index == nil {
		c.index = map[string]*Finding{}
	}
	if f := c.index[key]; f != nil {
		f.Message += "; " + msg
		return
	}
	f := &Finding{Check: check, Subject: s.Text, Element: s.Element, Place: s.Place, Message: msg, file: file, order: order}
	c.index[key] = f
	c.list = append(c.list, f)
}

// sound makes the checks of the first question on a definition file.
func sound(m *model, c *collector) {
	add := func(check string, subject Subject, format string, a ...any) {
		c.add(check, -1, 0, subject, format, a...)
	}
	leaves := m.leaves()

	// one-owner
	writers := map[string][]string{}
	for _, e := range leaves {
		for _, w := range e.Writes {
			p := m.place[w.Place]
			writers[p.Name] = append(writers[p.Name], e.Name)
			switch {
			case p.Outside:
				add("one-owner", sWrite(e.Name, p.Name), "%s is from outside the system, and no element writes it", p.Name)
			case !m.ownedBy(e.Name, p):
				add("one-owner", sWrite(e.Name, p.Name), "%s belongs to %s, and %s is not below %s", p.Name, p.BelongsTo, e.Name, p.BelongsTo)
			}
		}
	}
	for _, p := range m.f.Places {
		ws := writers[p.Name]
		for i := 0; i < len(ws); i++ {
			for j := i + 1; j < len(ws); j++ {
				if n := m.elem[m.nearest(ws[i], ws[j])]; n != nil && (n.Shape == "include" || n.Shape == "join" && !p.Lasting) {
					add("one-owner", sPlace(p.Name), "%s and %s both write it, and the nearest element above both, %s, is %s", ws[i], ws[j], n.Name, article(n.Shape))
				}
			}
		}
	}

	for _, e := range leaves {
		for _, w := range e.Writes {
			p := m.place[w.Place]
			exists := w.check("exists")
			// reads-for-reference
			if e.reads(p.Name) && !p.Lasting && w.Mode != "replace" && !(w.Mode == "create" && exists != nil) {
				add("reads-for-reference", sWrite(e.Name, p.Name), "%s reads %s and also writes it", e.Name, p.Name)
			}
			// create-never-replaces
			if w.Mode == "create" && p.OnExisting == "replace" && exists == nil {
				add("create-never-replaces", sWrite(e.Name, p.Name), "the store of %s replaces a record that is already there, and the write has no exists check", p.Name)
			}
			if exists != nil && !e.reads(p.Name) {
				add("create-never-replaces", sCheck(e.Name, p.Name, "exists"), "%s checks what is in %s and does not read it", e.Name, p.Name)
			}
			// replace-checks-owner
			if w.Mode == "replace" && w.check("belongs") == nil {
				add("replace-checks-owner", sWrite(e.Name, p.Name), "the write replaces a record and has no belongs check")
			}
			// key-from-content
			if w.Key == "content" && w.Records != "one" && w.check("repeated") == nil {
				add("key-from-content", sWrite(e.Name, p.Name), "the key comes from what %s reads, one run can make many records, and the write has no repeated check", e.Name)
			}
		}
	}

	// rejects-declared and declines-declared
	for _, e := range leaves {
		if len(e.Reads) > 0 && e.Rejects == nil {
			add("rejects-declared", sRejects(e.Name), "%s reads %s and has no rejects", e.Name, strings.Join(readNames(e), ", "))
		}
		asks := false
		for _, w := range e.Writes {
			if ch := w.check("exists"); ch != nil && ch.Then == "ask" {
				asks = true
			}
		}
		if asks && e.Declines == nil {
			add("declines-declared", sDeclines(e.Name), "%s asks a person and has no declines", e.Name)
		}
	}
	for _, e := range m.f.Elements {
		parent := m.elem[m.parent[e.Name]]
		if parent == nil {
			continue
		}
		if e.Rejects != nil && e.Rejects.To == "parent" && parent.Rejects == nil {
			add("rejects-declared", sRejects(parent.Name), "%s passes its refusal to %s, which has no rejects", e.Name, parent.Name)
		}
		if e.Declines != nil && e.Declines.To == "parent" && parent.Declines == nil {
			add("declines-declared", sDeclines(parent.Name), "%s passes the person's answer to %s, which has no declines", e.Name, parent.Name)
		}
	}

	// reject-values
	for _, p := range m.f.Places {
		if p.Invalid == "" {
			add("reject-values", sPlace(p.Name), "the place does not say what makes a value of it invalid")
		}
	}

	for _, e := range m.f.Elements {
		switch e.Shape {
		case "include":
			for i, a := range e.Children {
				for _, b := range e.Children[i+1:] {
					aw, bw, ar, br := m.writesOf(a), m.writesOf(b), m.readsOf(a), m.readsOf(b)
					for _, p := range sortedKeys(aw) {
						if bw[p] {
							add("include-independent", sElement(e.Name), "%s and %s both write %s", a, b, p)
						}
						if br[p] {
							add("include-independent", sElement(e.Name), "%s reads %s, which %s writes", b, p, a)
						}
					}
					for _, p := range sortedKeys(bw) {
						if ar[p] {
							add("include-independent", sElement(e.Name), "%s reads %s, which %s writes", a, p, b)
						}
					}
				}
			}
		case "join":
			for i, child := range e.Children {
				for _, p := range sortedKeys(m.readsOf(child)) {
					if m.place[p].Lasting {
						continue
					}
					earlier, later := false, ""
					for j, other := range e.Children {
						if !m.writesOf(other)[p] {
							continue
						}
						if j < i {
							earlier = true
						} else if j > i && later == "" {
							later = other
						}
					}
					if later != "" && !earlier {
						add("join-ordered", sElement(e.Name), "%s reads %s, which the later child %s writes and no earlier child writes", child, p, later)
					}
				}
			}
		case "or":
			if e.Otherwise == "" {
				add("or-conditions", sElement(e.Name), "the or has no otherwise")
			}
			seen := map[string]bool{}
			for _, b := range e.Branches {
				if seen[b.When] {
					add("or-conditions", sElement(e.Name), "two branches have the same condition, %q", b.When)
				}
				seen[b.When] = true
			}
		}
	}
}

// held makes the second question's check on every rule of a definition
// file: each names a test, in a file that exists and contains its name.
func held(m *model, root string, c *collector) {
	for _, r := range rules(m) {
		if why := missing(root, r.Ev, "hold", "test", false); why != "" {
			c.add("held", -1, 0, r.Subject, "%s", why)
		}
	}
}

// sourced makes the source check on every statement of a description
// file, and reports each entry in its unsaid that the definition file
// does not also state, word for word.
func sourced(m *model, required map[string]bool, root string, file int, c *collector) {
	stmt := func(subject Subject, ev *Evidence) {
		if why := missing(root, ev, "source", "function", true); why != "" {
			c.add("source", file, file, subject, "%s", why)
		}
	}
	for _, p := range m.f.Places {
		stmt(sPlace(p.Name), p.Ev)
	}
	for _, e := range m.f.Elements {
		stmt(sElement(e.Name), e.Ev)
		for _, r := range e.Reads {
			stmt(sRead(e.Name, r.Place), r.Ev)
		}
		for _, w := range e.Writes {
			stmt(sWrite(e.Name, w.Place), w.Ev)
			for _, ch := range w.Before {
				stmt(sCheck(e.Name, w.Place, ch.Kind), ch.Ev)
			}
		}
		if e.Rejects != nil {
			stmt(sRejects(e.Name), e.Rejects.Ev)
		}
		if e.Declines != nil {
			stmt(sDeclines(e.Name), e.Declines.Ev)
		}
		for _, b := range e.Branches {
			stmt(sBranch(e.Name, b.Do), b.Ev)
		}
	}
	for _, u := range m.f.Unsaid {
		stmt(sUnsaid(u.What), u.Ev)
		if !required[u.What] {
			c.add("unsaid", file, file, sUnsaid(u.What), "the code does this, and the fields cannot state it")
		}
	}
}

// missing says what is wrong with a hold or a source, or "" when the
// file exists and contains the name. A function's name is matched as a
// whole word; a test's name is matched exactly as written.
func missing(root string, ev *Evidence, word, name string, wholeWord bool) string {
	if ev == nil {
		return "no " + word
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(ev.File)))
	if err != nil {
		return fmt.Sprintf("%s does not exist", ev.File)
	}
	text := string(data)
	if (wholeWord && hasWord(text, ev.Name)) || (!wholeWord && ev.Name != "" && strings.Contains(text, ev.Name)) {
		return ""
	}
	return fmt.Sprintf("%s does not contain the %s %q", ev.File, name, ev.Name)
}

// hasWord reports whether text contains word with no letter, digit or
// underscore directly before or after it.
func hasWord(text, word string) bool {
	if word == "" {
		return false
	}
	wordChar := func(b byte) bool {
		return b == '_' || b >= '0' && b <= '9' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= 0x80
	}
	for from := 0; ; {
		i := strings.Index(text[from:], word)
		if i < 0 {
			return false
		}
		start, end := from+i, from+i+len(word)
		if (start == 0 || !wordChar(text[start-1])) && (end == len(text) || !wordChar(text[end])) {
			return true
		}
		from = start + 1
	}
}

func readNames(e *Element) []string {
	var out []string
	for _, r := range e.Reads {
		out = append(out, r.Place)
	}
	return out
}

func sortedKeys(set map[string]bool) []string {
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
