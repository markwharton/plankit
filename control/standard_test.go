package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The standard's structure is the reference for changing it: every rule
// sits under a principle and ends with how it is checked, every case
// names the principle it shows, and every principle has a case. This
// carries that decision in docs/plans/2026-11-22-001-control-contain-
// drift-in-the-standard-and-the-checker.md.
func TestEveryRuleHasACheckAndEveryCaseAPrinciple(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "docs", "control.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, problem := range standardProblems(string(data)) {
		t.Error("docs/control.md: " + problem)
	}
}

// standardProblems reads the standard by its headings and bullets and
// returns each way its structure is broken. The principles are the third
// level headings under "## Principles and rules"; the cases are the
// bullets under "## Cases".
func standardProblems(text string) []string {
	var problems []string
	rules := map[string][]string{} // principle → its rule bullets
	var principles []string
	var cases []string
	section, principle := "", ""
	var bullet *string
	end := func() { bullet = nil }
	for _, line := range strings.Split(text, "\n") {
		switch {
		case strings.HasPrefix(line, "## "):
			end()
			section, principle = strings.TrimPrefix(line, "## "), ""
		case strings.HasPrefix(line, "### ") && section == "Principles and rules":
			end()
			principle = strings.TrimPrefix(line, "### ")
			principles = append(principles, principle)
			rules[principle] = nil
		case strings.HasPrefix(line, "- **"):
			end()
			switch {
			case section == "Principles and rules" && principle != "":
				rules[principle] = append(rules[principle], line)
				bullet = &rules[principle][len(rules[principle])-1]
			case section == "Cases":
				cases = append(cases, line)
			}
		case strings.HasPrefix(line, "  ") && bullet != nil:
			*bullet += " " + strings.TrimSpace(line)
		default:
			end()
		}
	}
	if len(principles) == 0 {
		return []string{`no principles: this test reads "### " headings under "## Principles and rules"`}
	}
	hasCase := map[string]bool{}
	for _, c := range cases {
		name, _, ok := strings.Cut(strings.TrimPrefix(c, "- **"), ": ")
		if !ok || !contains(principles, name) {
			problems = append(problems, "a case names no principle: "+c)
			continue
		}
		hasCase[name] = true
	}
	for _, p := range principles {
		if len(rules[p]) == 0 {
			problems = append(problems, "principle "+p+" has no rules")
		}
		for _, r := range rules[p] {
			if !strings.Contains(r, "Check:") {
				title, _, _ := strings.Cut(strings.TrimPrefix(r, "- **"), "**")
				problems = append(problems, "the rule \""+title+"\" under "+p+" does not end with its Check")
			}
		}
		if !hasCase[p] {
			problems = append(problems, "principle "+p+" has no case")
		}
	}
	return problems
}

// The instrument is standardProblems: it must find each kind of break,
// and find nothing in a well-formed standard.
func TestStandardProblemsReadsTheStructure(t *testing.T) {
	good := "## Principles and rules\n\n### Alpha\n\nWhy.\n\n- **A rule.** Text\n  more. Check: a test.\n\n### Beta\n\n- **B rule.** Check: review.\n\n## Cases\n\n- **Alpha: one.** x\n- **Beta: two.** y\n"
	if got := standardProblems(good); len(got) != 0 {
		t.Fatalf("well formed: want no problems, got %q", got)
	}
	for _, tc := range []struct{ name, text, want string }{
		{"a rule with no Check", strings.Replace(good, " Check: a test.", "", 1), `"A rule." under Alpha does not end with its Check`},
		{"a case naming no principle", strings.Replace(good, "- **Beta: two.**", "- **Gamma: two.**", 1), "a case names no principle"},
		{"a principle with no case", strings.Replace(good, "- **Beta: two.** y\n", "", 1), "principle Beta has no case"},
		{"a principle with no rules", strings.Replace(good, "- **B rule.** Check: review.\n", "", 1), "principle Beta has no rules"},
		{"no principles", "## Cases\n", "no principles"},
	} {
		got := strings.Join(standardProblems(tc.text), "\n")
		if !strings.Contains(got, tc.want) {
			t.Errorf("%s: want a problem containing %q, got %q", tc.name, tc.want, got)
		}
	}
}
