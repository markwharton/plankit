package changelog

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/markwharton/plankit/internal/version"
)

func TestLoadConfig(t *testing.T) {
	t.Run("full config", func(t *testing.T) {
		cfg, err := LoadConfig(func(name string) ([]byte, error) {
			return []byte(`{"changelog":{
				"types": [{"type":"feat","section":"Features"}],
				"versionFiles": [{"path":"package.json","type":"json"}],
				"hooks": {"postVersion":"echo done","preCommit":"echo pre"}
			}}`), nil
		})
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}
		if len(cfg.Types) != 1 || cfg.Types[0].Section != "Features" {
			t.Errorf("types = %v, want Features", cfg.Types)
		}
		if len(cfg.VersionFiles) != 1 || cfg.VersionFiles[0].Path != "package.json" {
			t.Errorf("versionFiles = %v, want package.json", cfg.VersionFiles)
		}
		if cfg.Hooks.PostVersion != "echo done" {
			t.Errorf("hooks.postVersion = %q, want echo done", cfg.Hooks.PostVersion)
		}
		if cfg.Hooks.PreCommit != "echo pre" {
			t.Errorf("hooks.preCommit = %q, want echo pre", cfg.Hooks.PreCommit)
		}
	})

	t.Run("types only", func(t *testing.T) {
		cfg, err := LoadConfig(func(name string) ([]byte, error) {
			return []byte(`{"changelog":{"types":[{"type":"fix","section":"Fixed"}]}}`), nil
		})
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}
		if len(cfg.Types) != 1 || cfg.Types[0].Type != "fix" {
			t.Errorf("types = %v, want fix", cfg.Types)
		}
	})

	t.Run("missing file returns defaults", func(t *testing.T) {
		cfg, err := LoadConfig(func(name string) ([]byte, error) {
			return nil, os.ErrNotExist
		})
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}
		if len(cfg.Types) != len(defaultTypes) {
			t.Errorf("types count = %d, want %d", len(cfg.Types), len(defaultTypes))
		}
	})

	t.Run("malformed JSON returns error", func(t *testing.T) {
		_, err := LoadConfig(func(name string) ([]byte, error) {
			return []byte(`{not json}`), nil
		})
		if err == nil {
			t.Error("expected error for malformed JSON")
		}
	})

	t.Run("empty types uses defaults", func(t *testing.T) {
		cfg, err := LoadConfig(func(name string) ([]byte, error) {
			return []byte(`{"changelog":{"types":[]}}`), nil
		})
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}
		if len(cfg.Types) != len(defaultTypes) {
			t.Errorf("types count = %d, want %d", len(cfg.Types), len(defaultTypes))
		}
	})
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		tag    string
		want   version.Semver
		wantOK bool
	}{
		{"v1.2.3", version.Semver{Major: 1, Minor: 2, Patch: 3}, true},
		{"v0.0.0", version.Semver{}, true},
		{"1.2.3", version.Semver{Major: 1, Minor: 2, Patch: 3}, true},
		{"v10.20.30", version.Semver{Major: 10, Minor: 20, Patch: 30}, true},
		{"invalid", version.Semver{}, false},
		{"v1.2", version.Semver{}, false},
		{"v1.2.x", version.Semver{}, false},
		{"", version.Semver{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			got, ok := version.ParseSemver(tt.tag)
			if ok != tt.wantOK || got != tt.want {
				t.Errorf("ParseSemver(%q) = %v, %v; want %v, %v", tt.tag, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestFormatVersion(t *testing.T) {
	tests := []struct {
		v    version.Semver
		want string
	}{
		{version.Semver{Major: 1, Minor: 2, Patch: 3}, "v1.2.3"},
		{version.Semver{}, "v0.0.0"},
		{version.Semver{Major: 10}, "v10.0.0"},
	}
	for _, tt := range tests {
		if got := tt.v.String(); got != tt.want {
			t.Errorf("Semver%v.String() = %q, want %q", tt.v, got, tt.want)
		}
	}
}

func TestParseCommit(t *testing.T) {
	tests := []struct {
		name    string
		hash    string
		subject string
		body    string
		want    Commit
		wantOK  bool
	}{
		{
			"feat",
			"abc1234", "feat: add new feature", "",
			Commit{Hash: "abc1234", Type: "feat", Message: "add new feature"},
			true,
		},
		{
			"fix with scope",
			"def5678", "fix(auth): handle nil token", "",
			Commit{Hash: "def5678", Type: "fix", Scope: "auth", Message: "handle nil token"},
			true,
		},
		{
			"breaking via bang",
			"ghi9012", "feat!: breaking API change", "",
			Commit{Hash: "ghi9012", Type: "feat", Breaking: true, Message: "breaking API change"},
			true,
		},
		{
			"breaking via scope and bang",
			"jkl3456", "feat(api)!: remove endpoint", "",
			Commit{Hash: "jkl3456", Type: "feat", Scope: "api", Breaking: true, Message: "remove endpoint"},
			true,
		},
		{
			"breaking via BREAKING CHANGE trailer",
			"mno7890", "feat: add new API", "BREAKING CHANGE: old API removed",
			Commit{Hash: "mno7890", Type: "feat", Breaking: true, Message: "add new API"},
			true,
		},
		{
			"breaking via BREAKING-CHANGE trailer",
			"pqr1234", "fix: update config", "BREAKING-CHANGE: config format changed",
			Commit{Hash: "pqr1234", Type: "fix", Breaking: true, Message: "update config"},
			true,
		},
		{
			"docs type",
			"stu5678", "docs: update README", "",
			Commit{Hash: "stu5678", Type: "docs", Message: "update README"},
			true,
		},
		{
			"non-conventional",
			"vwx9012", "Merge branch 'main'", "",
			Commit{},
			false,
		},
		{
			"random message",
			"yza3456", "random commit message", "",
			Commit{},
			false,
		},
		{
			"empty subject",
			"bcd6789", "", "",
			Commit{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseCommit(tt.hash, tt.subject, tt.body)
			if ok != tt.wantOK {
				t.Errorf("parseCommit ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && (got.Hash != tt.want.Hash || got.Type != tt.want.Type || got.Scope != tt.want.Scope ||
				got.Message != tt.want.Message || got.Breaking != tt.want.Breaking) {
				t.Errorf("parseCommit = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestDetectBump(t *testing.T) {
	tests := []struct {
		name    string
		commits []Commit
		want    int
	}{
		{"all fix", []Commit{{Type: "fix"}, {Type: "fix"}}, version.BumpPatch},
		{"has feat", []Commit{{Type: "fix"}, {Type: "feat"}}, version.BumpMinor},
		{"has breaking", []Commit{{Type: "fix"}, {Type: "feat", Breaking: true}}, version.BumpMajor},
		{"docs and chore", []Commit{{Type: "docs"}, {Type: "chore"}}, version.BumpPatch},
		{"single feat", []Commit{{Type: "feat"}}, version.BumpMinor},
		{"empty", []Commit{}, version.BumpPatch},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := detectBump(tt.commits); got != tt.want {
				t.Errorf("detectBump = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestResolveBump(t *testing.T) {
	commits := []Commit{{Type: "feat"}}

	t.Run("auto detect", func(t *testing.T) {
		got, err := resolveBump("", commits)
		if err != nil || got != version.BumpMinor {
			t.Errorf("resolveBump empty = %d, %v; want minor", got, err)
		}
	})

	t.Run("override major", func(t *testing.T) {
		got, err := resolveBump("major", commits)
		if err != nil || got != version.BumpMajor {
			t.Errorf("resolveBump major = %d, %v", got, err)
		}
	})

	t.Run("override minor", func(t *testing.T) {
		got, err := resolveBump("minor", commits)
		if err != nil || got != version.BumpMinor {
			t.Errorf("resolveBump minor = %d, %v", got, err)
		}
	})

	t.Run("override patch", func(t *testing.T) {
		got, err := resolveBump("patch", commits)
		if err != nil || got != version.BumpPatch {
			t.Errorf("resolveBump patch = %d, %v", got, err)
		}
	})

	t.Run("invalid flag", func(t *testing.T) {
		_, err := resolveBump("invalid", commits)
		if err == nil {
			t.Error("resolveBump invalid should error")
		}
	})
}

func TestGroupCommits(t *testing.T) {
	types := []TypeConfig{
		{Type: "feat", Section: "Added"},
		{Type: "fix", Section: "Fixed"},
		{Type: "docs", Hidden: true},
		{Type: "refactor", Section: "Changed"},
	}

	commits := []Commit{
		{Type: "fix", Message: "bug1"},
		{Type: "feat", Message: "feature1"},
		{Type: "docs", Message: "readme"},
		{Type: "refactor", Message: "cleanup"},
		{Type: "feat", Message: "feature2"},
		{Type: "unknown", Message: "skip me"},
	}

	groups := groupCommits(commits, types)

	if len(groups) != 3 {
		t.Fatalf("groups count = %d, want 3", len(groups))
	}
	if groups[0].Heading != "Added" || len(groups[0].Items) != 2 {
		t.Errorf("group 0 = %s (%d items), want Added (2)", groups[0].Heading, len(groups[0].Items))
	}
	if groups[1].Heading != "Fixed" || len(groups[1].Items) != 1 {
		t.Errorf("group 1 = %s (%d items), want Fixed (1)", groups[1].Heading, len(groups[1].Items))
	}
	if groups[2].Heading != "Changed" || len(groups[2].Items) != 1 {
		t.Errorf("group 2 = %s (%d items), want Changed (1)", groups[2].Heading, len(groups[2].Items))
	}
}

func TestGroupCommits_sectionOrdering(t *testing.T) {
	// Two types map to same section — only appears once.
	types := []TypeConfig{
		{Type: "refactor", Section: "Changed"},
		{Type: "perf", Section: "Changed"},
		{Type: "feat", Section: "Added"},
	}
	commits := []Commit{
		{Type: "perf", Message: "speed"},
		{Type: "refactor", Message: "cleanup"},
		{Type: "feat", Message: "new"},
	}
	groups := groupCommits(commits, types)
	if len(groups) != 2 {
		t.Fatalf("groups count = %d, want 2", len(groups))
	}
	if groups[0].Heading != "Changed" || len(groups[0].Items) != 2 {
		t.Errorf("group 0 = %s (%d items), want Changed (2)", groups[0].Heading, len(groups[0].Items))
	}
	if groups[1].Heading != "Added" {
		t.Errorf("group 1 = %s, want Added", groups[1].Heading)
	}
}

func TestFormatSection(t *testing.T) {
	groups := []CommitGroup{
		{Heading: "Added", Items: []Commit{
			{Hash: "abc1234", Message: "new feature"},
		}},
		{Heading: "Fixed", Items: []Commit{
			{Hash: "def5678", Message: "bug fix"},
			{Hash: "ghi9012", Message: "breaking change", Breaking: true},
		}},
	}

	got := formatSection("v1.0.0", "2026-04-03", groups, false)

	if !strings.Contains(got, "## [v1.0.0] - 2026-04-03") {
		t.Error("missing version header")
	}
	if !strings.Contains(got, "### Added") {
		t.Error("missing Added heading")
	}
	if !strings.Contains(got, "- new feature (abc1234)") {
		t.Error("missing feature entry")
	}
	if !strings.Contains(got, "### Fixed") {
		t.Error("missing Fixed heading")
	}
	if !strings.Contains(got, "- **BREAKING:** breaking change (ghi9012)") {
		t.Error("missing breaking change entry")
	}
}

func TestFormatSection_scopedCommits(t *testing.T) {
	groups := []CommitGroup{
		{Heading: "Fixed", Items: []Commit{
			{Hash: "dab3f6d", Scope: "flow", Message: "resolve Object-in-String-Context pattern"},
			{Hash: "abc1234", Scope: "auth", Message: "handle nil token", Breaking: true},
			{Hash: "def5678", Message: "typo in error message"},
		}},
	}

	t.Run("showScope false omits scope", func(t *testing.T) {
		got := formatSection("v1.1.0", "2026-04-09", groups, false)

		if !strings.Contains(got, "- resolve Object-in-String-Context pattern (dab3f6d)") {
			t.Errorf("scoped commit missing or malformed: %s", got)
		}
		if !strings.Contains(got, "- **BREAKING:** handle nil token (abc1234)") {
			t.Errorf("scoped breaking commit missing or malformed: %s", got)
		}
		if !strings.Contains(got, "- typo in error message (def5678)") {
			t.Errorf("unscoped commit missing: %s", got)
		}
		if strings.Contains(got, "**flow:**") || strings.Contains(got, "**auth:**") {
			t.Errorf("scope should not appear when showScope is false: %s", got)
		}
	})

	t.Run("showScope true includes scope prefix", func(t *testing.T) {
		got := formatSection("v1.1.0", "2026-04-09", groups, true)

		if !strings.Contains(got, "- **flow:** resolve Object-in-String-Context pattern (dab3f6d)") {
			t.Errorf("scoped commit missing scope prefix: %s", got)
		}
		if !strings.Contains(got, "- **BREAKING:** **auth:** handle nil token (abc1234)") {
			t.Errorf("breaking+scoped commit wrong format: %s", got)
		}
		// Unscoped commit should not get a scope prefix.
		if !strings.Contains(got, "- typo in error message (def5678)") {
			t.Errorf("unscoped commit missing: %s", got)
		}
	})
}

func TestInsertSection(t *testing.T) {
	section := "## [v1.0.0] - 2026-04-03\n\n### Added\n\n- feature (abc1234)\n"

	t.Run("empty file", func(t *testing.T) {
		got := insertSection("", section)
		if !strings.HasPrefix(got, "# Changelog") {
			t.Error("missing header")
		}
		if !strings.Contains(got, section) {
			t.Error("missing section")
		}
	})

	t.Run("existing with versions", func(t *testing.T) {
		existing := changelogHeader + "\n## [v0.1.0] - 2026-03-01\n\n### Added\n\n- old feature\n"
		got := insertSection(existing, section)
		v1Idx := strings.Index(got, "## [v1.0.0]")
		v01Idx := strings.Index(got, "## [v0.1.0]")
		if v1Idx < 0 || v01Idx < 0 || v1Idx >= v01Idx {
			t.Error("new section should appear before old section")
		}
	})

	t.Run("header only", func(t *testing.T) {
		got := insertSection(changelogHeader, section)
		if !strings.Contains(got, section) {
			t.Error("missing section")
		}
	})
}

func TestParseLog(t *testing.T) {
	t.Run("normal output", func(t *testing.T) {
		output := "abc1234\x00feat: add feature\x00\x00def5678\x00fix: bug fix\x00\x00"
		commits := parseLog(output)
		if len(commits) != 2 {
			t.Fatalf("commits = %d, want 2", len(commits))
		}
		if commits[0].Type != "feat" || commits[1].Type != "fix" {
			t.Errorf("types = %s, %s", commits[0].Type, commits[1].Type)
		}
	})

	t.Run("with body", func(t *testing.T) {
		output := "abc1234\x00feat: new API\x00BREAKING CHANGE: old API removed\x00"
		commits := parseLog(output)
		if len(commits) != 1 || !commits[0].Breaking {
			t.Error("expected 1 breaking commit")
		}
	})

	t.Run("empty output", func(t *testing.T) {
		commits := parseLog("")
		if len(commits) != 0 {
			t.Error("expected no commits")
		}
	})

	t.Run("non-conventional skipped", func(t *testing.T) {
		output := "abc1234\x00Merge branch main\x00\x00def5678\x00feat: feature\x00\x00"
		commits := parseLog(output)
		if len(commits) != 1 || commits[0].Type != "feat" {
			t.Errorf("commits = %+v, want 1 feat", commits)
		}
	})
}

func TestAppendRefLink(t *testing.T) {
	t.Run("to content with trailing newline", func(t *testing.T) {
		got := appendRefLink("# Changelog\n\n## [v0.1.0]\n", "[v0.1.0]: https://example.com")
		if !strings.HasSuffix(got, "[v0.1.0]: https://example.com\n") {
			t.Errorf("got %q", got)
		}
		// First ref link should have double newline separator from content.
		if !strings.Contains(got, "## [v0.1.0]\n\n[v0.1.0]:") {
			t.Errorf("expected double newline before first ref link, got %q", got)
		}
	})

	t.Run("to content without trailing newline", func(t *testing.T) {
		got := appendRefLink("# Changelog", "[v0.1.0]: https://example.com")
		if !strings.Contains(got, "# Changelog\n\n[v0.1.0]") {
			t.Errorf("got %q", got)
		}
	})

	t.Run("after existing ref link uses single newline", func(t *testing.T) {
		existing := "# Changelog\n\n[v0.1.0]: https://example.com/v0.1.0\n"
		got := appendRefLink(existing, "[v0.2.0]: https://example.com/v0.2.0")
		want := "[v0.1.0]: https://example.com/v0.1.0\n[v0.2.0]: https://example.com/v0.2.0\n"
		if !strings.HasSuffix(got, want) {
			t.Errorf("got %q, want suffix %q", got, want)
		}
	})

	t.Run("duplicate ref link skipped", func(t *testing.T) {
		existing := "# Changelog\n\n[v0.1.0]: https://example.com/v0.1.0\n"
		got := appendRefLink(existing, "[v0.1.0]: https://example.com/v0.1.0")
		if got != existing {
			t.Errorf("expected no change, got %q", got)
		}
	})
}

func TestSpliceJSONVersion(t *testing.T) {
	t.Run("package.json style", func(t *testing.T) {
		input := []byte(`{
  "name": "my-app",
  "version": "1.2.3",
  "description": "test"
}
`)
		got, err := spliceJSONVersion(input, "2.0.0")
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(got, []byte(`"version": "2.0.0"`)) {
			t.Errorf("version not updated: %s", got)
		}
		// Verify formatting preserved.
		if !bytes.Contains(got, []byte(`"name": "my-app"`)) {
			t.Error("name field lost")
		}
		if !bytes.Contains(got, []byte(`"description": "test"`)) {
			t.Error("description field lost")
		}
	})

	t.Run("version length change", func(t *testing.T) {
		input := []byte(`{"version": "9.9.9"}`)
		got, err := spliceJSONVersion(input, "10.0.0")
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(got, []byte(`"10.0.0"`)) {
			t.Errorf("version not updated: %s", got)
		}
	})

	t.Run("preserves key order", func(t *testing.T) {
		input := []byte(`{"z": 1, "version": "1.0.0", "a": 2}`)
		got, err := spliceJSONVersion(input, "2.0.0")
		if err != nil {
			t.Fatal(err)
		}
		zIdx := bytes.Index(got, []byte(`"z"`))
		aIdx := bytes.Index(got, []byte(`"a"`))
		if zIdx >= aIdx {
			t.Errorf("key order not preserved: z at %d, a at %d", zIdx, aIdx)
		}
	})

	t.Run("no version field", func(t *testing.T) {
		input := []byte(`{"name": "test"}`)
		_, err := spliceJSONVersion(input, "1.0.0")
		if err == nil {
			t.Error("expected error for missing version")
		}
	})

	t.Run("not JSON object", func(t *testing.T) {
		input := []byte(`[1, 2, 3]`)
		_, err := spliceJSONVersion(input, "1.0.0")
		if err == nil {
			t.Error("expected error for non-object")
		}
	})
}

// --- Integration tests for Run ---

func fixedTime() time.Time {
	return time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)
}

func TestRun_notAGitRepo(t *testing.T) {
	var stderr bytes.Buffer
	cfg := Config{
		Stderr: &stderr,
		GitExec: func(dir string, args ...string) (string, error) {
			if args[0] == "rev-parse" {
				return "", fmt.Errorf("fatal: not a git repository")
			}
			return "", nil
		},
		ReadFile: func(name string) ([]byte, error) { return nil, os.ErrNotExist },
		Now:      fixedTime,
	}
	code := Run(cfg)
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "not a git repository") {
		t.Errorf("stderr = %q, want 'not a git repository' message", stderr.String())
	}
	if strings.Contains(stderr.String(), "failed to list tags") {
		t.Errorf("stderr = %q, should not surface the raw git-tag error when the repo check catches it first", stderr.String())
	}
}

func TestRun_noTags(t *testing.T) {
	var stderr bytes.Buffer
	cfg := Config{
		Stderr: &stderr,
		GitExec: func(dir string, args ...string) (string, error) {
			if args[0] == "tag" {
				return "", nil // no tags
			}
			if args[0] == "ls-remote" {
				return "", nil // origin has no tags either
			}
			return "", nil
		},
		ReadFile: func(name string) ([]byte, error) { return nil, os.ErrNotExist },
		Now:      fixedTime,
	}
	code := Run(cfg)
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "no version tags found") {
		t.Errorf("stderr = %q, want no version tags message", stderr.String())
	}
	if !strings.Contains(stderr.String(), "pk setup --baseline") {
		t.Errorf("stderr = %q, want baseline advice when origin has no tags either", stderr.String())
	}
}

func TestRun_noLocalTagsButOriginHas(t *testing.T) {
	var stderr bytes.Buffer
	cfg := Config{
		Stderr: &stderr,
		GitExec: func(dir string, args ...string) (string, error) {
			if args[0] == "tag" {
				return "", nil // no local tags
			}
			if args[0] == "ls-remote" {
				return "abc123\trefs/tags/v0.1.0\ndef456\trefs/tags/v0.2.0\n", nil
			}
			return "", nil
		},
		ReadFile: func(name string) ([]byte, error) { return nil, os.ErrNotExist },
		Now:      fixedTime,
	}
	code := Run(cfg)
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "no version tags found locally") {
		t.Errorf("stderr = %q, want 'locally' qualifier", stderr.String())
	}
	if !strings.Contains(stderr.String(), "git fetch --tags") {
		t.Errorf("stderr = %q, want fetch advice", stderr.String())
	}
	if strings.Contains(stderr.String(), "pk setup --baseline") {
		t.Errorf("stderr = %q, should not show baseline advice when origin has tags", stderr.String())
	}
}

func TestRun_firstRelease(t *testing.T) {
	var stderr bytes.Buffer
	var gitCalls []string
	var writtenFile string
	var writtenContent []byte

	cfg := Config{
		Stderr: &stderr,
		GitExec: func(dir string, args ...string) (string, error) {
			call := strings.Join(args, " ")
			gitCalls = append(gitCalls, call)
			if args[0] == "tag" && args[1] == "--list" {
				return "v0.0.0", nil
			}
			if args[0] == "log" {
				return "abc1234\x00feat: add feature\x00\x00def5678\x00fix: fix bug\x00\x00", nil
			}
			if args[0] == "remote" {
				return "git@github.com:owner/repo.git", nil
			}
			return "", nil
		},
		ReadFile: func(name string) ([]byte, error) {
			if name == ".pk.json" {
				return nil, os.ErrNotExist
			}
			return nil, os.ErrNotExist // no existing CHANGELOG.md
		},
		WriteFile: func(name string, data []byte, perm os.FileMode) error {
			writtenFile = name
			writtenContent = data
			return nil
		},
		RunScript: func(command string, env map[string]string) error { return nil },
		Now:       fixedTime,
	}

	code := Run(cfg)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", code, stderr.String())
	}

	if writtenFile != "CHANGELOG.md" {
		t.Errorf("written file = %q, want CHANGELOG.md", writtenFile)
	}
	content := string(writtenContent)
	if !strings.Contains(content, "# Changelog") {
		t.Error("missing changelog header")
	}
	if !strings.Contains(content, "## [v0.1.0] - 2026-04-03") {
		t.Error("missing version section")
	}
	if !strings.Contains(content, "add feature (abc1234)") {
		t.Error("missing feature entry")
	}
	if !strings.Contains(content, "fix bug (def5678)") {
		t.Error("missing fix entry")
	}
	if !strings.Contains(content, "[v0.1.0]: https://github.com/owner/repo/compare/v0.0.0...v0.1.0") {
		t.Error("missing comparison link")
	}

	// Verify git operations: commit carries Release-Tag trailer, no git tag created.
	hasCommit := false
	hasTag := false
	for _, call := range gitCalls {
		if strings.HasPrefix(call, "commit -m chore: release v0.1.0") &&
			strings.Contains(call, "--trailer Release-Tag: v0.1.0") {
			hasCommit = true
		}
		if call == "tag v0.1.0" {
			hasTag = true
		}
	}
	if !hasCommit {
		t.Errorf("missing git commit with Release-Tag trailer; calls: %v", gitCalls)
	}
	if hasTag {
		t.Error("pk changelog should not create a git tag (that's pk release's job)")
	}
	if !strings.Contains(stderr.String(), "Committed v0.1.0") {
		t.Errorf("missing committed message; stderr: %s", stderr.String())
	}
}

func TestRun_noNewCommits(t *testing.T) {
	var stderr bytes.Buffer
	cfg := Config{
		Stderr: &stderr,
		GitExec: func(dir string, args ...string) (string, error) {
			if args[0] == "tag" && args[1] == "--list" {
				return "v1.0.0", nil
			}
			if args[0] == "log" {
				return "", nil
			}
			return "", nil
		},
		ReadFile: func(name string) ([]byte, error) { return nil, os.ErrNotExist },
		Now:      fixedTime,
	}

	code := Run(cfg)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stderr.String(), "No new conventional commits") {
		t.Errorf("stderr = %q, want no commits message", stderr.String())
	}
}

func TestRun_dryRun(t *testing.T) {
	var stderr bytes.Buffer
	writeFileCalled := false

	cfg := Config{
		Stderr: &stderr,
		GitExec: func(dir string, args ...string) (string, error) {
			if args[0] == "tag" && args[1] == "--list" {
				return "v1.0.0", nil
			}
			if args[0] == "log" {
				return "abc1234\x00feat: new feature\x00\x00", nil
			}
			return "", nil
		},
		ReadFile: func(name string) ([]byte, error) { return nil, os.ErrNotExist },
		WriteFile: func(name string, data []byte, perm os.FileMode) error {
			writeFileCalled = true
			return nil
		},
		RunScript: func(command string, env map[string]string) error {
			t.Error("RunScript should not be called in dry run")
			return nil
		},
		Now:    fixedTime,
		DryRun: true,
	}

	code := Run(cfg)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if writeFileCalled {
		t.Error("WriteFile should not be called in dry run")
	}
	if !strings.Contains(stderr.String(), "## [v1.1.0]") {
		t.Errorf("stderr should contain section preview, got: %s", stderr.String())
	}
}

func TestRun_bumpOverride(t *testing.T) {
	var stderr bytes.Buffer
	var gitCalls []string

	cfg := Config{
		Stderr: &stderr,
		GitExec: func(dir string, args ...string) (string, error) {
			gitCalls = append(gitCalls, strings.Join(args, " "))
			if args[0] == "tag" && args[1] == "--list" {
				return "v1.0.0", nil
			}
			if args[0] == "log" {
				return "abc1234\x00fix: small fix\x00\x00", nil
			}
			return "", nil
		},
		ReadFile:  func(name string) ([]byte, error) { return nil, os.ErrNotExist },
		WriteFile: func(name string, data []byte, perm os.FileMode) error { return nil },
		RunScript: func(command string, env map[string]string) error { return nil },
		Now:       fixedTime,
		Bump:      "major",
	}

	code := Run(cfg)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	hasCommit := false
	for _, call := range gitCalls {
		if strings.Contains(call, "commit -m chore: release v2.0.0") &&
			strings.Contains(call, "--trailer Release-Tag: v2.0.0") {
			hasCommit = true
		}
	}
	if !hasCommit {
		t.Errorf("expected commit with Release-Tag: v2.0.0 trailer, git calls: %v", gitCalls)
	}
}

func TestRun_breakingViaBang(t *testing.T) {
	var stderr bytes.Buffer
	var gitCalls []string

	cfg := Config{
		Stderr: &stderr,
		GitExec: func(dir string, args ...string) (string, error) {
			gitCalls = append(gitCalls, strings.Join(args, " "))
			if args[0] == "tag" && args[1] == "--list" {
				return "v1.0.0", nil
			}
			if args[0] == "log" {
				return "abc1234\x00feat!: new breaking API\x00\x00", nil
			}
			return "", nil
		},
		ReadFile:  func(name string) ([]byte, error) { return nil, os.ErrNotExist },
		WriteFile: func(name string, data []byte, perm os.FileMode) error { return nil },
		RunScript: func(command string, env map[string]string) error { return nil },
		Now:       fixedTime,
	}

	code := Run(cfg)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	hasCommit := false
	for _, call := range gitCalls {
		if strings.Contains(call, "commit -m chore: release v2.0.0") &&
			strings.Contains(call, "--trailer Release-Tag: v2.0.0") {
			hasCommit = true
		}
	}
	if !hasCommit {
		t.Errorf("expected major bump to v2.0.0, git calls: %v", gitCalls)
	}
}

func TestRun_breakingViaTrailer(t *testing.T) {
	var stderr bytes.Buffer
	var gitCalls []string

	cfg := Config{
		Stderr: &stderr,
		GitExec: func(dir string, args ...string) (string, error) {
			gitCalls = append(gitCalls, strings.Join(args, " "))
			if args[0] == "tag" && args[1] == "--list" {
				return "v1.0.0", nil
			}
			if args[0] == "log" {
				return "abc1234\x00feat: add new feature\x00BREAKING CHANGE: old feature removed\x00", nil
			}
			return "", nil
		},
		ReadFile:  func(name string) ([]byte, error) { return nil, os.ErrNotExist },
		WriteFile: func(name string, data []byte, perm os.FileMode) error { return nil },
		RunScript: func(command string, env map[string]string) error { return nil },
		Now:       fixedTime,
	}

	code := Run(cfg)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	hasCommit := false
	for _, call := range gitCalls {
		if strings.Contains(call, "commit -m chore: release v2.0.0") &&
			strings.Contains(call, "--trailer Release-Tag: v2.0.0") {
			hasCommit = true
		}
	}
	if !hasCommit {
		t.Errorf("expected major bump from BREAKING CHANGE trailer, git calls: %v", gitCalls)
	}
}

func TestRun_customConfigHiddenTypes(t *testing.T) {
	var stderr bytes.Buffer
	var writtenContent []byte

	cfg := Config{
		Stderr: &stderr,
		GitExec: func(dir string, args ...string) (string, error) {
			if args[0] == "tag" && args[1] == "--list" {
				return "v0.0.0", nil
			}
			if args[0] == "log" {
				return "abc1234\x00feat: visible\x00\x00def5678\x00docs: hidden\x00\x00", nil
			}
			return "", nil
		},
		ReadFile: func(name string) ([]byte, error) {
			if name == ".pk.json" {
				return []byte(`{"changelog":{"types":[{"type":"feat","section":"Added"},{"type":"docs","hidden":true}]}}`), nil
			}
			return nil, os.ErrNotExist
		},
		WriteFile: func(name string, data []byte, perm os.FileMode) error {
			writtenContent = data
			return nil
		},
		RunScript: func(command string, env map[string]string) error { return nil },
		Now:       fixedTime,
	}

	code := Run(cfg)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	content := string(writtenContent)
	if !strings.Contains(content, "visible") {
		t.Error("visible commit should be in changelog")
	}
	if strings.Contains(content, "hidden") {
		t.Error("hidden docs commit should not be in changelog")
	}
}

func TestRun_versionFiles(t *testing.T) {
	var stderr bytes.Buffer
	files := map[string][]byte{
		".pk.json":     []byte(`{"changelog":{"versionFiles":[{"path":"package.json","type":"json"}]}}`),
		"package.json": []byte(`{"name":"test","version":"0.0.0"}`),
	}
	var updatedPkg []byte

	cfg := Config{
		Stderr: &stderr,
		GitExec: func(dir string, args ...string) (string, error) {
			if args[0] == "tag" && args[1] == "--list" {
				return "v0.0.0", nil
			}
			if args[0] == "log" {
				return "abc1234\x00feat: add feature\x00\x00", nil
			}
			return "", nil
		},
		ReadFile: func(name string) ([]byte, error) {
			if data, ok := files[name]; ok {
				return data, nil
			}
			return nil, os.ErrNotExist
		},
		WriteFile: func(name string, data []byte, perm os.FileMode) error {
			if name == "package.json" {
				updatedPkg = data
			}
			return nil
		},
		RunScript: func(command string, env map[string]string) error { return nil },
		Now:       fixedTime,
	}

	code := Run(cfg)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", code, stderr.String())
	}

	if !bytes.Contains(updatedPkg, []byte(`"0.1.0"`)) {
		t.Errorf("package.json version not updated: %s", updatedPkg)
	}
}

func TestRun_hooks(t *testing.T) {
	var stderr bytes.Buffer
	type hookCall struct {
		command string
		env     map[string]string
	}
	var hookCalls []hookCall

	cfg := Config{
		Stderr: &stderr,
		GitExec: func(dir string, args ...string) (string, error) {
			if args[0] == "tag" && args[1] == "--list" {
				return "v0.0.0", nil
			}
			if args[0] == "log" {
				return "abc1234\x00feat: feature\x00\x00", nil
			}
			return "", nil
		},
		ReadFile: func(name string) ([]byte, error) {
			if name == ".pk.json" {
				return []byte(`{"changelog":{"hooks":{"postVersion":"echo post","preCommit":"echo pre"}}}`), nil
			}
			return nil, os.ErrNotExist
		},
		WriteFile: func(name string, data []byte, perm os.FileMode) error { return nil },
		RunScript: func(command string, env map[string]string) error {
			hookCalls = append(hookCalls, hookCall{command, env})
			return nil
		},
		Now: fixedTime,
	}

	code := Run(cfg)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", code, stderr.String())
	}

	if len(hookCalls) != 2 {
		t.Fatalf("hook calls = %d, want 2: %v", len(hookCalls), hookCalls)
	}
	if hookCalls[0].command != "echo post" {
		t.Errorf("postVersion hook command = %q, want 'echo post'", hookCalls[0].command)
	}
	if hookCalls[0].env["VERSION"] != "0.1.0" {
		t.Errorf("postVersion hook VERSION = %q, want '0.1.0'", hookCalls[0].env["VERSION"])
	}
	if hookCalls[1].command != "echo pre" {
		t.Errorf("preCommit hook command = %q, want 'echo pre'", hookCalls[1].command)
	}
	if hookCalls[1].env["VERSION"] != "0.1.0" {
		t.Errorf("preCommit hook VERSION = %q, want '0.1.0'", hookCalls[1].env["VERSION"])
	}
}

func TestRun_hookFailure(t *testing.T) {
	var stderr bytes.Buffer

	cfg := Config{
		Stderr: &stderr,
		GitExec: func(dir string, args ...string) (string, error) {
			if args[0] == "tag" && args[1] == "--list" {
				return "v0.0.0", nil
			}
			if args[0] == "log" {
				return "abc1234\x00feat: feature\x00\x00", nil
			}
			return "", nil
		},
		ReadFile: func(name string) ([]byte, error) {
			if name == ".pk.json" {
				return []byte(`{"changelog":{"hooks":{"postVersion":"fail"}}}`), nil
			}
			return nil, os.ErrNotExist
		},
		WriteFile: func(name string, data []byte, perm os.FileMode) error { return nil },
		RunScript: func(command string, env map[string]string) error {
			return fmt.Errorf("hook failed")
		},
		Now: fixedTime,
	}

	code := Run(cfg)
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "postVersion hook failed") {
		t.Errorf("stderr = %q, want hook failure message", stderr.String())
	}
}

func TestRun_gitCommitFailure(t *testing.T) {
	var stderr bytes.Buffer

	cfg := Config{
		Stderr: &stderr,
		GitExec: func(dir string, args ...string) (string, error) {
			if args[0] == "tag" && args[1] == "--list" {
				return "v0.0.0", nil
			}
			if args[0] == "log" {
				return "abc1234\x00feat: feature\x00\x00", nil
			}
			if args[0] == "commit" {
				return "", fmt.Errorf("commit failed")
			}
			return "", nil
		},
		ReadFile:  func(name string) ([]byte, error) { return nil, os.ErrNotExist },
		WriteFile: func(name string, data []byte, perm os.FileMode) error { return nil },
		RunScript: func(command string, env map[string]string) error { return nil },
		Now:       fixedTime,
	}

	code := Run(cfg)
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
}

func TestRun_subsequentRelease(t *testing.T) {
	var writtenContent []byte

	existing := changelogHeader + "\n## [v0.1.0] - 2026-03-01\n\n### Added\n\n- old feature (xyz9999)\n"

	cfg := Config{
		Stderr: &bytes.Buffer{},
		GitExec: func(dir string, args ...string) (string, error) {
			if args[0] == "tag" && args[1] == "--list" {
				return "v0.1.0", nil
			}
			if args[0] == "log" {
				return "abc1234\x00fix: important fix\x00\x00", nil
			}
			return "", nil
		},
		ReadFile: func(name string) ([]byte, error) {
			if name == "CHANGELOG.md" {
				return []byte(existing), nil
			}
			return nil, os.ErrNotExist
		},
		WriteFile: func(name string, data []byte, perm os.FileMode) error {
			if name == "CHANGELOG.md" {
				writtenContent = data
			}
			return nil
		},
		RunScript: func(command string, env map[string]string) error { return nil },
		Now:       fixedTime,
	}

	code := Run(cfg)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	content := string(writtenContent)
	// New section should be before old section.
	newIdx := strings.Index(content, "## [v0.1.1]")
	oldIdx := strings.Index(content, "## [v0.1.0]")
	if newIdx < 0 || oldIdx < 0 || newIdx >= oldIdx {
		t.Error("new section should appear before old section")
	}
	if !strings.Contains(content, "important fix") {
		t.Error("missing fix entry")
	}
	if !strings.Contains(content, "old feature") {
		t.Error("old entries should be preserved")
	}
}

func TestRun_guardedBranch(t *testing.T) {
	var stderr bytes.Buffer
	cfg := Config{
		Stderr: &stderr,
		GitExec: func(dir string, args ...string) (string, error) {
			if args[0] == "branch" {
				return "main\n", nil
			}
			return "", nil
		},
		ReadFile: func(name string) ([]byte, error) {
			return []byte(`{"guard":{"branches":["main"]}}`), nil
		},
		Now: fixedTime,
	}
	code := Run(cfg)
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "protected branch") {
		t.Errorf("stderr = %q, want protected branch warning", stderr.String())
	}
}

func TestRun_guardedBranchAllowsUnprotected(t *testing.T) {
	var stderr bytes.Buffer
	cfg := Config{
		Stderr: &stderr,
		GitExec: func(dir string, args ...string) (string, error) {
			if args[0] == "branch" {
				return "dev\n", nil
			}
			if args[0] == "tag" && args[1] == "--list" {
				return "v0.0.0", nil
			}
			if args[0] == "log" {
				return "abc1234\x00feat: add feature\x00\x00", nil
			}
			return "", nil
		},
		ReadFile: func(name string) ([]byte, error) {
			return []byte(`{"guard":{"branches":["main"]}}`), nil
		},
		Now:    fixedTime,
		DryRun: true,
	}
	code := Run(cfg)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if strings.Contains(stderr.String(), "protected branch") {
		t.Errorf("should not warn on unprotected branch")
	}
}

func TestRun_dirtyWorkingTree(t *testing.T) {
	var stderr bytes.Buffer

	cfg := Config{
		Stderr: &stderr,
		GitExec: func(dir string, args ...string) (string, error) {
			if args[0] == "status" && args[1] == "--porcelain" {
				return " M dirty.go", nil
			}
			return "", nil
		},
		ReadFile: func(name string) ([]byte, error) { return nil, os.ErrNotExist },
	}

	code := Run(cfg)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "working tree is not clean") {
		t.Errorf("stderr = %q, want dirty tree message", stderr.String())
	}
}

func TestRun_dirtyWorkingTreeAllowedInDryRun(t *testing.T) {
	var stderr bytes.Buffer

	cfg := Config{
		Stderr: &stderr,
		GitExec: func(dir string, args ...string) (string, error) {
			if args[0] == "status" && args[1] == "--porcelain" {
				t.Error("status --porcelain should not be called in dry run")
			}
			if args[0] == "tag" && args[1] == "--list" {
				return "v1.0.0", nil
			}
			if args[0] == "log" {
				return "abc1234\x00feat: new feature\x00\x00", nil
			}
			return "", nil
		},
		ReadFile: func(name string) ([]byte, error) { return nil, os.ErrNotExist },
		Now:      fixedTime,
		DryRun:   true,
	}

	code := Run(cfg)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", code, stderr.String())
	}
}

func TestRun_showScope(t *testing.T) {
	var stderr bytes.Buffer
	var writtenContent []byte

	cfg := Config{
		Stderr: &stderr,
		GitExec: func(dir string, args ...string) (string, error) {
			if args[0] == "tag" && args[1] == "--list" {
				return "v1.0.0", nil
			}
			if args[0] == "log" {
				return "dab3f6d\x00fix(flow): resolve pattern\x00\x00abc1234\x00feat: plain feature\x00\x00", nil
			}
			return "", nil
		},
		ReadFile: func(name string) ([]byte, error) {
			if name == ".pk.json" {
				return []byte(`{"changelog":{"showScope":true}}`), nil
			}
			return nil, os.ErrNotExist
		},
		WriteFile: func(name string, data []byte, perm os.FileMode) error {
			writtenContent = data
			return nil
		},
		RunScript: func(command string, env map[string]string) error { return nil },
		Now:       fixedTime,
	}

	code := Run(cfg)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", code, stderr.String())
	}

	content := string(writtenContent)
	if !strings.Contains(content, "**flow:** resolve pattern") {
		t.Errorf("scoped commit should have scope prefix, got: %s", content)
	}
	if strings.Contains(content, "**plain feature**") {
		t.Errorf("unscoped commit should not have scope prefix, got: %s", content)
	}
}

// --- Undo tests ---

// undoGit returns a base stub for Undo() tests. trailer is the Release-Tag
// value returned by git log; empty means the trailer is absent.
func undoGit(trailer string) func(dir string, args ...string) (string, error) {
	return func(dir string, args ...string) (string, error) {
		switch args[0] {
		case "log":
			// log -1 --format=%(trailers:...) HEAD
			if len(args) >= 3 && strings.HasPrefix(args[2], "--format=%(trailers") {
				return trailer, nil
			}
			// log @{u}..HEAD --oneline — HEAD is ahead of upstream (unpushed).
			return "abc1234 chore: release " + trailer, nil
		case "status":
			return "", nil
		case "rev-parse":
			// Upstream exists.
			return "origin/dev", nil
		case "reset":
			return "", nil
		}
		return "", nil
	}
}

func TestUndo_happyPath(t *testing.T) {
	var stderr bytes.Buffer
	var resetCalled bool

	base := undoGit("v0.6.2")
	cfg := Config{
		Stderr: &stderr,
		GitExec: func(dir string, args ...string) (string, error) {
			if args[0] == "reset" {
				resetCalled = true
				if args[1] != "--hard" || args[2] != "HEAD~1" {
					t.Errorf("reset args = %v, want [reset --hard HEAD~1]", args)
				}
			}
			return base(dir, args...)
		},
	}

	code := Undo(cfg)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", code, stderr.String())
	}
	if !resetCalled {
		t.Error("git reset --hard HEAD~1 was not called")
	}
	if !strings.Contains(stderr.String(), "Undid release commit v0.6.2") {
		t.Errorf("stderr = %q, want undo confirmation", stderr.String())
	}
}

