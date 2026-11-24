package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// subjects returns each printed line up to its message: the check and
// the subject, with "accepted" in front of an accepted one.
func subjects(stdout string) []string {
	var out []string
	for _, line := range strings.Split(strings.TrimRight(stdout, "\n"), "\n") {
		if line == "" {
			continue
		}
		head, _, _ := strings.Cut(line, ": ")
		out = append(out, head)
	}
	return out
}

func sameOrder(t *testing.T, name string, got, want []string) {
	t.Helper()
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("%s:\n got:\n  %s\n want:\n  %s", name, strings.Join(got, "\n  "), strings.Join(want, "\n  "))
	}
}

// The three trials are a design and the code as it was, for an upload,
// an import and a save. Each run prints its known findings, in the
// order checks.md fixes, and fails. The lists for upload and save are
// the ones fresh readers produced by hand from the three pages.
func TestTheTrialsPrintTheirKnownFindings(t *testing.T) {
	trials := []struct {
		dir   string
		files []string
		want  []string
	}{
		{"upload", []string{"documents.json", "upload.json", "manual-file.json"}, []string{
			"one-owner  place register",
			"one-owner  report writes register",
			"key-from-content  record writes register",
			"held  record rejects",
			"follows  file reads files",
			"follows  file writes files, check exists",
			"follows  file writes files, check repeated",
			"follows  parse rejects",
			"follows  report writes register",
			"follows  element manual-file",
			"source  report writes status",
			`unsaid  "a document with no pages is filed as an empty file"`,
		}},
		{"import", []string{"saved-files.json", "import.json"}, []string{
			"follows  import reads volume",
			"follows  import writes volume, check repeated",
		}},
		{"save", []string{"games.json", "save.json"}, []string{
			"replace-checks-owner  save in place writes disk",
			"rejects-declared  mark saved rejects",
			"declines-declared  save as declines",
			"join-ordered  element saving",
			`held  "a name of only spaces is refused"`,
			"held  store rejects",
			"follows  element retitle",
			"follows  element saving",
			"follows  place window title",
			"follows  save as declines",
			"follows  save in place writes disk",
			"follows  saving rejects",
			"source  save branch to say full",
			"stale  held  save rejects",
			"accepted  follows  place disk",
			"accepted  follows  element engine",
		}},
	}
	for _, tr := range trials {
		root := filepath.Join("examples", tr.dir)
		args := []string{"check", "--root", root}
		for _, f := range tr.files {
			args = append(args, filepath.Join(root, f))
		}
		code, stdout, _ := runCLI(args...)
		if code != 2 {
			t.Errorf("%s: exit %d, want 2", tr.dir, code)
		}
		sameOrder(t, tr.dir, subjects(stdout), tr.want)
	}
}

var jsonBlock = regexp.MustCompile("(?s)```json\n(.*?)```")
var plainBlock = regexp.MustCompile("(?s)```\n(.*?)```")

