package brief

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/markwharton/plankit/internal/cli"
	"github.com/markwharton/plankit/internal/config"
	"github.com/markwharton/plankit/internal/git"
)

// TestTextFollowsTheDials pins what each dial changes in the text.
func TestTextFollowsTheDials(t *testing.T) {
	base := func() *config.PkConfig { return config.Default("main") }
	cases := []struct {
		name    string
		mutate  func(*config.PkConfig)
		want    []string
		notWant []string
	}{
		{"default", func(*config.PkConfig) {},
			[]string{"Types: feat, fix, deprecate", "guard asks before one is committed", "main is protected: commits and pushes there are blocked.", "do not push", "releases merge into main with pk ship", "preserve mode is manual", "pk help lists every command."},
			[]string{"style, plan", "automatically"}}, // hidden plan type never listed
		{"empty types resolve to defaults", func(c *config.PkConfig) { c.Changelog.Types = nil },
			[]string{"Types: feat, fix, deprecate"}, nil},
		{"custom types", func(c *config.PkConfig) {
			c.Changelog.Types = []config.TypeConfig{{Type: "feat", Section: "New"}, {Type: "wip", Section: "x", Hidden: true}}
		}, []string{"Types: feat (from"}, []string{"wip"}},
		{"breaking off keeps the rule, drops the ask", func(c *config.PkConfig) { c.Guard.Breaking = "off" },
			[]string{"only on explicit user direction."}, []string{"guard asks"}},
		{"guard ask", func(c *config.PkConfig) { c.Guard.Mode = "ask"; c.Guard.Push = "ask" },
			[]string{"prompt for confirmation.", "Pushing prompts for confirmation."}, []string{"blocked"}},
		{"guard off drops the branch paragraph", func(c *config.PkConfig) { c.Guard.Mode = "off" },
			nil, []string{"protected", "pk ship"}},
		{"two branches", func(c *config.PkConfig) { c.Guard.Branches = []string{"main", "release"} },
			[]string{"main, release are protected"}, nil},
		{"trunk flow", func(c *config.PkConfig) { c.Release.Branch = "" },
			[]string{"Releases run from the default branch with pk ship."}, []string{"merge into"}},
		{"preserve auto", func(c *config.PkConfig) { c.Preserve.Mode = "auto" },
			[]string{"committed there automatically"}, []string{"/plankit:preserve"}},
		{"preserve off", func(c *config.PkConfig) { c.Preserve.Mode = "off" },
			[]string{"docs/plans/ is immutable.\n"}, []string{"Approved plans"}},
	}
	for _, tc := range cases {
		cfg := base()
		tc.mutate(cfg)
		got := Text(cfg)
		for _, w := range tc.want {
			if !strings.Contains(got, w) {
				t.Errorf("%s: missing %q in:\n%s", tc.name, w, got)
			}
		}
		for _, nw := range tc.notWant {
			if strings.Contains(got, nw) {
				t.Errorf("%s: unexpected %q in:\n%s", tc.name, nw, got)
			}
		}
	}
}

func scratch(t *testing.T, configured bool) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{{"init", "-q", "-b", "main"}, {"config", "user.email", "t@t"}, {"config", "user.name", "t"}} {
		if _, err := git.Exec(dir, args...); err != nil {
			t.Fatal(err)
		}
	}
	if configured {
		if err := config.Write(dir, config.Default("main")); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestHookShapeInjectsContextOnlyWhenConfigured(t *testing.T) {
	t.Setenv("CLAUDE_PROJECT_DIR", "")
	run := func(dir string) string {
		payload, _ := json.Marshal(map[string]any{"cwd": dir, "session_id": "s", "hook_event_name": "SessionStart"})
		var out, errw bytes.Buffer
		if code := cli.RunIO([]string{"pk", "brief"}, []*cli.Command{Cmd}, bytes.NewReader(payload), &out, &errw); code != 0 {
			t.Fatalf("hook exit %d: %s", code, errw.String())
		}
		return out.String()
	}
	if out := run(scratch(t, false)); out != "" {
		t.Fatalf("unconfigured repo must be silent, got %q", out)
	}
	out := run(scratch(t, true))
	var resp struct {
		HookSpecificOutput struct{ HookEventName, AdditionalContext string }
	}
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("not a hook envelope: %v\n%s", err, out)
	}
	if resp.HookSpecificOutput.HookEventName != "SessionStart" || !strings.Contains(resp.HookSpecificOutput.AdditionalContext, "plankit is configured") {
		t.Fatalf("wrong envelope: %+v", resp)
	}
}

func TestExplicitShapePrintsTextOrRefuses(t *testing.T) {
	dir := scratch(t, true)
	var out, errw bytes.Buffer
	if code := cli.RunIO([]string{"pk", "brief", "--project-dir", dir}, []*cli.Command{Cmd}, nil, &out, &errw); code != 0 {
		t.Fatalf("exit %d: %s", code, errw.String())
	}
	if !strings.HasPrefix(out.String(), "plankit is configured in this repository.") || strings.Contains(out.String(), "hookSpecificOutput") {
		t.Fatalf("explicit shape must print plain text:\n%s", out.String())
	}
	if _, err := os.Stat(filepath.Join(dir, ".pk.json")); err != nil {
		t.Fatal(err)
	}
	// --format json at a terminal is the hook envelope itself.
	out.Reset()
	if code := cli.RunIO([]string{"pk", "brief", "--project-dir", dir, "--format", "json"}, []*cli.Command{Cmd}, nil, &out, &errw); code != 0 {
		t.Fatalf("json exit %d: %s", code, errw.String())
	}
	var env struct {
		HookSpecificOutput struct{ HookEventName, AdditionalContext string }
	}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil || env.HookSpecificOutput.HookEventName != "SessionStart" {
		t.Fatalf("--format json must be the SessionStart envelope: %v\n%s", err, out.String())
	}

	dir = scratch(t, false)
	out.Reset()
	errw.Reset()
	if code := cli.RunIO([]string{"pk", "brief", "--project-dir", dir}, []*cli.Command{Cmd}, nil, &out, &errw); code != cli.ExitState || !strings.Contains(errw.String(), "pk init") {
		t.Fatalf("unconfigured explicit run: code=%d errw=%q", code, errw.String())
	}
}

// TestHookActsWhereTheSessionIs pins the precedence bug: with
// CLAUDE_PROJECT_DIR pointing at an unconfigured directory and the
// payload's cwd inside a configured repository, the hook must answer
// for the repository the session is actually in.
func TestHookActsWhereTheSessionIs(t *testing.T) {
	started := scratch(t, false)
	here := scratch(t, true)
	t.Setenv("CLAUDE_PROJECT_DIR", started)
	payload, _ := json.Marshal(map[string]any{"cwd": here, "hook_event_name": "SessionStart"})
	var out, errw bytes.Buffer
	if code := cli.RunIO([]string{"pk", "brief"}, []*cli.Command{Cmd}, bytes.NewReader(payload), &out, &errw); code != 0 {
		t.Fatalf("exit %d: %s", code, errw.String())
	}
	if !strings.Contains(out.String(), "plankit is configured") {
		t.Fatalf("hook followed CLAUDE_PROJECT_DIR instead of the session's cwd: %q", out.String())
	}
}
