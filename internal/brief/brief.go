// Package brief implements pk brief: the repository's plankit policy,
// rendered as instructions for a session. It is pk status for a
// different reader. The SessionStart hook injects the text as
// additionalContext on every start, resume, and compaction; a person
// runs pk brief to read exactly what sessions are told. Both shapes
// render the same resolved config, so the brief cannot disagree with
// what the other hooks then enforce.
//
// Hook contracts as for the others: exit 0 always, and an unconfigured
// repository produces no output at all.
package brief

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/markwharton/plankit/internal/cli"
	"github.com/markwharton/plankit/internal/config"
	"github.com/markwharton/plankit/internal/git"
	"github.com/markwharton/plankit/internal/hookio"
	"github.com/markwharton/plankit/internal/msg"
)

// Cmd is the brief command: hook-driven and explicitly invocable.
var Cmd = &cli.Command{
	Name:    "brief",
	Summary: "Hook: tell the session this repository's plankit policy",
	Hook:    true,
	Flags:   []cli.FlagSpec{cli.FormatFlag},
	Run:     run,
}

func run(ctx *cli.Context) error {
	input, err := hookio.ReadInput(ctx.Stdin)
	hookInvocation := err == nil
	payloadCWD := ""
	if hookInvocation {
		payloadCWD = input.CWD
	}
	dir := hookio.ResolveDir(os.Getenv, payloadCWD, ctx.ProjectDir, ctx.ProjectDirExplicit)
	root, ok := git.FindRoot(dir)

	if hookInvocation {
		if !ok {
			return nil
		}
		cfg, err := config.Load(root)
		if errors.Is(err, config.ErrNotConfigured) {
			return nil // off here; the hook fires everywhere
		}
		if err != nil {
			msg.Hookf(ctx.Stderr, "brief", "%v", err)
			return cli.Silent(hookio.ExitReport) // shown at session start, not blocking
		}
		if err := hookio.WriteSessionStart(ctx.Stdout, Text(cfg)); err != nil {
			msg.Hookf(ctx.Stderr, "brief", "%v", err)
		}
		return nil
	}

	// A person at a terminal: show the text sessions receive.
	if !ok {
		return cli.Statef("not a git repository: %s", dir)
	}
	cfg, err := config.Load(root)
	if errors.Is(err, config.ErrNotConfigured) {
		return cli.WithHint(cli.Statef("plankit is not configured in %s", root), "run pk init to configure it")
	}
	if err != nil {
		return err
	}
	if ctx.Format == "json" {
		// The exact envelope the SessionStart hook emits.
		return hookio.WriteSessionStart(ctx.Stdout, Text(cfg))
	}
	fmt.Fprint(ctx.Stdout, Text(cfg))
	return nil
}

// Text renders the policy as prose for a session. Two sentences are
// constant, the first and the last; everything between is the resolved
// config in words, and paragraphs come and go with the dials.
func Text(cfg *config.PkConfig) string {
	var b strings.Builder
	b.WriteString("plankit is configured in this repository.\n\n")

	var types []string
	for _, tc := range cfg.Changelog.ResolvedTypes() {
		if !tc.Hidden {
			types = append(types, tc.Type)
		}
	}
	fmt.Fprintf(&b, "Commits follow Conventional Commits, type(scope): subject. Types: %s (from .pk.json changelog.types).\n\n", strings.Join(types, ", "))

	switch cfg.Guard.ResolvedBreaking() {
	case "ask":
		b.WriteString("Never add a breaking marker (! or BREAKING CHANGE) on your own judgment; only on explicit user direction. guard asks before one is committed.\n\n")
	default:
		b.WriteString("Add a breaking marker (! or BREAKING CHANGE) only on explicit user direction.\n\n")
	}

	if mode := cfg.Guard.ResolvedMode(); mode != "off" && len(cfg.Guard.Branches) > 0 {
		branches := strings.Join(cfg.Guard.Branches, ", ")
		verb := "is"
		if len(cfg.Guard.Branches) > 1 {
			verb = "are"
		}
		switch mode {
		case "block":
			fmt.Fprintf(&b, "%s %s protected: commits and pushes there are blocked.", branches, verb)
		case "ask":
			fmt.Fprintf(&b, "%s %s protected: commits and pushes there prompt for confirmation.", branches, verb)
		}
		switch cfg.Guard.ResolvedPush() {
		case "block":
			b.WriteString(" Pushing is the developer's action; do not push.")
		case "ask":
			b.WriteString(" Pushing prompts for confirmation.")
		}
		if cfg.Release.Branch != "" {
			fmt.Fprintf(&b, " Work on the development branch; releases merge into %s with pk ship.\n\n", cfg.Release.Branch)
		} else {
			b.WriteString(" Releases run from the default branch with pk ship.\n\n")
		}
	}

	switch cfg.Preserve.ResolvedMode() {
	case "auto":
		b.WriteString("docs/plans/ is immutable. Approved plans are preserved and committed there automatically.\n\n")
	case "manual":
		b.WriteString("docs/plans/ is immutable. Approved plans are preserved there; preserve mode is manual, so run /plankit:preserve to commit a pending plan.\n\n")
	default:
		b.WriteString("docs/plans/ is immutable.\n\n")
	}

	b.WriteString("pk help lists every command.\n")
	return b.String()
}
