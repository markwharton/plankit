package help

import (
	"fmt"
	"strings"

	"github.com/markwharton/plankit/internal/cli"
)

// Cmd is the help command: the terminal reader for the same documents
// the plankit plugin ships as skills.
var Cmd = &cli.Command{
	Name:    "help",
	Summary: "Show documentation for pk and its commands",
	Run:     run,
}

func run(ctx *cli.Context) error {
	args := ctx.Args()
	if len(args) == 0 {
		return toc(ctx)
	}
	name := args[0]
	doc, raw, ok := Topic(name)
	if !ok {
		names, _ := topicNames()
		return cli.WithHint(
			cli.Usagef("unknown help topic %q", name),
			"available topics: %s", strings.Join(names, ", "))
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
	for _, m := range metas {
		pad := strings.Repeat(" ", width-len(m.Name))
		fmt.Fprintf(ctx.Stdout, "  %s%s%s%s  %s\n", strong, m.Name, reset, pad, m.Description)
	}
	fmt.Fprintf(ctx.Stdout, "\nRun 'pk help <topic>'. The same pages ship as /plankit: skills in Claude Code.\n")
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
