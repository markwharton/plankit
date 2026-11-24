package main

import (
	"fmt"
	"strings"
)

const (
	onlyDefinition  = "in the definition file only"
	onlyDescription = "in the description file only"
)

// follows compares a description file with the definition file:
// checks.md, "Does the code follow the design?".
func follows(def, desc *model, file int, c *collector) {
	add := func(subject Subject, format string, a ...any) { c.add("follows", file, file, subject, format, a...) }

	top := def.elem[desc.f.Top]
	switch {
	case top == nil:
		add(sElement(desc.f.Top), "the definition file has no element of this name")
		return
	case def.parent[top.Name] != def.f.Top:
		add(sElement(desc.f.Top), "the element is not a child of the top element, %s", def.f.Top)
		return
	}

	for _, p := range desc.f.Places {
		d := def.place[p.Name]
		if d == nil {
			add(sPlace(p.Name), onlyDescription)
			continue
		}
		if d.Keyed != p.Keyed {
			add(sPlace(p.Name), "keyed is %v in the definition file and %v in the description file", d.Keyed, p.Keyed)
		}
		if d.OnExisting != p.OnExisting {
			add(sPlace(p.Name), "onExisting is %s in the definition file and %s in the description file", orNone(d.OnExisting), orNone(p.OnExisting))
		}
	}

	// In each file, only the described element and the elements below it.
	inDef := map[string]*Element{}
	for _, e := range def.under(top.Name) {
		inDef[e.Name] = e
	}
	inDesc := map[string]*Element{}
	for _, e := range desc.f.Elements {
		inDesc[e.Name] = e
	}
	// An element in one file only is reported once, at the highest
	// element that the other file lacks.
	for _, e := range def.under(top.Name) {
		if inDesc[e.Name] == nil && inDesc[def.parent[e.Name]] != nil {
			add(sElement(e.Name), onlyDefinition)
		}
	}
	for _, e := range desc.f.Elements {
		if inDef[e.Name] == nil && (e.Name == desc.f.Top || inDef[desc.parent[e.Name]] != nil) {
			add(sElement(e.Name), onlyDescription)
		}
	}

	for _, d := range def.under(top.Name) {
		s := inDesc[d.Name]
		if s == nil {
			continue
		}
		if why := shapeDiffers(d, s); why != "" {
			add(sElement(d.Name), "%s", why)
		}
		compareRefusal(add, sRejects(d.Name), d.Rejects, s.Rejects)
		compareRefusal(add, sDeclines(d.Name), d.Declines, s.Declines)
		if d.Shape != "leaf" || s.Shape != "leaf" {
			continue
		}
		for _, r := range d.Reads {
			if !s.reads(r.Place) {
				add(sRead(d.Name, r.Place), onlyDefinition)
			}
		}
		for _, r := range s.Reads {
			if !d.reads(r.Place) {
				add(sRead(d.Name, r.Place), onlyDescription)
			}
		}
		for _, dw := range d.Writes {
			sw := s.write(dw.Place)
			if sw == nil {
				add(sWrite(d.Name, dw.Place), onlyDefinition)
				continue
			}
			for _, f := range []struct{ name, def, desc string }{
				{"mode", dw.Mode, sw.Mode}, {"key", dw.Key, sw.Key}, {"records", dw.Records, sw.Records},
			} {
				if f.def != f.desc {
					add(sWrite(d.Name, dw.Place), "%s is %s in the definition file and %s in the description file", f.name, orNone(f.def), orNone(f.desc))
				}
			}
			for _, dc := range dw.Before {
				sc := sw.check(dc.Kind)
				switch {
				case sc == nil:
					add(sCheck(d.Name, dw.Place, dc.Kind), onlyDefinition)
				case sc.Then != dc.Then:
					add(sCheck(d.Name, dw.Place, dc.Kind), "then is %s in the definition file and %s in the description file", dc.Then, sc.Then)
				}
			}
			for _, sc := range sw.Before {
				if dw.check(sc.Kind) == nil {
					add(sCheck(d.Name, dw.Place, sc.Kind), onlyDescription)
				}
			}
		}
		for _, sw := range s.Writes {
			if d.write(sw.Place) == nil {
				add(sWrite(d.Name, sw.Place), onlyDescription)
			}
		}
	}
}

// undescribed reports, once for the run, each child of the top element
// that no description file describes.
func undescribed(def *model, described map[string]bool, order int, c *collector) {
	for _, child := range def.elem[def.f.Top].Children {
		if !described[child] {
			c.add("follows", -1, order, sElement(child), "no description file for this element was given")
		}
	}
}

// shapeDiffers says how two elements of the same name differ in shape
// or in children, or "" when they do not.
func shapeDiffers(d, s *Element) string {
	if d.Shape != s.Shape {
		return fmt.Sprintf("it is %s in the definition file and %s in the description file", article(d.Shape), article(s.Shape))
	}
	switch d.Shape {
	case "join":
		if strings.Join(d.Children, "\x00") != strings.Join(s.Children, "\x00") {
			return fmt.Sprintf("its children are %s in the definition file and %s in the description file", strings.Join(d.Children, ", "), strings.Join(s.Children, ", "))
		}
	case "include", "or":
		if d.Otherwise != s.Otherwise {
			return fmt.Sprintf("its otherwise is %s in the definition file and %s in the description file", orNone(d.Otherwise), orNone(s.Otherwise))
		}
		dc, sc := set(branchChildren(d)), set(branchChildren(s))
		if strings.Join(sortedKeys(dc), "\x00") != strings.Join(sortedKeys(sc), "\x00") {
			return fmt.Sprintf("its children are %s in the definition file and %s in the description file", strings.Join(sortedKeys(dc), ", "), strings.Join(sortedKeys(sc), ", "))
		}
	}
	return ""
}

// branchChildren is the children an include lists, or the children of
// an or's branches, without its otherwise.
func branchChildren(e *Element) []string {
	if e.Shape != "or" {
		return e.Children
	}
	var out []string
	for _, b := range e.Branches {
		out = append(out, b.Do)
	}
	return out
}

func compareRefusal(add func(subject Subject, format string, a ...any), subject Subject, d, s *Refusal) {
	switch {
	case d == nil && s == nil:
	case s == nil:
		add(subject, onlyDefinition)
	case d == nil:
		add(subject, onlyDescription)
	case d.To != s.To:
		add(subject, "to is %s in the definition file and %s in the description file", orNone(d.To), orNone(s.To))
	}
}

func (e *Element) write(place string) *Write {
	for _, w := range e.Writes {
		if w.Place == place {
			return w
		}
	}
	return nil
}

func set(list []string) map[string]bool {
	m := map[string]bool{}
	for _, s := range list {
		m[s] = true
	}
	return m
}

func orNone(s string) string {
	if s == "" {
		return "left out"
	}
	return s
}

func article(shape string) string {
	switch shape {
	case "include", "or":
		return "an " + shape
	case "steps":
		return "the top element"
	}
	return "a " + shape
}
