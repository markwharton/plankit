// Package preserve implements the preserve PostToolUse hook command.
// It captures approved Claude Code plans as timestamped files in docs/plans/
// and commits them to git.
package preserve

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	pkgit "github.com/markwharton/plankit/internal/git"
	"github.com/markwharton/plankit/internal/hooks"
)

// Config holds injectable dependencies for testing.
type Config struct {
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
	Env     func(string) string
	HomeDir func() (string, error)
	Now     func() time.Time
	GitExec func(projectDir string, args ...string) (string, error)

	Getwd     func() (string, error)
	ReadFile  func(string) ([]byte, error)
	WriteFile func(string, []byte, os.FileMode) error
	Stat      func(string) (os.FileInfo, error)
	MkdirAll  func(string, os.FileMode) error
	ReadDir   func(string) ([]os.DirEntry, error)
	Remove    func(string) error

	// Notify outputs a systemMessage prompt without preserving.
	Notify bool
	// DryRun previews the preserve without writing, committing, or pushing.
	DryRun bool
	// Push pushes to origin after committing. Default is commit only.
	Push bool
	// CheckUpdate returns an update notice string, or "".
	CheckUpdate func() string
}

// DefaultConfig returns a Config wired to real implementations.
func DefaultConfig() Config {
	return Config{
		Stdin:     os.Stdin,
		Stdout:    os.Stdout,
		Stderr:    os.Stderr,
		Env:       os.Getenv,
		HomeDir:   os.UserHomeDir,
		Now:       time.Now,
		GitExec:   pkgit.Exec,
		Getwd:     os.Getwd,
		ReadFile:  os.ReadFile,
		WriteFile: os.WriteFile,
		Stat:      os.Stat,
		MkdirAll:  os.MkdirAll,
		ReadDir:   os.ReadDir,
		Remove:    os.Remove,
	}
}

// minPlanSize is the minimum byte length for a plan to be preserved.
// Real plans have a title, context section, and at least a few lines of
// substance. Plans below this threshold are typically empty templates
// or aborted drafts.
const minPlanSize = 50

// pointerFilename names the per-repo pending-plan pointer written by
// --notify mode and consumed by the /preserve skill invocation. It lives
// under .git/ because that directory is always untracked by git, so no
// .gitignore coordination is needed and the file is naturally scoped to
// the repo. ~/.claude/plans/ is shared across Claude sessions, so an
// mtime-based selection in findLatestPlan() can grab a rival session's
// plan — the pointer records the exact plan that was approved so /preserve
// can pick it up even if the user runs it minutes later.
const pointerFilename = "pk-pending-plan"