func TestUndo_noTrailer(t *testing.T) {
	var stderr bytes.Buffer

	cfg := Config{
		Stderr:  &stderr,
		GitExec: undoGit(""),
	}

	code := Undo(cfg)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "no Release-Tag trailer on HEAD") {
		t.Errorf("stderr = %q, want no-trailer message", stderr.String())
	}
}

func TestUndo_invalidTrailerValue(t *testing.T) {
	var stderr bytes.Buffer

	cfg := Config{
		Stderr:  &stderr,
		GitExec: undoGit("v0.6"),
	}

	code := Undo(cfg)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "not valid semver") {
		t.Errorf("stderr = %q, want invalid-semver message", stderr.String())
	}
}

func TestUndo_dirtyTree(t *testing.T) {
	var stderr bytes.Buffer

	base := undoGit("v0.6.2")
	cfg := Config{
		Stderr: &stderr,
		GitExec: func(dir string, args ...string) (string, error) {
			if args[0] == "status" {
				return " M CHANGELOG.md", nil
			}
			return base(dir, args...)
		},
	}

	code := Undo(cfg)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "working tree is not clean") {
		t.Errorf("stderr = %q, want dirty-tree message", stderr.String())
	}
}

func TestUndo_alreadyPushed(t *testing.T) {
	var stderr bytes.Buffer

	base := undoGit("v0.6.2")
	cfg := Config{
		Stderr: &stderr,
		GitExec: func(dir string, args ...string) (string, error) {
			// log @{u}..HEAD returns empty — HEAD is at or behind upstream.
			if args[0] == "log" && len(args) >= 2 && args[1] == "@{u}..HEAD" {
				return "", nil
			}
			return base(dir, args...)
		},
	}

	code := Undo(cfg)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "already on the remote") {
		t.Errorf("stderr = %q, want pushed-commit message", stderr.String())
	}
}

