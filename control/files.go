package main

import (
	"fmt"
	"strings"
)

// Evidence is a hold in a definition file, a file and a test's name, or
// a source in a description file, a file and a function's name.
type Evidence struct {
	File string
	Name string
}

// File is a definition file or a description file. Top is the name its
// first key gives: the top element of a definition file, the described
// element of a description file.
type File struct {
	Path        string
	Description bool
	Top         string
	Places      []*Place
	Elements    []*Element
	Unsaid      []*Unsaid
}

type Place struct {
	Name       string
	BelongsTo  string
	Outside    bool
	Keyed      bool
	Lasting    bool
	OnExisting string
	Invalid    string
	Ev         *Evidence
}

// Element is a leaf, or combines its children as a join, an include, an
// or, or the top element's steps. Children holds the children of any
// shape; for an or, the child of each branch and then the otherwise.
type Element struct {
	Name      string
	Shape     string
	Reads     []*Read
	Writes    []*Write
	Children  []string
	Branches  []*Branch
	Otherwise string
	Rejects   *Refusal
	Declines  *Refusal
	Ev        *Evidence
}

type Read struct {
	Place string
	Ev    *Evidence
}

type Write struct {
	Place   string
	Mode    string
	Key     string
	Records string
	Before  []*Check
	Ev      *Evidence
}

type Check struct {
	Kind string
	Then string
	Ev   *Evidence
}

// Refusal is a rejects or a declines.
type Refusal struct {
	Reason string
	To     string
	Ev     *Evidence
}

type Branch struct {
	When string
	Do   string
	Ev   *Evidence
}

type Unsaid struct {
	What string
	Ev   *Evidence
}

type Acceptance struct {
	Finding string
	Reason  string
}

// kindOf says which of the three kinds of file a JSON object is, by its
// first-level keys.
func kindOf(n *node) string {
	switch {
	case n.vals["system"] != nil:
		return "definition"
	case n.vals["describes"] != nil:
		return "description"
	case n.vals["accepted"] != nil:
		return "acceptances"
	}
	return ""
}

// parseFile reads a definition file or a description file. It returns
// every reason the file is refused as JSON in the format; whether it
// makes one tree is checked afterwards, by wellFormed.
func parseFile(path string, data []byte, description bool) (*File, []string) {
	n, err := readJSON(data)
	if err != nil {
		return nil, []string{err.Error()}
	}
	want := "definition"
	if description {
		want = "description"
	}
	if got := kindOf(n); got != want {
		if got == "" {
			return nil, []string{fmt.Sprintf("it is not a %s file: it has no %q", want, keysOf[want][0])}
		}
		return nil, []string{fmt.Sprintf("it is a %s file, and a %s file was expected here", got, want)}
	}
	r := &reader{description: description}
	f := &File{Path: path, Description: description}
	r.object(n, want, "the file")
	f.Top = r.text(n, keysOf[want][0], "the file", true)

	places, _ := r.list(n, "places", "the file", true)
	for i, pn := range places {
		if p := r.place(pn, i); p != nil {
			f.Places = append(f.Places, p)
		}
	}
	elements, _ := r.list(n, "elements", "the file", true)
	for i, en := range elements {
		if e := r.element(en, i); e != nil {
			f.Elements = append(f.Elements, e)
		}
	}
	unsaid, _ := r.list(n, "unsaid", "the file", true)
	for i, un := range unsaid {
		where := fmt.Sprintf("entry %d of unsaid", i+1)
		if !r.object(un, "unsaid", where) {
			continue
		}
		f.Unsaid = append(f.Unsaid, &Unsaid{What: r.text(un, "what", where, true), Ev: r.evidence(un, where)})
	}
	return f, r.problems
}

func (r *reader) place(n *node, i int) *Place {
	where := fmt.Sprintf("place %d", i+1)
	if n.kind == 'o' {
		if v := n.vals["name"]; v != nil && v.kind == 's' && v.str != "" {
			where = fmt.Sprintf("place %q", v.str)
		}
	}
	if !r.object(n, "place", where) {
		return nil
	}
	p := &Place{
		Name:       r.text(n, "name", where, true),
		Keyed:      r.flag(n, "keyed", where),
		OnExisting: r.oneOf(n, "onExisting", where, "replace", "refuse"),
	}
	if p.Keyed && n.vals["onExisting"] == nil {
		r.bad("%s is keyed and has no \"onExisting\"", where)
	}
	if !p.Keyed && n.vals["onExisting"] != nil {
		r.bad("%s has \"onExisting\" and is not keyed", where)
	}
	if r.description {
		p.Ev = r.evidence(n, where)
		return p
	}
	p.Outside = r.flag(n, "outside", where)
	p.Lasting = r.flag(n, "lasting", where)
	p.BelongsTo = r.text(n, "belongsTo", where, false)
	if n.vals["belongsTo"] != nil && p.BelongsTo == "" {
		r.bad("%s: \"belongsTo\" is empty", where)
	}
	if (n.vals["belongsTo"] != nil) == p.Outside {
		r.bad("%s has both \"belongsTo\" and \"outside\", or neither; it has exactly one", where)
	}
	p.Invalid = r.text(n, "invalid", where, false)
	if n.vals["hold"] != nil {
		if p.OnExisting != "refuse" {
			r.bad("%s has a \"hold\", which a place has only when its \"onExisting\" is \"refuse\"", where)
		}
		p.Ev = r.evidence(n, where)
	}
	return p
}

