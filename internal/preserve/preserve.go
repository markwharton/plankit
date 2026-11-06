// Package preserve implements the preserve hook: it captures approved
// Claude Code plans, byte for byte, as dated and sequenced files in
// the plans directory (.pk.json preserve.dir), and commits them.
//
// Three invocation shapes share this command. The automatic PostToolUse
// hook on ExitPlanMode arrives with a payload on stdin and honors
// .pk.json preserve.mode: auto commits immediately, manual records a
// pending-plan pointer and tells the person to run /plankit:preserve,
// off does nothing. An explicit invocation (the /plankit:preserve skill,
// or a person at a terminal) arrives with no payload, consumes the
// pointer, and always commits. A --plan invocation names the plan
// itself, for a plan revised after its pointer was consumed, and
// commits it under the same rules.
//
// Ported from v1. The v2 changes: an unconfigured repository is a fast
// no-op for the automatic hook (v1's absent-config default was manual,
// but the plugin's hooks fire in every repository), messages say
// /plankit:preserve (plugin-namespaced), and a CRLF plan no longer
// carries \r into its title.
package preserve

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/markwharton/plankit/internal/cli"
	"github.com/markwharton/plankit/internal/config"
	"github.com/markwharton/plankit/internal/git"
	"github.com/markwharton/plankit/internal/hookio"
	"github.com/markwharton/plankit/internal/msg"
)

// minPlanSize is the minimum byte length for a plan to be preserved.
// Real plans have a title, context, and some substance; anything below
// this is an empty template or an aborted draft.
const minPlanSize = 50

// pointerFilename names the per-repo pending-plan pointer written by
// manual mode and consumed by the explicit invocation. It lives under
// .git/ because that directory is always untracked, so no .gitignore
// coordination is needed and the file is naturally scoped to the repo.
// The pointer records the exact plan that was approved, so the explicit
// run picks it up even minutes later.
const pointerFilename = "pk-pending-plan"

// now is stubbed in tests to pin the date prefix.
var now = time.Now

// configDirEnv is Claude Code's own variable for a relocated configuration
// directory (a second account on one machine, for example). Plans live in
// plans/ under it; with it unset they live under ~/.claude.
const configDirEnv = "CLAUDE_CONFIG_DIR"

// Cmd is the preserve command: hook-driven and explicitly invocable.
var Cmd = &cli.Command{
	Name:    "preserve",
	Summary: "Hook: preserve the approved plan as a record and commit it",
	Hook:    true,
	Flags: []cli.FlagSpec{
		{Name: "discard", Type: cli.BoolFlag, Usage: "Remove the pending plan pointer without preserving the plan it names"},
		{Name: "dry-run", Type: cli.BoolFlag, Usage: "Preview without writing or committing"},
		{Name: "plan", Type: cli.StringFlag, Usage: "Plan file to preserve, for a plan revised after approval (a path under ~/.claude/plans, or plans/ under CLAUDE_CONFIG_DIR when that is set)"},
		{Name: "push", Type: cli.BoolFlag, Usage: "Push to origin after committing"},
	},
	Run: run,
}

