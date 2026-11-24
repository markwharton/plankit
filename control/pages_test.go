package main

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

func page(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// firstColumn returns the first cell of each body row of the table
// after marker, with its backquotes and quotes taken off.
func firstColumn(t *testing.T, text, marker string) []string {
	t.Helper()
	rows, ok := tableRows(text, marker)
	if !ok {
		t.Fatalf("no table after %q", marker)
	}
	var out []string
	for _, r := range rows[1:] {
		out = append(out, strings.Trim(r[0], "`\""))
	}
	return out
}

// fields.md lists the format's fields for whoever writes a file, and
// the reader's table of keys is what the checker accepts. Each table on
// the page names the same keys as the reader, no more and no fewer.
func TestTheFieldsPageListsEveryKeyOfTheFormat(t *testing.T) {
	text := page(t, "fields.md")
	for _, tb := range []struct{ marker, object string }{
		{"## The definition file", "definition"},
		{"### A place", "place"},
		{"### An element", "element"},
		{"### A write", "write"},
	} {
		got, want := firstColumn(t, text, tb.marker), append([]string{}, keysOf[tb.object]...)
		sort.Strings(got)
		sort.Strings(want)
		if strings.Join(got, " ") != strings.Join(want, " ") {
			t.Errorf("fields.md, the table after %q:\n page   %q\n reader %q", tb.marker, got, want)
		}
	}
	// The values a field takes are on the page too.
	got := firstColumn(t, text, "### A check before a write")
	if strings.Join(got, " ") != "exists repeated belongs" {
		t.Errorf("fields.md: the checks before a write are %q", got)
	}
	// Every key is named somewhere on the page, in backquotes or as a
	// quoted key of an object shown in braces.
	for object, keys := range keysOf {
		for _, k := range keys {
			if !strings.Contains(text, "`"+k+"`") && !strings.Contains(text, `"`+k+`":`) {
				t.Errorf("fields.md does not name the key %s of %s", k, object)
			}
		}
	}
}

// checks.md names every check, and the checks of the first question are
// in its table in the order the checker prints them.
func TestTheChecksPageListsEveryCheck(t *testing.T) {
	text := page(t, "checks.md")
	var order []string
	for _, id := range firstColumn(t, text, "## Is the design sound?") {
		if len(order) == 0 || order[len(order)-1] != id {
			order = append(order, id)
		}
	}
	if strings.Join(order, " ") != strings.Join(soundChecks, " ") {
		t.Errorf("checks.md, the table of the first question:\n page    %q\n checker %q", order, soundChecks)
	}
	for _, id := range checkOrder {
		if !strings.Contains(text, "`"+id+"`") {
			t.Errorf("checks.md does not name the check %s", id)
		}
	}
	// The page states the order of the other checks in one sentence.
	if !strings.Contains(strings.Join(strings.Fields(text), " "), "then `held`, `follows`, `source`, `unsaid` and `stale`") {
		t.Error("checks.md does not give the order held, follows, source, unsaid, stale")
	}
}

// The subjects the checker prints are the ones in the page's table,
// and each names the element and the place the table says it does.
func TestTheSubjectsAreThePages(t *testing.T) {
	rows, ok := tableRows(page(t, "checks.md"), "The subject says which statement")
	if !ok {
		t.Fatal("checks.md has no table of subjects")
	}
	subjects := []Subject{
		sPlace("volume"), sElement("import"), sRead("import", "image"), sWrite("import", "volume"),
		sCheck("import", "volume", "repeated"), sRejects("import"), sDeclines("import"),
		sBranch("save", "say full"), sUnsaid("a long name is cut short"),
	}
	if len(rows)-1 != len(subjects) {
		t.Fatalf("checks.md has %d subjects, the checker %d", len(rows)-1, len(subjects))
	}
	field := regexp.MustCompile("`(element|place)`")
	for i, r := range rows[1:] {
		s := subjects[i]
		if got := strings.Trim(r[2], "`"); got != s.Text {
			t.Errorf("checks.md, row %d: the example is %q, the checker prints %q", i+1, got, s.Text)
		}
		var page, made []string
		for _, m := range field.FindAllStringSubmatch(r[3], -1) {
			page = append(page, m[1])
		}
		if s.Element != "" {
			made = append(made, "element")
		}
		if s.Place != "" {
			made = append(made, "place")
		}
		if strings.Join(page, " ") != strings.Join(made, " ") {
			t.Errorf("checks.md, row %d (%s): the page names the fields %q, the checker sets %q", i+1, s.Text, page, made)
		}
	}
}

// design.md sorts the checks of the first question into three groups.
// Every check is in exactly one of them.
func TestTheDesignPageGroupsEveryCheck(t *testing.T) {
	rows, ok := tableRows(page(t, "design.md"), "## What the checker looks at")
	if !ok {
		t.Fatal("design.md has no table of groups")
	}
	var got []string
	for _, r := range rows[1:] {
		for _, m := range regexp.MustCompile("`([a-z-]+)`").FindAllStringSubmatch(r[1], -1) {
			got = append(got, m[1])
		}
	}
	want := append([]string{}, soundChecks...)
	sort.Strings(got)
	sort.Strings(want)
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("design.md, the groups of checks:\n page    %q\n checker %q", got, want)
	}
}