// The example at the end of fields.md is read from the page, run, and
// compared with the line checks.md says the run prints.
func TestThePagesExamplePrintsTheLineThePageShows(t *testing.T) {
	fields, err := os.ReadFile("fields.md")
	if err != nil {
		t.Fatal(err)
	}
	checks, err := os.ReadFile("checks.md")
	if err != nil {
		t.Fatal(err)
	}
	_, example, found := strings.Cut(string(fields), "\n## An example\n")
	blocks := jsonBlock.FindAllStringSubmatch(example, -1)
	if !found || len(blocks) != 2 {
		t.Fatalf("fields.md: want two JSON blocks under \"An example\", got %d", len(blocks))
	}
	dir := t.TempDir()
	var files []obj
	for i, name := range []string{"definition.json", "description.json"} {
		var o obj
		if err := json.Unmarshal([]byte(blocks[i][1]), &o); err != nil {
			t.Fatalf("fields.md, example block %d: %v", i+1, err)
		}
		files = append(files, o)
		writeFiles(t, dir, map[string]obj{name: o})
	}
	evidence(t, dir, files...)

	_, tail, found := strings.Cut(string(checks), "\n## An example\n")
	shown := plainBlock.FindStringSubmatch(tail)
	if !found || shown == nil {
		t.Fatal("checks.md: no output block under \"An example\"")
	}
	code, stdout, _ := runCLI("check", "--root", dir, filepath.Join(dir, "definition.json"), filepath.Join(dir, "description.json"))
	if code != 2 || stdout != shown[1] {
		t.Errorf("exit %d\n got  %q\n want %q", code, stdout, shown[1])
	}

	// With the acceptance the page shows beside the description file,
	// the run passes.
	accept := jsonBlock.FindStringSubmatch(tail)
	if accept == nil {
		t.Fatal("checks.md: no acceptance block under \"An example\"")
	}
	if err := os.WriteFile(filepath.Join(dir, "description.accepted.json"), []byte(accept[1]), 0o644); err != nil {
		t.Fatal(err)
	}
	code, stdout, _ = runCLI("check", "--root", dir, filepath.Join(dir, "definition.json"), filepath.Join(dir, "description.json"))
	want := "accepted  follows  import writes volume, check repeated: a disk image cannot hold two files of one name\n"
	if code != 0 || stdout != want {
		t.Errorf("with the acceptance: exit %d\n got  %q\n want %q", code, stdout, want)
	}
}

// baselineRun writes the baseline into a directory, with every hold and
// source in place, after change has altered it, and runs check on it.
func baselineRun(t *testing.T, change func(def, intake, audit obj), accepted map[string]string, extra ...string) (int, string, string) {
	t.Helper()
	return baselineCommand(t, "check", change, accepted, extra...)
}

