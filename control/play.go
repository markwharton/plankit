package main

// Playing a run: view.md, "Playing a run", gives the rules. play is the
// only code that applies them; the sweep, the tests from the gaps and
// the view read the traces it returns.

// The states a place can hold, the outcomes of a step, and the verdicts
// on a fault, in the order view.md's tables give them.
var (
	playStates = []string{"given", "present", "empty", "written", "invalid", "refusal"}
	outcomes   = []string{"done", "refused", "passed-on", "not-run"}
	verdicts   = []string{"refused", "lost", "output", "passed", "unread"}
)

// Break is what a play changes: a place that holds its invalid value,
// or a leaf made to refuse or to be declined whatever it reads.
type Break struct {
	Place    string `json:"place,omitempty"`
	Refuses  string `json:"refuses,omitempty"`
	Declines string `json:"declines,omitempty"`
}

// Pick is the child an or runs on one path.
type Pick struct {
	Or    string `json:"or"`
	Child string `json:"child"`
}

// Seen is a place and the state it holds.
type Seen struct {
	Place string `json:"place"`
	State string `json:"state"`
}

// StepTrace is one leaf of a play: what it read, what it wrote, and its
// outcome.
type StepTrace struct {
	Element string `json:"element"`
	Outcome string `json:"outcome"`
	Reads   []Seen `json:"reads"`
	Writes  []Seen `json:"writes"`
}

// Climb is the path of one refusal or decline: the leaf it leaves, the
// elements that pass it on or report it, and the place it reaches, or
// the element where it stops because that element declares no refusal.
type Climb struct {
	Kind    string   `json:"kind"`
	From    string   `json:"from"`
	Through []string `json:"through"`
	To      string   `json:"to,omitempty"`
	LostAt  string   `json:"lostAt,omitempty"`
}

// reached is every element a climb stops: the leaf, the elements it
// passes through, and the element where it is lost.
func (c Climb) reached() []string {
	out := append([]string{c.From}, c.Through...)
	if c.LostAt != "" {
		out = append(out, c.LostAt)
	}
	return out
}

// Trace is one play of a run on one path.
type Trace struct {
	Run      string      `json:"run"`
	Break    Break       `json:"break"`
	Path     []Pick      `json:"path"`
	Start    []Seen      `json:"start"`
	Steps    []StepTrace `json:"steps"`
	End      []Seen      `json:"end"`
	Refusals []Climb     `json:"refusals"`
}

// paths is every choice of branches below an element: one list of
// picks for each path through its ors, in the order of their children.
func (m *model) paths(name string) [][]Pick {
	e := m.elem[name]
	switch e.Shape {
	case "leaf":
		return [][]Pick{{}}
	case "or":
		var out [][]Pick
		for _, c := range e.Children {
			for _, p := range m.paths(c) {
				out = append(out, append([]Pick{{Or: name, Child: c}}, p...))
			}
		}
		return out
	}
	out := [][]Pick{{}}
	for _, c := range e.Children {
		var next [][]Pick
		for _, head := range out {
			for _, tail := range m.paths(c) {
				next = append(next, append(append([]Pick{}, head...), tail...))
			}
		}
		out = next
	}
	return out
}

// touched is every place the leaves under an element read or write, in
// the file's order.
func (m *model) touched(name string) []string {
	reads, writes := m.readsOf(name), m.writesOf(name)
	var out []string
	for _, p := range m.f.Places {
		if reads[p.Name] || writes[p.Name] {
			out = append(out, p.Name)
		}
	}
	return out
}

type cell struct {
	state string
	bad   bool
}

func (c cell) seen() string {
	if c.bad && c.state != "empty" {
		return "invalid"
	}
	return c.state
}

type player struct {
	m     *model
	brk   Break
	pick  map[string]string
	cells map[string]*cell
	t     *Trace
}

// play runs one element of the top element with one break, on one path.
func play(m *model, run string, brk Break, path []Pick) Trace {
	t := &Trace{Run: run, Break: brk, Path: append([]Pick{}, path...), Start: []Seen{}, Steps: []StepTrace{}, End: []Seen{}, Refusals: []Climb{}}
	p := &player{m: m, brk: brk, pick: map[string]string{}, cells: map[string]*cell{}, t: t}
	for _, pk := range path {
		p.pick[pk.Or] = pk.Child
	}
	for _, pl := range m.f.Places {
		c := &cell{state: "empty"}
		switch {
		case pl.Outside:
			c.state = "given"
		case pl.Lasting:
			c.state = "present"
		}
		c.bad = pl.Name == brk.Place && c.state != "empty"
		p.cells[pl.Name] = c
	}
	places := m.touched(run)
	for _, name := range places {
		p.t.Start = append(p.t.Start, Seen{name, p.cells[name].seen()})
	}
	p.exec(run)
	for _, name := range places {
		p.t.End = append(p.t.End, Seen{name, p.cells[name].seen()})
	}
	return *p.t
}

