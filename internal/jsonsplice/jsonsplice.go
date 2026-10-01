// Package jsonsplice edits one key in a JSON object document by splicing
// bytes, so every other byte of the document survives: key order,
// indentation, spacing, and fields the caller has never heard of. It is
// how pk writes into files the developer owns. Bytes in, bytes out; the
// caller reads and writes the file.
package jsonsplice

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// defaultUnit indents entries when the document gives no example.
const defaultUnit = "  "

// Get returns the raw bytes of the value at path, and whether the key is
// present. A missing object on the way counts as not present. A document
// that is not a JSON object, or a value on the way that is not one, is an
// error naming it.
func Get(content []byte, path ...string) (json.RawMessage, bool, error) {
	l, err := locate(content, path)
	if err != nil {
		return nil, false, err
	}
	if !l.found {
		return nil, false, nil
	}
	return json.RawMessage(content[l.valueStart:l.valueEnd]), true, nil
}

// Replace returns content with the value at path replaced by the
// encoding of value. The key must be present.
func Replace(content []byte, value any, path ...string) ([]byte, error) {
	l, err := locate(content, path)
	if err != nil {
		return nil, err
	}
	if !l.found {
		return nil, fmt.Errorf("%s: not present", strings.Join(path, "."))
	}
	encoded, err := encode(value, lineIndent(content, l.valueStart), l.unit, l.multiline)
	if err != nil {
		return nil, err
	}
	return splice(content, l.valueStart, l.valueEnd, encoded), nil
}

// Insert returns content with the last element of path added, with the
// encoding of value, to the object the rest of path names. The key must
// be absent; objects missing on the way are created around the value.
// The entry copies its siblings' indentation and key-value spacing, or
// the document's when the object is empty, and goes last.
func Insert(content []byte, value any, path ...string) ([]byte, error) {
	l, err := locate(content, path)
	if err != nil {
		return nil, err
	}
	if l.found {
		return nil, fmt.Errorf("%s: already present", strings.Join(path, "."))
	}
	for i := len(path) - 1; i > l.depth; i-- {
		value = map[string]any{path[i]: value}
	}
	key, err := encode(path[l.depth], "", "", false)
	if err != nil {
		return nil, err
	}
	ws := leadingWhitespace(content[l.parentOpen:])
	if l.lastValueEnd < 0 {
		// Empty object: it becomes a multi-line object with one entry,
		// indented one unit past the line that opens it.
		outer := lineIndent(content, l.parentOpen)
		inner := outer + l.unit
		encoded, err := encode(value, inner, l.unit, true)
		if err != nil {
			return nil, err
		}
		entry := "\n" + inner + string(key) + ": " + string(encoded) + "\n" + outer
		return splice(content, l.parentOpen, l.closeStart, []byte(entry)), nil
	}
	if bytes.Contains(ws, []byte("\n")) {
		indent := lineIndent(content, l.parentOpen+len(ws))
		encoded, err := encode(value, indent, l.unit, true)
		if err != nil {
			return nil, err
		}
		entry := ",\n" + indent + string(key) + l.sep + string(encoded)
		return splice(content, l.lastValueEnd, l.lastValueEnd, []byte(entry)), nil
	}
	encoded, err := encode(value, "", "", false)
	if err != nil {
		return nil, err
	}
	entry := "," + string(ws) + string(key) + l.sep + string(encoded)
	return splice(content, l.lastValueEnd, l.lastValueEnd, []byte(entry)), nil
}

// location is what one walk of the document finds for a path: the value
// span when the key is present, and otherwise the shape of the object
// that lacks it.
type location struct {
	found      bool
	valueStart int
	valueEnd   int
	depth      int // when not found, the index in path of the first absent key

	parentOpen   int    // just past the parent object's '{'
	closeStart   int    // at the parent object's '}'
	lastValueEnd int    // just past the parent's last value, -1 when empty
	sep          string // ": " or ":", after the parent's first entry; ": " when empty

	unit      string // the document's indentation unit
	multiline bool   // the root object spans lines
}

