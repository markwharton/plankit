package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Read reads a design written as text in a form. file names the text in a refusal, with its line.
func Read(f Form, file, text string) (doc *Obj, err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(refusal); ok {
				err = e
				return
			}
			panic(r)
		}
	}()
	top := lines(f, file, text)
	return (&reader{f: f, file: file}).design(top), nil
}

// line is one line of the text, with the lines indented below it.
type line struct {
	n     int
	toks  []token
	kids  []*line
	file  string
	f     Form
	i     int // the next token to read
	depth int
}

func (l *line) refuse(format string, a ...any) {
	refuse("%s:%d: %s", l.file, l.n, fmt.Sprintf(format, a...))
}

// lines reads the text into lines, each with the lines indented below it.
func lines(f Form, file, text string) []*line {
	var top []*line
	var open []*line // the last line read at each depth
	for i, s := range strings.Split(text, "\n") {
		if strings.TrimSpace(s) == "" {
			continue
		}
		body := strings.TrimLeft(s, " ")
		l := &line{n: i + 1, file: file, f: f}
		spaces := len(s) - len(body)
		if spaces%4 != 0 {
			l.refuse("indented by %d spaces; a level is four", spaces)
		}
		l.depth = spaces / 4
		toks, err := lex(f, body)
		if err != nil {
			l.refuse("%v", err)
		}
		l.toks = toks
		if l.depth > len(open) || (l.depth > 0 && open[l.depth-1] == nil) {
			l.refuse("this line is indented under nothing")
		}
		if l.depth == 0 {
			top = append(top, l)
		} else {
			parent := open[l.depth-1]
			parent.kids = append(parent.kids, l)
		}
		open = append(open[:l.depth], l)
	}
	return top
}

func (l *line) more() bool { return l.i < len(l.toks) }

func (l *line) peek() token {
	if !l.more() {
		return token{kind: -1}
	}
	return l.toks[l.i]
}

// is consumes the keywords, in order, when they come next.
func (l *line) is(words ...string) bool {
	for j, w := range words {
		if l.i+j >= len(l.toks) || l.toks[l.i+j].kind != tKeyword || l.toks[l.i+j].text != w {
			return false
		}
	}
	l.i += len(words)
	return true
}

func (l *line) show(t token) string {
	switch t.kind {
	case -1:
		return "the end of the line"
	case tKeyword:
		return l.f.kw(t.text)
	case tName:
		return "[" + t.text + "]"
	}
	return t.text
}

// last is the keyword read before the next token, as the form writes it.
func (l *line) last() string {
	if l.i == 0 {
		return "the start of the line"
	}
	return l.show(l.toks[l.i-1])
}

// startsName is true when a name comes next.
func (l *line) startsName() bool {
	k := l.peek().kind
	return (l.f == A && k == tWord) || (l.f == B && k == tName)
}

func (l *line) name() string {
	if !l.startsName() {
		if t := l.peek(); t.kind == tKeyword {
			l.refuse("a name was expected after %s, and %s is a keyword", l.last(), l.show(t))
		}
		l.refuse("a name was expected after %s; got %s", l.last(), l.show(l.peek()))
	}
	if l.f == B {
		l.i++
		return l.toks[l.i-1].text
	}
	var words []string
	for l.more() && l.peek().kind == tWord {
		words = append(words, l.toks[l.i].text)
		l.i++
	}
	return strings.Join(words, " ")
}

// word is a word as written: a path, a test's name or a number.
func (l *line) word(what string) string {
	if l.peek().kind != tWord && !(l.f == A && l.peek().kind == tName) {
		l.refuse("%s was expected after %s", what, l.last())
	}
	l.i++
	return l.toks[l.i-1].text
}

func (l *line) text() string {
	if l.peek().kind != tText {
		return ""
	}
	l.i++
	return l.toks[l.i-1].text
}

func (l *line) keyword(from ...string) string {
	t := l.peek()
	if t.kind == tKeyword {
		for _, w := range from {
			if t.text == w {
				l.i++
				return w
			}
		}
	}
	shown := make([]string, len(from))
	for i, w := range from {
		shown[i] = l.f.kw(w)
	}
	l.refuse("one of %s was expected after %s; got %s", strings.Join(shown, ", "), l.last(), l.show(t))
	return ""
}

func (l *line) end() {
	if l.more() {
		l.refuse("%s was not expected after %s", l.show(l.peek()), l.last())
	}
}

