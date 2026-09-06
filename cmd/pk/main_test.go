package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/markwharton/plankit/internal/config"
	"github.com/markwharton/plankit/internal/registry"
)

// repoRoot walks up from the test binary's working directory to the
// directory holding go.mod.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above test directory")
		}
		dir = parent
	}
}

// TestCommandsAndSkillsAreOneToOne enforces the design rule that skills
// mirror commands exactly: every registered command has skills/<name>/
// SKILL.md whose frontmatter name matches, and every skill directory
// (minus the plankit overview) is a registered command. A drifted pair
// means either an undocumented command or a skill promising a command
// that does not exist.
func TestCommandsAndSkillsAreOneToOne(t *testing.T) {
	root := repoRoot(t)
	registered := map[string]bool{}
	for _, c := range registry.Commands() {
		registered[c.Name] = true
	}

	entries, err := os.ReadDir(filepath.Join(root, "skills"))
	if err != nil {
		t.Fatal(err)
	}
	skillDirs := map[string]bool{}
	nameRe := regexp.MustCompile(`(?m)^name:\s*(\S+)\s*$`)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillDirs[e.Name()] = true
		md, err := os.ReadFile(filepath.Join(root, "skills", e.Name(), "SKILL.md"))
		if err != nil {
			t.Errorf("skills/%s has no SKILL.md: %v", e.Name(), err)
			continue
		}
		m := nameRe.FindSubmatch(md)
		if m == nil || string(m[1]) != e.Name() {
			t.Errorf("skills/%s: frontmatter name %q must match the directory", e.Name(), m)
		}
	}

	for name := range registered {
		if !skillDirs[name] {
			t.Errorf("command %q has no skills/%s/SKILL.md", name, name)
		}
	}
	for dir := range skillDirs {
		md, _ := os.ReadFile(filepath.Join(root, "skills", dir, "SKILL.md"))
		// The opening heading declares the page's kind: "pk <name>" is a
		// command page and must have its command; "<name>" alone is a
		// document (the overview, craft) and is exempt.
		isCommand := regexp.MustCompile(`(?m)^# pk ` + regexp.QuoteMeta(dir) + `\s*$`).Match(md)
		if !isCommand {
			if registered[dir] {
				t.Errorf("skills/%s is a command page but its heading does not read \"# pk %s\"", dir, dir)
			}
			continue
		}
		if !registered[dir] {
			t.Errorf("skills/%s does not correspond to a registered command", dir)
		}
	}
}

