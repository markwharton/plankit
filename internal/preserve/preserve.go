// Package preserve implements the preserve hook: it captures approved
// Claude Code plans, byte for byte, as dated and sequenced files in
// docs/plans/, and commits them.
//
// Two invocation shapes share this command. The automatic PostToolUse
// hook on ExitPlanMode arrives with a payload on stdin and honors
// .pk.json preserve.mode: auto commits immediately, manual records a
// pending-plan pointer and tells the person to run /plankit:preserve,
// off does nothing. An explicit invocation (the /plankit:preserve skill,
// or a person at a terminal) arrives with no payload, consumes the
// pointer, and always commits.
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
	"github.com/markwharton/plankit/internal/paths"
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

// Cmd is the preserve command: hook-driven and explicitly invocable.
var Cmd = &cli.Command{
	Name:    "preserve",
	Summary: "Hook: preserve the approved plan into docs/plans and commit it",
	Hook:    true,
	Flags: []cli.FlagSpec{
		{Name: "push", Type: cli.BoolFlag, Usage: "Push to origin after committing"},
		{Name: "dry-run", Type: cli.BoolFlag, Usage: "Preview without writing or committing"},
	},
	Run: run,
}

func run(ctx *cli.Context) error {
	var planPath, reason, payloadCWD string

	input, err := hookio.ReadInput(ctx.Stdin)
	if err == nil {
		payloadCWD = input.CWD
	}
	// A valid payload means the automatic ExitPlanMode hook, which honors
	// preserve.mode. No payload means an explicit invocation, which
	// consumes the pointer and always commits.
	hookInvocation := err == nil

	dir := hookio.ResolveDir(os.Getenv, payloadCWD, ctx.ProjectDir)
	root, rootOK := git.FindRoot(dir)

	if hookInvocation {
		planPath = extractPlanPath(input.ToolResponse, os.UserHomeDir)
		if planPath == "" {
			reason = "tool_response did not contain a .claude/plans/*.md path"
		} else if _, err := os.Stat(planPath); err != nil {
			reason = fmt.Sprintf("plan file not found: %s", planPath)
			planPath = ""
		}
	} else if rootOK {
		if p, ok := readPointer(root); ok {
			planPath = p
		} else {
			reason = "stdin had no hook payload and no pending-plan pointer was found"
		}
	}

	if planPath == "" {
		if ctx.Bool("dry-run") {
			msg.Hookf(ctx.Stderr, "preserve --dry-run", "no plan found (%s)", reason)
		}
		return nil
	}

	content, err := os.ReadFile(planPath)
	if err != nil {
		msg.Hookf(ctx.Stderr, "preserve", "failed to read plan: %v", err)
		return nil
	}
	if len(content) < minPlanSize {
		return nil
	}

	if !rootOK {
		msg.Hookf(ctx.Stderr, "preserve", "not a git repository: %s", dir)
		return nil
	}

	// The automatic hook honors policy; the explicit invocation is the
	// person acting, which is consent enough.
	mode := "auto"
	if hookInvocation {
		cfg, err := config.Load(root)
		switch {
		case errors.Is(err, config.ErrNotConfigured):
			return nil // plankit is off here; the hook fires everywhere
		case err != nil:
			msg.Hookf(ctx.Stderr, "preserve", "%v", err)
			return nil
		}
		mode = cfg.Preserve.ResolvedMode()
	}

	title := extractTitle(string(content))
	switch mode {
	case "off":
		return nil
	case "manual":
		writePointer(ctx, root, planPath)
		writeResponse(ctx,
			fmt.Sprintf("Plan '%s' ready. Type /plankit:preserve to save it.", title),
			fmt.Sprintf("The user's plan '%s' has been approved. Inform the user that they can type /plankit:preserve to save it to docs/plans/.", title))
		return nil
	}

	// auto, or explicit invocation: commit.
	datePrefix := now().Format("2006-01-02")
	slug := slugify(title, 60)
	if slug == "" {
		slug = "untitled"
	}
	destDir := paths.Plans(root)

	dupName, seq := scanDestDir(destDir, datePrefix, content)
	if dupName != "" {
		removePointer(root)
		writeResponse(ctx, fmt.Sprintf("Plan already preserved as %s", paths.PlanRel(dupName)), "")
		return nil
	}
	filename := fmt.Sprintf("%s-%03d-%s.md", datePrefix, seq, slug)
	relPath := paths.PlanRel(filename)

	if ctx.Bool("dry-run") {
		fmt.Fprintf(ctx.Stderr, "pk preserve --dry-run:\n")
		fmt.Fprintf(ctx.Stderr, "  Plan:   %s\n", title)
		fmt.Fprintf(ctx.Stderr, "  File:   %s\n", relPath)
		fmt.Fprintf(ctx.Stderr, "  Commit: plan: %s [skip ci]\n", title)
		if ctx.Bool("push") {
			fmt.Fprintf(ctx.Stderr, "  Push:   git push origin HEAD\n")
		}
		return nil
	}

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		msg.Hookf(ctx.Stderr, "preserve", "failed to create directory: %v", err)
		return nil
	}
	if err := os.WriteFile(filepath.Join(destDir, filename), content, 0o644); err != nil {
		msg.Hookf(ctx.Stderr, "preserve", "failed to write plan: %v", err)
		return nil
	}
	if _, err := git.Exec(root, "add", relPath); err != nil {
		msg.Hookf(ctx.Stderr, "preserve", "git add failed: %v", err)
		return nil
	}
	// Nothing staged means the identical bytes were already committed.
	if _, err := git.Exec(root, "diff", "--cached", "--quiet"); err == nil {
		removePointer(root)
		writeResponse(ctx, "Plan unchanged, no commit needed.", "")
		return nil
	}
	if _, err := git.Exec(root, "commit", "-m", fmt.Sprintf("%s: %s [skip ci]", config.PlanType, title)); err != nil {
		msg.Hookf(ctx.Stderr, "preserve", "git commit failed: %v", err)
		return nil
	}
	removePointer(root)

	if ctx.Bool("push") {
		if _, err := git.Exec(root, "push", "origin", "HEAD"); err != nil {
			writeResponse(ctx, fmt.Sprintf("Plan committed locally but push failed: %s", relPath), "")
		} else {
			writeResponse(ctx, fmt.Sprintf("Approved plan committed and pushed: %s", relPath), "")
		}
	} else {
		writeResponse(ctx, fmt.Sprintf("Approved plan committed: %s", relPath), "")
	}
	return nil
}

