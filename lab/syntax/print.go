package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Print writes a design as text in a form. Every key of every object is written, or the design is
// refused with the key named; so is anything the text could not carry back the same.
func Print(f Form, doc *Obj) (out string, err error) {
	p := &printer{f: f, seen: map[*Obj]map[string]bool{}}
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(refusal); ok {
				err = e
				return
			}
			panic(r)
		}
	}()
	p.design(doc)
	return strings.Join(p.lines, "\n") + "\n", nil
}

type refusal struct{ msg string }

func (r refusal) Error() string { return r.msg }

func refuse(format string, a ...any) { panic(refusal{fmt.Sprintf(format, a...)}) }

type printer struct {
	f     Form
	lines []string
	seen  map[*Obj]map[string]bool
}

func (p *printer) line(depth int, s string) {
	p.lines = append(p.lines, strings.Repeat("    ", depth)+s)
}

func (p *printer) kw(words ...string) string {
	out := make([]string, len(words))
	for i, w := range words {
		out[i] = p.f.kw(w)
	}
	return strings.Join(out, " ")
}

// take is a key's value, marked as written.
func (p *printer) take(o *Obj, k string) (any, bool) {
	v, ok := o.Get(k)
	if ok {
		if p.seen[o] == nil {
			p.seen[o] = map[string]bool{}
		}
		p.seen[o][k] = true
	}
	return v, ok
}

func (p *printer) str(o *Obj, k string) string {
	v, _ := p.take(o, k)
	s, ok := v.(string)
	if !ok {
		refuse("%q is a string", k)
	}
	return s
}

// done refuses an object with a key nothing wrote.
func (p *printer) done(o *Obj, where string) {
	for _, k := range o.Keys() {
		if !p.seen[o][k] {
			refuse("%s has %q, which the text form does not write", where, k)
		}
	}
}

func (p *printer) name(n string) string {
	if n == "" || n != strings.TrimSpace(n) || strings.Contains(n, "  ") {
		refuse("the name %q has spaces the text form would not keep", n)
	}
	if strings.ContainsAny(n, "=()[]\t\n") {
		refuse("the name %q holds one of = ( ) [ ], a tab or a line break, which a name may not hold", n)
	}
	if p.f == B {
		return "[" + n + "]"
	}
	for _, w := range strings.Fields(n) {
		if k, ok := A.keyword(w); ok {
			refuse("the name %q holds the keyword %s: form A cannot tell it from the keyword; write it in lower case", n, A.kw(k))
		}
	}
	return n
}

func (p *printer) text(s string) string {
	if s != strings.TrimSpace(s) || strings.Contains(s, "\n") {
		refuse("the text %q has spaces or a line break the text form would not keep", s)
	}
	return s
}

func list(v any) []string {
	switch x := v.(type) {
	case string:
		return []string{x}
	case []any:
		out := make([]string, len(x))
		for i, e := range x {
			s, ok := e.(string)
			if !ok {
				refuse("a list of names holds %v", e)
			}
			out[i] = s
		}
		return out
	}
	refuse("a name, or a list of names, was expected; got %v", v)
	return nil
}

// names writes head and a list of names: on the line in form B, or when there is one;
// one to a line below it in form A, since a name may hold a comma.
func (p *printer) names(depth int, head string, v any) {
	ns := list(v)
	if len(ns) == 0 {
		refuse("%s has an empty list", head)
	}
	if p.f == B || len(ns) == 1 {
		out := make([]string, len(ns))
		for i, n := range ns {
			out[i] = p.name(n)
		}
		p.line(depth, head+" "+strings.Join(out, ", "))
		return
	}
	p.line(depth, head)
	for _, n := range ns {
		p.line(depth+1, p.name(n))
	}
}

func (p *printer) cond(c any, inner bool) string {
	switch x := c.(type) {
	case string:
		return p.name(x)
	case *Obj:
		if len(x.Keys()) == 1 {
			k := x.Keys()[0]
			v, _ := x.Get(k)
			if k == "not" {
				return p.kw("not") + " " + p.cond(v, true)
			}
			if l, ok := v.([]any); ok && (k == "all" || k == "any") {
				if len(l) < 2 {
					refuse("a condition that joins fewer than two facts is written as the fact itself")
				}
				parts := make([]string, len(l))
				for i, e := range l {
					parts[i] = p.cond(e, true)
				}
				s := strings.Join(parts, " "+p.kw(map[string]string{"all": "and", "any": "or"}[k])+" ")
				if inner {
					return "(" + s + ")"
				}
				return s
			}
		}
	}
	refuse("a condition is a fact's name, not, all or any; got %v", c)
	return ""
}

