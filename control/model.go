package main

import (
	"fmt"
	"sort"
)

// model is a file whose names resolve and whose elements make one tree.
type model struct {
	f      *File
	place  map[string]*Place
	elem   map[string]*Element
	parent map[string]string
}

// wellFormed builds the model, or returns every reason the file is
// refused: checks.md, "When a file is refused".
func wellFormed(f *File) (*model, []string) {
	var bad []string
	add := func(format string, a ...any) { bad = append(bad, fmt.Sprintf(format, a...)) }
	m := &model{f: f, place: map[string]*Place{}, elem: map[string]*Element{}, parent: map[string]string{}}

	for _, p := range f.Places {
		if p.Name == "" {
			continue
		}
		if p.Name == "parent" {
			add("a place is named \"parent\"")
		}
		if m.place[p.Name] != nil {
			add("two places are named %q", p.Name)
		}
		m.place[p.Name] = p
	}
	for _, e := range f.Elements {
		if e.Name == "" {
			continue
		}
		if m.elem[e.Name] != nil {
			add("two elements are named %q", e.Name)
		}
		m.elem[e.Name] = e
	}
	seen := map[string]bool{}
	for _, u := range f.Unsaid {
		if u.What != "" && seen[u.What] {
			add("two entries in unsaid have the same \"what\": %q", u.What)
		}
		seen[u.What] = true
	}

	// Every name that is used is listed in this file.
	topKey := "system"
	if f.Description {
		topKey = "describes"
	}
	if f.Top != "" && m.elem[f.Top] == nil {
		add("%q names %q, which is not an element in this file", topKey, f.Top)
	}
	for _, p := range f.Places {
		if p.BelongsTo != "" && m.elem[p.BelongsTo] == nil {
			add("place %q belongs to %q, which is not an element in this file", p.Name, p.BelongsTo)
		}
	}
	for _, e := range f.Elements {
		for _, c := range e.Children {
			if c != "" && m.elem[c] == nil {
				add("element %q names the child %q, which is not an element in this file", e.Name, c)
			}
		}
		for _, r := range e.Reads {
			if r.Place != "" && m.place[r.Place] == nil {
				add("element %q reads %q, which is not a place in this file", e.Name, r.Place)
			}
		}
		for _, w := range e.Writes {
			if w.Place != "" && m.place[w.Place] == nil {
				add("element %q writes %q, which is not a place in this file", e.Name, w.Place)
			}
		}
		for word, r := range map[string]*Refusal{"rejects": e.Rejects, "declines": e.Declines} {
			if r != nil && r.To != "" && r.To != "parent" && m.place[r.To] == nil {
				add("element %q: the \"to\" of its %s is %q, which is not a place in this file", e.Name, word, r.To)
			}
		}
	}
	if len(bad) > 0 {
		sort.Strings(bad)
		return nil, bad
	}

	// The elements make one tree under the top element.
	for _, e := range f.Elements {
		for _, c := range e.Children {
			if p, has := m.parent[c]; has && p != e.Name {
				add("element %q is the child of two elements, %q and %q", c, p, e.Name)
			}
			m.parent[c] = e.Name
		}
		twice := map[string]bool{}
		for _, b := range e.Branches {
			if twice[b.Do] {
				add("element %q has two branches with the same child, %q", e.Name, b.Do)
			}
			twice[b.Do] = true
		}
		if e.Otherwise != "" && twice[e.Otherwise] {
			add("element %q has a branch and an \"otherwise\" with the same child, %q", e.Name, e.Otherwise)
		}
		if len(e.Children) > 1 && e.Shape != "or" {
			dup := map[string]bool{}
			for _, c := range e.Children {
				if dup[c] {
					add("element %q lists the child %q twice", e.Name, c)
				}
				dup[c] = true
			}
		}
	}
	for _, e := range f.Elements {
		// Walk up from each element; meeting it again means it is below itself.
		at, steps := e.Name, 0
		for m.parent[at] != "" && steps <= len(f.Elements) {
			at = m.parent[at]
			steps++
			if at == e.Name {
				add("element %q is below itself", e.Name)
				break
			}
		}
	}
	if len(bad) > 0 {
		sort.Strings(bad)
		return nil, bad
	}
	for _, e := range f.Elements {
		if e.Name != f.Top && !m.below(e.Name, f.Top) {
			if f.Description {
				add("element %q is not the described element and is not below it", e.Name)
			} else {
				add("element %q is not the top element and is not below it", e.Name)
			}
		}
		switch {
		case f.Description && e.Shape == "steps":
			add("element %q has \"steps\", which only the top element of a definition file has", e.Name)
		case !f.Description && e.Shape == "steps" && e.Name != f.Top:
			add("element %q has \"steps\" and is not the top element", e.Name)
		case !f.Description && e.Name == f.Top && e.Shape != "steps":
			add("the top element %q has no \"steps\"; it has another shape", e.Name)
		}
	}
	if !f.Description && m.parent[f.Top] != "" {
		add("the top element %q is the child of %q", f.Top, m.parent[f.Top])
	}

	// Each field holds a value its table allows.
	for _, e := range f.Elements {
		reads := map[string]bool{}
		for _, r := range e.Reads {
			if reads[r.Place] {
				add("element %q reads %q twice", e.Name, r.Place)
			}
			reads[r.Place] = true
		}
		writes := map[string]bool{}
		for _, w := range e.Writes {
			if writes[w.Place] {
				add("element %q writes %q twice", e.Name, w.Place)
			}
			writes[w.Place] = true
			p := m.place[w.Place]
			if p == nil {
				continue
			}
			where := fmt.Sprintf("element %q, the write to %q", e.Name, w.Place)
			if p.Keyed {
				if w.Mode == "write" {
					add("%s: the place is keyed, so its \"mode\" is \"create\" or \"replace\"", where)
				}
				if w.Key == "" {
					add("%s: the place is keyed, and the write has no \"key\"", where)
				}
			} else {
				if w.Mode == "create" || w.Mode == "replace" {
					add("%s: the place is not keyed, so its \"mode\" is \"write\"", where)
				}
				if w.Key != "" || w.Records != "" || len(w.Before) > 0 {
					add("%s: the place is not keyed, so the write has no \"key\", \"records\" or \"before\"", where)
				}
			}
			kinds := map[string]bool{}
			for _, c := range w.Before {
				if kinds[c.Kind] {
					add("%s has two %q checks", where, c.Kind)
				}
				kinds[c.Kind] = true
			}
		}
		for word, r := range map[string]*Refusal{"rejects": e.Rejects, "declines": e.Declines} {
			if r == nil || r.To == "" {
				continue
			}
			if r.To == "parent" {
				if !f.Description && e.Name == f.Top {
					add("the top element %q: the \"to\" of its %s is \"parent\", and it has no parent", e.Name, word)
				}
				continue
			}
			if !m.writesOf(e.Name)[r.To] {
				add("element %q: the \"to\" of its %s is %q, which neither it nor an element below it writes", e.Name, word, r.To)
			}
		}
	}
	if len(bad) > 0 {
		sort.Strings(bad)
		return nil, bad
	}
	return m, nil
}

