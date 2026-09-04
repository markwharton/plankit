package version

import (
	"fmt"
	"strconv"
	"strings"
)

// Semver is a parsed semantic version. PreRelease and Build keep their
// original dot-separated strings; empty means not present.
type Semver struct {
	Major, Minor, Patch int
	PreRelease          string
	Build               string
}

// Bump levels for Semver.Bump.
const (
	BumpPatch = iota + 1
	BumpMinor
	BumpMajor
)

// ParseSemver parses a semantic version, accepting an optional "v"
// prefix (as git tags carry). ok is false for anything the spec rejects.
func ParseSemver(s string) (Semver, bool) {
	s = strings.TrimPrefix(s, "v")

	var build string
	if idx := strings.IndexByte(s, '+'); idx >= 0 {
		build = s[idx+1:]
		s = s[:idx]
		if !validBuildMetadata(build) {
			return Semver{}, false
		}
	}
	var preRelease string
	if idx := strings.IndexByte(s, '-'); idx >= 0 {
		preRelease = s[idx+1:]
		s = s[:idx]
		if !validPreRelease(preRelease) {
			return Semver{}, false
		}
	}
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
	return Semver{Major: maj, Minor: min, Patch: pat, PreRelease: preRelease, Build: build}, true
}

// Bump returns a new Semver with the given level incremented and lower
// fields zeroed. Pre-release and build metadata drop: the bumped version
// is a fresh release at that level.
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

// String renders with the "v" prefix: vX.Y.Z[-pre][+build].
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

// parseNumericID parses a non-negative integer with no leading zeros.
func parseNumericID(s string) (int, bool) {
	if s == "" || (len(s) > 1 && s[0] == '0') {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}

// validPreRelease validates dot-separated pre-release identifiers:
// non-empty, [0-9A-Za-z-] only, numeric identifiers without leading zeros.
func validPreRelease(s string) bool {
	if s == "" {
		return false
	}
	for _, id := range strings.Split(s, ".") {
		if id == "" || !allAlphanumericHyphen(id) {
			return false
		}
		if isNumeric(id) && len(id) > 1 && id[0] == '0' {
			return false
		}
	}
	return true
}

// validBuildMetadata validates dot-separated build identifiers:
// non-empty, [0-9A-Za-z-] only.
func validBuildMetadata(s string) bool {
	if s == "" {
		return false
	}
	for _, id := range strings.Split(s, ".") {
		if id == "" || !allAlphanumericHyphen(id) {
			return false
		}
	}
	return true
}

func allAlphanumericHyphen(s string) bool {
	for _, c := range s {
		switch {
		case c >= '0' && c <= '9':
		case c >= 'a' && c <= 'z':
		case c >= 'A' && c <= 'Z':
		case c == '-':
		default:
			return false
		}
	}
	return true
}

func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}