// Run reads a PostToolUse hook payload from stdin and preserves the approved plan.
// Returns the process exit code (always 0 for hook commands).
func Run(cfg Config) int {
	var planPath string
	var inputCWD string

	input, err := hooks.ReadInput(cfg.Stdin)
	if err == nil {
		inputCWD = input.CWD
	}
	projectDir := resolveProjectDir(cfg, inputCWD)

	if err != nil {
		// stdin has no hook payload (e.g., /preserve skill invocation).
		// Prefer the project-local pointer written by --notify; fall back
		// to mtime-based findLatestPlan only when the pointer is absent.
		if projectDir != "" {
			if p, ok := readPointer(cfg, projectDir); ok {
				planPath = p
			}
		}
		if planPath == "" {
			planPath = findLatestPlan(cfg)
		}
	} else {
		planPath = extractPlanPath(input.ToolResponseString())
		if planPath == "" || !fileExists(cfg, planPath) {
			planPath = findLatestPlan(cfg)
		}
	}

	if planPath == "" {
		return 0
	}

	// Read plan content.
	content, err := cfg.ReadFile(planPath)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "pk preserve: failed to read plan: %v\n", err)
		return 0
	}

	// Skip trivially short plans.
	if len(content) < minPlanSize {
		return 0
	}

	// Notify-only mode: output plan title, record the pointer for later /preserve, don't commit.
	if cfg.Notify {
		title := extractTitle(string(content))
		if projectDir != "" {
			writePointer(cfg, projectDir, planPath)
		}
		cfg.writeHookResponse(
			fmt.Sprintf("Plan '%s' ready. Type /preserve to save it.", title),
			fmt.Sprintf("The user's plan '%s' has been approved. Inform the user that they can type /preserve to save it to docs/plans/.", title),
		)
		return 0
	}

	if projectDir == "" {
		fmt.Fprintf(cfg.Stderr, "pk preserve: could not determine project directory\n")
		return 0
	}

	// Verify git repo.
	if _, err := cfg.GitExec(projectDir, "rev-parse", "--is-inside-work-tree"); err != nil {
		fmt.Fprintf(cfg.Stderr, "pk preserve: not a git repository: %s\n", projectDir)
		return 0
	}

	// Extract title and build destination path.
	title := extractTitle(string(content))
	datePrefix := cfg.Now().Format("2006-01-02")
	slug := slugify(title, 60)
	if slug == "" {
		slug = "untitled"
	}
	destDir := filepath.Join(projectDir, "docs", "plans")

	// Scan destination directory for duplicates and next sequence number.
	// scanDestDir handles a missing directory gracefully (returns "", 1).
	dupName, seq := scanDestDir(cfg, destDir, datePrefix, content)
	if dupName != "" {
		removePointer(cfg, projectDir)
		cfg.writeSystemMessage(fmt.Sprintf("Plan already preserved as docs/plans/%s", dupName))
		return 0
	}
	filename := fmt.Sprintf("%s-%03d-%s.md", datePrefix, seq, slug)

	// Dry-run mode: preview without writing, committing, or pushing.
	if cfg.DryRun {
		relPath := filepath.Join("docs", "plans", filename)
		fmt.Fprintf(cfg.Stderr, "pk preserve --dry-run:\n")
		fmt.Fprintf(cfg.Stderr, "  Plan:   %s\n", title)
		fmt.Fprintf(cfg.Stderr, "  File:   %s\n", relPath)
		fmt.Fprintf(cfg.Stderr, "  Commit: plan: %s [skip ci]\n", title)
		if cfg.Push {
			fmt.Fprintf(cfg.Stderr, "  Push:   git push origin HEAD\n")
		}
		return 0
	}

	// Create directory before writing.
	if err := cfg.MkdirAll(destDir, 0755); err != nil {
		fmt.Fprintf(cfg.Stderr, "pk preserve: failed to create directory: %v\n", err)
		return 0
	}

	destFile := filepath.Join(destDir, filename)
	if err := cfg.WriteFile(destFile, content, 0644); err != nil {
		fmt.Fprintf(cfg.Stderr, "pk preserve: failed to write plan: %v\n", err)
		return 0
	}

	// Git add.
	relPath := filepath.Join("docs", "plans", filename)
	if _, err := cfg.GitExec(projectDir, "add", relPath); err != nil {
		fmt.Fprintf(cfg.Stderr, "pk preserve: git add failed: %v\n", err)
		return 0
	}

	// Check for staged changes.
	if _, err := cfg.GitExec(projectDir, "diff", "--cached", "--quiet"); err == nil {
		removePointer(cfg, projectDir)
		cfg.writeSystemMessage("Plan unchanged, no commit needed.")
		return 0
	}

	// Git commit.
	commitMsg := fmt.Sprintf("plan: %s [skip ci]", title)
	if _, err := cfg.GitExec(projectDir, "commit", "-m", commitMsg); err != nil {
		fmt.Fprintf(cfg.Stderr, "pk preserve: git commit failed: %v\n", err)
		return 0
	}
	removePointer(cfg, projectDir)

	// Git push (only when --push is set).
	if cfg.Push {
		if _, err := cfg.GitExec(projectDir, "push", "origin", "HEAD"); err != nil {
			cfg.writeSystemMessage(fmt.Sprintf("Plan committed locally but push failed: docs/plans/%s", filename))
		} else {
			cfg.writeSystemMessage(fmt.Sprintf("Approved plan committed and pushed: docs/plans/%s", filename))
		}
	} else {
		cfg.writeSystemMessage(fmt.Sprintf("Approved plan committed: docs/plans/%s", filename))
	}

	return 0
}

// resolveProjectDir determines the project directory from CLAUDE_PROJECT_DIR,
// the hook payload's CWD, or os.Getwd(). Returns "" when no source yields a path.
func resolveProjectDir(cfg Config, inputCWD string) string {
	projectDir := hooks.ResolveProjectDir(cfg.Env, inputCWD)
	if projectDir == "" && cfg.Getwd != nil {
		if wd, err := cfg.Getwd(); err == nil {
			projectDir = wd
		}
	}
	return projectDir
}

// pointerPath returns the absolute path to the pending-plan pointer for a repo.
func pointerPath(projectDir string) string {
	return filepath.Join(projectDir, ".git", pointerFilename)
}