// names reads a list of names: on the line, separated by commas in form B, or one to a line below it.
func (l *line) names() []string {
	if l.more() {
		out := []string{l.name()}
		for l.f == B && l.peek().kind == tComma {
			l.i++
			out = append(out, l.name())
		}
		l.end()
		return out
	}
	if len(l.kids) == 0 {
		l.refuse("%s names nothing", l.last())
	}
	var out []string
	for _, k := range l.kids {
		out = append(out, k.names()...)
		if len(k.kids) > 0 {
			k.kids[0].refuse("a name in a list has nothing below it")
		}
	}
	l.kids = nil
	return out
}

// one is a list read back as one name when it has one, as the design wrote it.
func one(ns []string) any {
	if len(ns) == 1 {
		return ns[0]
	}
	out := make([]any, len(ns))
	for i, n := range ns {
		out[i] = n
	}
	return out
}

func many(ns []string) []any {
	out := make([]any, len(ns))
	for i, n := range ns {
		out[i] = n
	}
	return out
}

// cond reads a condition to the end of the line: OR of AND of NOT, with brackets.
func (l *line) cond() any {
	if !l.more() {
		l.refuse("a condition was expected after %s", l.last())
	}
	c := l.or()
	return c
}

func (l *line) or() any {
	return l.join("or", "any", l.and)
}

func (l *line) and() any {
	return l.join("and", "all", l.not)
}

func (l *line) join(word, key string, next func() any) any {
	first := next()
	parts := []any{first}
	for l.is(word) {
		parts = append(parts, next())
	}
	if len(parts) == 1 {
		return first
	}
	o := newObj()
	o.Set(key, parts)
	return o
}

func (l *line) not() any {
	if !l.more() {
		l.refuse("the condition ends where a fact was expected")
	}
	if l.is("not") {
		o := newObj()
		o.Set("not", l.not())
		return o
	}
	if l.peek().kind == tOpen {
		l.i++
		c := l.or()
		if l.peek().kind != tClose {
			l.refuse("a bracket opened in the condition is not closed")
		}
		l.i++
		return c
	}
	return l.name()
}

// holds is true when the rest of the line has the keyword.
func (l *line) holds(w string) bool {
	for _, t := range l.toks[l.i:] {
		if t.kind == tKeyword && t.text == w {
			return true
		}
	}
	return false
}

// block reads KEY and a condition on the line, or KEY ANY or KEY ALL alone with the parts one to a line below it.
func (l *line) block(key string) any {
	l.keyword(key)
	if t := l.peek(); t.kind == tKeyword && (t.text == "any" || t.text == "all") && l.i+1 == len(l.toks) {
		join := t.text
		l.i++
		if len(l.kids) < 2 {
			l.refuse("%s joins the lines below it, and there are fewer than two", l.f.kw(join))
		}
		var parts []any
		for _, k := range l.kids {
			parts = append(parts, k.cond())
			k.end()
			if len(k.kids) > 0 {
				k.kids[0].refuse("a part of a condition has nothing below it")
			}
		}
		l.kids = nil
		o := newObj()
		o.Set(join, parts)
		return o
	}
	c := l.cond()
	l.end()
	return c
}

type reader struct {
	f    Form
	file string
}

// details reads says, because and hold, the lines every statement can carry. It is true when it read one.
func (r *reader) details(o *Obj, k *line, says *[]string) bool {
	switch {
	case k.is("says"):
		*says = append(*says, k.text())
	case k.is("because"):
		o.Set("because", k.text())
	case k.is("hold"):
		if !k.more() {
			k.refuse("%s names a file and a test", r.f.kw("hold"))
		}
		h := newObj()
		h.Set("file", k.word("a file"))
		if !k.more() {
			k.refuse("%s names a file and a test", r.f.kw("hold"))
		}
		h.Set("test", k.word("a test"))
		o.Set("hold", h)
	default:
		return false
	}
	k.end()
	if len(k.kids) > 0 {
		k.kids[0].refuse("this line is indented under %s, which has nothing below it", k.show(k.toks[0]))
	}
	return true
}

func setSays(o *Obj, says []string) {
	if len(says) > 0 {
		o.Set("says", one(says))
	}
}

