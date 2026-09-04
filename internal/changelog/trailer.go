package changelog

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/markwharton/plankit/internal/git"
	"github.com/markwharton/plankit/internal/version"
)

// TrailerKey is the git trailer pk changelog writes into release commits
// and pk release / pk changelog --undo read back. The value is the
// pending version that pk release turns into a real git tag.
const TrailerKey = "Release-Tag"

// ErrNoTrailer reports that HEAD has no Release-Tag trailer. The message
// is deliberately neutral: pk release wraps it with "run pk changelog
// first", while --undo prints it as-is.
var ErrNoTrailer = errors.New("no Release-Tag trailer on HEAD")

// ErrInvalidTrailer reports a trailer value that does not round-trip
// through version.ParseSemver.
var ErrInvalidTrailer = errors.New("Release-Tag trailer value is not valid semver")

// ReadReleaseTagTrailer reads and validates the trailer on HEAD: trimmed,
// non-empty, parses as semver, and re-renders to exactly itself (which
// catches trailing garbage and a missing v prefix). The returned string
// equals the parsed form, so callers use either interchangeably.
func ReadReleaseTagTrailer(dir string) (version.Semver, string, error) {
	out, err := git.Exec(dir, "log", "-1", "--format=%(trailers:key="+TrailerKey+",valueonly)", "HEAD")
	if err != nil {
		return version.Semver{}, "", fmt.Errorf("git log failed: %w", err)
	}
	value := strings.TrimSpace(out)
	if value == "" {
		return version.Semver{}, "", ErrNoTrailer
	}
	parsed, ok := version.ParseSemver(value)
	if !ok || parsed.String() != value {
		return version.Semver{}, "", fmt.Errorf("%w: %q", ErrInvalidTrailer, value)
	}
	return parsed, value, nil
}

// readFile and writeFile isolate the two filesystem touches for tests.
func readFile(path string) ([]byte, error) { return os.ReadFile(path) }
func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}