func run(ctx *cli.Context) error {
	if ctx.Bool("discard") {
		return runDiscard(ctx)
	}
	if typed := ctx.String("plan"); typed != "" {
		return runNamed(ctx, typed)
	}

	var planPath string

	// A payload means the automatic ExitPlanMode hook, which honors
	// preserve.mode. No payload means an explicit invocation, which
	// consumes the pointer and always commits. A payload that does not
	// parse is neither, and nothing is preserved on its word.
	input, hookInvocation, err := hookio.ReadInput(ctx.Stdin)
	if err != nil {
		return hookio.Report(ctx.Stderr, "preserve", err) // shown, not blocking; nothing preserved
	}

	dir := hookio.ResolveDir(os.Getenv, input.CWD, ctx.ProjectDir, ctx.ProjectDirExplicit)

	// Both shapes need the repository and its policy file, for the mode
	// and the plans directory. A person who typed the command is told
	// what is missing; the hook declines without either.
	root, rootErr := cli.Root(dir)
	var cfg *config.PkConfig
	switch {
	case !hookInvocation:
		if rootErr != nil {
			return rootErr
		}
		if cfg, err = cli.Policy(root); err != nil {
			return err
		}
	case rootErr == nil:
		if cfg, err = config.Load(root); err != nil {
			if errors.Is(err, config.ErrNotConfigured) {
				return nil // plankit is off here; the hook fires everywhere
			}
			return hookio.Report(ctx.Stderr, "preserve", err) // shown, not blocking; nothing preserved
		}
	}

	if hookInvocation {
		planPath = extractPlanPath(input.ToolResponse, sessionPlansDir(), os.UserHomeDir)
		if planPath != "" {
			if _, err := os.Stat(planPath); err != nil {
				planPath = "" // the payload named a plan that is gone
			}
		}
	} else if p, ok := readPointer(root); ok {
		planPath = p
	}

	if planPath == "" {
		// The hook says nothing when it takes no action: its stdout is
		// the PostToolUse envelope, and silence is how it declines. A
		// person who typed the command hears the outcome instead, the
		// way the other commands narrate one.
		if !hookInvocation {
			fmt.Fprintln(ctx.Stderr, "No pending plan to preserve.")
		}
		return nil
	}

	content, err := os.ReadFile(planPath)
	if err != nil {
		return outcome(ctx, hookInvocation, cli.WithHint(unreadable(planPath, err), "%s", discardHint))
	}
	if len(content) < minPlanSize {
		// The hook declines a draft in silence. A person who typed the
		// command is told which plan it was and how to drop it.
		if hookInvocation {
			return nil
		}
		return cli.WithHint(tooSmall(planPath, len(content)), "%s", discardHint)
	}

	if rootErr != nil {
		msg.Hookf(ctx.Stderr, "preserve", "%v", rootErr)
		return nil
	}

	// The automatic hook honors policy; the explicit invocation is the
	// person acting, which is consent enough.
	mode := "auto"
	if hookInvocation {
		mode = cfg.Preserve.ResolvedMode()
	}

	title := extractTitle(string(content))
	switch mode {
	case "off":
		return nil
	case "manual":
		writePointer(ctx, root, planPath)
		return outcome(ctx, hookInvocation, writeResponse(ctx,
			fmt.Sprintf("Plan '%s' ready. Type /plankit:preserve to save it.", title),
			fmt.Sprintf("The user's plan '%s' has been approved. Inform the user that they can type /plankit:preserve to save it to %s/.", title, cfg.Preserve.ResolvedDir())))
	}

	// auto, or explicit invocation: commit.
	return outcome(ctx, hookInvocation, preserve(ctx, root, cfg.Preserve, content, title))
}

// discardHint names the way out for a pending plan that cannot be
// preserved.
const discardHint = "to drop the pending plan: pk preserve --discard"

// unreadable is the refusal for a plan file that could not be read.
func unreadable(planPath string, err error) error {
	return cli.Statef("failed to read plan %s: %v", planPath, err)
}

// tooSmall is the refusal for a plan under minPlanSize.
func tooSmall(planPath string, size int) error {
	return cli.Statef("plan %s is %d bytes, below the minimum of %d", planPath, size, minPlanSize)
}

// outcome turns a failed preserve into the exit its caller is owed. A
// typed run gets the error back, and the frame prints it with its exit
// code. The automatic hook reports it and exits with hookio.ExitReport:
// Claude Code shows the message and continues, where exit 0 would hide
// it and exit 2 would block.
func outcome(ctx *cli.Context, hook bool, err error) error {
	if err == nil || !hook {
		return err
	}
	return hookio.Report(ctx.Stderr, "preserve", err)
}

// runNamed is the --plan path: a person names the plan, so stdin is not
// read, the pointer is not consulted except to refuse a conflict, and
// every reason not to act is an error naming it. A pointer naming the
// same file is consumed as an explicit run consumes it.
func runNamed(ctx *cli.Context, typed string) error {
	plans := sessionPlansDir()
	planPath := matchPlanPath(typed, plans, os.UserHomeDir)
	if planPath == "" {
		where := plans + "/"
		if plans == "" {
			where = "~/.claude/plans/"
		}
		return cli.Usagef("--plan %s is not a Claude Code plan: the path must be a .md file under %s (the session's plans directory: plans/ under %s when that is set, else ~/.claude/plans)", typed, where, configDirEnv)
	}
	content, err := os.ReadFile(planPath)
	if err != nil {
		return unreadable(planPath, err)
	}
	if len(content) < minPlanSize {
		return tooSmall(planPath, len(content))
	}
	dir := hookio.ResolveDir(os.Getenv, "", ctx.ProjectDir, ctx.ProjectDirExplicit)
	root, err := cli.Root(dir)
	if err != nil {
		return err
	}
	cfg, err := cli.Policy(root)
	if err != nil {
		return err
	}
	if pending, ok := readPointer(root); ok && pending != planPath {
		return cli.WithHint(
			cli.Statef("a pending plan is waiting: %s; --plan names %s", pending, planPath),
			"run pk preserve to commit the pending plan first, then preserve the other with --plan")
	}
	return preserve(ctx, root, cfg.Preserve, content, extractTitle(string(content)))
}

