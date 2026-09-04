// pk is the plan-driven development toolkit for Claude Code.
//
// The binary is distributed by the plankit Claude Code plugin, which also
// carries the documentation as skills; pk help renders the same pages in
// the terminal.
package main

import (
	"os"

	"github.com/markwharton/plankit/internal/cli"
	"github.com/markwharton/plankit/internal/version"
)

func main() {
	os.Exit(cli.Run(os.Args, commands()))
}

// commands is the explicit registry. Each layer appends here; there is no
// init-time magic, so reading this list is reading the product surface.
func commands() []*cli.Command {
	return []*cli.Command{
		version.Cmd,
	}
}