// below reports whether name is below top in the tree.
func (m *model) below(name, top string) bool {
	for at, steps := m.parent[name], 0; at != "" && steps <= len(m.f.Elements); at, steps = m.parent[at], steps+1 {
		if at == top {
			return true
		}
	}
	return false
}

// under is the element and every element below it, in the file's order.
func (m *model) under(name string) []*Element {
	var out []*Element
	for _, e := range m.f.Elements {
		if e.Name == name || m.below(e.Name, name) {
			out = append(out, e)
		}
	}
	return out
}

// leaves are every leaf of the file, in the file's order.
func (m *model) leaves() []*Element {
	var out []*Element
	for _, e := range m.f.Elements {
		if e.Shape == "leaf" {
			out = append(out, e)
		}
	}
	return out
}

// readsOf is every place the element or an element below it reads.
func (m *model) readsOf(name string) map[string]bool {
	set := map[string]bool{}
	for _, e := range m.under(name) {
		for _, r := range e.Reads {
			set[r.Place] = true
		}
	}
	return set
}

// writesOf is every place the element or an element below it writes.
func (m *model) writesOf(name string) map[string]bool {
	set := map[string]bool{}
	for _, e := range m.under(name) {
		for _, w := range e.Writes {
			set[w.Place] = true
		}
	}
	return set
}

// nearest is the nearest element above both a and b.
func (m *model) nearest(a, b string) string {
	above := map[string]bool{}
	for at := m.parent[a]; at != ""; at = m.parent[at] {
		above[at] = true
	}
	for at := m.parent[b]; at != ""; at = m.parent[at] {
		if above[at] {
			return at
		}
	}
	return ""
}

func (e *Element) reads(place string) bool {
	for _, r := range e.Reads {
		if r.Place == place {
			return true
		}
	}
	return false
}

func (w *Write) check(kind string) *Check {
	for _, c := range w.Before {
		if c.Kind == kind {
			return c
		}
	}
	return nil
}