// readPointer reads the pending-plan pointer. Returns the plan path and true
// when the pointer is present and the pointed-to file still exists. A stale
// pointer (missing target) is deleted and reported as absent.
func readPointer(cfg Config, projectDir string) (string, bool) {
	path := pointerPath(projectDir)
	data, err := cfg.ReadFile(path)
	if err != nil {
		return "", false
	}
	planPath := strings.TrimSpace(string(data))
	if planPath == "" {
		cfg.Remove(path)
		return "", false
	}
	if _, err := cfg.Stat(planPath); err != nil {
		cfg.Remove(path)
		return "", false
	}
	return planPath, true
}

// writePointer records the pending-plan pointer. Best-effort: a write failure
// (e.g., .git/ is a worktree file or the directory is read-only) is logged and
// the caller continues — /preserve will still work via the mtime fallback.
func writePointer(cfg Config, projectDir, planPath string) {
	path := pointerPath(projectDir)
	if err := cfg.WriteFile(path, []byte(planPath+"\n"), 0644); err != nil {
		fmt.Fprintf(cfg.Stderr, "pk preserve: failed to write pending-plan pointer: %v\n", err)
	}
}

// removePointer deletes the pending-plan pointer. Best-effort.
func removePointer(cfg Config, projectDir string) {
	cfg.Remove(pointerPath(projectDir))
}

// planPathRegex matches paths to Claude Code plan files.
var planPathRegex = regexp.MustCompile(`/[^ "]*\.claude/plans/[^ "]*\.md`)

// extractPlanPath searches the tool response text for a .claude/plans/*.md path.
func extractPlanPath(text string) string {
	match := planPathRegex.FindString(text)
	return match
}

// extractTitle finds the first H1 heading in the plan content.
func extractTitle(content string) string {
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
	}
	return "untitled plan"
}

// slugify converts a title to a URL-friendly slug.
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

	s := b.String()
	s = strings.TrimRight(s, "-")

	if utf8.RuneCountInString(s) > maxLen {
		runes := []rune(s)
		s = string(runes[:maxLen])
		s = strings.TrimRight(s, "-")
	}

	return s
}

// scanDestDir reads destDir once and returns both a duplicate filename (if any
// existing file for the given date has identical content) and the next sequence
// number. This avoids two separate ReadDir calls on the same directory.
func scanDestDir(cfg Config, destDir, datePrefix string, content []byte) (dupName string, nextSeq int) {
	entries, err := cfg.ReadDir(destDir)
	if err != nil {
		return "", 1
	}
	maxSeq := 0
	prefix := datePrefix + "-"
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasPrefix(name, prefix) {
			continue
		}

		// Parse variable-width sequence digits followed by '-'.
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

		// Check for duplicate content, using file size as a fast path.
		if dupName == "" {
			if info, err := entry.Info(); err == nil && info.Size() == int64(len(content)) {
				if existing, err := cfg.ReadFile(filepath.Join(destDir, name)); err == nil {
					if bytes.Equal(existing, content) {
						dupName = name
					}
				}
			}
		}
	}
	return dupName, maxSeq + 1
}

// findLatestPlan returns the most recently modified .md file in ~/.claude/plans/.
func findLatestPlan(cfg Config) string {
	home, err := cfg.HomeDir()
	if err != nil {
		return ""
	}

	plansDir := filepath.Join(home, ".claude", "plans")
	entries, err := cfg.ReadDir(plansDir)
	if err != nil {
		return ""
	}

	var latestPath string
	var latestTime time.Time

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(latestTime) {
			latestTime = info.ModTime()
			latestPath = filepath.Join(plansDir, entry.Name())
		}
	}

	return latestPath
}

func fileExists(cfg Config, path string) bool {
	_, err := cfg.Stat(path)
	return err == nil
}

// writeSystemMessage outputs a hook systemMessage JSON to stdout.
// If CheckUpdate is configured and returns a notice, it is appended.
func (cfg Config) writeSystemMessage(msg string) {
	cfg.writeHookResponse(msg, "")
}

// writeHookResponse outputs a hook response with both systemMessage (shown to user)
// and additionalContext (injected into Claude's next turn).
func (cfg Config) writeHookResponse(msg, context string) {
	if cfg.CheckUpdate != nil {
		if notice := cfg.CheckUpdate(); notice != "" {
			msg += " | " + notice
		}
	}
	if err := hooks.WritePostToolUse(cfg.Stdout, msg, context); err != nil {
		fmt.Fprintf(cfg.Stderr, "pk preserve: write error: %v\n", err)
	}
}