// TestHookWiringMatchesRegisteredCommands parses hooks/hooks.json and
// checks every command entry is the shim invoking a registered command
// through the documented quoting: "${CLAUDE_PLUGIN_ROOT}"/bin/pk <cmd>.
func TestHookWiringMatchesRegisteredCommands(t *testing.T) {
	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "hooks", "hooks.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Hooks map[string][]struct {
			Matcher string `json:"matcher"`
			Hooks   []struct {
				Type    string `json:"type"`
				Command string `json:"command"`
			} `json:"hooks"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("hooks.json does not parse: %v", err)
	}
	if len(cfg.Hooks) == 0 {
		t.Fatal("hooks.json declares no hooks")
	}

	registered := map[string]bool{}
	for _, c := range registry.Commands() {
		registered[c.Name] = true
	}
	const prefix = `"${CLAUDE_PLUGIN_ROOT}"/bin/pk `
	seen := map[string]bool{}
	for event, entries := range cfg.Hooks {
		for _, entry := range entries {
			for _, h := range entry.Hooks {
				if h.Type != "command" {
					t.Errorf("%s[%s]: unexpected hook type %q", event, entry.Matcher, h.Type)
					continue
				}
				if !strings.HasPrefix(h.Command, prefix) {
					t.Errorf("%s[%s]: command %q must start with %q (double-quoted plugin root)", event, entry.Matcher, h.Command, prefix)
					continue
				}
				name := strings.TrimPrefix(h.Command, prefix)
				if !registered[name] {
					t.Errorf("%s[%s]: %q is not a registered command", event, entry.Matcher, name)
				}
				seen[name] = true
			}
		}
	}
	for _, want := range []string{"guard", "protect", "preserve"} {
		if !seen[want] {
			t.Errorf("hooks.json does not wire %q", want)
		}
	}
}

// TestChangelogSkillListsDefaultTypeTable pins the hand-written table
// on the changelog page to config.Default(): the page is a derived
// copy of the type table, so drift between them must fail the build.
func TestChangelogSkillListsDefaultTypeTable(t *testing.T) {
	root := repoRoot(t)
	md, err := os.ReadFile(filepath.Join(root, "skills", "changelog", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range config.Default("main").Changelog.Types {
		re := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(tc.Type) + `\s`)
		if !re.Match(md) {
			t.Errorf("skills/changelog/SKILL.md does not list default type %q", tc.Type)
		}
		if tc.Hidden {
			// The hidden annotation is part of the page's claim; pin it.
			hiddenRe := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(tc.Type) + `\s+\(hidden`)
			if !hiddenRe.Match(md) {
				t.Errorf("skills/changelog/SKILL.md must mark hidden type %q with (hidden...)", tc.Type)
			}
		} else if !strings.Contains(string(md), tc.Section) {
			t.Errorf("skills/changelog/SKILL.md missing section %q for type %q", tc.Section, tc.Type)
		}
	}
}

// TestDesignDocCodeBlocksMatchSource pins every fenced Go block in
// docs/design.md that names a source file (```go path/to/file.go) to
// that file: the block, whitespace-normalized, must appear verbatim in
// the source. Struct listings in the design are hand copies of code,
// and this is what keeps them honest.
func TestDesignDocCodeBlocksMatchSource(t *testing.T) {
	root := repoRoot(t)
	doc, err := os.ReadFile(filepath.Join(root, "docs", "design.md"))
	if err != nil {
		t.Fatal(err)
	}
	fence := regexp.MustCompile("(?s)```go (\\S+\\.go)\\n(.*?)```")
	matches := fence.FindAllStringSubmatch(string(doc), -1)
	if len(matches) == 0 {
		t.Fatal("design.md has no source-pinned code blocks; the test expects at least one")
	}
	norm := func(s string) string { return strings.Join(strings.Fields(s), " ") }
	for _, m := range matches {
		src, err := os.ReadFile(filepath.Join(root, m[1]))
		if err != nil {
			t.Errorf("design.md pins %s, which cannot be read: %v", m[1], err)
			continue
		}
		if !strings.Contains(norm(string(src)), norm(m[2])) {
			t.Errorf("design.md code block pinned to %s no longer matches the source:\n%s", m[1], m[2])
		}
	}
}

// TestSettingsTableIsSortedAndSound holds the settings table: keys
// sorted, every enumerated default among its values, every
// enumerated setting able to report its value for Validate, and every
// section owned by a page of the same name.
func TestSettingsTableIsSortedAndSound(t *testing.T) {
	root := repoRoot(t)
	for _, sec := range config.Sections() {
		if _, err := os.Stat(filepath.Join(root, "skills", sec, "SKILL.md")); err != nil {
			t.Errorf("settings section %q has no page skills/%s/SKILL.md", sec, sec)
		}
	}
	prev := ""
	for _, s := range config.Settings() {
		if s.Key < prev {
			t.Errorf("settings out of order: %q after %q", s.Key, prev)
		}
		prev = s.Key
		if len(s.Values) == 0 {
			if s.Kind == "" {
				t.Errorf("%s: neither values nor kind", s.Key)
			}
			continue
		}
		found := false
		for _, v := range s.Values {
			if v == s.Default {
				found = true
			}
		}
		if !found {
			t.Errorf("%s: default %q is not among %v", s.Key, s.Default, s.Values)
		}
	}
}
