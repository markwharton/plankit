// Package help embeds plankit's documentation as a compiled intermediate
// representation and renders it for the terminal.
//
// skills/<topic>/SKILL.md is the single documentation source. At build
// time tools/docgen compiles each topic into data/ir/<topic>.json (this
// package's schema) and copies the authored bytes to data/raw/<topic>.md.
// The renderer walks the IR with a style table; it never parses markdown.
// Non-TTY readers get the raw authored bytes, so what Claude reads is
// exactly the skill file.
package help

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// SchemaVersion is the IR schema this package understands. docgen stamps
// it into every document; Load rejects anything else.
const SchemaVersion = 1

// Doc is one compiled documentation topic.
type Doc struct {
	Schema int     `json:"schema"`
	Meta   Meta    `json:"meta"`
	Blocks []Block `json:"blocks"`
}

// Meta mirrors the skill frontmatter.
type Meta struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Block is one block-level node. Type selects which fields are set:
//
//	heading    level (1..3), id, inlines
//	para       inlines
//	codeblock  lang (optional), and text or runs (exactly one)
//	list       ordered (optional), items
//	quote      blocks
//	rule       nothing
type Block struct {
	Type    string    `json:"type"`
	Level   int       `json:"level,omitempty"`
	ID      string    `json:"id,omitempty"`
	Inlines []Span    `json:"inlines,omitempty"`
	Lang    string    `json:"lang,omitempty"`
	Text    string    `json:"text,omitempty"`
	Runs    []Run     `json:"runs,omitempty"`
	Ordered bool      `json:"ordered,omitempty"`
	Items   [][]Block `json:"items,omitempty"`
	Blocks  []Block   `json:"blocks,omitempty"`
}

// Span is one inline node. Type selects the shape:
//
//	text  text, with optional strong, em, url
//	code  text, with optional url
//	br    a hard line break
type Span struct {
	Type   string `json:"type"`
	Text   string `json:"text,omitempty"`
	Strong bool   `json:"strong,omitempty"`
	Em     bool   `json:"em,omitempty"`
	URL    string `json:"url,omitempty"`
}

// Run is one highlight run inside a codeblock. Kind is a collapsed token
// class (kw, str, com, num, name); empty means unstyled.
type Run struct {
	Kind string `json:"kind,omitempty"`
	Text string `json:"text"`
}

// Load decodes and validates one IR document. Unknown keys are rejected:
// the schema is the subset contract, and drift fails loudly.
func Load(data []byte) (*Doc, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var d Doc
	if err := dec.Decode(&d); err != nil {
		return nil, err
	}
	if err := d.validate(); err != nil {
		return nil, err
	}
	return &d, nil
}

func (d *Doc) validate() error {
	if d.Schema != SchemaVersion {
		return fmt.Errorf("schema %d, want %d", d.Schema, SchemaVersion)
	}
	if d.Meta.Name == "" || d.Meta.Description == "" {
		return fmt.Errorf("meta.name and meta.description are required")
	}
	return validateBlocks(d.Blocks)
}

func validateBlocks(blocks []Block) error {
	for i, b := range blocks {
		if err := b.validate(); err != nil {
			return fmt.Errorf("block %d: %w", i, err)
		}
	}
	return nil
}

func (b *Block) validate() error {
	switch b.Type {
	case "heading":
		if b.Level < 1 || b.Level > 3 {
			return fmt.Errorf("heading level %d, want 1..3", b.Level)
		}
		if b.ID == "" {
			return fmt.Errorf("heading missing id")
		}
		return validateSpans(b.Inlines)
	case "para":
		return validateSpans(b.Inlines)
	case "codeblock":
		if (b.Text == "") == (len(b.Runs) == 0) {
			return fmt.Errorf("codeblock needs exactly one of text or runs")
		}
		return nil
	case "list":
		if len(b.Items) == 0 {
			return fmt.Errorf("list has no items")
		}
		for _, item := range b.Items {
			if err := validateBlocks(item); err != nil {
				return err
			}
		}
		return nil
	case "quote":
		return validateBlocks(b.Blocks)
	case "rule":
		return nil
	default:
		return fmt.Errorf("unknown block type %q", b.Type)
	}
}

func validateSpans(spans []Span) error {
	for _, s := range spans {
		switch s.Type {
		case "text", "code", "br":
		default:
			return fmt.Errorf("unknown span type %q", s.Type)
		}
	}
	return nil
}