// runDiscard is the --discard path: the pointer is removed without
// preserving the plan it names, for the case the plan belongs to
// another repository and was preserved there. A person typed it, so
// stdin is not read; the command commits nothing and narrates on
// stderr. A stale pointer is discarded like a live one, and said to be.
func runDiscard(ctx *cli.Context) error {
	if ctx.String("plan") != "" || ctx.Bool("push") {
		return cli.Usagef("--discard removes the pending plan pointer and takes no --plan or --push")
	}
	dir := hookio.ResolveDir(os.Getenv, "", ctx.ProjectDir, ctx.ProjectDirExplicit)
	root, err := cli.Root(dir)
	if err != nil {
		return err
	}
	planPath, present, stale := peekPointer(root)
	if !present {
		removePointer(root) // an empty pointer file is nothing pending too
		fmt.Fprintln(ctx.Stderr, "No pending plan to discard.")
		return nil
	}
	missing := ""
	if stale {
		missing = " (the plan file no longer exists)"
	}
	if ctx.Bool("dry-run") {
		fmt.Fprintf(ctx.Stderr, "pk preserve --dry-run:\n")
		fmt.Fprintf(ctx.Stderr, "  Discard: %s%s\n", planPath, missing)
		return nil
	}
	if err := os.Remove(pointerPath(root)); err != nil {
		return cli.Statef("failed to remove the pending plan pointer: %v", err)
	}
	fmt.Fprintf(ctx.Stderr, "Discarded the pending plan: %s%s\n", planPath, missing)
	return nil
}

// preserve writes the plan into the plans directory under the next
// filename for the day and commits it, reporting through the response
// envelope.
// Identical bytes already committed report the existing file instead.
// A failure is returned: a refusal as a state error, a git command or
// a write that failed as an internal one. The typed runs and the
// automatic hook share this path, and outcome gives each its exit.
func preserve(ctx *cli.Context, root string, policy config.PreserveConfig, content []byte, title string) error {
	datePrefix := now().Format("2006-01-02")
	slug := slugify(title, 60)
	if slug == "" {
		slug = "untitled"
	}
	destDir := policy.DirPath(root)

	// The file is the one already holding these bytes, when there is
	// one, else the next name for the day.
	filename, seq := scanDestDir(destDir, datePrefix, content)
	leftover := filename != ""
	if !leftover {
		filename = fmt.Sprintf("%s-%03d-%s.md", datePrefix, seq, slug)
	}
	relPath := path.Join(policy.ResolvedDir(), filename)

	// A plan is preserved when it is committed, and git commits nothing
	// it ignores, so an ignored target is refused before any write.
	ignored, err := git.Ignored(root, relPath)
	if err != nil {
		return err
	}
	if ignored {
		return cli.Statef("%s is ignored by git, so a plan cannot be committed there: change preserve.dir in %s, or the ignore rule", relPath, config.FileName)
	}
	if leftover {
		// The same bytes, committed, are the plan already preserved. Not
		// committed, the file is what a run whose commit failed left
		// behind, and this run commits it as it stands.
		clean, err := git.Clean(root, relPath)
		if err != nil {
			return err
		}
		if clean {
			removePointer(root)
			return writeResponse(ctx, fmt.Sprintf("Plan already preserved as %s", relPath), "")
		}
	}

	// The marker skips CI for the whole push it lands in, so it goes on
	// only when the plan commit is the only thing the next push carries:
	// HEAD is on its upstream. Unpushed commits before it, or no upstream
	// to compare with, leave the marker off and the output says why.
	ahead, known, err := git.Ahead(root)
	if err != nil {
		return err
	}
	subject := fmt.Sprintf("%s: %s", config.PlanType, title)
	reason := ""
	switch {
	case !known:
		reason = "the branch has no upstream"
	case ahead > 0:
		reason = "unpushed commits precede it"
	default:
		subject += " [skip ci]"
	}
	why := ""
	if reason != "" {
		why = fmt.Sprintf(" ([skip ci] left off: %s)", reason)
	}

	if ctx.Bool("dry-run") {
		fmt.Fprintf(ctx.Stderr, "pk preserve --dry-run:\n")
		fmt.Fprintf(ctx.Stderr, "  Plan:   %s\n", title)
		fmt.Fprintf(ctx.Stderr, "  File:   %s\n", relPath)
		fmt.Fprintf(ctx.Stderr, "  Commit: %s\n", subject)
		if reason != "" {
			fmt.Fprintf(ctx.Stderr, "  CI:     [skip ci] left off: %s\n", reason)
		}
		if ctx.Bool("push") {
			fmt.Fprintf(ctx.Stderr, "  Push:   git push origin HEAD\n")
		}
		return nil
	}

	if !leftover {
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory: %v", err)
		}
		if err := os.WriteFile(filepath.Join(destDir, filename), content, 0o644); err != nil {
			return fmt.Errorf("failed to write plan: %v", err)
		}
	}
	// The commit holds the plan alone; other staged work stays staged.
	if err := git.CommitPaths(root, subject, relPath); err != nil {
		return err
	}
	removePointer(root)

	if ctx.Bool("push") {
		if _, err := git.Exec(root, "push", "origin", "HEAD"); err != nil {
			return writeResponse(ctx, fmt.Sprintf("Plan committed locally but push failed: %s%s", relPath, why), "")
		}
		return writeResponse(ctx, fmt.Sprintf("Approved plan committed and pushed: %s%s", relPath, why), "")
	}
	return writeResponse(ctx, fmt.Sprintf("Approved plan committed: %s%s", relPath, why), "")
}