// child refuses a line a statement does not take.
func (r *reader) child(k *line, what string) {
	if t := k.peek(); t.kind == tKeyword {
		k.refuse("%s does not take %s", what, k.show(t))
	}
	k.refuse("a line in %s starts with a keyword; got %s", what, k.show(k.peek()))
}

func (r *reader) design(top []*line) *Obj {
	d := newObj()
	var on []any
	var runs []any
	places, facts, parts := []any{}, []any{}, newObj()
	var expects, rules []any
	for _, l := range top {
		switch {
		case l.is("system"):
			d.Set("system", l.text())
		case l.is("on"):
			on = append(on, l.word("a path"))
		case l.is("reports"):
			d.Set("reports", l.name())
		case l.is("faults"):
			w := l.word("a whole number")
			n := json.Number(w)
			if _, err := n.Int64(); err != nil {
				l.refuse("%s is a whole number", r.f.kw("faults"))
			}
			d.Set("faults", n)
		case l.is("runs"):
			runs = append(runs, many(l.names())...)
		case l.is("place"):
			places = append(places, r.place(l))
		case l.peek().kind == tKeyword && (l.peek().text == "fact" || l.peek().text == "fault" || l.peek().text == "open"):
			facts = append(facts, r.fact(l))
		case l.peek().kind == tKeyword && (l.peek().text == "seq" || l.peek().text == "all" || l.peek().text == "choose" || l.peek().text == "action"):
			key, p := r.part(l, true)
			if _, ok := parts.Get(key); ok {
				l.refuse("two parts are named %q", key)
			}
			parts.Set(key, p)
		case l.is("expect"):
			expects = append(expects, r.expect(l))
		case l.is("rule"):
			rules = append(rules, r.rule(l))
		default:
			r.child(l, "the design")
		}
		l.end()
		if len(l.kids) > 0 {
			l.kids[0].refuse("this line is indented under %s, which has nothing below it", l.show(l.toks[0]))
		}
	}
	if len(top) == 0 || !has(d, "system") {
		refuse("%s: a design starts with %s", r.file, r.f.kw("system"))
	}
	out := newObj()
	out.Set("system", d.vals["system"])
	if on != nil {
		out.Set("on", on)
	}
	for _, k := range []string{"reports", "faults"} {
		if v, ok := d.Get(k); ok {
			out.Set(k, v)
		}
	}
	out.Set("places", places)
	out.Set("facts", facts)
	out.Set("parts", parts)
	out.Set("runs", runs)
	if expects != nil {
		out.Set("expects", expects)
	}
	if rules != nil {
		out.Set("rules", rules)
	}
	return out
}

func has(o *Obj, k string) bool { _, ok := o.Get(k); return ok }

func (r *reader) place(l *line) *Obj {
	p := newObj()
	p.Set("name", l.name())
	if l.is("one") || l.is("many") || l.is("series") {
		p.Set("is", l.toks[l.i-1].text)
	}
	if l.is("outside") || l.is("lasting") || l.is("run") || l.is("report") {
		p.Set("kind", l.toks[l.i-1].text)
	}
	if l.is("in") {
		p.Set("in", l.name())
	}
	if l.is("given") {
		p.Set("given", true)
	}
	if l.is("owner") {
		p.Set("owner", l.name())
	}
	l.end()
	var says []string
	for _, k := range l.kids {
		if !r.details(p, k, &says) {
			r.child(k, "a place")
		}
	}
	l.kids = nil
	setSays(p, says)
	return p
}

func (r *reader) fact(l *line) *Obj {
	f := newObj()
	kind := l.keyword("fact", "fault", "open")
	f.Set("name", l.name())
	switch kind {
	case "fault":
		f.Set("fault", true)
	case "open":
		f.Set("open", true)
	}
	if l.is("about") {
		f.Set("about", l.name())
	}
	for _, k := range []string{"only", "means"} {
		if l.is(k) {
			f.Set(k, l.cond())
		}
	}
	l.end()
	var says []string
	for _, k := range l.kids {
		switch {
		case k.peek().kind == tKeyword && (k.peek().text == "only" || k.peek().text == "means"):
			f.Set(k.peek().text, k.block(k.peek().text))
		case k.is("at"):
			f.Set("at", one(k.names()))
		case r.details(f, k, &says):
		default:
			r.child(k, "a fact")
		}
	}
	l.kids = nil
	setSays(f, says)
	return f
}

