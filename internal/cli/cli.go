// Package cli is plankit's execution frame: the command registry, the
// resolved invocation context, and the output contract.
//
// Commands are declared as data (Command, FlagSpec) and handed to Run.
// The frame materializes flag sets, layers the universal flags onto every
// command, resolves everything once into a Context, dispatches, and maps
// returned errors onto the exit-code taxonomy. Because the declarations
// are data, the documentation compiler imports this package and derives
// each command's flag reference from the same source that drives dispatch.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/markwharton/plankit/internal/msg"
	"strings"
)

// FlagType enumerates the flag kinds a command may declare.
type FlagType int

const (
	BoolFlag FlagType = iota
	StringFlag
)

// FlagSpec declares one command flag as data.
type FlagSpec struct {
	Name    string // kebab-case, without leading dashes
	Type    FlagType
	Default string // literal default; empty for bools means false
	Usage   string // one line, sentence case, no trailing period
}

// Command declares one pk command as data.
type Command struct {
	Name    string
	Summary string     // one line for the CLI usage index; the skill's frontmatter description is deliberately separate
	Hook    bool       // hook-driven: invoked by Claude Code, not by people
	Flags   []FlagSpec // each spec both registers its flag and documents it: --help derives from here
	MaxArgs int        // positional arguments accepted; zero for most commands
	Run     func(*Context) error
}

// universalFlags are layered onto every command by the frame. Commands
// receive their resolved values through Context and never redeclare them.
var universalFlags = []FlagSpec{
	{Name: "project-dir", Type: StringFlag, Usage: "Project directory (default: PK_PROJECT_DIR, else the current directory)"},
	{Name: "plain", Type: BoolFlag, Usage: "Undecorated output: no color, no wrapping"},
	{Name: "quiet", Type: BoolFlag, Usage: "Suppress notes and hints (errors still print)"},
}

// Run dispatches argv against the registered commands and returns the
// process exit code. Streams default to the process's; tests substitute
// buffers via RunIO.
func Run(argv []string, cmds []*Command) int {
	return RunIO(argv, cmds, os.Stdin, os.Stdout, os.Stderr)
}

// RunIO is Run with explicit streams.
func RunIO(argv []string, cmds []*Command, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(argv) < 2 {
		printUsage(stderr, cmds)
		return ExitUsage
	}

	name := argv[1]
	switch name {
	case "--help", "-h":
		name = "help"
	case "--version", "-v":
		name = "version"
	}
	args := argv[2:]

	cmd := lookup(cmds, name)
	if cmd == nil {
		if name == "help" { // help not registered yet (layer 1): fall back
			printUsage(stdout, cmds)
			return ExitOK
		}
		msg.Errorf(stderr, "unknown command %q", name)
		fmt.Fprintln(stderr)
		printUsage(stderr, cmds)
		return ExitUsage
	}

	ctx, err := parse(cmd, cmds, args, stdin, stdout, stderr)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printCommandUsage(stdout, cmd)
			return ExitOK
		}
		msg.Errorf(stderr, "%v", err)
		fmt.Fprintln(stderr)
		printCommandUsage(stderr, cmd)
		return ExitUsage
	}

	return report(ctx, cmd.Run(ctx))
}

// report maps a command's returned error onto the exit-code taxonomy and
// prints it through the message contract.
func report(ctx *Context, err error) int {
	if err == nil {
		return ExitOK
	}
	var ee *exitError
	if errors.As(err, &ee) {
		if ee.msg != "" {
			msg.Errorf(ctx.Stderr, "%s", ee.msg)
		}
		if ee.hint != "" && !ctx.Quiet {
			msg.Hintf(ctx.Stderr, "%s", ee.hint)
		}
		return ee.code
	}
	msg.Errorf(ctx.Stderr, "%v", err)
	return ExitInternal
}