// long is a condition that joins more than three facts at its top: it is written as a block.
func long(c any) (string, []any, bool) {
	o, ok := c.(*Obj)
	if !ok || len(o.Keys()) != 1 {
		return "", nil, false
	}
	k := o.Keys()[0]
	v, _ := o.Get(k)
	l, ok := v.([]any)
	if !ok || (k != "all" && k != "any") || len(l) <= 3 {
		return "", nil, false
	}
	return k, l, true
}

// attrs is a line with conditions: short ones go on it, and a long one goes below it as a block.
type attrs struct {
	head   string
	blocks [][3]any // key, all or any, the facts
}

func (p *printer) on(a *attrs, key string, c any, block bool) {
	if k, l, ok := long(c); ok && block {
		a.blocks = append(a.blocks, [3]any{key, k, l})
		return
	}
	a.head += " " + p.kw(key) + " " + p.cond(c, false)
}

func (p *printer) write(depth int, a *attrs) {
	p.line(depth, a.head)
	for _, b := range a.blocks {
		p.line(depth+1, p.kw(b[0].(string), b[1].(string)))
		for _, e := range b[2].([]any) {
			p.line(depth+2, p.cond(e, false))
		}
	}
}

// details writes says, because and hold: the lines every statement can carry.
func (p *printer) details(depth int, o *Obj) {
	if v, ok := p.take(o, "says"); ok {
		for _, s := range list(v) {
			p.line(depth, strings.TrimRight(p.kw("says")+" "+p.text(s), " "))
		}
	}
	if v, ok := p.take(o, "because"); ok {
		s, isString := v.(string)
		if !isString {
			refuse("\"because\" is one sentence")
		}
		p.line(depth, strings.TrimRight(p.kw("because")+" "+p.text(s), " "))
	}
	if v, ok := p.take(o, "hold"); ok {
		h, isObj := v.(*Obj)
		if !isObj {
			refuse("a hold is an object with \"file\" and \"test\"")
		}
		file, test := p.str(h, "file"), p.str(h, "test")
		if strings.ContainsAny(file+test, " \t") || file == "" || test == "" {
			refuse("a hold's file and test are each one word")
		}
		p.done(h, "a hold")
		p.line(depth, p.kw("hold")+" "+file+" "+test)
	}
}

func (p *printer) objs(v any, what string) []*Obj {
	l, ok := v.([]any)
	if !ok {
		refuse("%s is a list", what)
	}
	out := make([]*Obj, len(l))
	for i, e := range l {
		o, ok := e.(*Obj)
		if !ok {
			refuse("%s holds %v, where an object was expected", what, e)
		}
		out[i] = o
	}
	return out
}

// word is a value that is one of a key's words, as the form writes it.
func (p *printer) word(o *Obj, k string, allowed ...string) string {
	v := p.str(o, k)
	for _, w := range allowed {
		if v == w {
			return p.kw(w)
		}
	}
	refuse("%q is one of %s; got %q", k, strings.Join(allowed, ", "), v)
	return ""
}

var stateWords = []string{"untouched", "created", "replaced", "destroyed", "undone", "changed", "unchanged"}

func (p *printer) yes(o *Obj, k string) bool {
	v, ok := p.take(o, k)
	if !ok {
		return false
	}
	if v != true {
		refuse("%q is written only as true", k)
	}
	return true
}