func TestUndo_noUpstream(t *testing.T) {
	var stderr bytes.Buffer
	var resetCalled bool

	base := undoGit("v0.6.2")
	cfg := Config{
		Stderr: &stderr,
		GitExec: func(dir string, args ...string) (string, error) {
			// rev-parse --abbrev-ref HEAD@{upstream} errors: no upstream.
			if args[0] == "rev-parse" {
				return "", fmt.Errorf("no upstream configured for branch")
			}
			if args[0] == "reset" {
				resetCalled = true
			}
			return base(dir, args...)
		},
	}

	code := Undo(cfg)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", code, stderr.String())
	}
	if !resetCalled {
		t.Error("reset should run when branch has no upstream")
	}
}

// --- Exclude tests ---

// excludeLog builds a synthetic git log output containing the given commits
// in the NUL-delimited format pk changelog expects. Each entry becomes
// "<hash>\x00<subject>\x00\x00" (no body).
func excludeLog(entries ...[2]string) string {
	var b strings.Builder
	for _, e := range entries {
		b.WriteString(e[0])
		b.WriteByte(0)
		b.WriteString(e[1])
		b.WriteByte(0)
		b.WriteByte(0)
	}
	return b.String()
}

// excludeConfig wires a Config with a log that returns the given commits
// and captures all git commit calls into the returned slice pointer.
func excludeConfig(t *testing.T, log string, exclude []string) (Config, *[]string, *bytes.Buffer) {
	t.Helper()
	var stderr bytes.Buffer
	gitCalls := make([]string, 0)
	cfg := Config{
		Stderr: &stderr,
		GitExec: func(dir string, args ...string) (string, error) {
			gitCalls = append(gitCalls, strings.Join(args, " "))
			if args[0] == "tag" && args[1] == "--list" {
				return "v1.0.0", nil
			}
			if args[0] == "log" {
				return log, nil
			}
			return "", nil
		},
		ReadFile:  func(name string) ([]byte, error) { return nil, os.ErrNotExist },
		WriteFile: func(name string, data []byte, perm os.FileMode) error { return nil },
		RunScript: func(command string, env map[string]string) error { return nil },
		Now:       fixedTime,
		Exclude:   exclude,
	}
	return cfg, &gitCalls, &stderr
}

