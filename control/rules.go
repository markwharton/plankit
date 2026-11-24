package main

// rule is one statement of a definition file that says the system stops
// something (format.md, "Rule"), with the object it is on. Kind is
// "store", "check", "rejects", "declines" or "unsaid".
type rule struct {
	Kind    string
	Subject Subject
	Ev      *Evidence
	Place   *Place
	Elem    *Element
	Write   *Write
	Check   *Check
	Refusal *Refusal
	Unsaid  *Unsaid
}

// rules is every rule of a file, in one order: the places whose store
// refuses, then each element's checks before its writes, its rejects
// and its declines, then the entries in unsaid. held and the tests
// from the gaps both walk this list.
func rules(m *model) []rule {
	var out []rule
	for _, p := range m.f.Places {
		if p.OnExisting == "refuse" {
			out = append(out, rule{Kind: "store", Subject: sPlace(p.Name), Ev: p.Ev, Place: p})
		}
	}
	for _, e := range m.f.Elements {
		for _, w := range e.Writes {
			for _, ch := range w.Before {
				out = append(out, rule{Kind: "check", Subject: sCheck(e.Name, w.Place, ch.Kind), Ev: ch.Ev, Elem: e, Write: w, Check: ch, Place: m.place[w.Place]})
			}
		}
		if e.Rejects != nil {
			out = append(out, rule{Kind: "rejects", Subject: sRejects(e.Name), Ev: e.Rejects.Ev, Elem: e, Refusal: e.Rejects})
		}
		if e.Declines != nil {
			out = append(out, rule{Kind: "declines", Subject: sDeclines(e.Name), Ev: e.Declines.Ev, Elem: e, Refusal: e.Declines})
		}
	}
	for _, u := range m.f.Unsaid {
		out = append(out, rule{Kind: "unsaid", Subject: sUnsaid(u.What), Ev: u.Ev, Unsaid: u})
	}
	return out
}

// ownedBy reports whether an element may write a place: the place
// belongs to it or to an element above it. A place from outside the
// system is owned by no element.
func (m *model) ownedBy(element string, p *Place) bool {
	return !p.Outside && (p.BelongsTo == element || m.below(element, p.BelongsTo))
}
