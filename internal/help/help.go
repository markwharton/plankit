package help

import (
	"fmt"
	"strings"

	"github.com/markwharton/plankit/internal/cli"
)

// Document is the reserved topic name for every page in index order,
// read as one document. It is not a skill: run intercepts it before the
// topic lookup, and a test holds the name free.
const Document = "document"

// Cmd is the help command: the terminal reader for the same documents
// the plankit plugin ships as skills.
//
// --format is declared here rather than taken from cli.FormatFlag: this
// command's artifact is a document, so its formats are text and man,
// and the usage line has to say so.
var Cmd = &cli.Command{
	Name:    "help",
	Summary: "Show documentation for pk and its commands",
	MaxArgs: 1,
	Flags: []cli.FlagSpec{
		{Name: "format", Type: cli.StringFlag, Default: "text", Usage: "Output format: text or man"},
	},
	Formats: []string{"text", "man"},
	Run:     run,
}

func run(ctx *cli.Context) error {
	args := ctx.Args()
	if len(args) == 0 {
		return toc(ctx)
	}
	name := args[0]
	if name == Document {
		return document(ctx)
	}
	doc, raw, ok := Topic(name)
	if !ok {
		names, _ := topicNames()
		return cli.WithHint(
			cli.Usagef("unknown help topic %q", name),
			"available topics: %s", strings.Join(names, ", "))
	}

	if ctx.Format == "man" {
		out, err := RenderMan(doc)
		if err != nil {
			return err
		}
		_, err = fmt.Fprint(ctx.Stdout, out)
		return err
	}

	// Non-TTY readers get the exact authored bytes: what Claude reads is
	// the skill file, byte for byte. Terminals (and forced color) get the
	// rendered IR.
	if !ctx.IsTTY && ctx.Style == cli.StyleNone {
		_, err := ctx.Stdout.Write(raw)
		return err
	}
	_, err := fmt.Fprint(ctx.Stdout, Render(doc, ctx.Style, ctx.Width))
	return err
}

// document writes every page in index order as one document. Unlike a
// single topic, which is the authored bytes, this is a reader's
// document: frontmatter is metadata for the index and the plugin
// loader, not prose to meet once per page.
func document(ctx *cli.Context) error {
	metas, err := Topics()
	if err != nil {
		return err
	}
	docs := make([]*Doc, 0, len(metas))
	raws := make([][]byte, 0, len(metas))
	for _, m := range metas {
		d, raw, ok := Topic(m.Name)
		if !ok {
			return fmt.Errorf("topic %s is indexed but not embedded", m.Name)
		}
		docs = append(docs, d)
		raws = append(raws, raw)
	}

	if ctx.Format == "man" {
		out, err := RenderManDocument(docs)
		if err != nil {
			return err
		}
		_, err = fmt.Fprint(ctx.Stdout, out)
		return err
	}

	// The same fork a single topic takes: a flag outranks the terminal
	// probe, and the probe decides rendered against source.
	var sb strings.Builder
	if !ctx.IsTTY && ctx.Style == cli.StyleNone {
		for i, raw := range raws {
			if i > 0 {
				sb.WriteString("\n---\n\n")
			}
			sb.Write(stripFrontmatter(raw))
		}
	} else {
		for i, d := range docs {
			if i > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(Render(d, ctx.Style, ctx.Width))
		}
	}
	_, err = fmt.Fprint(ctx.Stdout, sb.String())
	return err
}

// stripFrontmatter removes a leading --- block. A page without one is
// returned unchanged.
func stripFrontmatter(raw []byte) []byte {
	s := string(raw)
	if !strings.HasPrefix(s, "---\n") {
		return raw
	}
	i := strings.Index(s[4:], "\n---\n")
	if i < 0 {
		return raw
	}
	return []byte(strings.TrimLeft(s[4+i+len("\n---\n"):], "\n"))
}

func toc(ctx *cli.Context) error {
	metas, err := Topics()
	if err != nil {
		return err
	}
	width := 0
	for _, m := range metas {
		if len(m.Name) > width {
			width = len(m.Name)
		}
	}
	strong, reset := "", ""
	if ctx.Style == cli.StyleANSI {
		strong, reset = "\x1b[1m", "\x1b[0m"
	}
	fmt.Fprintf(ctx.Stdout, "Documentation topics:\n\n")
	for i, m := range metas {
		if i > 0 && m.Command && !metas[i-1].Command {
			fmt.Fprintln(ctx.Stdout) // documents above, commands below
		}
		pad := strings.Repeat(" ", width-len(m.Name))
		fmt.Fprintf(ctx.Stdout, "  %s%s%s%s  %s\n", strong, m.Name, reset, pad, m.Description)
	}
	fmt.Fprintf(ctx.Stdout, "\nRun 'pk help <topic>', or 'pk help %s' for every page in order. The same pages ship as /plankit: skills in Claude Code.\n", Document)
	return nil
}

func topicNames() ([]string, error) {
	metas, err := Topics()
	if err != nil {
		return nil, err
	}
	names := make([]string, len(metas))
	for i, m := range metas {
		names[i] = m.Name
	}
	return names, nil
}