// commitMatch returns true if gitCalls contains a "commit -m ... --trailer Release-Tag: <tag>" call.
func commitMatch(gitCalls []string, tag string) bool {
	for _, c := range gitCalls {
		if strings.Contains(c, "commit -m chore: release "+tag) &&
			strings.Contains(c, "--trailer Release-Tag: "+tag) {
			return true
		}
	}
	return false
}

func TestRun_excludeByShortSHA(t *testing.T) {
	log := excludeLog(
		[2]string{"aaa1111", "fix: real fix"},
		[2]string{"bbb2222", "fix: should be excluded"},
	)
	cfg, gitCalls, stderr := excludeConfig(t, log, []string{"bbb2222"})

	var writtenContent []byte
	cfg.WriteFile = func(name string, data []byte, perm os.FileMode) error {
		writtenContent = data
		return nil
	}

	code := Run(cfg)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", code, stderr.String())
	}
	if strings.Contains(string(writtenContent), "should be excluded") {
		t.Errorf("excluded commit should not appear in CHANGELOG.md, got: %s", writtenContent)
	}
	if !strings.Contains(string(writtenContent), "real fix") {
		t.Errorf("non-excluded commit should appear in CHANGELOG.md, got: %s", writtenContent)
	}
	if !commitMatch(*gitCalls, "v1.0.1") {
		t.Errorf("expected patch bump (v1.0.1), git calls: %v", *gitCalls)
	}
}

