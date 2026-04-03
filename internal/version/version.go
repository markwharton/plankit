// Package version provides build version information.
//
// The version is determined by the build path:
//   - go install @latest: Go embeds the module version via debug.ReadBuildInfo()
//   - make build VERSION=x.y.z: ldflags override sets a specific version
//   - make build: ldflags sets "dev" for development builds
package version

import "runtime/debug"

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
