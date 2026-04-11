// Package protect implements the protect-plans PreToolUse hook command.
// It blocks Claude Code Edit/Write operations targeting files in docs/plans/.
package protect

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/markwharton/plankit/internal/hooks"
)

// Config holds the dependencies for the protect command.
type Config struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	Env    func(string) string
}

// DefaultConfig returns a Config wired to real OS resources.
func DefaultConfig() Config {
	return Config{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Env:    os.Getenv,
	}
}

// Run reads a PreToolUse hook payload from stdin and blocks edits to docs/plans/.
// Returns the process exit code (always 0 for hook commands).
func Run(cfg Config) int {
	input, err := hooks.ReadInput(cfg.Stdin)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "pk protect: failed to read input: %v\n", err)
		return 0
	}

	if input.ToolInput == nil || input.ToolInput.FilePath == "" {
		return 0
	}

	projectDir := cfg.Env("CLAUDE_PROJECT_DIR")
	if projectDir == "" {
		projectDir = input.CWD
	}
	if projectDir == "" {
		return 0
	}

	if isUnderPlansDir(input.ToolInput.FilePath, projectDir) {
		hooks.WritePermissionDecision(cfg.Stdout, hooks.PermissionDeny, "docs/plans/ files are immutable historical records. They must not be edited or overwritten after creation.")
	}

	return 0
}

// isUnderPlansDir checks whether filePath is under projectDir/docs/plans/.
// Resolves symlinks to prevent bypass via symbolic links.
func isUnderPlansDir(filePath, projectDir string) bool {
	plansDir := filepath.Join(projectDir, "docs", "plans")

	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(projectDir, filePath)
	}

	cleanFile := filepath.Clean(filePath)
	cleanPlans := filepath.Clean(plansDir)

	// Resolve symlinks if possible (best-effort; fall back to cleaned paths).
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