func TestRun_excludeMultiple(t *testing.T) {
	log := excludeLog(
		[2]string{"aaa1111", "fix: keep this"},
		[2]string{"bbb2222", "fix: drop this"},
		[2]string{"ccc3333", "fix: drop that"},
	)
	cfg, _, stderr := excludeConfig(t, log, []string{"bbb2222", "ccc3333"})

	var writtenContent []byte
	cfg.WriteFile = func(name string, data []byte, perm os.FileMode) error {
		writtenContent = data
		return nil
	}

	code := Run(cfg)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", code, stderr.String())
	}
	content := string(writtenContent)
	if !strings.Contains(content, "keep this") {
		t.Errorf("kept commit missing from changelog: %s", content)
	}
	if strings.Contains(content, "drop this") {
		t.Errorf("first excluded commit present in changelog: %s", content)
	}
	if strings.Contains(content, "drop that") {
		t.Errorf("second excluded commit present in changelog: %s", content)
	}
}

func TestRun_excludeUnknownSHA(t *testing.T) {
	log := excludeLog(
		[2]string{"aaa1111", "fix: one real fix"},
	)
	cfg, gitCalls, stderr := excludeConfig(t, log, []string{"deadbee"})

	code := Run(cfg)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "warning: --exclude deadbee did not match any commit") {
		t.Errorf("stderr missing unknown-sha warning: %s", stderr.String())
	}
	// Unknown SHA should not block the release.
	if !commitMatch(*gitCalls, "v1.0.1") {
		t.Errorf("expected release to proceed despite unknown exclude, git calls: %v", *gitCalls)
	}
}

