// Package registry is the command list: one place every command is
// declared, read by main to dispatch, by the invariant tests, and by
// docgen to write each page's Flags section from the same structs
// --help prints.
package registry

import (
	"github.com/markwharton/plankit/internal/brief"
	"github.com/markwharton/plankit/internal/changelog"
	"github.com/markwharton/plankit/internal/cli"
	"github.com/markwharton/plankit/internal/guard"
	"github.com/markwharton/plankit/internal/help"
	"github.com/markwharton/plankit/internal/preserve"
	"github.com/markwharton/plankit/internal/protect"
	"github.com/markwharton/plankit/internal/release"
	"github.com/markwharton/plankit/internal/repo"
	"github.com/markwharton/plankit/internal/ship"
	"github.com/markwharton/plankit/internal/version"
)

// Commands returns every pk command.
func Commands() []*cli.Command {
	return []*cli.Command{
		changelog.Cmd,
		guard.Cmd,
		brief.Cmd,
		help.Cmd,
		preserve.Cmd,
		protect.Cmd,
		release.Cmd,
		release.PinCmd,
		ship.Cmd,
		repo.InitCmd,
		repo.StatusCmd,
		version.Cmd,
	}
}
