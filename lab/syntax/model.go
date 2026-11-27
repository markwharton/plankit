package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// Obj is a JSON object that keeps the order of its keys, so a design read and printed
// again comes out in the order it was written.
type Obj struct {
	keys []string
	vals map[string]any
}

func newObj() *Obj { return &Obj{vals: map[string]any{}} }

// Set gives a key its value, adding the key at the end when it is new.
func (o *Obj) Set(k string, v any) {
	if _, ok := o.vals[k]; !ok {
		o.keys = append(o.keys, k)
	}
	o.vals[k] = v
}

// Get is a key's value, and whether the object has the key.
func (o *Obj) Get(k string) (any, bool) {
	v, ok := o.vals[k]
	return v, ok
}

// Keys are the object's keys, in order.
func (o *Obj) Keys() []string { return o.keys }

// MarshalJSON writes the keys in order, with no escaping of <, > and &.
func (o *Obj) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, k := range o.keys {
		if i > 0 {
			b.WriteByte(',')
		}
		for _, v := range []any{k, o.vals[k]} {
			data, err := encode(v)
			if err != nil {
				return nil, err
			}
			b.Write(data)
			if v == any(k) {
				b.WriteByte(':')
			}
		}
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

func encode(v any) ([]byte, error) {
	var b bytes.Buffer
	e := json.NewEncoder(&b)
	e.SetEscapeHTML(false)
	if err := e.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(b.Bytes(), "\n"), nil
}

// Decode reads a design's JSON, keeping the order of every object's keys. Numbers stay as written.
func Decode(data []byte) (*Obj, error) {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	v, err := value(d)
	if err != nil {
		return nil, err
	}
	if _, err := d.Token(); err != io.EOF {
		return nil, fmt.Errorf("more follows the design's closing brace")
	}
	o, ok := v.(*Obj)
	if !ok {
		return nil, fmt.Errorf("a design is a JSON object")
	}
	return o, nil
}

func value(d *json.Decoder) (any, error) {
	t, err := d.Token()
	if err != nil {
		return nil, err
	}
	switch t {
	case json.Delim('{'):
		o := newObj()
		for d.More() {
			k, err := d.Token()
			if err != nil {
				return nil, err
			}
			v, err := value(d)
			if err != nil {
				return nil, err
			}
			o.Set(k.(string), v)
		}
		_, err := d.Token()
		return o, err
	case json.Delim('['):
		l := []any{}
		for d.More() {
			v, err := value(d)
			if err != nil {
				return nil, err
			}
			l = append(l, v)
		}
		_, err := d.Token()
		return l, err
	}
	return t, nil
}