func writeResponse(ctx *cli.Context, message, context string) {
	if err := hookio.WritePostToolUse(ctx.Stdout, message, context); err != nil {
		msg.Hookf(ctx.Stderr, "preserve", "write error: %v", err)
	}
}

// pointerPath returns the pending-plan pointer path for a repo.
func pointerPath(root string) string {
	return filepath.Join(root, ".git", pointerFilename)
}

// readPointer reads the pending-plan pointer. Returns the plan path and
// true when the pointer exists and its target still does. A stale
// pointer is deleted and reported absent.
func readPointer(root string) (string, bool) {
	path := pointerPath(root)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	planPath := strings.TrimSpace(string(data))
	if planPath == "" {
		os.Remove(path)
		return "", false
	}
	if _, err := os.Stat(planPath); err != nil {
		os.Remove(path)
		return "", false
	}
	return planPath, true
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

// planPathRegex matches Claude Code plan file paths. The optional drive
// letter handles Windows paths after backslash normalization.
var planPathRegex = regexp.MustCompile(`(?:[A-Za-z]:)?/[^ "]*\.claude/plans/[^ "]*\.md`)

// extractPlanPath finds the plan file path in a PostToolUse
// tool_response. The harness sends an object carrying the path in a
// filePath field; older versions sent a plain string. Parsing the JSON
// structurally lets encoding/json handle escaping, so a Windows path's
// escaped backslashes decode to single separators before normalization;
// matching raw bytes would turn each \\ into // and break the path.
func extractPlanPath(toolResponse json.RawMessage, homeDir func() (string, error)) string {
	if len(toolResponse) == 0 {
		return ""
	}
	var obj struct {
		FilePath string `json:"filePath"`
	}
	if json.Unmarshal(toolResponse, &obj) == nil && obj.FilePath != "" {
		if p := matchPlanPath(obj.FilePath, homeDir); p != "" {
			return p
		}
	}
	var s string
	if json.Unmarshal(toolResponse, &s) == nil {
		return matchPlanPath(s, homeDir)
	}
	return ""
}

// matchPlanPath normalizes backslashes to forward slashes so Windows
// paths match, expands a leading ~/ to the home directory, then returns
// the first .claude/plans/*.md path found, or "".
func matchPlanPath(text string, homeDir func() (string, error)) string {
	normalized := strings.ReplaceAll(text, `\`, "/")
	if home, err := homeDir(); err == nil {
		home = strings.ReplaceAll(home, `\`, "/")
		normalized = strings.ReplaceAll(normalized, "~/", home+"/")
	}
	return planPathRegex.FindString(normalized)
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
