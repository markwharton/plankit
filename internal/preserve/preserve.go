// Package preserve implements the preserve PostToolUse hook command.
// It captures approved Claude Code plans as timestamped files in docs/plans/
// and commits them to git.
package preserve

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"

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

	Getwd func() (string, error)

	// Notify outputs a systemMessage prompt without preserving.
	Notify bool
	// DryRun previews the preserve without writing, committing, or pushing.
	DryRun bool
	// CheckUpdate returns an update notice string, or "".
	CheckUpdate func() string
}

// DefaultConfig returns a Config wired to real implementations.
func DefaultConfig() Config {
	return Config{
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Env:     os.Getenv,
		HomeDir: os.UserHomeDir,
		Now:     time.Now,
		GitExec: defaultGitExec,
		Getwd:   os.Getwd,
	}
}

// Run reads a PostToolUse hook payload from stdin and preserves the approved plan.
// Returns the process exit code (always 0 for hook commands).
func Run(cfg Config) int {
	var planPath string
	var inputCWD string

	input, err := hooks.ReadInput(cfg.Stdin)
	if err != nil {
		// stdin may not have a hook payload (e.g., invoked via skill).
		// Fall back to finding the latest plan.
		planPath = findLatestPlan(cfg.HomeDir)
	} else {
		inputCWD = input.CWD
		planPath = extractPlanPath(input.ToolResponseString())
		if planPath == "" || !fileExists(planPath) {
			planPath = findLatestPlan(cfg.HomeDir)
		}
	}

	if planPath == "" {
		return 0
	}

	// Read plan content.
	content, err := os.ReadFile(planPath)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "pk preserve: failed to read plan: %v\n", err)
		return 0
	}

	// Skip trivially short plans (< 50 bytes). Real plans have a title,
	// context section, and at least a few lines of substance. Plans below
	// this threshold are typically empty templates or aborted drafts.
	if len(content) < 50 {
		return 0
	}

	// Notify-only mode: output plan title, don't preserve.
	if cfg.Notify {
		title := extractTitle(string(content))
		cfg.writeSystemMessage(fmt.Sprintf("Plan '%s' ready. Type /preserve to save it.", title))
		return 0
	}

	// Determine project directory.
	projectDir := cfg.Env("CLAUDE_PROJECT_DIR")
	if projectDir == "" {
		projectDir = inputCWD
	}
	if projectDir == "" {
		if wd, err := cfg.Getwd(); err == nil {
			projectDir = wd
		}
	}
	if projectDir == "" {
		return 0
	}

	// Verify git repo.
	if _, err := cfg.GitExec(projectDir, "rev-parse", "--is-inside-work-tree"); err != nil {
		return 0
	}

	// Extract title and build destination path.
	title := extractTitle(string(content))
	datePrefix := cfg.Now().Format("2006-01-02")
	slug := slugify(title, 60)
	destDir := filepath.Join(projectDir, "docs", "plans")

	// Ensure directory exists before scanning for sequence number.
	if err := os.MkdirAll(destDir, 0755); err != nil {
		fmt.Fprintf(cfg.Stderr, "pk preserve: failed to create directory: %v\n", err)
		return 0
	}

	// Scan destination directory for duplicates and next sequence number.
	dupName, seq := scanDestDir(destDir, datePrefix, content)
	if dupName != "" {
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
		fmt.Fprintf(cfg.Stderr, "  Push:   git push origin HEAD\n")
		return 0
	}

	destFile := filepath.Join(destDir, filename)
	if err := os.WriteFile(destFile, content, 0644); err != nil {
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
		cfg.writeSystemMessage("Plan unchanged, no commit needed.")
		return 0
	}

	// Git commit.
	commitMsg := fmt.Sprintf("plan: %s [skip ci]", title)
	if _, err := cfg.GitExec(projectDir, "commit", "-m", commitMsg); err != nil {
		fmt.Fprintf(cfg.Stderr, "pk preserve: git commit failed: %v\n", err)
		return 0
	}

	// Git push (non-fatal).
	if _, err := cfg.GitExec(projectDir, "push", "origin", "HEAD"); err != nil {
		cfg.writeSystemMessage(fmt.Sprintf("Plan committed locally but push failed: docs/plans/%s", filename))
	} else {
		cfg.writeSystemMessage(fmt.Sprintf("Approved plan committed and pushed: docs/plans/%s", filename))
	}

	return 0
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

	if len(s) > maxLen {
		s = s[:maxLen]
		s = strings.TrimRight(s, "-")
	}

	return s
}

// scanDestDir reads destDir once and returns both a duplicate filename (if any
// existing file for the given date has identical content) and the next sequence
// number. This avoids two separate os.ReadDir calls on the same directory.
func scanDestDir(destDir, datePrefix string, content []byte) (dupName string, nextSeq int) {
	entries, err := os.ReadDir(destDir)
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
				if existing, err := os.ReadFile(filepath.Join(destDir, name)); err == nil {
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
func findLatestPlan(homeDir func() (string, error)) string {
	home, err := homeDir()
	if err != nil {
		return ""
	}

	plansDir := filepath.Join(home, ".claude", "plans")
	entries, err := os.ReadDir(plansDir)
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

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// writeSystemMessage outputs a hook systemMessage JSON to stdout.
// If CheckUpdate is configured and returns a notice, it is appended.
func (cfg Config) writeSystemMessage(msg string) {
	if cfg.CheckUpdate != nil {
		if notice := cfg.CheckUpdate(); notice != "" {
			msg += " | " + notice
		}
	}
	data, _ := json.Marshal(map[string]string{"systemMessage": msg})
	fmt.Fprint(cfg.Stdout, string(data))
}

func defaultGitExec(projectDir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", projectDir}, args...)...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}
