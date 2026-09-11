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

// TagState says where a Release-Tag trailer's tag stands. A release is
// pending only while its tag is absent: a release commit keeps its
// trailer after it ships, so the trailer alone cannot tell a staged
// release from a finished one.
type TagState int

const (
	TagAbsent    TagState = iota // no such tag: the release is pending
	TagOnHead                    // the tag is on HEAD: the release has been made
	TagElsewhere                 // the tag is on another commit: the trailer conflicts with it
)

// ReleaseTagState reports where tag stands relative to HEAD. An error
// means git could not answer; the state is meaningless then.
func ReleaseTagState(dir, tag string) (TagState, error) {
	existing, err := git.Exec(dir, "tag", "--list", tag)
	if err != nil {
		return TagAbsent, fmt.Errorf("git tag --list failed: %w", err)
	}
	if strings.TrimSpace(existing) == "" {
		return TagAbsent, nil
	}
	tagged, err := git.Exec(dir, "rev-parse", "--verify", "--quiet", tag+"^{commit}")
	if err != nil {
		return TagAbsent, fmt.Errorf("git rev-parse %s failed: %w", tag, err)
	}
	head, err := git.Exec(dir, "rev-parse", "--verify", "--quiet", "HEAD^{commit}")
	if err != nil {
		return TagAbsent, fmt.Errorf("git rev-parse HEAD failed: %w", err)
	}
	if strings.TrimSpace(tagged) == strings.TrimSpace(head) {
		return TagOnHead, nil
	}
	return TagElsewhere, nil
}

// readFile and writeFile isolate the two filesystem touches for tests.
func readFile(path string) ([]byte, error) { return os.ReadFile(path) }
func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}