func (p *printer) design(d *Obj) {
	p.line(0, p.kw("system")+" "+p.text(p.str(d, "system")))
	if v, ok := p.take(d, "on"); ok {
		for _, path := range list(v) {
			p.line(0, p.kw("on")+" "+path)
		}
	}
	if _, ok := d.Get("reports"); ok {
		p.line(0, p.kw("reports")+" "+p.name(p.str(d, "reports")))
	}
	if v, ok := p.take(d, "faults"); ok {
		n, isNumber := v.(json.Number)
		if _, err := n.Int64(); !isNumber || err != nil {
			refuse("\"faults\" is a whole number")
		}
		p.line(0, p.kw("faults")+" "+string(n))
	}
	runs, _ := p.take(d, "runs")
	p.names(0, p.kw("runs"), runs)
	p.line(0, "")
	if v, ok := p.take(d, "places"); ok {
		for _, pl := range p.objs(v, "places") {
			p.place(pl)
		}
	}
	p.line(0, "")
	if v, ok := p.take(d, "facts"); ok {
		for _, f := range p.objs(v, "facts") {
			p.fact(f)
		}
	}
	parts, _ := p.take(d, "parts")
	po, ok := parts.(*Obj)
	if !ok {
		refuse("\"parts\" is an object")
	}
	for _, k := range po.Keys() {
		v, _ := po.Get(k)
		o, ok := v.(*Obj)
		if !ok {
			refuse("part %q is an object", k)
		}
		p.line(0, "")
		p.part(0, k, o)
	}
	if v, ok := p.take(d, "expects"); ok {
		for _, e := range p.objs(v, "expects") {
			p.line(0, "")
			p.expect(e)
		}
	}
	if v, ok := p.take(d, "rules"); ok {
		for _, r := range p.objs(v, "rules") {
			p.line(0, "")
			p.rule(r)
		}
	}
	p.done(d, "the design")
}

func (p *printer) place(pl *Obj) {
	head := p.kw("place") + " " + p.name(p.str(pl, "name"))
	for _, k := range []string{"is", "kind"} {
		if _, ok := pl.Get(k); ok {
			head += " " + p.word(pl, k, map[string][]string{"is": {"one", "many", "series"}, "kind": {"outside", "lasting", "run", "report"}}[k]...)
		}
	}
	if _, ok := pl.Get("in"); ok {
		head += " " + p.kw("in") + " " + p.name(p.str(pl, "in"))
	}
	if p.yes(pl, "given") {
		head += " " + p.kw("given")
	}
	if _, ok := pl.Get("owner"); ok {
		head += " " + p.kw("owner") + " " + p.name(p.str(pl, "owner"))
	}
	p.line(0, head)
	p.details(1, pl)
	p.done(pl, "place "+fmt.Sprint(pl.vals["name"]))
}

func (p *printer) fact(f *Obj) {
	kind := "fact"
	if p.yes(f, "fault") {
		kind = "fault"
	}
	if p.yes(f, "open") {
		if kind == "fault" {
			refuse("a fact is not both open and a fault")
		}
		kind = "open"
	}
	a := &attrs{head: p.kw(kind) + " " + p.name(p.str(f, "name"))}
	if _, ok := f.Get("about"); ok {
		a.head += " " + p.kw("about") + " " + p.name(p.str(f, "about"))
	}
	for _, k := range []string{"only", "means"} {
		if v, ok := p.take(f, k); ok {
			p.on(a, k, v, true)
		}
	}
	p.write(0, a)
	if v, ok := p.take(f, "at"); ok {
		p.names(1, p.kw("at"), v)
	}
	p.details(1, f)
	p.done(f, "fact "+fmt.Sprint(f.vals["name"]))
}

func kindOf(o *Obj) string {
	for _, k := range []string{"do", "seq", "all", "choose", "use"} {
		if _, ok := o.Get(k); ok {
			return k
		}
	}
	return ""
}

// part writes a part: key is its name in "parts", or "" for one written in place.
func (p *printer) part(depth int, key string, o *Obj) {
	switch kind := kindOf(o); kind {
	case "do":
		p.action(depth, key, o)
	case "seq", "all", "choose":
		head := p.kw(kind)
		if v, ok := p.take(o, "name"); ok {
			if key != "" && v != key {
				refuse("part %q is named %q: a part in \"parts\" is named by its key", key, v)
			}
			head += " " + p.name(v.(string))
		} else if key != "" {
			head += " " + p.name(key)
		}
		a := &attrs{head: head}
		if v, ok := p.take(o, "when"); ok {
			p.on(a, "when", v, false)
		}
		p.write(depth, a)
		v, _ := p.take(o, kind)
		if kind == "choose" {
			for _, b := range p.objs(v, "a choice") {
				w, _ := p.take(b, "when")
				run, _ := p.take(b, "run")
				p.done(b, "a branch")
				p.line(depth+1, p.kw("when")+" "+p.cond(w, false))
				p.ref(depth+2, run)
			}
			if other, ok := p.take(o, "otherwise"); ok {
				p.line(depth+1, p.kw("otherwise"))
				p.ref(depth+2, other)
			}
		} else {
			l, ok := v.([]any)
			if !ok {
				refuse("%q is a list of parts", kind)
			}
			for _, c := range l {
				p.ref(depth+1, c)
			}
		}
		p.answers(depth+1, o)
		p.details(depth+1, o)
		p.done(o, kind+" "+key)
	default:
		refuse("part %q is not an action, a sequence, parts in no order or a choice", key)
	}
}

