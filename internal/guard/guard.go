// Package guard implements the guard PreToolUse hook command.
// It blocks git mutations (commit, push, merge) on protected branches.
package guard

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/markwharton/plankit/internal/hooks"
)

// GuardConfig holds the guard section of .pk.json.
type GuardConfig struct {
	ProtectedBranches []string `json:"protectedBranches,omitempty"`
}

// PkConfig reads just the guard portion of .pk.json.
type PkConfig struct {
	Guard GuardConfig `json:"guard,omitempty"`
}

// Config holds the dependencies for the guard command.
type Config struct {
	Stdin    io.Reader
	Stdout   io.Writer
	Stderr   io.Writer
	Env      func(string) string
	ReadFile func(string) ([]byte, error)
	GitExec  func(projectDir string, args ...string) (string, error)
}

// DefaultConfig returns a Config wired to real OS resources.
func DefaultConfig() Config {
	return Config{
		Stdin:    os.Stdin,
		Stdout:   os.Stdout,
		Stderr:   os.Stderr,
		Env:      os.Getenv,
		ReadFile: os.ReadFile,
		GitExec:  defaultGitExec,
	}
}

// Run reads a PreToolUse hook payload from stdin and blocks git mutations
// on protected branches. Returns the process exit code (always 0 for hooks).
func Run(cfg Config) int {
	input, err := hooks.ReadInput(cfg.Stdin)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "pk guard: failed to read input: %v\n", err)
		return 0
	}

	if input.ToolInput == nil || input.ToolInput.Command == "" {
		return 0
	}

	command := input.ToolInput.Command

	// Only check git mutation commands.
	if !isGitMutation(command) {
		return 0
	}

	// Determine project directory.
	projectDir := cfg.Env("CLAUDE_PROJECT_DIR")
	if projectDir == "" {
		projectDir = input.CWD
	}
	if projectDir == "" {
		return 0
	}

	// Load guard config.
	config := loadGuardConfig(cfg.ReadFile, projectDir)
	if len(config.ProtectedBranches) == 0 {
		return 0
	}

	// Get current branch.
	branch, err := cfg.GitExec(projectDir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return 0
	}
	branch = strings.TrimSpace(branch)

	// Check if current branch is protected.
	for _, protected := range config.ProtectedBranches {
		if branch == protected {
			reason := fmt.Sprintf("Branch %q is protected. Switch to a development branch before committing.", branch)
			fmt.Fprintf(cfg.Stdout, `{"decision":"block","reason":%q}`, reason)
			return 0
		}
	}

	return 0
}

// isGitMutation checks if the command is a git operation that mutates the branch.
func isGitMutation(command string) bool {
	// Normalize: trim leading whitespace, handle chained commands.
	cmd := strings.TrimSpace(command)

	// Check for git commands that mutate the branch.
	mutations := []string{
		"git commit",
		"git push",
		"git merge",
		"git rebase",
	}
	for _, m := range mutations {
		if strings.HasPrefix(cmd, m+" ") || cmd == m {
			return true
		}
	}

	// Also block force push variants.
	if strings.Contains(cmd, "git push") && (strings.Contains(cmd, "--force") || strings.Contains(cmd, " -f")) {
		return true
	}

	return false
}

// loadGuardConfig reads .pk.json from the project directory and returns the guard config.
func loadGuardConfig(readFile func(string) ([]byte, error), projectDir string) GuardConfig {
	data, err := readFile(projectDir + "/.pk.json")
	if err != nil {
		return GuardConfig{}
	}
	var pk PkConfig
	if err := json.Unmarshal(data, &pk); err != nil {
		return GuardConfig{}
	}
	return pk.Guard
}

func defaultGitExec(projectDir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", projectDir}, args...)...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}