// namedHold is a test a page names as a hold: the file, from the
// repository root, and the test names in it.
type namedHold struct {
	path  string
	tests []string
}

// holdPattern is a backquoted Go file followed by quoted test names, as
// in: Held by `x_test.go`, "TestA" and "TestB".
var holdPattern = regexp.MustCompile("`([A-Za-z0-9_./-]+\\.go)`((?:,? (?:and )?\"Test[A-Za-z]+\")*)")
var testName = regexp.MustCompile(`"(Test[A-Za-z]+)"`)

// namedHolds returns the holds a page names, and each "Held by" claim
// that names a file and no test.
func namedHolds(text string) (holds []namedHold, bare []string) {
	text = strings.Join(strings.Fields(text), " ")
	for _, m := range holdPattern.FindAllStringSubmatchIndex(text, -1) {
		path := text[m[2]:m[3]]
		var tests []string
		for _, n := range testName.FindAllStringSubmatch(text[m[4]:m[5]], -1) {
			tests = append(tests, n[1])
		}
		if len(tests) > 0 {
			holds = append(holds, namedHold{path, tests})
		} else if strings.HasSuffix(text[:m[0]], "Held by ") {
			bare = append(bare, path)
		}
	}
	return holds, bare
}

// Each claim names its hold. This finds every hold named on the
// checker's pages and in the standard, and fails when a named test is
// not in the named file. It reads no plan.
func TestEveryNamedHoldExists(t *testing.T) {
	root := ".."
	for _, p := range []struct {
		path     string
		mustName bool
	}{{"control/README.md", true}, {"control/design.md", false}, {"control/axioms.md", true}, {"control/view.md", true}, {"docs/control.md", true}} {
		data, err := os.ReadFile(filepath.Join(root, p.path))
		if err != nil {
			t.Fatal(err)
		}
		holds, bare := namedHolds(string(data))
		if p.mustName && len(holds) == 0 {
			t.Errorf("%s names no holds; this test reads `<file>_test.go`, \"TestName\"", p.path)
		}
		for _, b := range bare {
			t.Errorf("%s: a claim held by %s names no test", p.path, b)
		}
		for _, h := range holds {
			src, err := os.ReadFile(filepath.Join(root, h.path))
			if err != nil {
				t.Errorf("%s names a hold in %s, which does not exist", p.path, h.path)
				continue
			}
			for _, name := range h.tests {
				if !strings.Contains(string(src), "func "+name+"(t *testing.T)") {
					t.Errorf("%s names %s in %s, which has no such test", p.path, name, h.path)
				}
			}
		}
	}
}

func TestNamedHoldsReadsBothForms(t *testing.T) {
	holds, bare := namedHolds("Held by `a/x_test.go`, \"TestOne\" and\n\"TestTwo\". | Pass | `b/y_test.go`, \"TestThree\" |. Held by `c/z_test.go`. See `d/w.go`.")
	if len(holds) != 2 || holds[0].path != "a/x_test.go" || strings.Join(holds[0].tests, ",") != "TestOne,TestTwo" ||
		holds[1].path != "b/y_test.go" || strings.Join(holds[1].tests, ",") != "TestThree" {
		t.Errorf("holds: got %+v", holds)
	}
	if len(bare) != 1 || bare[0] != "c/z_test.go" {
		t.Errorf("bare: got %q, want only c/z_test.go", bare)
	}
}