// writeResponse writes the response envelope. A write that fails is
// returned like any other failure of a preserve, and outcome gives it
// the exit its caller is owed.
func writeResponse(ctx *cli.Context, message, context string) error {
	return hookio.WritePostToolUse(ctx.Stdout, message, context)
}

// pointerPath returns the pending-plan pointer path for a repo.
func pointerPath(root string) string {
	return filepath.Join(root, ".git", pointerFilename)
}

// peekPointer reads the pending-plan pointer and leaves it as it is:
// present when the file names a plan, stale when that plan no longer
// exists. readPointer consumes a stale one; pk status and --discard
// report it instead.
func peekPointer(root string) (planPath string, present, stale bool) {
	data, err := os.ReadFile(pointerPath(root))
	if err != nil {
		return "", false, false
	}
	planPath = strings.TrimSpace(string(data))
	if planPath == "" {
		return "", false, false
	}
	_, err = os.Stat(planPath)
	return planPath, true, err != nil
}

// readPointer reads the pending-plan pointer. Returns the plan path and
// true when the pointer exists and its target still does. A stale or
// empty pointer is deleted and reported absent.
func readPointer(root string) (string, bool) {
	planPath, present, stale := peekPointer(root)
	if !present || stale {
		removePointer(root)
		return "", false
	}
	return planPath, true
}

// PendingPlan reports the plan the pending pointer names, for pk status:
// ok when a pointer is present, stale when its plan file is gone. The
// pointer is left in place either way.
func PendingPlan(root string) (path string, stale, ok bool) {
	path, ok, stale = peekPointer(root)
	return path, stale, ok
}

// writePointer records the pending-plan pointer. Best-effort: a failure
// (worktree .git file, read-only dir) is logged and life goes on.
func writePointer(ctx *cli.Context, root, planPath string) {
	if err := os.WriteFile(pointerPath(root), []byte(planPath+"\n"), 0o644); err != nil {
		msg.Hookf(ctx.Stderr, "preserve", "failed to write pending-plan pointer: %v", err)
	}
}

func removePointer(root string) {
	os.Remove(pointerPath(root))
}

// sessionPlansDir is the plans directory of the Claude Code session the
// hook or the person runs in: claudePlansDir over the process environment.
func sessionPlansDir() string {
	return claudePlansDir(os.Getenv(configDirEnv), os.UserHomeDir)
}

