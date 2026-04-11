package changelog

import (
	"errors"
	"fmt"
	"strings"

	"github.com/markwharton/plankit/internal/version"
)

// TrailerKey is the git trailer key pk changelog writes into release commits
// and pk release / pk changelog --undo read back. The value is the pending
// version (e.g. v0.6.2) that pk release will turn into a real git tag.
const TrailerKey = "Release-Tag"

// ErrNoTrailer is returned when HEAD has no Release-Tag trailer. The message
// is intentionally neutral — pk release wraps it with "run 'pk changelog'
// first", but pk changelog --undo (which is trying to unwind a changelog
// commit, not run one) prints it as-is.
var ErrNoTrailer = errors.New("no Release-Tag trailer on HEAD")

// ErrInvalidTrailer is returned when the trailer value exists but doesn't
// round-trip through version.ParseSemver. The %s placeholder is filled with
// the offending value when wrapped via fmt.Errorf.
var ErrInvalidTrailer = errors.New("Release-Tag trailer value is not valid semver")

// ReadReleaseTagTrailer reads the Release-Tag trailer from HEAD and validates
// it. Returns the parsed Semver, the original (canonical) string form, or an
// error. The string form equals parsed.String() — round-trip equality is
// part of the validation, so callers can use either field interchangeably.
//
// Validation:
//  1. git log -1 --format=%(trailers:key=Release-Tag,valueonly) HEAD
//  2. trim whitespace
//  3. non-empty (else ErrNoTrailer)
//  4. parses via version.ParseSemver
//  5. parsed.String() == trimmed value (catches trailing garbage and
//     missing v prefix)
//
// Steps 4 and 5 together yield ErrInvalidTrailer wrapped with the value.
func ReadReleaseTagTrailer(gitExec func(dir string, args ...string) (string, error)) (version.Semver, string, error) {
	out, err := gitExec("", "log", "-1", "--format=%(trailers:key="+TrailerKey+",valueonly)", "HEAD")
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
