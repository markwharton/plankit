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
	"github.com/markwharton/plankit/internal/msg"
	"github.com/markwharton/plankit/internal/paths"
)

// Config holds the dependencies for the guard command.
type Config struct {
	Stdin    io.Reader
	Stdout   io.Writer
	Stderr   io.Writer
	Env      func(string) string
	ReadFile func(string) ([]byte, error)
	GitExec  func(projectDir string, args ...string) (string, error)

	// Ask and PushGuard are DEPRECATED flag overrides. Modes now live in
	// .pk.json (guard.mode, guard.push); these are honored only when an old
	// hook still passes --ask / --push-guard, so existing installs keep working
	// until re-setup normalizes their hooks. Removed in a later release.
	//
	// Ask, when true, forces ask mode (overrides guard.mode).
	Ask bool
	// PushGuard, when non-empty, forces the push policy (overrides guard.push).
	PushGuard string
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

// Run reads a PreToolUse hook payload from stdin and enforces two policies on git
// mutations: a branch policy (block/ask mutations on protected branches) and a push
// policy (block/ask any `git push` regardless of branch). The strongest applicable
// decision wins (deny > ask > allow), so a push to a protected branch is never
// downgraded. Returns the process exit code (always 0 for hooks).
func Run(cfg Config) int {
	input, err := hooks.ReadInput(cfg.Stdin)
	if err != nil {
		msg.Hookf(cfg.Stderr, "guard", "failed to read input: %v", err)
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
		msg.Hookf(cfg.Stderr, "guard", "%v", err)
		return 0
	}

	// Resolve modes from .pk.json, honoring the deprecated flag overrides.
	mode := guardCfg.ResolvedMode()
	if cfg.Ask {
		mode = "ask"
	}
	push := guardCfg.ResolvedPush()
	if cfg.PushGuard != "" {
		push = cfg.PushGuard
	}

	// Branch policy: a mutation on a protected branch (skipped when mode is off).
	branchDeny, branchAsk := false, false
	var protectedBranch string
	if mode != "off" && len(guardCfg.Branches) > 0 {
		branch, err := cfg.GitExec(projectDir, "rev-parse", "--abbrev-ref", "HEAD")
		if err != nil {
			return 0
		}
		branch = strings.TrimSpace(branch)
		for _, protected := range guardCfg.Branches {
			if branch == protected {
				protectedBranch = branch
				if mode == "ask" {
					branchAsk = true
				} else {
					branchDeny = true
				}
				break
			}
		}
	}

	// Push policy: any `git push`, regardless of branch.
	pushDeny, pushAsk := false, false
	if isGitPush(command) {
		switch push {
		case "block":
			pushDeny = true
		case "ask":
			pushAsk = true
		}
	}

	// Strongest decision wins (deny > ask). Reason comes from whichever drove it.
	const pushDenyReason = "pk guard: push blocked. Pushing is the developer's explicit action; the commit is local. Push it yourself, or use pk preserve / pk release, when ready."
	const pushAskReason = "pk guard: the agent is about to git push. Pushing is the developer's call. Allow this push?"
	switch {
	case pushDeny:
		writeDecision(cfg, hooks.PermissionDeny, pushDenyReason)
	case branchDeny:
		writeDecision(cfg, hooks.PermissionDeny, fmt.Sprintf("Branch %q is protected by pk guard. Switch to your development branch and use pk release from there.", protectedBranch))
	case pushAsk:
		writeDecision(cfg, hooks.PermissionAsk, pushAskReason)
	case branchAsk:
		writeDecision(cfg, hooks.PermissionAsk, fmt.Sprintf("Branch %q is protected by pk guard. Switch to your development branch and use pk release from there. Only proceed here for emergency hotfix or manual recovery.", protectedBranch))
	}

	return 0
}

// writeDecision emits a PreToolUse permission decision, logging write errors.
func writeDecision(cfg Config, decision, reason string) {
	if err := hooks.WritePermissionDecision(cfg.Stdout, decision, reason); err != nil {
		msg.Hookf(cfg.Stderr, "guard", "write error: %v", err)
	}
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
	switch gitSubcommand(cmd) {
	case "commit", "merge", "push", "rebase", "reset":
		return true
	}
	return false
}

// isGitPush reports whether any subcommand in a (possibly compound) command is a git push.
func isGitPush(command string) bool {
	for _, sub := range splitShellCommands(command) {
		if gitSubcommand(sub) == "push" {
			return true
		}
	}
	return false
}

// gitSubcommand returns the git subcommand for a single command, skipping git's
// global options so forms like "git -C dir push" or "git -c k=v push" are recognized.
// Returns "" if the command is not a git invocation. -C and -c take a separate-word
// value; other global options (--git-dir=..., --work-tree=..., etc.) are self-contained.
func gitSubcommand(cmd string) string {
	fields := strings.Fields(cmd)
	if len(fields) == 0 || fields[0] != "git" {
		return ""
	}
	for i := 1; i < len(fields); i++ {
		f := fields[i]
		if !strings.HasPrefix(f, "-") {
			return f
		}
		if f == "-C" || f == "-c" {
			i++ // skip its value
		}
	}
	return ""
}

// loadGuardConfig reads .pk.json from the project directory and returns the guard config.
// Returns an error if the file exists but contains malformed JSON.
func loadGuardConfig(readFile func(string) ([]byte, error), projectDir string) (config.GuardConfig, error) {
	pk, err := config.Load(readFile, filepath.Join(projectDir, paths.PkConfig))
	if err != nil {
		return config.GuardConfig{}, err
	}
	return pk.Guard, nil
}