// baselineCommand is baselineRun for any command that reads the files.
func baselineCommand(t *testing.T, command string, change func(def, intake, audit obj), accepted map[string]string, extra ...string) (int, string, string) {
	t.Helper()
	def, intake, audit := loadBaseline(t, "library.json"), loadBaseline(t, "intake.json"), loadBaseline(t, "audit.json")
	dir := t.TempDir()
	evidence(t, dir, def, intake, audit)
	if change != nil {
		change(def, intake, audit)
	}
	writeFiles(t, dir, map[string]obj{"library.json": def, "intake.json": intake, "audit.json": audit})
	for name, text := range accepted {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	args := append([]string{command}, extra...)
	args = append(args, "--root", dir, filepath.Join(dir, "library.json"), filepath.Join(dir, "intake.json"), filepath.Join(dir, "audit.json"))
	return runCLI(args...)
}

func TestACleanRunExitsZeroWithNothingOnStdout(t *testing.T) {
	code, stdout, stderr := baselineRun(t, nil, nil)
	if code != 0 || stdout != "" || !strings.Contains(stderr, "0 findings, 0 accepted") {
		t.Errorf("exit %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func noBelongs(def, _, _ obj) { delete(writeTo(element(def, "update"), "shelf"), "before") }

func TestFindingsGoToStdoutAndTheSummaryToStderr(t *testing.T) {
	code, stdout, stderr := baselineRun(t, noBelongs, nil)
	sameOrder(t, "one break", subjects(stdout), []string{
		"replace-checks-owner  update writes shelf",
		"follows  update writes shelf, check belongs",
	})
	if code != 2 || !strings.Contains(stderr, "2 findings, 0 accepted") || strings.Contains(stdout, "accepted") {
		t.Errorf("exit %d, stderr %q", code, stderr)
	}
}

func TestAnAcceptedFindingIsPrintedLastAndDoesNotFail(t *testing.T) {
	code, stdout, stderr := baselineRun(t, noBelongs, map[string]string{
		"library.accepted.json": `{"accepted": [{"finding": "replace-checks-owner  update writes shelf", "reason": "one reader owns every book"}]}`,
		"intake.accepted.json":  `{"accepted": [{"finding": "follows  update writes shelf, check belongs", "reason": "the code checks; the design will not ask for it"}]}`,
	})
	want := "accepted  replace-checks-owner  update writes shelf: one reader owns every book\n" +
		"accepted  follows  update writes shelf, check belongs: the code checks; the design will not ask for it\n"
	if code != 0 || stdout != want || !strings.Contains(stderr, "0 findings, 2 accepted") {
		t.Errorf("exit %d\n stdout %q\n want   %q\n stderr %q", code, stdout, want, stderr)
	}
}

func TestAStaleAcceptanceIsAFinding(t *testing.T) {
	code, stdout, _ := baselineRun(t, nil, map[string]string{
		"library.accepted.json": `{"accepted": [{"finding": "held  read rejects", "reason": "tested by hand"}]}`,
	})
	if code != 2 || stdout != "stale  held  read rejects: accepted, and no longer found\n" {
		t.Errorf("exit %d, stdout %q", code, stdout)
	}
}

// A finding made on a description file is accepted beside that file. An
// acceptance of it beside the definition file matches nothing.
func TestAnAcceptanceBelongsBesideItsFile(t *testing.T) {
	code, stdout, _ := baselineRun(t, noBelongs, map[string]string{
		"library.accepted.json": `{"accepted": [
			{"finding": "replace-checks-owner  update writes shelf", "reason": "one reader owns every book"},
			{"finding": "follows  update writes shelf, check belongs", "reason": "in the wrong file"}]}`,
	})
	sameOrder(t, "an acceptance in the wrong file", subjects(stdout), []string{
		"follows  update writes shelf, check belongs",
		"stale  follows  update writes shelf, check belongs",
		"accepted  replace-checks-owner  update writes shelf",
	})
	if code != 2 {
		t.Errorf("exit %d, want 2", code)
	}
}

func TestAChildOfTheTopElementNeedsADescriptionFile(t *testing.T) {
	def := loadBaseline(t, "library.json")
	dir := t.TempDir()
	evidence(t, dir, def)
	writeFiles(t, dir, map[string]obj{"library.json": def})
	code, stdout, _ := runCLI("check", "--root", dir, filepath.Join(dir, "library.json"))
	sameOrder(t, "the definition file alone", subjects(stdout), []string{"follows  element audit", "follows  element intake"})
	if code != 2 {
		t.Errorf("exit %d, want 2", code)
	}
}

// A refused file stops the run: every reason for every refused file
// goes to standard error as "<file>: <reason>", and no check is made.
func TestARefusedFileStopsTheRun(t *testing.T) {
	code, stdout, stderr := baselineRun(t, func(def, intake, _ obj) {
		noBelongs(def, nil, nil)
		place(def, "shelf")["colour"] = "red"
		place(def, "parsed")["keyed"] = false
		element(intake, "notify")["include"] = []any{"log", "post"}
	}, nil)
	if code != 2 || stdout != "" {
		t.Errorf("exit %d, stdout %q; want 2 and nothing printed", code, stdout)
	}
	for _, want := range []string{`library.json: place "shelf" has the key "colour"`, `library.json: place "parsed": "keyed" is written as false`, `intake.json: element "notify" names the child "post"`} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr does not have %q:\n%s", want, stderr)
		}
	}
	if strings.Contains(stderr, "finding") {
		t.Errorf("a refused run makes no checks, and stderr reports findings:\n%s", stderr)
	}

	code, _, stderr = baselineRun(t, nil, map[string]string{"audit.accepted.json": `{"accepted": [{"finding": "nonsense  x", "reason": "y"}]}`})
	if code != 2 || !strings.Contains(stderr, "audit.accepted.json: ") {
		t.Errorf("a bad acceptances file: exit %d, stderr %q", code, stderr)
	}

	code, _, stderr = runCLI("check", filepath.Join(t.TempDir(), "missing.json"))
	if code != 2 || !strings.Contains(stderr, "missing.json: the file cannot be read") {
		t.Errorf("a file that is not there: exit %d, stderr %q", code, stderr)
	}

	// view reads the files the same way, and draws no page from a
	// refused one.
	code, stdout, stderr = baselineCommand(t, "view", func(def, _, _ obj) { place(def, "shelf")["colour"] = "red" }, nil)
	if code != 2 || stdout != "" || !strings.Contains(stderr, `library.json: place "shelf" has the key "colour"`) {
		t.Errorf("view of a refused file: exit %d, stdout %d bytes, stderr %q", code, len(stdout), stderr)
	}
}

func TestADescriptionFileGivenFirstIsRefused(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]obj{"intake.json": loadBaseline(t, "intake.json")})
	code, _, stderr := runCLI("check", filepath.Join(dir, "intake.json"))
	if code != 2 || !strings.Contains(stderr, "it is a description file, and a definition file was expected here") {
		t.Errorf("exit %d, stderr %q", code, stderr)
	}
}

