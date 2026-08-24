package git

import (
	"errors"
	"testing"
)

func TestDefaultBranch_found(t *testing.T) {
	stub := func(dir string, args ...string) (string, error) {
		return "ref: refs/heads/main\tHEAD\nabc123def456\tHEAD\n", nil
	}

	branch, ok, err := DefaultBranch(stub, "")
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true when origin advertises a HEAD symref")
	}
	if branch != "main" {
		t.Errorf("expected branch 'main', got %q", branch)
	}
}

func TestDefaultBranch_slashedName(t *testing.T) {
	stub := func(dir string, args ...string) (string, error) {
		return "ref: refs/heads/release/main\tHEAD\nabc123\tHEAD\n", nil
	}

	branch, ok, err := DefaultBranch(stub, "")
	if err != nil || !ok {
		t.Fatalf("expected (branch, true, nil), got (%q, %v, %v)", branch, ok, err)
	}
	if branch != "release/main" {
		t.Errorf("expected branch 'release/main', got %q", branch)
	}
}

func TestDefaultBranch_noSymref(t *testing.T) {
	stub := func(dir string, args ...string) (string, error) {
		return "abc123def456\tHEAD\n", nil
	}

	branch, ok, err := DefaultBranch(stub, "")
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if ok {
		t.Error("expected ok=false when output has no ref: line")
	}
	if branch != "" {
		t.Errorf("expected empty branch, got %q", branch)
	}
}

func TestDefaultBranch_symrefForOtherRef(t *testing.T) {
	// A ref: line that is not HEAD's symref is ignored.
	stub := func(dir string, args ...string) (string, error) {
		return "ref: refs/heads/main\tOTHER\n", nil
	}

	branch, ok, err := DefaultBranch(stub, "")
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if ok || branch != "" {
		t.Errorf("expected (\"\", false), got (%q, %v)", branch, ok)
	}
}

func TestDefaultBranch_emptyOutput(t *testing.T) {
	stub := func(dir string, args ...string) (string, error) {
		return "", nil
	}

	branch, ok, err := DefaultBranch(stub, "")
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if ok || branch != "" {
		t.Errorf("expected (\"\", false), got (%q, %v)", branch, ok)
	}
}

func TestDefaultBranch_gitFailure(t *testing.T) {
	gitErr := errors.New("could not read from remote repository")
	stub := func(dir string, args ...string) (string, error) {
		return "", gitErr
	}

	branch, ok, err := DefaultBranch(stub, "")
	if !errors.Is(err, gitErr) {
		t.Errorf("expected the git error back, got: %v", err)
	}
	if ok || branch != "" {
		t.Errorf("expected (\"\", false) on git failure, got (%q, %v)", branch, ok)
	}
}

func TestDefaultBranch_passesDirAndArgs(t *testing.T) {
	var gotDir string
	var gotArgs []string
	stub := func(dir string, args ...string) (string, error) {
		gotDir = dir
		gotArgs = args
		return "ref: refs/heads/main\tHEAD\n", nil
	}

	DefaultBranch(stub, "/some/project")
	if gotDir != "/some/project" {
		t.Errorf("expected dir '/some/project', got %q", gotDir)
	}
	want := []string{"ls-remote", "--symref", "origin", "HEAD"}
	if len(gotArgs) != len(want) {
		t.Fatalf("expected args %v, got %v", want, gotArgs)
	}
	for i := range want {
		if gotArgs[i] != want[i] {
			t.Fatalf("expected args %v, got %v", want, gotArgs)
		}
	}
}
