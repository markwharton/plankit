// Package guard implements the guard PreToolUse hook command.
// It blocks git mutations (commit, push, merge) on protected branches.
package guard

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	pkgit "github.com/markwharton/plankit/internal/git"
	"github.com/markwharton/plankit/internal/hooks"
)

// GuardConfig holds the guard section of .pk.json.
// The legacy protectedBranches key is still accepted and promoted into
// Branches after unmarshal, so existing configs keep working.
type GuardConfig struct {
	Branches          []string `json:"branches,omitempty"`
	ProtectedBranches []string `json:"protectedBranches,omitempty"`
}

// Normalize promotes the legacy protectedBranches value into Branches if
// Branches is empty. New key wins if both are present. Callers should
// invoke this after json.Unmarshal so the rest of the code only reads
// from Branches.
func (g *GuardConfig) Normalize() {
	if len(g.Branches) == 0 && len(g.ProtectedBranches) > 0 {
		g.Branches = g.ProtectedBranches
	}
	g.ProtectedBranches = nil
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

	// Ask prompts the user instead of blocking outright.
	Ask bool
}

// DefaultConfig returns a Config wired to real OS resources.
func DefaultConfig() Config {
	return Config{
		Stdin:    os.Stdin,
		Stdout:   os.Stdout,
		Stderr:   os.Stderr,
		Env:      os.Getenv,
		ReadFile: os.ReadFile,
		GitExec:  pkgit.Exec,
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
	config, err := loadGuardConfig(cfg.ReadFile, projectDir)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "pk guard: %v\n", err)
		return 0
	}
	if len(config.Branches) == 0 {
		return 0
	}

	// Get current branch.
	branch, err := cfg.GitExec(projectDir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return 0
	}
	branch = strings.TrimSpace(branch)

	// Check if current branch is protected.
	for _, protected := range config.Branches {
		if branch == protected {
			if cfg.Ask {
				reason := fmt.Sprintf("Branch %q is protected by pk guard. Switch to your development branch and use pk release from there. Only proceed here for emergency hotfix or manual recovery.", branch)
				hooks.WritePermissionDecision(cfg.Stdout, hooks.PermissionAsk, reason)
			} else {
				reason := fmt.Sprintf("Branch %q is protected by pk guard. Switch to your development branch and use pk release from there.", branch)
				hooks.WritePermissionDecision(cfg.Stdout, hooks.PermissionDeny, reason)
			}
			return 0
		}
	}

	return 0
}

// isGitMutation checks if any subcommand in a (possibly compound) command
// is a git operation that mutates the branch. Splits on &&, ||, and ;
// to handle chained commands like "git checkout main && git merge dev".
func isGitMutation(command string) bool {
	for _, sub := range splitShellCommands(command) {
		if isGitMutationSingle(sub) {
			return true
		}
	}
	return false
}

// splitShellCommands splits a command string on shell operators (&&, ||, ;).
func splitShellCommands(command string) []string {
	// Replace operators with a common delimiter, then split.
	s := command
	s = strings.ReplaceAll(s, "&&", "\x00")
	s = strings.ReplaceAll(s, "||", "\x00")
	s = strings.ReplaceAll(s, ";", "\x00")
	parts := strings.Split(s, "\x00")

	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// isGitMutationSingle checks if a single command is a git mutation.
func isGitMutationSingle(cmd string) bool {
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
// Returns an error if the file exists but contains malformed JSON.
func loadGuardConfig(readFile func(string) ([]byte, error), projectDir string) (GuardConfig, error) {
	data, err := readFile(projectDir + "/.pk.json")
	if err != nil {
		return GuardConfig{}, nil
	}
	var pk PkConfig
	if err := json.Unmarshal(data, &pk); err != nil {
		return GuardConfig{}, fmt.Errorf("failed to parse .pk.json: %w", err)
	}
	pk.Guard.Normalize()
	return pk.Guard, nil
}