func lookup(cmds []*Command, name string) *Command {
	for _, c := range cmds {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// parse materializes the command's flag set (universal flags first),
// parses args, and resolves the Context.
func parse(cmd *Command, cmds []*Command, args []string, stdin io.Reader, stdout, stderr io.Writer) (*Context, error) {
	fs := flag.NewFlagSet(cmd.Name, flag.ContinueOnError)
	fs.SetOutput(io.Discard) // usage and errors are printed by the frame

	bools := map[string]*bool{}
	strs := map[string]*string{}
	declare := func(specs []FlagSpec) {
		for _, s := range specs {
			switch s.Type {
			case BoolFlag:
				bools[s.Name] = fs.Bool(s.Name, s.Default == "true", s.Usage)
			case StringFlag:
				strs[s.Name] = fs.String(s.Name, s.Default, s.Usage)
			}
		}
	}
	declare(universalFlags)
	declare(cmd.Flags)

	// Go's flag parser stops at the first positional; keep parsing
	// after each one so a flag may follow a positional (pk help craft
	// --plain). "--" still ends flags: everything after it is
	// positional.
	var positionals []string
	remaining := args
	for {
		if err := fs.Parse(remaining); err != nil {
			return nil, err
		}
		rest := fs.Args()
		if len(rest) == 0 {
			break
		}
		consumed := len(remaining) - len(rest)
		if consumed > 0 && remaining[consumed-1] == "--" {
			positionals = append(positionals, rest...)
			break
		}
		positionals = append(positionals, rest[0])
		remaining = rest[1:]
	}
	if len(positionals) > cmd.MaxArgs {
		return nil, fmt.Errorf("unexpected argument: %q", positionals[cmd.MaxArgs])
	}

	ctx := &Context{
		Command:  cmd.Name,
		Commands: cmds,
		Quiet:    *bools["quiet"],
		Stdin:    stdin,
		Stdout:   stdout,
		Stderr:   stderr,
		bools:    bools,
		strs:     strs,
		args:     positionals,
	}
	format := "text"
	if f, ok := strs["format"]; ok {
		format = *f
	}
	if err := ctx.resolve(*strs["project-dir"], format, *bools["plain"]); err != nil {
		return nil, err
	}
	return ctx, nil
}

// printUsage writes the top-level usage: user commands first, hook
// commands after, both aligned and alphabetical.
func printUsage(w io.Writer, cmds []*Command) {
	fmt.Fprint(w, "pk - plan-driven development toolkit for Claude Code\n\nUsage: pk <command> [flags]\n")

	user, hook := []*Command{}, []*Command{}
	width := 0
	for _, c := range cmds {
		if len(c.Name) > width {
			width = len(c.Name)
		}
		if c.Hook {
			hook = append(hook, c)
		} else {
			user = append(user, c)
		}
	}
	byName := func(s []*Command) {
		sort.Slice(s, func(i, j int) bool { return s[i].Name < s[j].Name })
	}
	byName(user)
	byName(hook)

	section := func(title string, s []*Command) {
		if len(s) == 0 {
			return
		}
		fmt.Fprintf(w, "\n%s\n", title)
		for _, c := range s {
			fmt.Fprintf(w, "  %-*s  %s\n", width, c.Name, c.Summary)
		}
	}
	section("Commands:", user)
	section("Hook commands (called by Claude Code, not directly):", hook)

	fmt.Fprint(w, "\nRun 'pk help <command>' for documentation, 'pk <command> --help' for flags.\n")
}

// printCommandUsage writes one command's usage with flags in the
// documented --kebab-case form; Go's default prints single-dash. The
// command's own flags print before the universal ones.
func printCommandUsage(w io.Writer, cmd *Command) {
	suffix := ""
	if len(cmd.Flags) > 0 || !cmd.Hook {
		suffix = " [flags]"
	}
	fmt.Fprintf(w, "Usage: pk %s%s\n\n%s\n", cmd.Name, suffix, cmd.Summary)
	if cmd.Hook {
		fmt.Fprint(w, "\nHook command: reads JSON on stdin, writes JSON on stdout.\n")
	}
	printFlagBlock(w, "Flags:", cmd.Flags)
	printFlagBlock(w, "Universal flags:", universalFlags)
	fmt.Fprintf(w, "\nDocumentation: pk help %s\n", cmd.Name)
}

func printFlagBlock(w io.Writer, title string, specs []FlagSpec) {
	if len(specs) == 0 {
		return
	}
	fmt.Fprintf(w, "\n%s\n%s", title, FlagBlock(specs))
}

// FlagBlock renders flag specs as the reference lines --help prints:
// the flag with <value> for string flags, then its usage and default.
// docgen writes the same block into each command's page, so the page
// and --help are one rendering of one declaration.
func FlagBlock(specs []FlagSpec) string {
	var b strings.Builder
	for _, s := range specs {
		line := "  --" + s.Name
		if s.Type == StringFlag {
			line += " <value>"
		}
		fmt.Fprintf(&b, "%s\n        %s", line, s.Usage)
		if s.Default != "" && s.Default != "false" {
			fmt.Fprintf(&b, " (default %s)", s.Default)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// UniversalFlags returns the flags every command accepts.
func UniversalFlags() []FlagSpec { return universalFlags }

// FormatFlag is --format, declared by the commands whose output has a
// structured form. It is not universal: a command that lists it
// honors it, and one that does not refuses it as an unknown flag.
var FormatFlag = FlagSpec{Name: "format", Type: StringFlag, Default: "text", Usage: "Output format: text or json"}
