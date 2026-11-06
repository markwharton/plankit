// Package protect implements the protect PreToolUse hook: it denies
// Edit/Write operations targeting files under the plans directory,
// which holds immutable historical records. Ported from v1; the v2 addition is the
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
)

// Cmd is the protect hook command.
var Cmd = &cli.Command{
	Name:    "protect",
	Summary: "Hook: keep preserved plans immutable",
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

	dir := hookio.ResolveDir(os.Getenv, input.CWD, ctx.ProjectDir, ctx.ProjectDirExplicit)
	root, ok := git.FindRoot(dir)
	if !ok {
		return nil
	}
	cfg, err := config.Load(root)
	if err != nil {
		if errors.Is(err, config.ErrNotConfigured) {
			return nil // off here; the hook fires everywhere
		}
		msg.Hookf(ctx.Stderr, "protect", "%v", err)
		return cli.Silent(hookio.ExitReport) // shown, not blocking; protect takes no action
	}

	if isUnderPlansDir(input.ToolInput.FilePath, root, cfg.Preserve.DirPath(root)) {
		err := hookio.WritePermissionDecision(ctx.Stdout, hookio.PermissionDeny,
			cfg.Preserve.ResolvedDir()+"/ files are immutable historical records. They must not be edited or overwritten after creation.")
		if err != nil {
			msg.Hookf(ctx.Stderr, "protect", "write error: %v", err)
		}
	}
	return nil
}

// isUnderPlansDir checks whether filePath is under plansDir, the
// configured plans directory under root, resolving symlinks to prevent
// bypass via links, and comparing case-insensitively on Windows.
//
// Both sides resolve through resolveExisting, never EvalSymlinks
// directly: the write target usually does not exist yet, and resolving
// only the side that exists produces mismatched prefixes wherever the
// path crosses a symlink. macOS makes that the common case, not the
// edge: /var is a symlink to /private/var, so a payload cwd under
// /var/folders (every TMPDIR) resolves to /private on the existing
// side and stays /var on the missing side. v1 had this asymmetry too.
//
// Spellings cannot say whether two paths are one directory, so a plans
// directory that exists is also recognised by identity (isInside). The
// comparison of paths stays for the directory that does not exist yet:
// it has no identity, and its first file is refused all the same.
func isUnderPlansDir(filePath, root, plansDir string) bool {
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(root, filePath)
	}
	cleanFile := resolveExisting(filePath)
	cleanPlans := resolveExisting(plansDir)
	if runtime.GOOS == "windows" {
		cleanFile = strings.ToLower(cleanFile)
		cleanPlans = strings.ToLower(cleanPlans)
	}
	if strings.HasPrefix(cleanFile, cleanPlans+string(filepath.Separator)) {
		return true
	}
	return isInside(filePath, plansDir)
}

// isInside reports whether a directory above file is dir itself, the
// same directory on disk, whatever either path is spelled like. The
// filesystem answers, so a spelling that differs only by case matches
// exactly where the filesystem ignores case. A dir that does not exist
// has no identity and nothing is inside it.
func isInside(file, dir string) bool {
	target, err := os.Stat(dir)
	if err != nil {
		return false
	}
	p := filepath.Dir(filepath.Clean(file))
	for {
		if info, err := os.Stat(p); err == nil && os.SameFile(info, target) {
			return true
		}
		parent := filepath.Dir(p)
		if parent == p {
			return false
		}
		p = parent
	}
}

// resolveExisting resolves symlinks through the nearest existing
// ancestor: EvalSymlinks the deepest path element that exists, then
// rejoin the missing tail. A path with no resolvable ancestor comes
// back cleaned but unresolved, which still compares consistently
// because both sides of the check go through this same function.
func resolveExisting(path string) string {
	p := filepath.Clean(path)
	suffix := ""
	for {
		if resolved, err := filepath.EvalSymlinks(p); err == nil {
			return filepath.Join(resolved, suffix)
		}
		parent := filepath.Dir(p)
		if parent == p {
			return filepath.Clean(path)
		}
		suffix = filepath.Join(filepath.Base(p), suffix)
		p = parent
	}
}