// claudePlansDir is the one directory Claude Code writes approved plans to:
// plans/ under configDir (CLAUDE_CONFIG_DIR) when that is set, else
// ~/.claude/plans. Only that directory counts, because a second account
// keeps its own ~/.claude/plans and a file of the same name there is
// another session's plan. Forward slashes throughout so a Windows path
// matches after normalization; a leading ~/ is expanded; "" when neither
// the variable nor the home directory is available.
func claudePlansDir(configDir string, homeDir func() (string, error)) string {
	home := ""
	if h, err := homeDir(); err == nil {
		home = strings.ReplaceAll(h, `\`, "/")
	}
	if configDir != "" {
		dir := strings.ReplaceAll(configDir, `\`, "/")
		if strings.HasPrefix(dir, "~/") && home != "" {
			dir = home + dir[1:]
		}
		return strings.TrimRight(path.Clean(dir), "/") + "/plans"
	}
	if home == "" {
		return ""
	}
	return home + "/.claude/plans"
}

// extractPlanPath finds the plan file path in a PostToolUse
// tool_response. The harness sends an object carrying the path in a
// filePath field; older versions sent a plain string. Parsing the JSON
// structurally lets encoding/json handle escaping, so a Windows path's
// escaped backslashes decode to single separators before normalization;
// matching raw bytes would turn each \\ into // and break the path.
func extractPlanPath(toolResponse json.RawMessage, dir string, homeDir func() (string, error)) string {
	if len(toolResponse) == 0 {
		return ""
	}
	var obj struct {
		FilePath string `json:"filePath"`
	}
	if json.Unmarshal(toolResponse, &obj) == nil && obj.FilePath != "" {
		if p := matchPlanPath(obj.FilePath, dir, homeDir); p != "" {
			return p
		}
	}
	var s string
	if json.Unmarshal(toolResponse, &s) == nil {
		return matchPlanPath(s, dir, homeDir)
	}
	return ""
}

// matchPlanPath normalizes backslashes to forward slashes so Windows
// paths match, expands a leading ~/ to the home directory, then returns
// the first <dir>/*.md path found in the text, or "". dir is claudePlansDir's
// answer; "" matches nothing. The path is returned clean, and one that
// climbs back out of dir (<dir>/../x.md) is not under it and matches
// nothing: this is where "inside the plans directory" is decided, for
// the hook's payload and --plan alike.
func matchPlanPath(text, dir string, homeDir func() (string, error)) string {
	if dir == "" {
		return ""
	}
	normalized := strings.ReplaceAll(text, `\`, "/")
	if home, err := homeDir(); err == nil {
		home = strings.ReplaceAll(home, `\`, "/")
		normalized = strings.ReplaceAll(normalized, "~/", home+"/")
	}
	re := regexp.MustCompile(regexp.QuoteMeta(dir) + `/[^ "]*\.md`)
	found := re.FindString(normalized)
	if found == "" {
		return ""
	}
	clean := path.Clean(found)
	if !strings.HasPrefix(clean, dir+"/") {
		return ""
	}
	return clean
}

// extractTitle finds the first H1 heading. CRLF content sheds its \r
// here, so titles never carry a carriage return into filenames or
// commit messages (a v1 gap).
func extractTitle(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
	}
	return "untitled plan"
}

// slugify converts a title to a filename-friendly slug: lowercase,
// letters and digits kept, runs of everything else collapsed to single
// hyphens, trimmed, capped at maxLen runes.
func slugify(title string, maxLen int) string {
	var b strings.Builder
	prevHyphen := false
	for _, r := range strings.ToLower(title) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevHyphen = false
		} else if !prevHyphen && b.Len() > 0 {
			b.WriteByte('-')
			prevHyphen = true
		}
	}
	s := strings.TrimRight(b.String(), "-")
	if utf8.RuneCountInString(s) > maxLen {
		runes := []rune(s)
		s = strings.TrimRight(string(runes[:maxLen]), "-")
	}
	return s
}

// scanDestDir reads destDir once and returns both a duplicate filename
// (an existing file with identical bytes) and the next sequence number
// for the date. Size is the fast path before comparing content. A
// missing directory returns ("", 1).
func scanDestDir(destDir, datePrefix string, content []byte) (dupName string, nextSeq int) {
	entries, err := os.ReadDir(destDir)
	if err != nil {
		return "", 1
	}
	maxSeq := 0
	prefix := datePrefix + "-"
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			continue
		}
		if strings.HasPrefix(name, prefix) {
			rest := name[len(prefix):]
			i := 0
			for i < len(rest) && rest[i] >= '0' && rest[i] <= '9' {
				i++
			}
			if i > 0 && i < len(rest) && rest[i] == '-' {
				seq := 0
				for _, c := range rest[:i] {
					seq = seq*10 + int(c-'0')
				}
				if seq > maxSeq {
					maxSeq = seq
				}
			}
		}
		if dupName == "" {
			if info, err := entry.Info(); err == nil && info.Size() == int64(len(content)) {
				if existing, err := os.ReadFile(filepath.Join(destDir, name)); err == nil && bytes.Equal(existing, content) {
					dupName = name
				}
			}
		}
	}
	return dupName, maxSeq + 1
}
