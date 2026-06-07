// Command calibrate measures the real characters-per-token ratio of plankit's
// shipped rule set against a specific Claude model, using the Anthropic
// count_tokens endpoint (tokenization, not inference). The zero-dep estimator in
// internal/rules approximates token counts with a single calibrated constant;
// this tool produces the real number that constant should be set to.
//
// It is maintainer-only eval infrastructure: it needs network access and
// ANTHROPIC_API_KEY, so it is NEVER run in the shipped binary or the release
// path. It lives in evals/ alongside the behavioral evals and is run by hand.
//
// Usage:
//
//	ANTHROPIC_API_KEY=... go run ./evals/calibrate --model claude-opus-4-8
//	ANTHROPIC_API_KEY=... go run ./evals/calibrate --model claude-opus-4-8 --write
//
// --write rewrites charsPerToken, calibrationModel, and calibrated in
// internal/rules/rules.go in place, then prints a suggested commit message.
// Token counts are model-specific, so --model is required and is stamped into
// the output and into the constant it writes.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const countTokensURL = "https://api.anthropic.com/v1/messages/count_tokens"

func main() {
	model := flag.String("model", "", "Claude model id to calibrate against (required; token counts are model-specific)")
	write := flag.Bool("write", false, "Rewrite the constants in internal/rules/rules.go with the measured values")
	repo := flag.String("repo", ".", "Path to the plankit repository root")
	flag.Parse()

	if err := run(*model, *repo, *write); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run(model, repo string, write bool) error {
	if model == "" {
		return fmt.Errorf("--model is required (e.g. --model claude-opus-4-8); token counts are model-specific")
	}
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("ANTHROPIC_API_KEY is not set; this eval calls the count_tokens endpoint")
	}

	alwaysOn, err := concatGroup(repo, alwaysOnFiles(repo))
	if err != nil {
		return err
	}
	skills, err := concatGroup(repo, skillFiles(repo))
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	alwaysTokens, err := countTokens(client, apiKey, model, alwaysOn)
	if err != nil {
		return err
	}
	skillsTokens := 0
	if skills != "" {
		if skillsTokens, err = countTokens(client, apiKey, model, skills); err != nil {
			return err
		}
	}

	alwaysChars := runeLen(alwaysOn)
	skillsChars := runeLen(skills)
	if alwaysTokens == 0 {
		return fmt.Errorf("count_tokens returned 0 tokens for the always-on set; nothing to calibrate")
	}
	// The constant comes from the always-on set only. The estimator is used for the
	// README headline (rules + CLAUDE.md) and the general `pk rules` footprint (any
	// project's rules + CLAUDE.md) — neither measures skills, so the denser, on-demand
	// skills ratio must not bias it. Skills are reported below as information only.
	alwaysRatio := float64(alwaysChars) / float64(alwaysTokens)

	fmt.Printf("Footprint calibration (model: %s)\n", model)
	fmt.Printf("  Note: token counts include a small per-message wrapper overhead (a few tokens), negligible at this scale.\n\n")
	rows := []ratioRow{{"Always-on (rules + CLAUDE.md) [constant]", alwaysChars, alwaysTokens}}
	if skills != "" {
		rows = append(rows, ratioRow{"On-demand (skills) [informational]", skillsChars, skillsTokens})
	}
	printRows(rows)
	fmt.Printf("\nSuggested constant: charsPerToken = %s  (calibrationModel = %q, calibrated = true)\n",
		formatRatio(alwaysRatio), model)

	if !write {
		fmt.Println("\nRun again with --write to apply these to internal/rules/rules.go.")
		return nil
	}

	rulesPath := filepath.Join(repo, "internal", "rules", "rules.go")
	if err := applyCalibration(rulesPath, alwaysRatio, model); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "\nWrote %s (charsPerToken=%s, calibrationModel=%q, calibrated=true)\n",
		rulesPath, formatRatio(alwaysRatio), model)
	fmt.Fprintln(os.Stderr, "Next: run `go run ./evals/footprint` (refresh README), then `make test`, then commit:")
	fmt.Fprintf(os.Stderr, "  git commit -m \"chore: recalibrate footprint estimator to %s chars/token (%s)\"\n",
		formatRatio(alwaysRatio), model)
	return nil
}

