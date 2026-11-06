// Package guard implements the guard PreToolUse hook: it blocks or
// questions git mutations on protected branches, pushes generally, and
// the release claims that are the developer's (a breaking marker, a
// --bump below what the commits infer), according to .pk.json policy.
// The command line is read once, by invocations, and every policy is
// given that reading. An unconfigured repository is a no-op, because
// the plugin's hooks fire in every repository.
package guard

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/markwharton/plankit/internal/changelog"
	"github.com/markwharton/plankit/internal/cli"
	"github.com/markwharton/plankit/internal/config"
	"github.com/markwharton/plankit/internal/git"
	"github.com/markwharton/plankit/internal/hookio"
	"github.com/markwharton/plankit/internal/msg"
	"github.com/markwharton/plankit/internal/version"
)

// Cmd is the guard hook command.
var Cmd = &cli.Command{
	Name:    "guard",
	Summary: "Hook: block or question git mutations on protected branches, pushes, breaking-change markers, and lowered version bumps",
	Hook:    true,
	Run:     run,
}

// run enforces four policies: a branch policy (git mutations on
// protected branches), a push policy (any git push, regardless of
// branch), a breaking policy (a commit message carrying a breaking
// marker), and a bump policy (a release command whose --bump is lower
// than the commits infer). The strongest applicable decision wins
// (deny > ask), so a push to a protected branch is never downgraded.
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
	calls := invocations(input.ToolInput.Command)
	mutation := isGitMutation(calls)
	level, excludes, override := bumpOverride(calls)
	if !mutation && !override {
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
		return cli.Silent(hookio.ExitReport) // shown, not blocking; guard takes no action
	}

	// Branch policy: a mutation while a protected branch is checked out.
	branchDeny, branchAsk := false, false
	var protectedBranch string
	if mode := cfg.Guard.ResolvedMode(); mutation && mode != "off" && len(cfg.Guard.Branches) > 0 {
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
	if cfg.Guard.ResolvedBreaking() == "ask" && hasBreakingCommit(calls) {
		breakingAsk = true
	}

	// Push policy: any git push, regardless of branch.
	pushDeny, pushAsk := false, false
	if isGitPush(calls) {
		switch cfg.Guard.ResolvedPush() {
		case "block":
			pushDeny = true
		case "ask":
			pushAsk = true
		}
	}

	// Bump policy: pk ship or pk changelog with a --bump lower than the
	// commits infer, measured the way changelog will measure it. The
	// commits decide the version; an override is the developer's call.
	// A rehearsal (--dry-run) writes nothing and passes; a value changelog
	// will reject passes to changelog's own usage error.
	bumpAsk, bumpReason := false, ""
	if override && cfg.Guard.ResolvedBump() == "ask" {
		if requested, ok := version.ParseBump(level); ok {
			inferred, err := changelog.InferBump(root, excludes)
			switch {
			case errors.Is(err, changelog.ErrNoCommits):
			case err != nil:
				msg.Hookf(ctx.Stderr, "guard", "cannot infer the bump: %v", err)
			case requested < inferred.Bump:
				bumpAsk, bumpReason = true, bumpAskReason(level, inferred)
			}
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
	case bumpAsk:
		writeDecision(ctx, hookio.PermissionAsk, bumpReason)
	}
	return nil
}

// bumpAskReason names the gap and the commits that call for the higher
// bump, so the person deciding sees what the override would hide.
func bumpAskReason(requested string, inferred changelog.Inference) string {
	var b strings.Builder
	name := version.BumpName(inferred.Bump)
	fmt.Fprintf(&b, "pk guard: --bump %s is lower than the %s the commits infer. The commits decide the version; --bump is the developer's override, never a way to match a number named earlier. Commits that call for %s:\n", requested, name, name)
	for _, c := range inferred.Drivers {
		fmt.Fprintf(&b, "  %s %s\n", c.Hash, c.Subject())
	}
	b.WriteString("Confirm the override, or drop --bump. A commit with the wrong type is fixed by amending its type, not by overriding the bump.")
	return b.String()
}

func writeDecision(ctx *cli.Context, decision, reason string) {
	if err := hookio.WritePermissionDecision(ctx.Stdout, decision, reason); err != nil {
		msg.Hookf(ctx.Stderr, "guard", "write error: %v", err)
	}
}

// gitMutations are the git subcommands that change the checked-out
// branch: each puts commits on it, moves it, or publishes it. The
// guard page and docs/architecture.md list them, and a test holds both
// to this list.
var gitMutations = []string{"am", "cherry-pick", "commit", "merge", "pull", "push", "rebase", "reset", "revert"}

// isGitMutation reports whether any command is a git operation that
// mutates the branch.
func isGitMutation(calls []invocation) bool {
	for _, call := range calls {
		sub, _, ok := gitCall(call)
		if !ok {
			continue
		}
		for _, m := range gitMutations {
			if sub == m {
				return true
			}
		}
	}
	return false
}

// bumpOverride finds a pk ship or pk changelog invocation carrying
// --bump and returns the level asked for with the --exclude hashes
// beside it, which change what the commits infer. A --dry-run
// invocation is not reported: a rehearsal writes nothing. pk is matched
// by path basename, so the plugin's shim form is recognized too.
func bumpOverride(calls []invocation) (level string, excludes []string, ok bool) {
	for _, call := range calls {
		if call.program != "pk" || len(call.args) == 0 {
			continue
		}
		if sub := call.args[0]; sub != "ship" && sub != "changelog" {
			continue
		}
		level, excludes, dryRun := "", []string(nil), false
		words := call.args[1:]
		for j := 0; j < len(words); j++ {
			w := words[j]
			switch {
			case w == "--bump":
				if j+1 < len(words) {
					level = words[j+1]
					j++
				}
			case strings.HasPrefix(w, "--bump="):
				level = strings.TrimPrefix(w, "--bump=")
			case w == "--exclude":
				if j+1 < len(words) {
					excludes = strings.Split(words[j+1], ",")
					j++
				}
			case strings.HasPrefix(w, "--exclude="):
				excludes = strings.Split(strings.TrimPrefix(w, "--exclude="), ",")
			case w == "--dry-run":
				dryRun = true
			}
		}
		if level != "" && !dryRun {
			return level, excludes, true
		}
	}
	return "", nil, false
}

// isGitPush reports whether any command is a git push.
func isGitPush(calls []invocation) bool {
	for _, call := range calls {
		if sub, _, ok := gitCall(call); ok && sub == "push" {
			return true
		}
	}
	return false
}

// splitShellCommands splits a command string into its simple commands,
// on the shell's operators (&&, ||, |&, ;, |, a background &, and
// newlines) and on parentheses, so a subshell and a command
// substitution are read as the commands inside them. Single and double
// quotes are respected: an operator inside a quoted string is not a
// delimiter. An & that belongs to a redirection (2>&1, &>) is not one
// either.
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
			} else if c == ';' || c == '|' || c == '\n' || c == '(' || c == ')' {
				flush()
			} else if c == '&' && !isRedirection(command, i) {
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

// isRedirection reports whether the & at command[i] belongs to a
// redirection: it follows > or < (2>&1, <&3) or leads one (&>, &>>).
func isRedirection(command string, i int) bool {
	if i > 0 && (command[i-1] == '>' || command[i-1] == '<') {
		return true
	}
	return i+1 < len(command) && command[i+1] == '>'
}

// invocation is one simple command as the shell would run it: the
// program's base name and the words after it.
type invocation struct {
	program string
	args    []string
}

// invocations reads a (possibly compound) command line into the simple
// commands it holds. It is the one reading of the command line: every
// policy is given its result, so they cannot disagree about what a
// command runs. The words before the program are dropped: forms like
// "GIT_DIR=. git push", "command git push", and "then git push" all
// run git.
//
// A command run through sudo, sh -c, or eval is not read: guard is a
// guardrail against an agent following its defaults, and those are not
// its defaults.
func invocations(command string) []invocation {
	var calls []invocation
	for _, cmd := range splitShellCommands(command) {
		words := shellWords(cmd)
		i := 0
		for i < len(words) && precedesProgram(words[i]) {
			i++
		}
		if i < len(words) {
			calls = append(calls, invocation{program: baseName(words[i]), args: words[i+1:]})
		}
	}
	return calls
}

// precedesProgram reports a word the shell allows before the program a
// simple command runs: an environment assignment, "command", or one of
// the shell's own words that open or continue a compound command.
func precedesProgram(word string) bool {
	switch word {
	case "!", "command", "do", "elif", "else", "if", "then", "until", "while", "{":
		return true
	}
	return isEnvAssignment(word)
}

// gitValueOptions are git's global options that take their value as
// the next word; every other global option is self-contained.
var gitValueOptions = map[string]bool{
	"-C":           true,
	"-c":           true,
	"--config-env": true,
	"--git-dir":    true,
	"--namespace":  true,
	"--work-tree":  true,
}

// gitCall returns the git subcommand an invocation runs and the words
// after it, skipping git's global options, so "git -C dir push" and
// "git -c k=v commit" are read as push and commit. The bool is false
// when the invocation is not git or names no subcommand.
func gitCall(call invocation) (string, []string, bool) {
	if call.program != "git" {
		return "", nil, false
	}
	for i := 0; i < len(call.args); i++ {
		word := call.args[i]
		if !strings.HasPrefix(word, "-") {
			return word, call.args[i+1:], true
		}
		if gitValueOptions[word] {
			i++ // skip its value
		}
	}
	return "", nil, false
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
func hasBreakingCommit(calls []invocation) bool {
	for _, call := range calls {
		sub, args, ok := gitCall(call)
		if !ok || sub != "commit" {
			continue
		}
		msgs := commitMessages(args)
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

// commitMessages extracts the -m/--message values from the words after
// git commit, in order (the first is the subject). Handled forms: -m
// msg, -mmsg, --message msg, --message=msg, and short clusters ending
// in m (-am msg), where git gives the trailing option the next argument.
func commitMessages(args []string) []string {
	var msgs []string
	for j := 0; j < len(args); j++ {
		w := args[j]
		switch {
		case w == "-m" || w == "--message":
			if j+1 < len(args) {
				msgs = append(msgs, args[j+1])
				j++
			}
		case strings.HasPrefix(w, "--message="):
			msgs = append(msgs, strings.TrimPrefix(w, "--message="))
		case strings.HasPrefix(w, "-m") && len(w) > 2 && w[2] != '-':
			msgs = append(msgs, w[2:])
		case isShortCluster(w) && strings.HasSuffix(w, "m"):
			if j+1 < len(args) {
				msgs = append(msgs, args[j+1])
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
// deliberately small: enough to read a program and a -m argument the
// way the shell hands them over. One departure: outside quotes, a
// backslash before a letter or a digit is kept, because the hook reads
// PowerShell commands too and there it is a path separator
// (C:\tools\git.exe), which nobody writes as an escape.
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
			if isAlnum(cmd[i+1]) {
				b.WriteByte(c)
			}
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

func isAlnum(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}