// exec runs an element and returns the climb of a refusal that reached
// it, or nil. An element a climb reaches stops.
func (p *player) exec(name string) *Climb {
	e := p.m.elem[name]
	switch e.Shape {
	case "leaf":
		return p.leaf(e)
	case "or":
		return p.through(name, p.exec(p.pick[name]))
	case "include":
		var first *Climb
		for _, c := range e.Children {
			if got := p.through(name, p.exec(c)); got != nil && first == nil {
				first = got
			}
		}
		return first
	}
	// A join, or the steps of the top element.
	for i, c := range e.Children {
		if got := p.through(name, p.exec(c)); got != nil {
			for _, rest := range e.Children[i+1:] {
				p.notRun(rest)
			}
			return got
		}
	}
	return nil
}

// through passes a climb on when it reached this element as well.
func (p *player) through(name string, c *Climb) *Climb {
	if c == nil {
		return nil
	}
	for _, r := range c.reached() {
		if r == name {
			return c
		}
	}
	return nil
}

// notRun records every leaf under an element, on this path, as not run.
func (p *player) notRun(name string) {
	e := p.m.elem[name]
	switch e.Shape {
	case "leaf":
		p.t.Steps = append(p.t.Steps, StepTrace{Element: name, Outcome: "not-run", Reads: []Seen{}, Writes: []Seen{}})
	case "or":
		p.notRun(p.pick[name])
	default:
		for _, c := range e.Children {
			p.notRun(c)
		}
	}
}

func (p *player) leaf(e *Element) *Climb {
	st := StepTrace{Element: e.Name, Reads: []Seen{}, Writes: []Seen{}}
	invalid := false
	for _, r := range e.Reads {
		s := p.cells[r.Place].seen()
		st.Reads = append(st.Reads, Seen{r.Place, s})
		invalid = invalid || s == "invalid"
	}
	kind, refusal := "", (*Refusal)(nil)
	switch {
	case p.brk.Declines == e.Name && e.Declines != nil:
		kind, refusal = "declines", e.Declines
	case (invalid || p.brk.Refuses == e.Name) && e.Rejects != nil:
		kind, refusal = "rejects", e.Rejects
	}
	if refusal != nil {
		st.Outcome = "refused"
		p.t.Steps = append(p.t.Steps, st)
		c := p.climb(e.Name, kind, refusal)
		if c.To != "" {
			p.cells[c.To] = &cell{state: "refusal"}
		}
		p.t.Refusals = append(p.t.Refusals, c)
		return &c
	}
	st.Outcome = "done"
	if invalid {
		st.Outcome = "passed-on"
	}
	for _, w := range e.Writes {
		c := &cell{state: "written", bad: invalid || w.Place == p.brk.Place}
		p.cells[w.Place] = c
		st.Writes = append(st.Writes, Seen{w.Place, c.seen()})
	}
	p.t.Steps = append(p.t.Steps, st)
	return nil
}

// climb follows a refusal up the tree: each "parent" passes it to the
// element above, which passes it on or reports it by its own refusal of
// the same kind, until a "to" names a place or an element has none.
func (p *player) climb(from, kind string, r *Refusal) Climb {
	c := Climb{Kind: kind, From: from, Through: []string{}}
	at, to := from, r.To
	for to == "parent" {
		above := p.m.elem[p.m.parent[at]]
		next := above.Rejects
		if kind == "declines" {
			next = above.Declines
		}
		if next == nil {
			c.LostAt = above.Name
			return c
		}
		c.Through = append(c.Through, above.Name)
		at, to = above.Name, next.To
	}
	c.To = to
	return c
}

// verdict decides a fault from its trace and the places the run reads:
// view.md, "Verdicts".
func (m *model) verdict(t Trace) string {
	if len(t.Refusals) > 0 {
		if t.Refusals[0].LostAt != "" {
			return "lost"
		}
		return "refused"
	}
	if !m.readsOf(t.Run)[t.Break.Place] {
		return "output"
	}
	for _, s := range t.Steps {
		if s.Outcome == "passed-on" {
			return "passed"
		}
	}
	return "unread"
}
