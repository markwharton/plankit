package rules

import (
	"strings"
	"testing"
)

func TestLintSafetyScan(t *testing.T) {
	zwsp := string(rune(0x200B)) // zero-width space

	// Clean tree: base --lint passes.
	cfg, buf := setupProject(t, "", map[string]string{
		"clean.md": managedRule("Clean", "craft", "# Clean\n\n- ok\n"),
	})
	cfg.Lint = true
	if code := Run(cfg); code != 0 {
		t.Fatalf("clean --lint returned %d, want 0", code)
	}
	if !strings.Contains(buf.String(), "No issues found") {
		t.Errorf("expected clean message, got: %s", buf.String())
	}

	// Hidden char: base --lint fails.
	cfg, buf = setupProject(t, "", map[string]string{
		"bad.md": localRule("Bad", "craft", "# Bad\n\n- has"+zwsp+"hidden\n"),
	})
	cfg.Lint = true
	if code := Run(cfg); code != 1 {
		t.Fatalf("hidden-char --lint returned %d, want 1", code)
	}
	if !strings.Contains(buf.String(), "[safety]") {
		t.Errorf("expected safety finding, got: %s", buf.String())
	}
}

func TestLintStyleIsOptIn(t *testing.T) {
	// An em dash and trailing whitespace in a rule.
	body := "# Styled\n\n- uses an " + emDash + " em dash  \n"

	// Base --lint must NOT flag style.
	cfg, buf := setupProject(t, "", map[string]string{
		"styled.md": localRule("Styled", "conduct", body),
	})
	cfg.Lint = true
	if code := Run(cfg); code != 0 {
		t.Fatalf("base --lint should ignore style, returned %d: %s", code, buf.String())
	}

	// --strict flags both the em dash and the trailing whitespace.
	cfg, buf = setupProject(t, "", map[string]string{
		"styled.md": localRule("Styled", "conduct", body),
	})
	cfg.Lint = true
	cfg.Strict = true
	if code := Run(cfg); code != 1 {
		t.Fatalf("--strict should flag style, returned %d", code)
	}
	out := buf.String()
	if !strings.Contains(out, "em dash") {
		t.Errorf("expected em dash finding:\n%s", out)
	}
	if !strings.Contains(out, "trailing whitespace") {
		t.Errorf("expected trailing-whitespace finding:\n%s", out)
	}
}

func TestStyleFindingsHardWrapAndFence(t *testing.T) {
	content := strings.Join([]string{
		"- a single-line bullet",
		"- a bullet that wraps",
		"  onto a continuation line",
		"",
		"```",
		"- this bullet is in a code fence " + emDash + " ignored",
		"```",
	}, "\n")

	findings := styleFindings("x.md", content)

	var hardWrap, emDashInFence bool
	for _, f := range findings {
		if strings.Contains(f.msg, "hard-wrapped bullet") {
			hardWrap = true
		}
		if strings.Contains(f.msg, "em dash") {
			emDashInFence = true // should stay false: the only em dash is inside the fence
		}
	}
	if !hardWrap {
		t.Errorf("expected a hard-wrapped bullet finding, got %v", findings)
	}
	if emDashInFence {
		t.Error("em dash inside a code fence must be ignored")
	}
}

func TestIsBullet(t *testing.T) {
	yes := []string{"- item", "* item", "+ item", "1. item", "42. item"}
	no := []string{"text", "-no space", "1.no space", "## heading", ".dot", "1) paren"}
	for _, s := range yes {
		if !isBullet(s) {
			t.Errorf("isBullet(%q) = false, want true", s)
		}
	}
	for _, s := range no {
		if isBullet(s) {
			t.Errorf("isBullet(%q) = true, want false", s)
		}
	}
}
