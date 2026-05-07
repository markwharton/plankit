package git

import (
	"errors"
	"strings"
	"testing"
)

func TestCheckCleanTree_clean(t *testing.T) {
	stub := func(dir string, args ...string) (string, error) {
		return "", nil
	}

	if err := CheckCleanTree(stub, ""); err != nil {
		t.Errorf("expected nil error for clean tree, got: %v", err)
	}
}

func TestCheckCleanTree_cleanWhitespaceOnly(t *testing.T) {
	stub := func(dir string, args ...string) (string, error) {
		return "  \n\t\n  ", nil
	}

	if err := CheckCleanTree(stub, ""); err != nil {
		t.Errorf("expected nil error for whitespace-only output, got: %v", err)
	}
}

func TestCheckCleanTree_dirty(t *testing.T) {
	stub := func(dir string, args ...string) (string, error) {
		return " M main.go\n?? untracked.txt\n", nil
	}

	err := CheckCleanTree(stub, "")
	if err == nil {
		t.Fatal("expected error for dirty tree, got nil")
	}
	if !errors.Is(err, ErrDirtyTree) {
		t.Errorf("expected ErrDirtyTree, got: %v", err)
	}
}

func TestCheckCleanTree_gitFailure(t *testing.T) {
	gitErr := errors.New("not a git repository")
	stub := func(dir string, args ...string) (string, error) {
		return "", gitErr
	}

	err := CheckCleanTree(stub, "")
	if err == nil {
		t.Fatal("expected error when git command fails, got nil")
	}
	if !errors.Is(err, gitErr) {
		t.Errorf("expected wrapped git error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "git status failed") {
		t.Errorf("expected 'git status failed' prefix, got: %v", err)
	}
}

func TestCheckCleanTree_passesDir(t *testing.T) {
	var gotDir string
	stub := func(dir string, args ...string) (string, error) {
		gotDir = dir
		return "", nil
	}

	CheckCleanTree(stub, "/some/project")
	if gotDir != "/some/project" {
		t.Errorf("expected dir '/some/project', got %q", gotDir)
	}
}

func TestCheckCleanTree_passesCorrectArgs(t *testing.T) {
	var gotArgs []string
	stub := func(dir string, args ...string) (string, error) {
		gotArgs = args
		return "", nil
	}

	CheckCleanTree(stub, "")
	if len(gotArgs) != 2 || gotArgs[0] != "status" || gotArgs[1] != "--porcelain" {
		t.Errorf("expected args [status --porcelain], got %v", gotArgs)
	}
}
