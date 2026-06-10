package readiness

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/markwharton/plankit/internal/config"
)

// fakeGit returns a GitExec that answers from canned responses keyed by the
// joined argument list. Unknown commands return an error, matching git's
// non-zero exit on missing refs.
func fakeGit(responses map[string]string) func(string, ...string) (string, error) {
	return func(dir string, args ...string) (string, error) {
		key := strings.Join(args, " ")
		if out, ok := responses[key]; ok {
			return out, nil
		}
		return "", errors.New("exit status 1")
	}
}

func mergeConf(branch string) config.PkConfig {
	var c config.PkConfig
	c.Release.Branch = branch
	return c
}

func find(t *testing.T, checks []Check, label string) Check {
	t.Helper()
	for _, c := range checks {
		if c.Label == label {
			return c
		}
	}
	t.Fatalf("no check labeled %q in %+v", label, checks)
	return Check{}
}

func TestEvaluate_mergeFlowAllReady(t *testing.T) {
	git := fakeGit(map[string]string{
		"tag --list v* --sort=-v:refname":                        "v1.2.3\n",
		"branch --show-current":                                  "develop\n",
		"rev-parse --verify --quiet refs/remotes/origin/develop": "abc123\n",
		"rev-parse --verify --quiet refs/remotes/origin/main":    "def456\n",
	})
	checks := Evaluate(git, "/repo", mergeConf("main"))
	if !Ready(checks) {
		t.Errorf("Ready = false, want true; checks: %+v", checks)
	}
	if c := find(t, checks, "baseline tag"); c.Value != "v1.2.3" {
		t.Errorf("baseline tag value = %q, want v1.2.3", c.Value)
	}
	find(t, checks, "develop on origin")
	find(t, checks, "main on origin")
}

func TestEvaluate_mergeFlowOnReleaseBranch(t *testing.T) {
	// The main-only dead-end: release.branch is main and the user is on main.
	git := fakeGit(map[string]string{
		"branch --show-current": "main\n",
	})
	checks := Evaluate(git, "/repo", mergeConf("main"))
	if Ready(checks) {
		t.Error("Ready = true, want false")
	}
	c := find(t, checks, "working branch")
	if c.OK {
		t.Error("working branch check OK, want failure")
	}
	if !strings.Contains(c.Hint, "git switch -c develop") {
		t.Errorf("working branch hint = %q, want create-develop command", c.Hint)
	}
	if c := find(t, checks, "baseline tag"); c.OK || c.Hint == "" || c.Or == "" {
		t.Errorf("baseline tag = %+v, want failed check with hint and git equivalent", c)
	}
	// No origin check for main: the working-branch gap comes first.
	for _, c := range checks {
		if c.Label == "main on origin" {
			t.Error("unexpected 'main on origin' check while on the release branch")
		}
	}
}

func TestEvaluate_mergeFlowOriginRefsMissing(t *testing.T) {
	git := fakeGit(map[string]string{
		"tag --list v* --sort=-v:refname": "v0.0.0\n",
		"branch --show-current":           "develop\n",
	})
	checks := Evaluate(git, "/repo", mergeConf("main"))
	if Ready(checks) {
		t.Error("Ready = true, want false")
	}
	for _, branch := range []string{"develop", "main"} {
		c := find(t, checks, branch+" on origin")
		if c.OK || c.Value != "missing" {
			t.Errorf("%s on origin = %+v, want missing", branch, c)
		}
		want := fmt.Sprintf("To publish it: git push -u origin %s", branch)
		if c.Hint != want {
			t.Errorf("hint = %q, want %q", c.Hint, want)
		}
	}
}

func TestEvaluate_trunkFlow(t *testing.T) {
	git := fakeGit(map[string]string{
		"tag --list v* --sort=-v:refname":                     "v2.0.0\n",
		"branch --show-current":                               "main\n",
		"rev-parse --verify --quiet refs/remotes/origin/main": "abc\n",
	})
	checks := Evaluate(git, "/repo", config.PkConfig{})
	if !Ready(checks) {
		t.Errorf("Ready = false, want true; checks: %+v", checks)
	}
	if len(checks) != 2 {
		t.Errorf("len(checks) = %d, want 2 (no merge-flow checks without release.branch)", len(checks))
	}
}

func TestEvaluate_detachedHead(t *testing.T) {
	// Detached HEAD: branch checks are skipped, baseline still evaluated.
	git := fakeGit(map[string]string{
		"tag --list v* --sort=-v:refname": "v1.0.0\n",
		"branch --show-current":           "\n",
	})
	checks := Evaluate(git, "/repo", config.PkConfig{})
	if len(checks) != 1 || checks[0].Label != "baseline tag" {
		t.Errorf("checks = %+v, want baseline tag only", checks)
	}
}

func TestEvaluate_gitFailures(t *testing.T) {
	// Every git call fails: checks degrade to not-ready, never panic.
	git := fakeGit(nil)
	checks := Evaluate(git, "/repo", mergeConf("main"))
	if Ready(checks) {
		t.Error("Ready = true with failing git, want false")
	}
	if c := find(t, checks, "baseline tag"); c.OK {
		t.Error("baseline tag OK with failing git")
	}
}

func TestValidSemverTag(t *testing.T) {
	tests := []struct {
		name    string
		out     string
		err     error
		wantTag string
		wantOK  bool
	}{
		{"valid", "v1.2.3\nv1.0.0", nil, "v1.2.3", true},
		{"skips invalid", "vNext\nv0.9.0", nil, "v0.9.0", true},
		{"none", "", nil, "", false},
		{"only invalid", "vNext\nvFinal", nil, "", false},
		{"git error", "", errors.New("boom"), "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			git := func(dir string, args ...string) (string, error) { return tt.out, tt.err }
			tag, ok := ValidSemverTag(git, "/repo")
			if tag != tt.wantTag || ok != tt.wantOK {
				t.Errorf("ValidSemverTag = (%q, %v), want (%q, %v)", tag, ok, tt.wantTag, tt.wantOK)
			}
		})
	}
}