func (r *reader) element(n *node, i int) *Element {
	where := fmt.Sprintf("element %d", i+1)
	if n.kind == 'o' {
		if v := n.vals["name"]; v != nil && v.kind == 's' && v.str != "" {
			where = fmt.Sprintf("element %q", v.str)
		}
	}
	if !r.object(n, "element", where) {
		return nil
	}
	e := &Element{Name: r.text(n, "name", where, true)}
	if r.description {
		e.Ev = r.evidence(n, where)
	}

	var shapes []string
	for _, s := range []string{"join", "include", "steps", "or"} {
		if n.vals[s] != nil {
			shapes = append(shapes, s)
		}
	}
	leaf := n.vals["reads"] != nil || n.vals["writes"] != nil
	if leaf {
		shapes = append(shapes, "leaf")
	}
	switch {
	case len(shapes) == 0:
		r.bad("%s is a leaf with no reads and no writes", where)
		e.Shape = "leaf"
	case len(shapes) > 1:
		r.bad("%s has more than one shape: %s", where, strings.Join(shapes, ", "))
		e.Shape = shapes[0]
	default:
		e.Shape = shapes[0]
	}

	if items, ok := r.list(n, "reads", where, false); ok {
		for _, it := range items {
			w := where + ", a read"
			if !r.object(it, "read", w) {
				continue
			}
			e.Reads = append(e.Reads, &Read{Place: r.text(it, "place", w, true), Ev: r.evidence(it, w)})
		}
	}
	if items, ok := r.list(n, "writes", where, false); ok {
		for _, it := range items {
			if w := r.write(it, where); w != nil {
				e.Writes = append(e.Writes, w)
			}
		}
	}
	switch e.Shape {
	case "join", "include", "steps":
		e.Children = r.names(n, e.Shape, where)
		if e.Shape == "join" && len(e.Children) == 1 {
			r.bad("%s is a join of one child; a join has at least two", where)
		}
	case "or":
		items, _ := r.list(n, "or", where, false)
		for _, it := range items {
			w := where + ", a branch"
			if !r.object(it, "branch", w) {
				continue
			}
			b := &Branch{When: r.text(it, "when", w, true), Do: r.text(it, "do", w, true), Ev: r.evidence(it, w)}
			e.Branches = append(e.Branches, b)
			e.Children = append(e.Children, b.Do)
		}
	}
	if n.vals["otherwise"] != nil {
		e.Otherwise = r.text(n, "otherwise", where, true)
		if e.Shape != "or" {
			r.bad("%s has \"otherwise\" and is not an or", where)
		} else if e.Otherwise != "" {
			e.Children = append(e.Children, e.Otherwise)
		}
	}
	e.Rejects = r.refusal(n, "rejects", where)
	e.Declines = r.refusal(n, "declines", where)
	return e
}

func (r *reader) write(n *node, element string) *Write {
	where := element + ", a write"
	if n.kind == 'o' {
		if v := n.vals["place"]; v != nil && v.kind == 's' && v.str != "" {
			where = fmt.Sprintf("%s, the write to %q", element, v.str)
		}
	}
	if !r.object(n, "write", where) {
		return nil
	}
	w := &Write{
		Place:   r.text(n, "place", where, true),
		Mode:    r.oneOf(n, "mode", where, "create", "replace", "write"),
		Key:     r.oneOf(n, "key", where, "content", "assigned"),
		Records: r.oneOf(n, "records", where, "one"),
		Ev:      r.evidence(n, where),
	}
	if n.vals["mode"] == nil {
		r.bad("%s has no \"mode\"", where)
	}
	if items, ok := r.list(n, "before", where, false); ok {
		for _, it := range items {
			cw := where + ", a check"
			if !r.object(it, "check", cw) {
				continue
			}
			c := &Check{Kind: r.oneOf(it, "check", cw, "exists", "repeated", "belongs"), Ev: r.evidence(it, cw)}
			if it.vals["check"] == nil {
				r.bad("%s has no \"check\"", cw)
			}
			then := []string{"refuse"}
			if c.Kind == "exists" {
				then = []string{"refuse", "ask"}
			}
			c.Then = r.oneOf(it, "then", cw, then...)
			if it.vals["then"] == nil {
				r.bad("%s has no \"then\"", cw)
			}
			w.Before = append(w.Before, c)
		}
	}
	return w
}

// refusal reads a rejects or a declines. A rejects always says where it
// goes; a declines can leave that out.
func (r *reader) refusal(n *node, key, element string) *Refusal {
	v, ok := n.vals[key]
	if !ok {
		return nil
	}
	where := element + ", " + key
	if !r.object(v, key, where) {
		return nil
	}
	return &Refusal{
		Reason: r.text(v, "reason", where, true),
		To:     r.text(v, "to", where, key == "rejects"),
		Ev:     r.evidence(v, where),
	}
}

// parseAcceptances reads an acceptances file. names are the checks an
// acceptance may name.
func parseAcceptances(data []byte, names []string) ([]Acceptance, []string) {
	n, err := readJSON(data)
	if err != nil {
		return nil, []string{err.Error()}
	}
	r := &reader{}
	r.object(n, "accepted", "the file")
	items, _ := r.list(n, "accepted", "the file", true)
	var out []Acceptance
	for i, it := range items {
		where := fmt.Sprintf("acceptance %d", i+1)
		if !r.object(it, "acceptance", where) {
			continue
		}
		a := Acceptance{Finding: r.text(it, "finding", where, true), Reason: r.text(it, "reason", where, true)}
		check, _, found := strings.Cut(a.Finding, "  ")
		if a.Finding != "" && (!found || !contains(names, check) || check == "stale") {
			r.bad("%s: %q does not begin with the name of a check that can be accepted and two spaces", where, a.Finding)
		}
		out = append(out, a)
	}
	return out, r.problems
}