// ref writes a part inside another: a name, a use of a part, nothing, or a part written in place.
func (p *printer) ref(depth int, v any) {
	switch x := v.(type) {
	case nil:
		p.line(depth, p.kw("nothing"))
	case string:
		p.line(depth, p.name(x))
	case *Obj:
		if kindOf(x) != "use" {
			p.part(depth, "", x)
			return
		}
		if len(x.Keys()) == 1 {
			refuse("a use with nothing but its part is written as the part's name")
		}
		a := &attrs{head: p.name(p.str(x, "use"))}
		if _, ok := x.Get("name"); ok {
			a.head += " " + p.kw("as") + " " + p.name(p.str(x, "name"))
		}
		if w, ok := p.take(x, "when"); ok {
			p.on(a, "when", w, false)
		}
		p.write(depth, a)
		p.details(depth+1, x)
		p.done(x, "a use of "+fmt.Sprint(x.vals["use"]))
	default:
		refuse("a part was expected; got %v", v)
	}
}

func (p *printer) action(depth int, key string, a *Obj) {
	do := p.str(a, "do")
	at := &attrs{head: p.kw("action") + " " + p.name(do)}
	if key != "" && key != do {
		at.head += " " + p.kw("as") + " " + p.name(key)
	}
	if v, ok := p.take(a, "when"); ok {
		p.on(at, "when", v, true)
	}
	p.write(depth, at)
	d := depth + 1
	if _, ok := a.Get("uses"); ok {
		p.line(d, p.kw("uses")+" "+p.name(p.str(a, "uses")))
		if v, ok := p.take(a, "with"); ok {
			w, isObj := v.(*Obj)
			if !isObj {
				refuse("\"with\" is an object")
			}
			for _, g := range w.Keys() {
				m, _ := w.Get(g)
				s, isString := m.(string)
				if !isString {
					refuse("\"with\" gives a place for a place")
				}
				p.line(d+1, p.kw("with")+" "+p.name(g)+" = "+p.name(s))
			}
		}
	}
	if v, ok := p.take(a, "reads"); ok {
		l, isList := v.([]any)
		if !isList {
			refuse("\"reads\" is a list")
		}
		var plain []any
		flush := func() {
			if len(plain) > 0 {
				p.names(d, p.kw("reads"), plain)
				plain = nil
			}
		}
		for _, r := range l {
			if o, isObj := r.(*Obj); isObj {
				flush()
				w, _ := p.take(o, "when")
				p.line(d, p.kw("reads")+" "+p.name(p.str(o, "place"))+" "+p.kw("when")+" "+p.cond(w, false))
				p.done(o, "a read")
			} else {
				plain = append(plain, r)
			}
		}
		flush()
	}
	if v, ok := p.take(a, "changes"); ok {
		for _, c := range p.objs(v, "changes") {
			head := p.word(c, "how", "create", "replace", "destroy") + " " + p.name(p.str(c, "place"))
			if p.yes(c, "many") {
				head += " " + p.kw("many")
			}
			if _, ok := c.Get("from"); ok {
				head += " " + p.kw("from") + " " + p.name(p.str(c, "from"))
			}
			ca := &attrs{head: head}
			if w, ok := p.take(c, "when"); ok {
				p.on(ca, "when", w, false)
			}
			p.write(d, ca)
			p.makes(d+1, c)
			p.later(d+1, c)
			p.details(d+1, c)
			p.done(c, "a change of "+fmt.Sprint(c.vals["place"]))
		}
	}
	p.makes(d, a)
	p.later(d, a)
	if v, ok := p.take(a, "rehearsal"); ok {
		p.line(d, p.kw("rehearsal")+" "+p.cond(v, false))
	}
	p.answers(d, a)
	p.details(d, a)
	p.done(a, "action "+do)
}