func locate(content []byte, path []string) (location, error) {
	if len(path) == 0 {
		return location{}, fmt.Errorf("empty path")
	}
	dec := json.NewDecoder(bytes.NewReader(content))
	if err := expectOpen(dec, "document"); err != nil {
		return location{}, err
	}
	l := location{sep: ": ", unit: defaultUnit}
	rootWS := leadingWhitespace(content[dec.InputOffset():])
	if i := bytes.LastIndexByte(rootWS, '\n'); i >= 0 {
		l.multiline = true
		if unit := string(rootWS[i+1:]); unit != "" {
			l.unit = unit
		}
	}
	for i, want := range path {
		last := i == len(path)-1
		l.parentOpen = int(dec.InputOffset())
		l.lastValueEnd = -1
		l.sep = ": "
		matched := false
		first := true
		for dec.More() {
			tok, err := dec.Token()
			if err != nil {
				return location{}, err
			}
			key, ok := tok.(string)
			if !ok {
				return location{}, fmt.Errorf("expected a string key, got %v", tok)
			}
			keyEnd := int(dec.InputOffset())
			if key == want {
				if !last {
					if err := expectOpen(dec, strings.Join(path[:i+1], ".")); err != nil {
						return location{}, err
					}
					matched = true
					break
				}
				start, end, err := rawSpan(dec, content)
				if err != nil {
					return location{}, err
				}
				l.found, l.valueStart, l.valueEnd = true, start, end
				return l, nil
			}
			start, end, err := rawSpan(dec, content)
			if err != nil {
				return location{}, err
			}
			if first {
				l.sep = separator(content[keyEnd:start])
				first = false
			}
			l.lastValueEnd = end
		}
		if matched {
			continue
		}
		if _, err := dec.Token(); err != nil { // the parent's '}'
			return location{}, err
		}
		l.closeStart = int(dec.InputOffset()) - 1
		l.depth = i
		return l, nil
	}
	return l, nil
}

// expectOpen consumes the next token, which must open an object.
func expectOpen(dec *json.Decoder, what string) error {
	tok, err := dec.Token()
	if err != nil {
		return fmt.Errorf("%s: expected a JSON object: %w", what, err)
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '{' {
		return fmt.Errorf("%s: expected a JSON object, got %v", what, tok)
	}
	return nil
}

// separator reduces the bytes between a key and its value to the style
// they show: a space after the colon, or none. Padding that aligns one
// value with its neighbours belongs to that entry, not to the object.
func separator(between []byte) string {
	if bytes.ContainsAny(between[bytes.IndexByte(between, ':')+1:], " \t") {
		return ": "
	}
	return ":"
}

// rawSpan decodes the next value and returns its byte span in content.
func rawSpan(dec *json.Decoder, content []byte) (int, int, error) {
	before := int(dec.InputOffset())
	var raw json.RawMessage
	if err := dec.Decode(&raw); err != nil {
		return 0, 0, err
	}
	after := int(dec.InputOffset())
	idx := bytes.Index(content[before:after], raw)
	if idx < 0 {
		return 0, 0, fmt.Errorf("could not locate a value in the source")
	}
	start := before + idx
	return start, start + len(raw), nil
}

// encode renders value without HTML escaping. With multiline, nested
// entries start on their own line at prefix plus one unit per level.
func encode(value any, prefix, unit string, multiline bool) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if multiline {
		enc.SetIndent(prefix, unit)
	}
	if err := enc.Encode(value); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}

func splice(content []byte, start, end int, in []byte) []byte {
	out := make([]byte, 0, len(content)-(end-start)+len(in))
	out = append(out, content[:start]...)
	out = append(out, in...)
	return append(out, content[end:]...)
}

func leadingWhitespace(b []byte) []byte {
	return b[:len(b)-len(bytes.TrimLeft(b, " \t\r\n"))]
}

// lineIndent returns the leading whitespace of the line containing offset.
func lineIndent(content []byte, offset int) string {
	start := bytes.LastIndexByte(content[:offset], '\n') + 1
	return string(leadingWhitespace(content[start:offset]))
}
