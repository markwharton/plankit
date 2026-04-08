// Package git provides shared git command execution.
package git

import (
	"os/exec"
	"strings"
)

// Exec runs a git command and returns trimmed output.
// If dir is non-empty, git runs with -C dir.
func Exec(dir string, args ...string) (string, error) {
	if dir != "" {
		args = append([]string{"-C", dir}, args...)
	}
	out, err := exec.Command("git", args...).CombinedOutput()
	return strings.TrimRight(string(out), "\n"), err
}
