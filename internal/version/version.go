// Package version reports the pk build version and registers the version
// command, the first command through the execution frame.
package version

import (
	"encoding/json"
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/markwharton/plankit/internal/cli"
)

// stamped is set at release time via:
//
//	-ldflags "-X github.com/markwharton/plankit/internal/version.stamped=1.2.3"
var stamped = ""

// Version returns the stamped release version, the module version for
// a go-install build, or a dev identifier derived from VCS metadata
// when building from a source checkout.
func Version() string {
	if stamped != "" {
		return stamped
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		return fromBuildInfo(info)
	}
	return "dev"
}

// fromBuildInfo resolves a version from build metadata. A module build
// (go install module@version) carries the tag as Main.Version; a
// source-checkout build reports "(devel)" there and carries vcs.*
// settings instead.
func fromBuildInfo(info *debug.BuildInfo) string {
	if v := info.Main.Version; v != "" && v != "(devel)" {
		return strings.TrimPrefix(v, "v")
	}
	rev, dirty := "", false
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	if rev != "" {
		v := "dev+" + rev[:min(9, len(rev))]
		if dirty {
			v += ".dirty"
		}
		return v
	}
	return "dev"
}

// Cmd is the version command.
var Cmd = &cli.Command{
	Name:    "version",
	Summary: "Print the pk version",
	Flags: []cli.FlagSpec{
		{Name: "verbose", Type: cli.BoolFlag, Usage: "Show build details"},
		cli.FormatFlag,
	},
	Run: run,
}

func run(ctx *cli.Context) error {
	v := Version()
	if ctx.Format == "json" {
		out := map[string]string{
			"version": v,
			"go":      runtime.Version(),
			"os":      runtime.GOOS,
			"arch":    runtime.GOARCH,
		}
		enc := json.NewEncoder(ctx.Stdout)
		return enc.Encode(out)
	}
	fmt.Fprintf(ctx.Stdout, "pk %s\n", v)
	if ctx.Bool("verbose") {
		fmt.Fprintf(ctx.Stdout, "go: %s\nplatform: %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