func (p *printer) makes(depth int, o *Obj) {
	v, ok := p.take(o, "makes")
	if !ok {
		return
	}
	m, isObj := v.(*Obj)
	if !isObj {
		refuse("\"makes\" is an object")
	}
	for _, f := range m.Keys() {
		b, _ := m.Get(f)
		switch b {
		case true:
			p.line(depth, p.kw("makes")+" "+p.name(f))
		case false:
			p.line(depth, p.kw("makes", "not")+" "+p.name(f))
		default:
			refuse("\"makes\" makes a fact true or false")
		}
	}
}

func (p *printer) later(depth int, o *Obj) {
	if _, ok := o.Get("later"); ok {
		p.line(depth, p.kw("later")+" "+p.word(o, "later", "undone", "stands"))
	}
}

func (p *printer) answers(depth int, o *Obj) {
	v, ok := p.take(o, "handles")
	if !ok {
		return
	}
	for _, h := range p.objs(v, "handles") {
		a := &attrs{head: p.word(h, "then", "refuse", "fail", "warn", "tell", "ask", "accept")}
		if _, ok := h.Get("no"); ok {
			a.head += " " + p.kw("no") + " " + p.word(h, "no", "stop", "skip")
		}
		if w, ok := p.take(h, "when"); ok {
			p.on(a, "when", w, true)
		}
		p.write(depth, a)
		if t, ok := p.take(h, "tells"); ok {
			p.names(depth+1, p.kw("tells"), t)
		}
		p.details(depth+1, h)
		p.done(h, "an answer")
	}
}

func (p *printer) expect(e *Obj) {
	p.line(0, p.kw("expect")+" "+p.name(p.str(e, "name")))
	if _, ok := e.Get("in"); ok {
		p.line(1, p.kw("in")+" "+p.name(p.str(e, "in")))
	}
	if w, ok := p.take(e, "when"); ok {
		a := &attrs{}
		p.on(a, "when", w, true)
		if a.blocks == nil {
			p.line(1, strings.TrimPrefix(a.head, " "))
		} else {
			b := a.blocks[0]
			p.line(1, p.kw("when", b[1].(string)))
			for _, c := range b[2].([]any) {
				p.line(2, p.cond(c, false))
			}
		}
	}
	if _, ok := e.Get("on"); ok {
		p.line(1, p.kw("on")+" "+p.word(e, "on", "any", "success", "failed", "declined"))
	}
	for _, k := range []string{"never changes", "never creates", "runs", "never runs"} {
		if v, ok := p.take(e, k); ok {
			p.names(1, p.kw(strings.Fields(k)...), v)
		}
	}
	if v, ok := p.take(e, "ends"); ok {
		m, isObj := v.(*Obj)
		if !isObj {
			refuse("\"ends\" is an object")
		}
		for _, pl := range m.Keys() {
			s, _ := m.Get(pl)
			st, isString := s.(string)
			if !isString || !isKeyword[st] || !contains(stateWords, st) {
				refuse("\"ends\" gives each place one of %s; got %v", strings.Join(stateWords, ", "), s)
			}
			p.line(1, p.kw("ends")+" "+p.name(pl)+" "+p.kw(st))
		}
	}
	if _, ok := e.Get("before"); ok {
		p.line(1, p.kw("before")+" "+p.name(p.str(e, "before"))+" "+p.kw("place")+" "+p.name(p.str(e, "place"))+" "+p.kw("is")+" "+p.word(e, "is", stateWords...))
	}
	p.details(1, e)
	p.done(e, "expectation "+fmt.Sprint(e.vals["name"]))
}

func (p *printer) rule(r *Obj) {
	p.line(0, p.kw("rule")+" "+p.text(p.str(r, "rule")))
	if _, ok := r.Get("about"); ok {
		p.line(1, p.kw("about")+" "+p.name(p.str(r, "about")))
	}
	if w, ok := p.take(r, "when"); ok {
		a := &attrs{}
		p.on(a, "when", w, true)
		if a.blocks == nil {
			p.line(1, strings.TrimPrefix(a.head, " "))
		} else {
			b := a.blocks[0]
			p.line(1, p.kw("when", b[1].(string)))
			for _, c := range b[2].([]any) {
				p.line(2, p.cond(c, false))
			}
		}
	}
	p.details(1, r)
	p.done(r, "rule")
}

func contains(l []string, s string) bool {
	for _, x := range l {
		if x == s {
			return true
		}
	}
	return false
}
