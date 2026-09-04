// docgen compiles plankit's documentation source (skills/<topic>/SKILL.md)
// into the help package's intermediate representation.
//
// For each topic it writes <out>/ir/<topic>.json (the compiled IR) and
// <out>/raw/<topic>.md (the authored bytes, untouched). Anything outside
// the authoring subset is a build failure: the schema is the contract,
// and "does it compile" is the validation rule.
//
// Run from the repository root via: make docs
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// The IR types mirror internal/help exactly. docgen is a separate module
// (it carries goldmark; the runtime carries nothing), so the schema is
// declared on both sides and the help package's strict loader keeps the
// two honest: drift fails the embed tests.
type doc struct {
	Schema int     `json:"schema"`
	Meta   meta    `json:"meta"`
	Blocks []block `json:"blocks"`
}

type meta struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type block struct {
	Type    string    `json:"type"`
	Level   int       `json:"level,omitempty"`
	ID      string    `json:"id,omitempty"`
	Inlines []span    `json:"inlines,omitempty"`
	Lang    string    `json:"lang,omitempty"`
	Text    string    `json:"text,omitempty"`
	Runs    []run     `json:"runs,omitempty"`
	Ordered bool      `json:"ordered,omitempty"`
	Items   [][]block `json:"items,omitempty"`
	Blocks  []block   `json:"blocks,omitempty"`
}

type span struct {
	Type   string `json:"type"`
	Text   string `json:"text,omitempty"`
	Strong bool   `json:"strong,omitempty"`
	Em     bool   `json:"em,omitempty"`
	URL    string `json:"url,omitempty"`
}

type run struct {
	Kind string `json:"kind,omitempty"`
	Text string `json:"text"`
}

func main() {
	skillsDir := flag.String("skills", "", "Directory of <topic>/SKILL.md sources (required)")
	outDir := flag.String("out", "", "Output directory for ir/ and raw/ (required)")
	flag.Parse()
	if *skillsDir == "" || *outDir == "" {
		fmt.Fprintln(os.Stderr, "usage: docgen -skills <dir> -out <dir>")
		os.Exit(2)
	}
	if err := compileAll(*skillsDir, *outDir); err != nil {
		fmt.Fprintf(os.Stderr, "docgen: %v\n", err)
		os.Exit(1)
	}
}

func compileAll(skillsDir, outDir string) error {
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return err
	}
	for _, sub := range []string{"ir", "raw"} {
		dir := filepath.Join(outDir, sub)
		if err := os.RemoveAll(dir); err != nil {
			return err
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		topic := e.Name()
		src := filepath.Join(skillsDir, topic, "SKILL.md")
		raw, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		d, err := compile(topic, raw)
		if err != nil {
			return fmt.Errorf("%s: %w", src, err)
		}
		out, err := json.MarshalIndent(d, "", "  ")
		if err != nil {
			return err
		}
		out = append(out, '\n')
		if err := os.WriteFile(filepath.Join(outDir, "ir", topic+".json"), out, 0o644); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(outDir, "raw", topic+".md"), raw, 0o644); err != nil {
			return err
		}
		n++
	}
	if n == 0 {
		return fmt.Errorf("no topics found under %s", skillsDir)
	}
	fmt.Printf("docgen: compiled %d topics\n", n)
	return nil
}

func compile(topic string, source []byte) (*doc, error) {
	fm, body, err := splitFrontmatter(source)
	if err != nil {
		return nil, err
	}
	if fm["name"] != topic {
		return nil, fmt.Errorf("frontmatter name %q must equal topic directory %q", fm["name"], topic)
	}
	if fm["description"] == "" {
		return nil, fmt.Errorf("frontmatter description is required")
	}

	root := goldmark.New().Parser().Parse(text.NewReader(body))
	c := &compiler{source: body, ids: map[string]int{}}
	blocks, err := c.blocks(root)
	if err != nil {
		return nil, err
	}
	if len(blocks) == 0 || blocks[0].Type != "heading" || blocks[0].Level != 1 {
		return nil, fmt.Errorf("document must open with a level-1 heading")
	}
	return &doc{
		Schema: 1,
		Meta:   meta{Name: fm["name"], Description: fm["description"]},
		Blocks: blocks,
	}, nil
}

