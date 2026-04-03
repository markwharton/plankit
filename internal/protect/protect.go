// Package protect implements the protect-plans PreToolUse hook command.
// It blocks Claude Code Edit/Write operations targeting files in docs/plans/.
package protect

import (
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/markwharton/plankit/internal/hooks"
)

// Run reads a PreToolUse hook payload from stdin and blocks edits to docs/plans/.
// Returns the process exit code (always 0 for hook commands).
func Run(stdin io.Reader, stdout io.Writer, stderr io.Writer, env func(string) string) int {
	input, err := hooks.ReadInput(stdin)
	if err != nil {
		fmt.Fprintf(stderr, "pk protect: failed to read input: %v\n", err)
		return 0
	}

	if input.ToolInput == nil || input.ToolInput.FilePath == "" {
		return 0
	}

	projectDir := env("CLAUDE_PROJECT_DIR")
	if projectDir == "" {
		projectDir = input.CWD
	}
	if projectDir == "" {
		return 0
	}

	if isUnderPlansDir(input.ToolInput.FilePath, projectDir) {
		fmt.Fprint(stdout, `{"decision":"block","reason":"docs/plans/ files are immutable historical records. They must not be edited or overwritten after creation."}`)
	}

	return 0
}

// isUnderPlansDir checks whether filePath is under projectDir/docs/plans/.
func isUnderPlansDir(filePath, projectDir string) bool {
	plansDir := filepath.Join(projectDir, "docs", "plans")

	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(projectDir, filePath)
	}

	cleanFile := filepath.Clean(filePath)
	cleanPlans := filepath.Clean(plansDir)

	if runtime.GOOS == "windows" {
		cleanFile = strings.ToLower(cleanFile)
		cleanPlans = strings.ToLower(cleanPlans)
	}

	return strings.HasPrefix(cleanFile, cleanPlans+string(filepath.Separator))
}
