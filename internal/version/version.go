// Package version provides build version information and semantic versioning
// per the Semantic Versioning 2.0.0 specification (https://semver.org).
//
// The build version is determined by the build path:
//   - go install @latest: Go embeds the module version via debug.ReadBuildInfo()
//   - make build VERSION=x.y.z: ldflags override sets a specific version
//   - make build: ldflags sets "dev" for development builds
package version

import (
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"
)

// version is set at build time via -ldflags for development and release builds.
// Empty means no ldflags were set (go install path).
var version string

// Version returns the build version as a bare semver string (no leading "v").
// A leading "v" is stripped so all three build paths report consistently —
// the release workflow already strips it, but go install surfaces tag names verbatim.
func Version() string {
	v := "dev"
	if version != "" {
		v = version
	} else if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		v = info.Main.Version
	}
	return strings.TrimPrefix(v, "v")
}

// IsDevBuild reports whether v is a development build (empty or "dev").
func IsDevBuild(v string) bool {
	return v == "" || v == "dev"
}

// Semver holds the components of a semantic version per semver.org/spec/v2.0.0.
//
// PreRelease and Build are stored as their original dot-separated strings.
// Empty string means not present.
type Semver struct {
	Major, Minor, Patch int
	PreRelease          string // dot-separated pre-release identifiers (e.g. "alpha.1")
	Build               string // dot-separated build metadata identifiers (e.g. "build.123")
}

// ParseSemver parses a semantic version string into a Semver.
// Accepts an optional "v" prefix (common in Git tags).
// Returns ok=false if the input is not a valid semver string.
func ParseSemver(s string) (Semver, bool) {
	s = strings.TrimPrefix(s, "v")

	// Split off build metadata (after '+').
	var build string
	if idx := strings.IndexByte(s, '+'); idx >= 0 {
		build = s[idx+1:]
		s = s[:idx]
		if !validBuildMetadata(build) {
			return Semver{}, false
		}
	}

	// Split off pre-release (after '-').
	var preRelease string
	if idx := strings.IndexByte(s, '-'); idx >= 0 {
		preRelease = s[idx+1:]
		s = s[:idx]
		if !validPreRelease(preRelease) {
			return Semver{}, false
		}
	}

	// Parse core version X.Y.Z.
	parts := strings.SplitN(s, ".", 3)
	if len(parts) != 3 {
		return Semver{}, false
	}
	maj, ok1 := parseNumericID(parts[0])
	min, ok2 := parseNumericID(parts[1])
	pat, ok3 := parseNumericID(parts[2])
	if !ok1 || !ok2 || !ok3 {
		return Semver{}, false
	}

	return Semver{
		Major: maj, Minor: min, Patch: pat,
		PreRelease: preRelease,
		Build:      build,
	}, true
}

// Bump level constants for Semver.Bump.
const (
	BumpPatch = iota + 1
	BumpMinor
	BumpMajor
)

// Bump returns a new Semver with the given level incremented and lower
// fields reset to zero. Pre-release and build metadata are dropped, since
// the bumped version represents a fresh release of that level.
func (v Semver) Bump(level int) Semver {
	switch level {
	case BumpMajor:
		return Semver{Major: v.Major + 1}
	case BumpMinor:
		return Semver{Major: v.Major, Minor: v.Minor + 1}
	case BumpPatch:
		return Semver{Major: v.Major, Minor: v.Minor, Patch: v.Patch + 1}
	default:
		return v
	}
}

// String returns the version with a "v" prefix: "vX.Y.Z[-prerelease][+build]".
func (v Semver) String() string {
	s := fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.PreRelease != "" {
		s += "-" + v.PreRelease
	}
	if v.Build != "" {
		s += "+" + v.Build
	}
	return s
}