// splitFrontmatter parses the leading "---" block as flat "key: value"
// lines. The frontmatter is deliberately not YAML: two required keys
// need no dependency, and anything fancier should fail here.
func splitFrontmatter(source []byte) (map[string]string, []byte, error) {
	const fence = "---\n"
	s := string(source)
	if !strings.HasPrefix(s, fence) {
		return nil, nil, fmt.Errorf("missing frontmatter (file must start with ---)")
	}
	rest := s[len(fence):]
	end := strings.Index(rest, "\n"+fence)
	if end < 0 {
		return nil, nil, fmt.Errorf("unterminated frontmatter")
	}
	fm := map[string]string{}
	for _, line := range strings.Split(rest[:end], "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return nil, nil, fmt.Errorf("frontmatter line %q is not key: value", line)
		}
		fm[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	for k := range fm {
		if k != "name" && k != "description" {
			return nil, nil, fmt.Errorf("unknown frontmatter key %q (allowed: name, description)", k)
		}
	}
	body := rest[end+1+len(fence):]
	return fm, []byte(body), nil
}

type compiler struct {
	source []byte
	ids    map[string]int
}

func (c *compiler) blocks(parent ast.Node) ([]block, error) {
	out := []block{}
	for n := parent.FirstChild(); n != nil; n = n.NextSibling() {
		b, err := c.block(n)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, nil
}

func (c *compiler) block(n ast.Node) (block, error) {
	switch v := n.(type) {
	case *ast.Heading:
		if v.Level > 3 {
			return block{}, fmt.Errorf("heading level %d: the subset allows 1..3", v.Level)
		}
		inl, err := c.spans(n, span{})
		if err != nil {
			return block{}, err
		}
		return block{Type: "heading", Level: v.Level, ID: c.slug(plain(inl)), Inlines: inl}, nil
	case *ast.Paragraph, *ast.TextBlock:
		inl, err := c.spans(n, span{})
		if err != nil {
			return block{}, err
		}
		return block{Type: "para", Inlines: inl}, nil
	case *ast.FencedCodeBlock:
		var sb strings.Builder
		lines := v.Lines()
		for i := 0; i < lines.Len(); i++ {
			seg := lines.At(i)
			sb.Write(seg.Value(c.source))
		}
		return block{Type: "codeblock", Lang: string(v.Language(c.source)), Text: sb.String()}, nil
	case *ast.CodeBlock:
		return block{}, fmt.Errorf("indented code block: use a fenced block")
	case *ast.List:
		items := [][]block{}
		for li := n.FirstChild(); li != nil; li = li.NextSibling() {
			item, err := c.blocks(li)
			if err != nil {
				return block{}, err
			}
			items = append(items, item)
		}
		return block{Type: "list", Ordered: v.IsOrdered(), Items: items}, nil
	case *ast.Blockquote:
		inner, err := c.blocks(n)
		if err != nil {
			return block{}, err
		}
		return block{Type: "quote", Blocks: inner}, nil
	case *ast.ThematicBreak:
		return block{Type: "rule"}, nil
	case *ast.HTMLBlock:
		return block{}, fmt.Errorf("HTML block: outside the subset")
	default:
		return block{}, fmt.Errorf("unsupported block %s: outside the subset", n.Kind())
	}
}

// spans flattens the inline tree onto flag-carrying spans: nesting
// resolves at compile time so the runtime walker is a single loop.
func (c *compiler) spans(parent ast.Node, state span) ([]span, error) {
	out := []span{}
	for n := parent.FirstChild(); n != nil; n = n.NextSibling() {
		switch v := n.(type) {
		case *ast.Text:
			t := string(v.Segment.Value(c.source))
			if v.SoftLineBreak() {
				t += " " // soft breaks collapse: the runtime re-wraps
			}
			out = appendText(out, state, t)
			if v.HardLineBreak() {
				out = append(out, span{Type: "br"})
			}
		case *ast.String:
			out = appendText(out, state, string(v.Value))
		case *ast.Emphasis:
			sub := state
			if v.Level >= 2 {
				sub.Strong = true
			} else {
				sub.Em = true
			}
			inner, err := c.spans(n, sub)
			if err != nil {
				return nil, err
			}
			out = append(out, inner...)
		case *ast.CodeSpan:
			out = append(out, span{Type: "code", Text: nodeText(n, c.source), URL: state.URL})
		case *ast.Link:
			sub := state
			sub.URL = string(v.Destination)
			inner, err := c.spans(n, sub)
			if err != nil {
				return nil, err
			}
			out = append(out, inner...)
		case *ast.AutoLink:
			u := string(v.URL(c.source))
			out = append(out, span{Type: "text", Text: u, Strong: state.Strong, Em: state.Em, URL: u})
		case *ast.RawHTML:
			return nil, fmt.Errorf("inline HTML: outside the subset")
		case *ast.Image:
			return nil, fmt.Errorf("image: outside the subset")
		default:
			return nil, fmt.Errorf("unsupported inline %s: outside the subset", n.Kind())
		}
	}
	return out, nil
}

// appendText merges consecutive same-styled text into one span.
func appendText(out []span, state span, t string) []span {
	if t == "" {
		return out
	}
	if n := len(out); n > 0 {
		last := &out[n-1]
		if last.Type == "text" && last.Strong == state.Strong && last.Em == state.Em && last.URL == state.URL {
			last.Text += t
			return out
		}
	}
	return append(out, span{Type: "text", Text: t, Strong: state.Strong, Em: state.Em, URL: state.URL})
}

func nodeText(n ast.Node, source []byte) string {
	var sb bytes.Buffer
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		switch v := c.(type) {
		case *ast.Text:
			sb.Write(v.Segment.Value(source))
		case *ast.String:
			sb.Write(v.Value)
		}
	}
	return sb.String()
}

func plain(spans []span) string {
	var sb strings.Builder
	for _, s := range spans {
		sb.WriteString(s.Text)
	}
	return sb.String()
}

var slugStrip = regexp.MustCompile(`[^a-z0-9]+`)

// slug produces GitHub-style heading ids, deduplicated per document.
func (c *compiler) slug(title string) string {
	s := strings.Trim(slugStrip.ReplaceAllString(strings.ToLower(title), "-"), "-")
	if s == "" {
		s = "section"
	}
	c.ids[s]++
	if n := c.ids[s]; n > 1 {
		return fmt.Sprintf("%s-%d", s, n)
	}
	return s
}
