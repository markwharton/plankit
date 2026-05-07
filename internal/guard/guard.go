// Package guard implements the guard PreToolUse hook command.
// It blocks git mutations (commit, push, merge) on protected branches.
package guard

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/markwharton/plankit/internal/config"
	pkgit "github.com/markwharton/plankit/internal/git"
	"github.com/markwharton/plankit/internal/hooks"
)

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
	projectDir := hooks.ResolveProjectDir(cfg.Env, input.CWD)
	if projectDir == "" {
		return 0
	}

	// Load guard config.
	guardCfg, err := loadGuardConfig(cfg.ReadFile, projectDir)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "pk guard: %v\n", err)
		return 0
	}
	if len(guardCfg.Branches) == 0 {
		return 0
	}

	// Get current branch.
	branch, err := cfg.GitExec(projectDir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return 0
	}
	branch = strings.TrimSpace(branch)

	// Check if current branch is protected.
	for _, protected := range guardCfg.Branches {
		if branch == protected {
			if cfg.Ask {
				reason := fmt.Sprintf("Branch %q is protected by pk guard. Switch to your development branch and use pk release from there. Only proceed here for emergency hotfix or manual recovery.", branch)
				if err := hooks.WritePermissionDecision(cfg.Stdout, hooks.PermissionAsk, reason); err != nil {
					fmt.Fprintf(cfg.Stderr, "pk guard: write error: %v\n", err)
				}
			} else {
				reason := fmt.Sprintf("Branch %q is protected by pk guard. Switch to your development branch and use pk release from there.", branch)
				if err := hooks.WritePermissionDecision(cfg.Stdout, hooks.PermissionDeny, reason); err != nil {
					fmt.Fprintf(cfg.Stderr, "pk guard: write error: %v\n", err)
				}
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

// splitShellCommands splits a command string on shell operators (&&, ||, ;),
// respecting single and double quotes so that operators inside quoted strings
// are not treated as delimiters.
func splitShellCommands(command string) []string {
	var result []string
	var current strings.Builder
	inSingle, inDouble := false, false

	for i := 0; i < len(command); i++ {
		c := command[i]
		if c == '\'' && !inDouble {
			inSingle = !inSingle
			current.WriteByte(c)
		} else if c == '"' && !inSingle {
			inDouble = !inDouble
			current.WriteByte(c)
		} else if !inSingle && !inDouble {
			if i+1 < len(command) && (command[i:i+2] == "&&" || command[i:i+2] == "||") {
				if trimmed := strings.TrimSpace(current.String()); trimmed != "" {
					result = append(result, trimmed)
				}
				current.Reset()
				i++
			} else if c == ';' {
				if trimmed := strings.TrimSpace(current.String()); trimmed != "" {
					result = append(result, trimmed)
				}
				current.Reset()
			} else {
				current.WriteByte(c)
			}
		} else {
			current.WriteByte(c)
		}
	}
	if trimmed := strings.TrimSpace(current.String()); trimmed != "" {
		result = append(result, trimmed)
	}
	return result
}

// isGitMutationSingle checks if a single command is a git mutation.
func isGitMutationSingle(cmd string) bool {
	mutations := []string{
		"git commit",
		"git merge",
		"git push",
		"git rebase",
		"git reset",
	}
	for _, m := range mutations {
		if strings.HasPrefix(cmd, m+" ") || cmd == m {
			return true
		}
	}
	return false
}

// loadGuardConfig reads .pk.json from the project directory and returns the guard config.
// Returns an error if the file exists but contains malformed JSON.
func loadGuardConfig(readFile func(string) ([]byte, error), projectDir string) (config.GuardConfig, error) {
	pk, err := config.Load(readFile, filepath.Join(projectDir, ".pk.json"))
	if err != nil {
		return config.GuardConfig{}, err
	}
	return pk.Guard, nil
}
