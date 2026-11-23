package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// node is one JSON value, read so that an object keeps its keys in
// order and a key given twice is an error.
type node struct {
	kind  byte // 'o' object, 'a' array, 's' string, 'b' bool, 'x' anything else
	keys  []string
	vals  map[string]*node
	items []*node
	str   string
	b     bool
}

// readJSON reads one JSON object and refuses anything after it.
func readJSON(data []byte) (*node, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	n, err := readValue(dec)
	if err != nil {
		return nil, err
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, fmt.Errorf("there is something after the object's end")
	}
	if n.kind != 'o' {
		return nil, fmt.Errorf("the file is not one JSON object")
	}
	return n, nil
}

func readValue(dec *json.Decoder) (*node, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, fmt.Errorf("the file is not valid JSON: %v", err)
	}
	switch t := tok.(type) {
	case json.Delim:
		if t == '{' {
			n := &node{kind: 'o', vals: map[string]*node{}}
			for dec.More() {
				kt, err := dec.Token()
				if err != nil {
					return nil, fmt.Errorf("the file is not valid JSON: %v", err)
				}
				key, _ := kt.(string)
				if _, twice := n.vals[key]; twice {
					return nil, fmt.Errorf("the key %q is given twice in one object", key)
				}
				v, err := readValue(dec)
				if err != nil {
					return nil, err
				}
				n.keys = append(n.keys, key)
				n.vals[key] = v
			}
			if _, err := dec.Token(); err != nil {
				return nil, fmt.Errorf("the file is not valid JSON: %v", err)
			}
			return n, nil
		}
		n := &node{kind: 'a'}
		for dec.More() {
			v, err := readValue(dec)
			if err != nil {
				return nil, err
			}
			n.items = append(n.items, v)
		}
		if _, err := dec.Token(); err != nil {
			return nil, fmt.Errorf("the file is not valid JSON: %v", err)
		}
		return n, nil
	case string:
		return &node{kind: 's', str: t}, nil
	case bool:
		return &node{kind: 'b', b: t}, nil
	}
	return &node{kind: 'x'}, nil
}

// keysOf is every key the format has, object by object, as fields.md
// lists them. A description file leaves out the fields that are the
// design's to decide and adds source; allowed works that out.
var keysOf = map[string][]string{
	"definition":  {"system", "places", "elements", "unsaid"},
	"description": {"describes", "places", "elements", "unsaid"},
	"place":       {"name", "belongsTo", "outside", "keyed", "lasting", "onExisting", "hold", "invalid"},
	"element":     {"name", "reads", "writes", "join", "include", "steps", "or", "otherwise", "rejects", "declines"},
	"read":        {"place"},
	"write":       {"place", "mode", "key", "records", "before"},
	"check":       {"check", "then", "hold"},
	"rejects":     {"reason", "to", "hold"},
	"declines":    {"reason", "to", "hold"},
	"branch":      {"when", "do"},
	"unsaid":      {"what", "hold"},
	"hold":        {"file", "test"},
	"source":      {"file", "function"},
	"accepted":    {"accepted"},
	"acceptance":  {"finding", "reason"},
}

// definitionOnly are the fields a description file does not have.
var definitionOnly = map[string]bool{"belongsTo": true, "outside": true, "lasting": true, "invalid": true, "hold": true}

// reader turns nodes into the format's objects and collects every
// reason the file is refused.
type reader struct {
	description bool
	problems    []string
}

func (r *reader) bad(format string, a ...any) {
	r.problems = append(r.problems, fmt.Sprintf(format, a...))
}

// allowed is the keys an object of this kind may have in this file.
func (r *reader) allowed(object string) []string {
	if !r.description {
		return keysOf[object]
	}
	var keys []string
	for _, k := range keysOf[object] {
		if !definitionOnly[k] {
			keys = append(keys, k)
		}
	}
	switch object {
	case "place", "element", "read", "write", "check", "rejects", "declines", "branch", "unsaid":
		keys = append(keys, "source")
	}
	return keys
}

// object checks that n is an object with only the keys its kind allows.
func (r *reader) object(n *node, object, where string) bool {
	if n.kind != 'o' {
		r.bad("%s is not an object", where)
		return false
	}
	allowed := r.allowed(object)
	for _, k := range n.keys {
		if contains(allowed, k) {
			continue
		}
		note := ""
		for _, a := range allowed {
			if strings.EqualFold(a, k) {
				note = fmt.Sprintf("; it is spelled %q", a)
			}
		}
		if r.description && (definitionOnly[k]) && contains(keysOf[object], k) {
			note = "; a description file does not have it"
		}
		if !r.description && k == "source" {
			note = "; a definition file does not have it"
		}
		r.bad("%s has the key %q, which the format does not list there%s", where, k, note)
	}
	return true
}

// text reads a string field. A required one that is missing or empty is
// a problem; an optional one that is missing is "".
func (r *reader) text(n *node, key, where string, required bool) string {
	v, ok := n.vals[key]
	if !ok {
		if required {
			r.bad("%s has no %q", where, key)
		}
		return ""
	}
	if v.kind != 's' {
		r.bad("%s: %q is not a string", where, key)
		return ""
	}
	if v.str == "" && required {
		r.bad("%s: %q is empty", where, key)
	}
	return v.str
}

// oneOf reads a string field that takes one of a few values.
func (r *reader) oneOf(n *node, key, where string, values ...string) string {
	if _, ok := n.vals[key]; !ok {
		return ""
	}
	s := r.text(n, key, where, true)
	if s != "" && !contains(values, s) {
		r.bad("%s: %q is %q, and it is one of %s", where, key, s, quoted(values))
		return ""
	}
	return s
}

// flag reads a field that is true or left out.
func (r *reader) flag(n *node, key, where string) bool {
	v, ok := n.vals[key]
	if !ok {
		return false
	}
	if v.kind != 'b' {
		r.bad("%s: %q is not true or false", where, key)
		return false
	}
	if !v.b {
		r.bad("%s: %q is written as false; a field that is not true is left out", where, key)
	}
	return v.b
}

// list reads a list field. Below the top level an empty list is a
// problem, since it is left out.
func (r *reader) list(n *node, key, where string, top bool) ([]*node, bool) {
	v, ok := n.vals[key]
	if !ok {
		if top {
			r.bad("%s has no %q", where, key)
		}
		return nil, false
	}
	if v.kind != 'a' {
		r.bad("%s: %q is not a list", where, key)
		return nil, false
	}
	if len(v.items) == 0 && !top {
		r.bad("%s: %q is an empty list; an empty list is left out", where, key)
	}
	return v.items, true
}

// names reads a list of names.
func (r *reader) names(n *node, key, where string) []string {
	items, _ := r.list(n, key, where, false)
	var out []string
	for _, it := range items {
		if it.kind != 's' || it.str == "" {
			r.bad("%s: %q holds something that is not a name", where, key)
			continue
		}
		out = append(out, it.str)
	}
	return out
}

// evidence reads a hold in a definition file or a source in a
// description file. Left out, it is nil, and a check reports it.
func (r *reader) evidence(n *node, where string) *Evidence {
	key, name := "hold", "test"
	if r.description {
		key, name = "source", "function"
	}
	v, ok := n.vals[key]
	if !ok {
		return nil
	}
	w := where + ", " + key
	if !r.object(v, key, w) {
		return nil
	}
	return &Evidence{File: r.text(v, "file", w, true), Name: r.text(v, name, w, true)}
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func quoted(values []string) string {
	q := make([]string, len(values))
	for i, v := range values {
		q[i] = fmt.Sprintf("%q", v)
	}
	return strings.Join(q, ", ")
}
