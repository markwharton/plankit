package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// The front page shows pk running. At build time the site generator
// scaffolds a scratch repository, executes the real binary in it, and
// embeds the verbatim output. The demo can never describe behavior
// the binary does not have, because the binary produced it.

type demoStep struct {
	Title, Command, Output string
}

func runDemo(pk string) ([]demoStep, error) {
	if pk == "" {
		return nil, nil
	}
	pk, err := filepath.Abs(pk)
	if err != nil {
		return nil, err
	}
	root, err := os.MkdirTemp("", "plankit-demo-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(root)
	work := filepath.Join(root, "acme-app")
	bare := filepath.Join(root, "origin.git")

	git := func(args ...string) error {
		cmd := exec.Command("git", args...)
		cmd.Dir = work
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=dev", "GIT_AUTHOR_EMAIL=dev@example.com",
			"GIT_COMMITTER_NAME=dev", "GIT_COMMITTER_EMAIL=dev@example.com",
			"GIT_AUTHOR_DATE=2026-09-01T09:00:00+10:00", "GIT_COMMITTER_DATE=2026-09-01T09:00:00+10:00")
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git %v: %v\n%s", args, err, out)
		}
		return nil
	}
	commit := func(file, msg string) error {
		if err := os.WriteFile(filepath.Join(work, file), []byte(msg+"\n"), 0o644); err != nil {
			return err
		}
		if err := git("add", "."); err != nil {
			return err
		}
		return git("commit", "-q", "-m", msg)
	}
	run := func(title, shown string, stdin string, args ...string) (demoStep, error) {
		cmd := exec.Command(pk, append([]string{}, args...)...)
		cmd.Dir = work
		cmd.Env = append(os.Environ(), "NO_COLOR=1", "CLAUDE_PROJECT_DIR=")
		if stdin != "" {
			cmd.Stdin = strings.NewReader(stdin)
		}
		out, _ := cmd.CombinedOutput() // demo commands are allowed to exit non-zero
		text := strings.ReplaceAll(string(out), work, "~/acme-app")
		return demoStep{Title: title, Command: shown, Output: strings.TrimRight(text, "\n")}, nil
	}

	if err := os.MkdirAll(work, 0o755); err != nil {
		return nil, err
	}
	if out, err := exec.Command("git", "init", "-q", "--bare", "-b", "main", bare).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("bare init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"remote", "add", "origin", bare},
	} {
		if err := git(args...); err != nil {
			return nil, err
		}
	}
	if err := commit("README.md", "chore: scaffold acme-app"); err != nil {
		return nil, err
	}

	var steps []demoStep
	s, _ := run("Configure a repository", "pk init", "", "init")
	steps = append(steps, s)
	if err := git("add", "."); err != nil {
		return nil, err
	}
	if err := git("commit", "-q", "-m", "chore: adopt plankit"); err != nil {
		return nil, err
	}
	if err := git("push", "-q", "-u", "origin", "main", "--tags"); err != nil {
		return nil, err
	}
	if err := git("switch", "-q", "-c", "develop"); err != nil {
		return nil, err
	}
	for _, c := range []struct{ f, m string }{
		{"auth.go", "feat(auth): add passkey sign-in"},
		{"cache.go", "fix(cache): expire entries on clock skew"},
		{"docs.md", "docs: describe the passkey flow"},
	} {
		if err := commit(c.f, c.m); err != nil {
			return nil, err
		}
	}
	if err := git("push", "-q", "-u", "origin", "develop"); err != nil {
		return nil, err
	}

	s, _ = run("See the state", "pk status", "", "status")
	steps = append(steps, s)
	s, _ = run("What every session is told at start", "pk brief", "", "brief")
	steps = append(steps, s)
	s, _ = run("Preview the release", "pk changelog --dry-run", "", "changelog", "--dry-run")
	steps = append(steps, s)

	payload, _ := json.Marshal(map[string]any{
		"cwd":        work,
		"tool_input": map[string]string{"command": `git commit -m "feat!: drop the legacy session cookie"`},
	})
	s, _ = run("Guard asks before a breaking marker is committed",
		`Claude Code runs: git commit -m "feat!: drop the legacy session cookie"`, string(payload), "guard")
	// The hook answers in JSON; show it readable.
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, []byte(strings.TrimSpace(s.Output)), "", "  "); err == nil {
		s.Output = pretty.String()
	}
	steps = append(steps, s)
	return steps, nil
}

// demoHTML renders the steps as terminal panels.
func demoHTML(steps []demoStep) template.HTML {
	if len(steps) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<section class="demo"><h2>What it looks like</h2>`)
	b.WriteString(`<p class="muted">Generated when this page was built, by running pk in a scratch repository.</p>`)
	for _, s := range steps {
		b.WriteString(`<figure class="terminal"><figcaption>` + template.HTMLEscapeString(s.Title) + `</figcaption>`)
		b.WriteString(`<pre><code><span class="prompt">$ </span>` + template.HTMLEscapeString(s.Command) + "\n" +
			template.HTMLEscapeString(s.Output) + `</code></pre></figure>`)
	}
	b.WriteString(`</section>`)
	return template.HTML(b.String())
}
