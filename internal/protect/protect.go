// Package protect implements the protect PreToolUse hook: it denies
// Edit/Write operations targeting files under docs/plans/, which are
// immutable historical records. Ported from v1; the v2 addition is the
// not-configured short-circuit, since the plugin's hooks fire in every
// repository (v1 protected unconditionally).
package protect

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/markwharton/plankit/internal/cli"
	"github.com/markwharton/plankit/internal/config"
	"github.com/markwharton/plankit/internal/git"
	"github.com/markwharton/plankit/internal/hookio"
	"github.com/markwharton/plankit/internal/msg"
	"github.com/markwharton/plankit/internal/paths"
)

// Cmd is the protect hook command.
var Cmd = &cli.Command{
	Name:    "protect",
	Summary: "Hook: keep preserved plans in docs/plans immutable",
	Hook:    true,
	Run:     run,
}

func run(ctx *cli.Context) error {
	input, err := hookio.ReadInput(ctx.Stdin)
	if err != nil {
		msg.Hookf(ctx.Stderr, "protect", "failed to read input: %v", err)
		return nil
	}
	if input.ToolInput == nil || input.ToolInput.FilePath == "" {
		return nil
	}

	dir := hookio.ResolveDir(os.Getenv, input.CWD, ctx.ProjectDir)
	root, ok := git.FindRoot(dir)
	if !ok {
		return nil
	}
	if _, err := config.Load(root); err != nil {
		if !errors.Is(err, config.ErrNotConfigured) {
			msg.Hookf(ctx.Stderr, "protect", "%v", err)
		}
		return nil // off, or policy unreadable: fail open
	}

	if isUnderPlansDir(input.ToolInput.FilePath, root) {
		err := hookio.WritePermissionDecision(ctx.Stdout, hookio.PermissionDeny,
			"docs/plans/ files are immutable historical records. They must not be edited or overwritten after creation.")
		if err != nil {
			msg.Hookf(ctx.Stderr, "protect", "write error: %v", err)
		}
	}
	return nil
}

// isUnderPlansDir checks whether filePath is under root/docs/plans/,
// resolving symlinks (best-effort) to prevent bypass via links, and
// comparing case-insensitively on Windows.
func isUnderPlansDir(filePath, root string) bool {
	plansDir := paths.Plans(root)
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(root, filePath)
	}
	cleanFile := filepath.Clean(filePath)
	cleanPlans := filepath.Clean(plansDir)
	if resolved, err := filepath.EvalSymlinks(cleanFile); err == nil {
		cleanFile = resolved
	}
	if resolved, err := filepath.EvalSymlinks(cleanPlans); err == nil {
		cleanPlans = resolved
	}
	if runtime.GOOS == "windows" {
		cleanFile = strings.ToLower(cleanFile)
		cleanPlans = strings.ToLower(cleanPlans)
	}
	return strings.HasPrefix(cleanFile, cleanPlans+string(filepath.Separator))
}
