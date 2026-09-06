// Package guard implements the guard PreToolUse hook: it blocks or
// questions git mutations on protected branches, and pushes generally,
// according to .pk.json policy. The command-recognition logic is ported
// from v1 unchanged; the v2 addition is the not-configured short-circuit,
// because the plugin's hooks fire in every repository.
package guard

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/markwharton/plankit/internal/cli"
	"github.com/markwharton/plankit/internal/config"
	"github.com/markwharton/plankit/internal/git"
	"github.com/markwharton/plankit/internal/hookio"
	"github.com/markwharton/plankit/internal/msg"
)

// Cmd is the guard hook command.
var Cmd = &cli.Command{
	Name:    "guard",
	Summary: "Hook: block or question git mutations on protected branches, pushes, and breaking-change markers",
	Hook:    true,
	Run:     run,
}

// run enforces two policies on git mutations: a branch policy (mutations
// on protected branches) and a push policy (any git push, regardless of
// branch). The strongest applicable decision wins (deny > ask), so a
// push to a protected branch is never downgraded.
//
// Internal errors (unreadable input, malformed .pk.json, git failures)
// fail open: guard logs to stderr and emits no decision, always exit 0.
// Stdin comes from the trusted harness, and failing closed would block
// every Bash call while, say, .pk.json is mid-edit. Guard is a guardrail
// against an agent following its defaults, not a security boundary.
func run(ctx *cli.Context) error {
	input, err := hookio.ReadInput(ctx.Stdin)
	if err != nil {
		msg.Hookf(ctx.Stderr, "guard", "failed to read input: %v", err)
		return nil
	}
	if input.ToolInput == nil || input.ToolInput.Command == "" {
		return nil
	}
	command := input.ToolInput.Command
	if !isGitMutation(command) {
		return nil
	}

	dir := hookio.ResolveDir(os.Getenv, input.CWD, ctx.ProjectDir, ctx.ProjectDirExplicit)
	root, ok := git.FindRoot(dir)
	if !ok {
		return nil
	}
	cfg, err := config.Load(root)
	if errors.Is(err, config.ErrNotConfigured) {
		return nil // plankit is off here; the hook fires everywhere
	}
	if err != nil {
		msg.Hookf(ctx.Stderr, "guard", "%v", err)
		return nil
	}

	// Branch policy: a mutation while a protected branch is checked out.
	branchDeny, branchAsk := false, false
	var protectedBranch string
	if mode := cfg.Guard.ResolvedMode(); mode != "off" && len(cfg.Guard.Branches) > 0 {
		branch, err := git.CurrentBranch(root)
		if err != nil {
			return nil
		}
		for _, protected := range cfg.Guard.Branches {
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

	// Breaking policy: a commit message carrying ! or BREAKING CHANGE,
	// on any branch. The marker is a user-approved claim, so the agent
	// writing one gets an ask.
	breakingAsk := false
	if cfg.Guard.ResolvedBreaking() == "ask" && hasBreakingCommit(command) {
		breakingAsk = true
	}

	// Push policy: any git push, regardless of branch.
	pushDeny, pushAsk := false, false
	if isGitPush(command) {
		switch cfg.Guard.ResolvedPush() {
		case "block":
			pushDeny = true
		case "ask":
			pushAsk = true
		}
	}

	const pushDenyReason = "pk guard: push blocked. Pushing is the developer's explicit action; the commit is local. Push it yourself, or use pk preserve / pk release, when ready."
	const pushAskReason = "pk guard: the agent is about to git push. Pushing is the developer's call. Allow this push?"
	switch {
	case pushDeny:
		writeDecision(ctx, hookio.PermissionDeny, pushDenyReason)
	case branchDeny:
		writeDecision(ctx, hookio.PermissionDeny, fmt.Sprintf("Branch %q is protected by pk guard. Switch to your development branch and use pk release from there.", protectedBranch))
	case pushAsk:
		writeDecision(ctx, hookio.PermissionAsk, pushAskReason)
	case branchAsk:
		writeDecision(ctx, hookio.PermissionAsk, fmt.Sprintf("Branch %q is protected by pk guard. Switch to your development branch and use pk release from there. Only proceed here for emergency hotfix or manual recovery.", protectedBranch))
	case breakingAsk:
		writeDecision(ctx, hookio.PermissionAsk, "pk guard: the commit message marks a breaking change (! or BREAKING CHANGE). That marker drives the next major version, and it is the developer's claim to make, not the agent's. Confirm the change is breaking, or reword the message without the marker.")
	}
	return nil
}

func writeDecision(ctx *cli.Context, decision, reason string) {
	if err := hookio.WritePermissionDecision(ctx.Stdout, decision, reason); err != nil {
		msg.Hookf(ctx.Stderr, "guard", "write error: %v", err)
	}
}

// isGitMutation reports whether any subcommand in a (possibly compound)
// command is a git operation that mutates the branch.
func isGitMutation(command string) bool {
	for _, sub := range splitShellCommands(command) {
		switch gitSubcommand(sub) {
		case "commit", "merge", "push", "rebase", "reset":
			return true
		}
	}
	return false
}

// isGitPush reports whether any subcommand is a git push.
func isGitPush(command string) bool {
	for _, sub := range splitShellCommands(command) {
		if gitSubcommand(sub) == "push" {
			return true
		}
	}
	return false
}

// splitShellCommands splits a command string on shell operators (&&, ||,
// |&, ;, |, and newlines), respecting single and double quotes so that
// operators inside quoted strings are not treated as delimiters.
func splitShellCommands(command string) []string {
	var result []string
	var current strings.Builder
	inSingle, inDouble := false, false

	flush := func() {
		if trimmed := strings.TrimSpace(current.String()); trimmed != "" {
			result = append(result, trimmed)
		}
		current.Reset()
	}
	for i := 0; i < len(command); i++ {
		c := command[i]
		switch {
		case c == '\'' && !inDouble:
			inSingle = !inSingle
			current.WriteByte(c)
		case c == '"' && !inSingle:
			inDouble = !inDouble
			current.WriteByte(c)
		case !inSingle && !inDouble:
			if i+1 < len(command) && (command[i:i+2] == "&&" || command[i:i+2] == "||" || command[i:i+2] == "|&") {
				flush()
				i++
			} else if c == ';' || c == '|' || c == '\n' {
				flush()
			} else {
				current.WriteByte(c)
			}
		default:
			current.WriteByte(c)
		}
	}
	flush()
	return result
}

// gitSubcommand returns the git subcommand for a single command, skipping
// leading VAR=value environment assignments and a leading "command" word,
// matching git by path basename, and skipping git's global options, so
// forms like "GIT_DIR=. git push", "command git push", "/usr/bin/git
// push", "git -C dir push", and "git -c k=v commit" are recognized.
// Returns "" when the command is not a git invocation. -C and -c take a
// separate-word value; other global options are self-contained.
func gitSubcommand(cmd string) string {
	fields := strings.Fields(cmd)
	start := 0
	for start < len(fields) && (isEnvAssignment(fields[start]) || fields[start] == "command") {
		start++
	}
	if start >= len(fields) || baseName(fields[start]) != "git" {
		return ""
	}
	for i := start + 1; i < len(fields); i++ {
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

// baseName is filepath.Base for both separators, with a case-insensitive
// .exe suffix stripped: hook commands may carry Windows paths and
// "git.exe" regardless of the platform pk runs on. v1 used
// filepath.Base, which let both slip past guard on the PowerShell side.
func baseName(word string) string {
	if i := strings.LastIndexAny(word, `/\`); i >= 0 {
		word = word[i+1:]
	}
	if len(word) > 4 && strings.EqualFold(word[len(word)-4:], ".exe") {
		word = word[:len(word)-4]
	}
	return word
}

// isEnvAssignment reports whether a word is a VAR=value prefix: a shell
// identifier (letters, digits, underscore, not starting with a digit)
// followed by "=".
func isEnvAssignment(word string) bool {
	eq := strings.IndexByte(word, '=')
	if eq <= 0 {
		return false
	}
	for i := 0; i < eq; i++ {
		c := word[i]
		switch {
		case c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z':
		case c >= '0' && c <= '9':
			if i == 0 {
				return false
			}
		default:
			return false
		}
	}
	return true
}

// hasBreakingCommit reports whether any git commit in the command line
// carries a breaking-change marker in a -m/--message argument: ! before
// the colon in the subject, or a BREAKING CHANGE / BREAKING-CHANGE line
// in any message. Only inline messages are inspectable; -F files and
// editor commits pass through, as does anything hidden behind command
// substitution. That covers the authored path this check exists for:
// agent commits are -m commits.
func hasBreakingCommit(command string) bool {
	for _, cmd := range splitShellCommands(command) {
		if gitSubcommand(cmd) != "commit" {
			continue
		}
		msgs := commitMessages(shellWords(cmd))
		if len(msgs) == 0 {
			continue
		}
		if breakingSubjectRe.MatchString(msgs[0]) {
			return true
		}
		for _, m := range msgs {
			if hasBreakingLine(m) {
				return true
			}
		}
	}
	return false
}

// breakingSubjectRe matches a conventional subject whose type carries
// the breaking !: type(scope)!: message.
var breakingSubjectRe = regexp.MustCompile(`^\s*\w+(?:\([^)]*\))?!\s*:`)

func hasBreakingLine(msg string) bool {
	for _, line := range strings.Split(msg, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "BREAKING CHANGE:") || strings.HasPrefix(t, "BREAKING-CHANGE:") {
			return true
		}
	}
	return false
}

// commitMessages extracts the -m/--message values from a tokenized git
// commit invocation, in order (the first is the subject). Handled
// forms: -m msg, -mmsg, --message msg, --message=msg, and short
// clusters ending in m (-am msg), where git gives the trailing option
// the next argument.
func commitMessages(words []string) []string {
	// Find the git word, skipping env assignments and "command".
	i := 0
	for i < len(words) && (isEnvAssignment(words[i]) || words[i] == "command") {
		i++
	}
	if i >= len(words) || baseName(words[i]) != "git" {
		return nil
	}
	// Advance past global options to the commit subcommand.
	i++
	for i < len(words) && words[i] != "commit" {
		if words[i] == "-C" || words[i] == "-c" {
			i++
		}
		i++
	}
	if i >= len(words) {
		return nil
	}
	var msgs []string
	for j := i + 1; j < len(words); j++ {
		w := words[j]
		switch {
		case w == "-m" || w == "--message":
			if j+1 < len(words) {
				msgs = append(msgs, words[j+1])
				j++
			}
		case strings.HasPrefix(w, "--message="):
			msgs = append(msgs, strings.TrimPrefix(w, "--message="))
		case strings.HasPrefix(w, "-m") && len(w) > 2 && w[2] != '-':
			msgs = append(msgs, w[2:])
		case isShortCluster(w) && strings.HasSuffix(w, "m"):
			if j+1 < len(words) {
				msgs = append(msgs, words[j+1])
				j++
			}
		}
	}
	return msgs
}

// isShortCluster reports a bundled short-option word like -am or -sm:
// a dash followed only by letters.
func isShortCluster(w string) bool {
	if len(w) < 3 || w[0] != '-' || w[1] == '-' {
		return false
	}
	for _, c := range w[1:] {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z') {
			return false
		}
	}
	return true
}

// shellWords splits one shell command into words, honoring single and
// double quotes and backslash escapes, and stripping the quotes. It is
// deliberately small: enough to read a -m argument the way the shell
// hands it to git.
func shellWords(cmd string) []string {
	var words []string
	var b strings.Builder
	inWord := false
	quote := byte(0)
	flush := func() {
		if inWord {
			words = append(words, b.String())
			b.Reset()
			inWord = false
		}
	}
	for i := 0; i < len(cmd); i++ {
		c := cmd[i]
		switch {
		case quote == '\'':
			if c == '\'' {
				quote = 0
			} else {
				b.WriteByte(c)
			}
		case quote == '"':
			if c == '"' {
				quote = 0
			} else if c == '\\' && i+1 < len(cmd) && (cmd[i+1] == '"' || cmd[i+1] == '\\' || cmd[i+1] == '$' || cmd[i+1] == '`') {
				b.WriteByte(cmd[i+1])
				i++
			} else {
				b.WriteByte(c)
			}
		case c == '\'' || c == '"':
			quote = c
			inWord = true
		case c == '\\' && i+1 < len(cmd):
			b.WriteByte(cmd[i+1])
			i++
			inWord = true
		case c == ' ' || c == '\t' || c == '\n':
			flush()
		default:
			b.WriteByte(c)
			inWord = true
		}
	}
	flush()
	return words
}