// alwaysOnFiles lists the shipped always-on source files: the template CLAUDE.md
// plus every rule, sorted for stable output.
func alwaysOnFiles(repo string) []string {
	files := []string{filepath.Join(repo, "internal", "setup", "template", "CLAUDE.md")}
	files = append(files, mdFiles(filepath.Join(repo, "internal", "setup", "rules"))...)
	return files
}

// skillFiles lists each shipped skill's SKILL.md, sorted.
func skillFiles(repo string) []string {
	skillsDir := filepath.Join(repo, "internal", "setup", "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return nil
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			files = append(files, filepath.Join(skillsDir, e.Name(), "SKILL.md"))
		}
	}
	sort.Strings(files)
	return files
}

// mdFiles returns every *.md file directly under dir, sorted.
func mdFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(files)
	return files
}

// concatGroup reads and newline-normalizes each file, concatenating them the way
// Claude Code would load them in one context.
func concatGroup(repo string, files []string) (string, error) {
	var b strings.Builder
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			return "", fmt.Errorf("read %s: %w", f, err)
		}
		b.WriteString(normalizeLF(string(data)))
	}
	return b.String(), nil
}

// countTokens calls the Anthropic count_tokens endpoint and returns input_tokens.
func countTokens(client *http.Client, apiKey, model, content string) (int, error) {
	payload, err := json.Marshal(map[string]any{
		"model":    model,
		"messages": []map[string]any{{"role": "user", "content": content}},
	})
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequest(http.MethodPost, countTokensURL, bytes.NewReader(payload))
	if err != nil {
		return 0, err
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("count_tokens %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var out struct {
		InputTokens int `json:"input_tokens"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return 0, fmt.Errorf("parse count_tokens response: %w", err)
	}
	return out.InputTokens, nil
}

// applyCalibration rewrites the charsPerToken, calibrationModel, and calibrated
// constants in rules.go and reformats the file with gofmt so no lint drift is
// introduced. Comment lines are skipped so only the assignments are touched.
func applyCalibration(path string, ratio float64, model string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	repl := map[string]string{
		"charsPerToken":    formatRatio(ratio),
		"calibrationModel": strconv.Quote(model),
		"calibrated":       "true",
	}
	done := map[string]bool{}
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}
		for name, val := range repl {
			if done[name] || !strings.HasPrefix(trimmed, name) {
				continue
			}
			if rest := strings.TrimSpace(trimmed[len(name):]); strings.HasPrefix(rest, "=") {
				indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
				lines[i] = indent + name + " = " + val
				done[name] = true
			}
		}
	}
	for name := range repl {
		if !done[name] {
			return fmt.Errorf("could not find const %s assignment in %s", name, path)
		}
	}
	formatted, err := format.Source([]byte(strings.Join(lines, "\n")))
	if err != nil {
		return fmt.Errorf("gofmt %s after edit: %w", path, err)
	}
	return os.WriteFile(path, formatted, 0644)
}

// ratioRow is one line of the calibration report.
type ratioRow struct {
	label         string
	chars, tokens int
}

// printRows prints the report with column widths computed from the values, so the
// label, chars, and tokens columns line up regardless of label length.
func printRows(rows []ratioRow) {
	labelW, charsW, tokenW := 0, 0, 0
	for _, r := range rows {
		labelW = max(labelW, len(r.label))
		charsW = max(charsW, len(strconv.Itoa(r.chars)))
		tokenW = max(tokenW, len(strconv.Itoa(r.tokens)))
	}
	for _, r := range rows {
		ratio := 0.0
		if r.tokens > 0 {
			ratio = float64(r.chars) / float64(r.tokens)
		}
		fmt.Printf("  %-*s  %*d chars  %*d tokens  ->  %s chars/token\n",
			labelW, r.label, charsW, r.chars, tokenW, r.tokens, formatRatio(ratio))
	}
}

// formatRatio renders a chars-per-token ratio with two decimals (e.g. "3.05").
func formatRatio(r float64) string {
	return strconv.FormatFloat(r, 'f', 2, 64)
}

func runeLen(s string) int { return len([]rune(s)) }

// normalizeLF collapses CRLF and lone CR to LF, matching internal/rules so the
// char counts line up with what the estimator measures.
func normalizeLF(s string) string {
	if !strings.Contains(s, "\r") {
		return s
	}
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
}