func TestRun_excludeAllFeats_fallsToPatch(t *testing.T) {
	log := excludeLog(
		[2]string{"aaa1111", "feat: new thing"},
		[2]string{"bbb2222", "feat: another thing"},
		[2]string{"ccc3333", "fix: a fix"},
	)
	cfg, gitCalls, stderr := excludeConfig(t, log, []string{"aaa1111", "bbb2222"})

	code := Run(cfg)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", code, stderr.String())
	}
	// Only a fix remains. Bump should be patch (v1.0.0 → v1.0.1), not minor.
	if !commitMatch(*gitCalls, "v1.0.1") {
		t.Errorf("expected patch bump v1.0.1 after excluding all feats, git calls: %v", *gitCalls)
	}
	if commitMatch(*gitCalls, "v1.1.0") {
		t.Errorf("bump should not have stayed at minor v1.1.0")
	}
}

func TestRun_excludeSomeFeats_staysMinor(t *testing.T) {
	log := excludeLog(
		[2]string{"aaa1111", "feat: keeper"},
		[2]string{"bbb2222", "feat: to drop"},
		[2]string{"ccc3333", "fix: a fix"},
	)
	cfg, gitCalls, stderr := excludeConfig(t, log, []string{"bbb2222"})

	code := Run(cfg)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", code, stderr.String())
	}
	// One feat remains. Bump should stay minor (v1.0.0 → v1.1.0).
	if !commitMatch(*gitCalls, "v1.1.0") {
		t.Errorf("expected minor bump v1.1.0 with one feat remaining, git calls: %v", *gitCalls)
	}
}

func TestRun_excludeBreaking_fallsFromMajor(t *testing.T) {
	// Breaking change via "!" suffix. Also include a regular feat so the
	// bump has somewhere to fall (minor) instead of collapsing to patch.
	log := excludeLog(
		[2]string{"aaa1111", "feat!: breaking change"},
		[2]string{"bbb2222", "feat: normal feat"},
	)
	cfg, gitCalls, stderr := excludeConfig(t, log, []string{"aaa1111"})

	code := Run(cfg)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", code, stderr.String())
	}
	// Breaking change excluded, one feat remains. Bump should be minor (v1.0.0 → v1.1.0).
	if !commitMatch(*gitCalls, "v1.1.0") {
		t.Errorf("expected bump to fall from major to minor v1.1.0, git calls: %v", *gitCalls)
	}
	if commitMatch(*gitCalls, "v2.0.0") {
		t.Errorf("bump should not have stayed at major v2.0.0")
	}
}