// Compare returns -1, 0, or 1 per semver precedence rules (spec rule 11).
// Build metadata is ignored (spec rule 10).
func (v Semver) Compare(other Semver) int {
	// 1. Compare core version numerically.
	if c := intCmp(v.Major, other.Major); c != 0 {
		return c
	}
	if c := intCmp(v.Minor, other.Minor); c != 0 {
		return c
	}
	if c := intCmp(v.Patch, other.Patch); c != 0 {
		return c
	}

	// 2. Pre-release precedence.
	// A version without pre-release has higher precedence.
	vPre := v.PreRelease != ""
	oPre := other.PreRelease != ""
	if !vPre && !oPre {
		return 0
	}
	if !vPre {
		return 1 // v is release, other is pre-release
	}
	if !oPre {
		return -1 // v is pre-release, other is release
	}

	// 3. Compare pre-release identifiers left to right.
	vIDs := strings.Split(v.PreRelease, ".")
	oIDs := strings.Split(other.PreRelease, ".")
	n := len(vIDs)
	if len(oIDs) < n {
		n = len(oIDs)
	}
	for i := 0; i < n; i++ {
		if c := comparePreReleaseID(vIDs[i], oIDs[i]); c != 0 {
			return c
		}
	}

	// 4. Larger set of identifiers has higher precedence.
	return intCmp(len(vIDs), len(oIDs))
}

// parseNumericID parses a non-negative integer with no leading zeros (spec rule 2).
func parseNumericID(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	if len(s) > 1 && s[0] == '0' {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}

// validPreRelease validates a dot-separated pre-release string (spec rule 9).
// Each identifier must be non-empty, contain only [0-9A-Za-z-], and
// numeric-only identifiers must not have leading zeros.
func validPreRelease(s string) bool {
	if s == "" {
		return false
	}
	for _, id := range strings.Split(s, ".") {
		if id == "" {
			return false
		}
		if !allAlphanumericHyphen(id) {
			return false
		}
		if allDigits(id) && len(id) > 1 && id[0] == '0' {
			return false
		}
	}
	return true
}

// validBuildMetadata validates a dot-separated build metadata string (spec rule 10).
// Each identifier must be non-empty and contain only [0-9A-Za-z-].
// Leading zeros are allowed in build metadata (unlike pre-release).
func validBuildMetadata(s string) bool {
	if s == "" {
		return false
	}
	for _, id := range strings.Split(s, ".") {
		if id == "" {
			return false
		}
		if !allAlphanumericHyphen(id) {
			return false
		}
	}
	return true
}

// comparePreReleaseID compares two pre-release identifiers per spec rule 11:
//   - Both numeric: compare as integers
//   - Both non-numeric: compare lexically (ASCII sort)
//   - Numeric always sorts before non-numeric
func comparePreReleaseID(a, b string) int {
	aDigits := allDigits(a)
	bDigits := allDigits(b)
	switch {
	case aDigits && bDigits:
		ai, _ := strconv.Atoi(a)
		bi, _ := strconv.Atoi(b)
		return intCmp(ai, bi)
	case aDigits:
		return -1
	case bDigits:
		return 1
	default:
		if a < b {
			return -1
		}
		if a > b {
			return 1
		}
		return 0
	}
}

func intCmp(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func allAlphanumericHyphen(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '-') {
			return false
		}
	}
	return true
}

func allDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return len(s) > 0
}

// VerboseInfo returns additional build information (Go version, build date, and commit).
// Returns empty string if build info is unavailable.
func VerboseInfo() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	goVer := info.GoVersion
	buildDate := "unknown"
	commitSHA := "unknown"
	dirty := false
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.time":
			buildDate = s.Value
		case "vcs.revision":
			commitSHA = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	if len(commitSHA) > 7 {
		commitSHA = commitSHA[:7]
	}
	if dirty && IsDevBuild(Version()) {
		commitSHA += " (dirty)"
	}
	return fmt.Sprintf("  go: %s\n  build: %s\n  commit: %s", goVer, buildDate, commitSHA)
}
