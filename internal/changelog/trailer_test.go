package changelog

import (
	"errors"
	"testing"

	"github.com/markwharton/plankit/internal/version"
)

func TestReadReleaseTagTrailer(t *testing.T) {
	t.Run("commit with Release-Tag trailer", func(t *testing.T) {
		gitExec := func(dir string, args ...string) (string, error) {
			return "v0.8.0\n", nil
		}
		parsed, tag, err := ReadReleaseTagTrailer(gitExec)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := version.Semver{Major: 0, Minor: 8, Patch: 0}
		if parsed != want {
			t.Errorf("parsed = %v, want %v", parsed, want)
		}
		if tag != "v0.8.0" {
			t.Errorf("tag = %q, want %q", tag, "v0.8.0")
		}
	})

	t.Run("commit without Release-Tag trailer", func(t *testing.T) {
		gitExec := func(dir string, args ...string) (string, error) {
			return "\n", nil
		}
		_, _, err := ReadReleaseTagTrailer(gitExec)
		if !errors.Is(err, ErrNoTrailer) {
			t.Errorf("err = %v, want ErrNoTrailer", err)
		}
	})

	t.Run("git log failure", func(t *testing.T) {
		gitExec := func(dir string, args ...string) (string, error) {
			return "", errors.New("fatal: not a git repository")
		}
		_, _, err := ReadReleaseTagTrailer(gitExec)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if errors.Is(err, ErrNoTrailer) {
			t.Error("error should not be ErrNoTrailer for git failure")
		}
	})

	t.Run("trailing whitespace in trailer value", func(t *testing.T) {
		gitExec := func(dir string, args ...string) (string, error) {
			return "  v1.2.3  \n", nil
		}
		parsed, tag, err := ReadReleaseTagTrailer(gitExec)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := version.Semver{Major: 1, Minor: 2, Patch: 3}
		if parsed != want {
			t.Errorf("parsed = %v, want %v", parsed, want)
		}
		if tag != "v1.2.3" {
			t.Errorf("tag = %q, want %q", tag, "v1.2.3")
		}
	})

	t.Run("empty output from git log", func(t *testing.T) {
		gitExec := func(dir string, args ...string) (string, error) {
			return "", nil
		}
		_, _, err := ReadReleaseTagTrailer(gitExec)
		if !errors.Is(err, ErrNoTrailer) {
			t.Errorf("err = %v, want ErrNoTrailer", err)
		}
	})

	t.Run("invalid semver in trailer", func(t *testing.T) {
		gitExec := func(dir string, args ...string) (string, error) {
			return "not-a-version\n", nil
		}
		_, _, err := ReadReleaseTagTrailer(gitExec)
		if !errors.Is(err, ErrInvalidTrailer) {
			t.Errorf("err = %v, want ErrInvalidTrailer", err)
		}
	})

	t.Run("missing v prefix in trailer", func(t *testing.T) {
		gitExec := func(dir string, args ...string) (string, error) {
			return "1.2.3\n", nil
		}
		_, _, err := ReadReleaseTagTrailer(gitExec)
		if !errors.Is(err, ErrInvalidTrailer) {
			t.Errorf("err = %v, want ErrInvalidTrailer", err)
		}
	})
}