func TestACommandGivenWronglyExitsOne(t *testing.T) {
	for _, args := range [][]string{{}, {"check"}, {"check", "--format", "xml", "a.json"}, {"check", "--nonsense"}, {"view"}, {"view", "--nonsense"}, {"frobnicate"}, {"axioms", "extra"}, {"checks", "extra"}} {
		if code, _, _ := runCLI(args...); code != 1 {
			t.Errorf("control %s: exit %d, want 1", strings.Join(args, " "), code)
		}
	}
}

func TestJSONFormatPrintsTheSameFindings(t *testing.T) {
	code, stdout, _ := baselineRun(t, noBelongs, map[string]string{
		"library.accepted.json": `{"accepted": [{"finding": "replace-checks-owner  update writes shelf", "reason": "one reader owns every book"}]}`,
	}, "--format", "json")
	var out struct {
		Findings []struct{ Check, Subject, Message string }
		Accepted []struct{ Check, Subject, Reason string }
	}
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, stdout)
	}
	if code != 2 || len(out.Findings) != 1 || len(out.Accepted) != 1 ||
		out.Findings[0].Check != "follows" || out.Findings[0].Subject != "update writes shelf, check belongs" || out.Findings[0].Message == "" ||
		out.Accepted[0].Check != "replace-checks-owner" || out.Accepted[0].Reason != "one reader owns every book" {
		t.Errorf("exit %d, got %+v", code, out)
	}
}

// A finding in JSON names its element and place in their own fields,
// so a reader never parses them out of the subject.
func TestAFindingNamesItsElementAndPlace(t *testing.T) {
	_, stdout, _ := baselineRun(t, func(def, intake, audit obj) {
		noBelongs(def, intake, audit)
		delete(place(def, "shelf"), "invalid")
	}, nil, "--format", "json")
	var out struct {
		Findings []struct{ Check, Subject, Element, Place string }
	}
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, stdout)
	}
	want := map[string][2]string{
		"replace-checks-owner  update writes shelf":   {"update", "shelf"},
		"reject-values  place shelf":                  {"", "shelf"},
		"follows  update writes shelf, check belongs": {"update", "shelf"},
	}
	seen := 0
	for _, f := range out.Findings {
		w, ok := want[f.Check+"  "+f.Subject]
		if !ok {
			continue
		}
		seen++
		if f.Element != w[0] || f.Place != w[1] {
			t.Errorf("%s  %s: element %q place %q, want %q %q", f.Check, f.Subject, f.Element, f.Place, w[0], w[1])
		}
	}
	if seen != len(want) {
		t.Errorf("found %d of the %d findings in %s", seen, len(want), stdout)
	}
}

func TestChecksListsEveryCheckOfTheFirstQuestion(t *testing.T) {
	code, stdout, _ := runCLI("checks")
	lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
	if code != 0 || len(lines) != len(soundChecks) {
		t.Fatalf("exit %d, %d lines, want %d", code, len(lines), len(soundChecks))
	}
	for i, id := range soundChecks {
		if !strings.HasPrefix(lines[i], id+" ") {
			t.Errorf("line %d is %q, want it to begin with %s", i+1, lines[i], id)
		}
	}
}
