package main

import (
	"fmt"
	"strings"
)

const (
	tWord    = iota // a word that is not a keyword: part of a name in form A, a path or a number in either
	tName           // a name in brackets, form B
	tKeyword        // a keyword, in lower case whatever the form
	tText           // the rest of a line after a keyword that takes text
	tOpen           // (
	tClose          // )
	tEq             // =
	tComma          // , between names in form B
)

type token struct {
	kind int
	text string
}

type lexer struct {
	form Form
	data string
	toks []token
	text int // where the text starts, after a keyword that takes it; -1 when there is none
}

func (l *lexer) add(kind, ts, te int) {
	l.toks = append(l.toks, token{kind, l.data[ts:te]})
}

// word adds a word, as a keyword when the form writes one so. It is true when the rest of the line is text.
func (l *lexer) word(ts, te int) bool {
	w := l.data[ts:te]
	if k, ok := l.form.keyword(w); ok {
		l.toks = append(l.toks, token{tKeyword, k})
		if texts[k] {
			l.text = te
			return true
		}
		return false
	}
	l.add(tWord, ts, te)
	return false
}

// lex reads one line, its indentation taken off.
func lex(form Form, line string) ([]token, error) {
	if err := brackets(form, line); err != nil {
		return nil, err
	}
	l := &lexer{form: form, data: line, text: -1}
	run := lexLineA
	if form == B {
		run = lexLineB
	}
	p, ok := run(l)
	if l.text >= 0 {
		if s := strings.TrimPrefix(line[l.text:], " "); s != "" {
			l.toks = append(l.toks, token{tText, s})
		}
		return l.toks, nil
	}
	if ok && p == len(line) {
		return l.toks, nil
	}
	rest := line[p:]
	switch {
	case strings.Contains(rest, "\t"):
		return nil, fmt.Errorf("a tab, where only spaces may be")
	case form == B && strings.HasPrefix(rest, "["):
		return nil, fmt.Errorf(`a name opened with "[" is not closed`)
	case form == B && strings.HasPrefix(rest, "]"):
		return nil, fmt.Errorf(`a "]" closes no name`)
	case form == A && strings.ContainsAny(rest, "[]"):
		return nil, fmt.Errorf("form A writes a name without brackets, and a name may not hold one")
	}
	return nil, fmt.Errorf("this cannot be read from %q", rest)
}

// brackets refuses a tab, and a bracket a name cannot have: in form A any, in form B one left open
// or one that closes nothing. It looks only up to the text a keyword starts, which may hold either.
func brackets(form Form, line string) error {
	open := false
	for i, r := range line {
		if r == ' ' && i > 0 {
			if w := lastWord(line[:i]); !open && texts[strings.ToLower(w)] && form.kw(strings.ToLower(w)) == w {
				return nil
			}
		}
		switch {
		case r == '\t':
			return fmt.Errorf("a tab, where only spaces may be")
		case form == A && (r == '[' || r == ']'):
			return fmt.Errorf("form A writes a name without brackets, and a name may not hold one")
		case form == B && r == '[':
			if open {
				return fmt.Errorf(`a name opened with "[" holds another "["`)
			}
			open = true
		case form == B && r == ']':
			if !open {
				return fmt.Errorf(`a "]" closes no name`)
			}
			open = false
		}
	}
	if open {
		return fmt.Errorf(`a name opened with "[" is not closed`)
	}
	return nil
}

func lastWord(s string) string {
	if i := strings.LastIndexAny(s, " ()=,[]"); i >= 0 {
		return s[i+1:]
	}
	return s
}