// part reads a part written at l. At the top it is named by its key, which it returns.
func (r *reader) part(l *line, top bool) (string, *Obj) {
	if l.is("action") {
		return r.action(l, top)
	}
	kind := l.keyword("seq", "all", "choose")
	p := newObj()
	name := ""
	if l.startsName() {
		name = l.name()
		p.Set("name", name)
	} else if top {
		l.refuse("a part at the top is named")
	}
	if l.is("when") {
		p.Set("when", l.cond())
	}
	l.end()
	var says []string
	var kids, branches []any
	var handles []any
	for _, k := range l.kids {
		switch {
		case kind == "choose" && k.is("when"):
			b := newObj()
			b.Set("when", k.cond())
			k.end()
			b.Set("run", r.only(k))
			branches = append(branches, b)
		case kind == "choose" && k.is("otherwise"):
			k.end()
			p.Set("otherwise", r.only(k))
		case isAnswer(k):
			handles = append(handles, r.answer(k))
		case r.details(p, k, &says):
		case kind != "choose":
			kids = append(kids, r.ref(k))
		default:
			r.child(k, "a choice")
		}
	}
	l.kids = nil
	if kind == "choose" {
		p.Set("choose", branches)
	} else {
		p.Set(kind, kids)
	}
	if handles != nil {
		p.Set("handles", handles)
	}
	setSays(p, says)
	return name, p
}

// only is the one part below a branch or OTHERWISE.
func (r *reader) only(k *line) any {
	if len(k.kids) != 1 {
		k.refuse("%s has one part below it", k.show(k.toks[0]))
	}
	v := r.ref(k.kids[0])
	k.kids = nil
	return v
}

// ref reads a part inside another: a part written in place, NOTHING, or a part's name with AS and WHEN.
func (r *reader) ref(k *line) any {
	if t := k.peek(); t.kind == tKeyword {
		switch t.text {
		case "seq", "all", "choose", "action":
			_, p := r.part(k, false)
			return p
		case "nothing":
			k.i++
			k.end()
			return nil
		}
		r.child(k, "a sequence")
	}
	name := k.name()
	u := newObj()
	u.Set("use", name)
	if k.is("as") {
		u.Set("name", k.name())
	}
	if k.is("when") {
		u.Set("when", k.cond())
	}
	k.end()
	var says []string
	for _, c := range k.kids {
		if !r.details(u, c, &says) {
			r.child(c, "a use of a part")
		}
	}
	k.kids = nil
	setSays(u, says)
	if len(u.Keys()) == 1 {
		return name
	}
	return u
}

func isAnswer(k *line) bool {
	t := k.peek()
	if t.kind != tKeyword {
		return false
	}
	switch t.text {
	case "refuse", "fail", "warn", "tell", "ask", "accept":
		return true
	}
	return false
}

func (r *reader) answer(k *line) *Obj {
	h := newObj()
	h.Set("then", k.keyword("refuse", "fail", "warn", "tell", "ask", "accept"))
	if k.is("no") {
		h.Set("no", k.keyword("stop", "skip"))
	}
	if k.is("when") {
		h.Set("when", k.cond())
	}
	k.end()
	var says []string
	for _, c := range k.kids {
		switch {
		case c.peek().kind == tKeyword && c.peek().text == "when":
			h.Set("when", c.block("when"))
		case c.is("tells"):
			h.Set("tells", one(c.names()))
		case r.details(h, c, &says):
		default:
			r.child(c, "an answer")
		}
	}
	k.kids = nil
	setSays(h, says)
	return h
}

