// Package version provides build version information.
//
// The version is determined by the build path:
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

// Version returns the build version.
// When ldflags are set (make build), returns the injected value.
// When not set (go install), reads the module version from Go's build info.
func Version() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}

// Semver holds the major, minor, and patch components of a semantic version.
type Semver struct {
	Major, Minor, Patch int
}

// ParseSemver parses "vX.Y.Z" or "X.Y.Z" into a Semver.
// Returns ok=false if the input is not a valid semver string.
func ParseSemver(s string) (Semver, bool) {
	s = strings.TrimPrefix(s, "v")
	parts := strings.SplitN(s, ".", 3)
	if len(parts) != 3 {
		return Semver{}, false
	}
	maj, err1 := strconv.Atoi(parts[0])
	min, err2 := strconv.Atoi(parts[1])
	pat, err3 := strconv.Atoi(parts[2])
	if err1 != nil || err2 != nil || err3 != nil {
		return Semver{}, false
	}
	return Semver{Major: maj, Minor: min, Patch: pat}, true
}

// String returns the "vX.Y.Z" representation.
func (v Semver) String() string {
	return fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Compare returns -1, 0, or 1 if v is less than, equal to, or greater than other.
func (v Semver) Compare(other Semver) int {
	pairs := [3][2]int{
		{v.Major, other.Major},
		{v.Minor, other.Minor},
		{v.Patch, other.Patch},
	}
	for _, p := range pairs {
		if p[0] < p[1] {
			return -1
		}
		if p[0] > p[1] {
			return 1
		}
	}
	return 0
}

// VerboseInfo returns additional build information (Go version and build date).
// Returns empty string if build info is unavailable.
func VerboseInfo() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	goVer := info.GoVersion
	buildDate := "unknown"
	for _, s := range info.Settings {
		if s.Key == "vcs.time" {
			buildDate = s.Value
			break
		}
	}
	return fmt.Sprintf("  go: %s\n  build: %s", goVer, buildDate)
}
