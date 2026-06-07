package main

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sampleRules mimics the relevant slice of internal/rules/rules.go: the three
// constants plus comment lines that mention those same identifiers (which must be
// skipped, not rewritten).
const sampleRules = `package rules

const (
	// calibrationModel is the model whose tokenizer charsPerToken was measured against.
	calibrationModel = "seed-model"
	// charsPerToken is the calibrated ratio; chars/4 runs low.
	charsPerToken = 3.1
	// calibrated reports whether charsPerToken came from a measurement.
	calibrated = false
)
`

func TestApplyCalibration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rules.go")
	writeFile(t, path, sampleRules)

	if err := applyCalibration(path, 2.93, "claude-opus-4-8"); err != nil {
		t.Fatalf("applyCalibration: %v", err)
	}
	got := readFile(t, path)

	for _, want := range []string{
		"charsPerToken = 2.93",
		`calibrationModel = "claude-opus-4-8"`,
		"calibrated = true",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	// Old assignment values must be gone.
	for _, gone := range []string{"= 3.1", `"seed-model"`, "= false"} {
		if strings.Contains(got, gone) {
			t.Errorf("stale value %q still present:\n%s", gone, got)
		}
	}
	// Comment lines (which also mention the identifiers) must be preserved verbatim.
	if !strings.Contains(got, "// charsPerToken is the calibrated ratio; chars/4 runs low.") {
		t.Errorf("comment line was mutated:\n%s", got)
	}
	// Result must still be valid Go (applyCalibration runs gofmt, which would fail otherwise).
	if _, err := parser.ParseFile(token.NewFileSet(), path, got, parser.AllErrors); err != nil {
		t.Errorf("output is not valid Go: %v\n%s", err, got)
	}
}

func TestApplyCalibrationMissingConst(t *testing.T) {
	// Source missing the calibrated const => clear error, nothing written silently wrong.
	src := "package rules\n\nconst (\n\tcharsPerToken = 3.1\n\tcalibrationModel = \"x\"\n)\n"
	path := filepath.Join(t.TempDir(), "rules.go")
	writeFile(t, path, src)
	if err := applyCalibration(path, 2.9, "m"); err == nil {
		t.Error("expected error when a constant is absent")
	}
}

func TestFormatRatio(t *testing.T) {
	cases := map[float64]string{2.93: "2.93", 3.0: "3.00", 2.9: "2.90"}
	for in, want := range cases {
		if got := formatRatio(in); got != want {
			t.Errorf("formatRatio(%v) = %q, want %q", in, got, want)
		}
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