func (r *reader) action(l *line, top bool) (string, *Obj) {
	a := newObj()
	do := l.name()
	a.Set("do", do)
	key := do
	if l.is("as") {
		if !top {
			l.refuse("%s names a part at the top by another key; a part written in place has none", r.f.kw("as"))
		}
		key = l.name()
	}
	if l.is("when") {
		a.Set("when", l.cond())
	}
	l.end()
	var says []string
	var reads, changes, handles []any
	for _, k := range l.kids {
		switch {
		case k.peek().kind == tKeyword && k.peek().text == "when":
			a.Set("when", k.block("when"))
		case k.is("uses"):
			a.Set("uses", k.name())
			k.end()
			if len(k.kids) > 0 {
				with := newObj()
				for _, w := range k.kids {
					if !w.is("with") {
						r.child(w, "a use of a run")
					}
					g := w.name()
					if w.peek().kind != tEq {
						w.refuse("%s gives a place for a place: %s, =, and a place", r.f.kw("with"), r.f.kw("with"))
					}
					w.i++
					with.Set(g, w.name())
					w.end()
				}
				k.kids = nil
				a.Set("with", with)
			}
		case k.is("reads"):
			if !k.holds("when") {
				reads = append(reads, many(k.names())...)
				continue
			}
			o := newObj()
			o.Set("place", k.name())
			k.keyword("when")
			o.Set("when", k.cond())
			k.end()
			reads = append(reads, o)
		case k.peek().kind == tKeyword && (k.peek().text == "create" || k.peek().text == "replace" || k.peek().text == "destroy"):
			changes = append(changes, r.change(k))
		case k.is("makes"):
			r.makes(a, k)
		case k.is("later"):
			a.Set("later", k.keyword("undone", "stands"))
			k.end()
		case k.is("rehearsal"):
			a.Set("rehearsal", k.cond())
			k.end()
		case isAnswer(k):
			handles = append(handles, r.answer(k))
		case r.details(a, k, &says):
		default:
			r.child(k, "an action")
		}
	}
	l.kids = nil
	if reads != nil {
		a.Set("reads", reads)
	}
	if changes != nil {
		a.Set("changes", changes)
	}
	if handles != nil {
		a.Set("handles", handles)
	}
	setSays(a, says)
	return key, a
}

func (r *reader) makes(o *Obj, k *line) {
	m, ok := o.Get("makes")
	if !ok {
		m = newObj()
		o.Set("makes", m)
	}
	v := !k.is("not")
	m.(*Obj).Set(k.name(), v)
	k.end()
}

func (r *reader) change(k *line) *Obj {
	c := newObj()
	c.Set("how", k.keyword("create", "replace", "destroy"))
	c.Set("place", k.name())
	if k.is("many") {
		c.Set("many", true)
	}
	if k.is("from") {
		c.Set("from", k.name())
	}
	if k.is("when") {
		c.Set("when", k.cond())
	}
	k.end()
	var says []string
	for _, x := range k.kids {
		switch {
		case x.is("makes"):
			r.makes(c, x)
		case x.is("later"):
			c.Set("later", x.keyword("undone", "stands"))
			x.end()
		case r.details(c, x, &says):
		default:
			r.child(x, "a change")
		}
	}
	k.kids = nil
	setSays(c, says)
	return c
}

var states = []string{"untouched", "created", "replaced", "destroyed", "undone", "changed", "unchanged"}

func (r *reader) expect(l *line) *Obj {
	e := newObj()
	e.Set("name", l.name())
	l.end()
	var says []string
	for _, k := range l.kids {
		switch {
		case k.is("in"):
			e.Set("in", k.name())
			k.end()
		case k.peek().kind == tKeyword && k.peek().text == "when":
			e.Set("when", k.block("when"))
		case k.is("on"):
			e.Set("on", k.keyword("any", "success", "failed", "declined"))
			k.end()
		case k.is("never", "changes"):
			e.Set("never changes", many(k.names()))
		case k.is("never", "creates"):
			e.Set("never creates", many(k.names()))
		case k.is("never", "runs"):
			e.Set("never runs", many(k.names()))
		case k.is("runs"):
			e.Set("runs", many(k.names()))
		case k.is("ends"):
			m, ok := e.Get("ends")
			if !ok {
				m = newObj()
				e.Set("ends", m)
			}
			pl := k.name()
			m.(*Obj).Set(pl, k.keyword(states...))
			k.end()
		case k.is("before"):
			e.Set("before", k.name())
			k.keyword("place")
			e.Set("place", k.name())
			k.keyword("is")
			e.Set("is", k.keyword(states...))
			k.end()
		case r.details(e, k, &says):
		default:
			r.child(k, "an expectation")
		}
	}
	l.kids = nil
	setSays(e, says)
	return e
}

func (r *reader) rule(l *line) *Obj {
	o := newObj()
	o.Set("rule", l.text())
	var says []string
	for _, k := range l.kids {
		switch {
		case k.is("about"):
			o.Set("about", k.name())
			k.end()
		case k.peek().kind == tKeyword && k.peek().text == "when":
			o.Set("when", k.block("when"))
		case r.details(o, k, &says):
		default:
			r.child(k, "a rule")
		}
	}
	l.kids = nil
	setSays(o, says)
	return o
}
