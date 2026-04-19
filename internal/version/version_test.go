package version

import "testing"

func TestVersion_LdflagsInjected(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"v-prefixed semver", "v0.1.0", "0.1.0"},
		{"bare semver", "0.1.0", "0.1.0"},
		{"v-prefixed with prerelease", "v1.2.3-alpha", "1.2.3-alpha"},
		{"bare with prerelease", "1.2.3-alpha", "1.2.3-alpha"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orig := version
			version = tt.in
			defer func() { version = orig }()
			if got := Version(); got != tt.want {
				t.Errorf("Version() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestVersion_FallbackToDev(t *testing.T) {
	orig := version
	version = ""
	defer func() { version = orig }()

	// No ldflags set. Under `go test`, debug.ReadBuildInfo().Main.Version
	// is "(devel)", which Version() treats as no real version and falls
	// back to "dev".
	if got := Version(); got != "dev" {
		t.Errorf("Version() with no ldflags = %q, want %q", got, "dev")
	}
}

func TestParseSemver(t *testing.T) {
	tests := []struct {
		input string
		want  Semver
		ok    bool
	}{
		// Core versions.
		{"1.0.0", Semver{1, 0, 0, "", ""}, true},
		{"v1.2.3", Semver{1, 2, 3, "", ""}, true},
		{"0.0.0", Semver{0, 0, 0, "", ""}, true},
		{"v10.20.30", Semver{10, 20, 30, "", ""}, true},

		// Pre-release.
		{"1.0.0-alpha", Semver{1, 0, 0, "alpha", ""}, true},
		{"1.0.0-alpha.1", Semver{1, 0, 0, "alpha.1", ""}, true},
		{"1.0.0-0.3.7", Semver{1, 0, 0, "0.3.7", ""}, true},
		{"1.0.0-x.7.z.92", Semver{1, 0, 0, "x.7.z.92", ""}, true},
		{"1.0.0-alpha-beta", Semver{1, 0, 0, "alpha-beta", ""}, true},
		{"v1.0.0-rc.1", Semver{1, 0, 0, "rc.1", ""}, true},

		// Build metadata.
		{"1.0.0+build", Semver{1, 0, 0, "", "build"}, true},
		{"1.0.0+20130313144700", Semver{1, 0, 0, "", "20130313144700"}, true},
		{"1.0.0+build.1.2", Semver{1, 0, 0, "", "build.1.2"}, true},
		{"1.0.0+001", Semver{1, 0, 0, "", "001"}, true}, // leading zeros OK in build

		// Pre-release + build.
		{"1.0.0-alpha+001", Semver{1, 0, 0, "alpha", "001"}, true},
		{"1.0.0-beta+exp.sha.5114f85", Semver{1, 0, 0, "beta", "exp.sha.5114f85"}, true},
		{"v1.0.0-rc.1+build.123", Semver{1, 0, 0, "rc.1", "build.123"}, true},

		// Invalid: not semver.
		{"invalid", Semver{}, false},
		{"v1.2", Semver{}, false},
		{"v1.2.abc", Semver{}, false},
		{"", Semver{}, false},
		{"1.0", Semver{}, false},
		{"1", Semver{}, false},

		// Invalid: leading zeros in core (spec rule 2).
		{"01.0.0", Semver{}, false},
		{"0.01.0", Semver{}, false},
		{"0.0.01", Semver{}, false},
		{"00.0.0", Semver{}, false},

		// Invalid: negative numbers.
		{"-1.0.0", Semver{}, false},

		// Invalid: leading zeros in numeric pre-release identifier.
		{"1.0.0-01", Semver{}, false},
		{"1.0.0-alpha.01", Semver{}, false},

		// Invalid: empty identifiers.
		{"1.0.0-", Semver{}, false},
		{"1.0.0+", Semver{}, false},
		{"1.0.0-alpha.", Semver{}, false},
		{"1.0.0-alpha..1", Semver{}, false},
		{"1.0.0+build.", Semver{}, false},

		// Invalid: bad characters.
		{"1.0.0-alpha@1", Semver{}, false},
		{"1.0.0+build!1", Semver{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, ok := ParseSemver(tt.input)
			if ok != tt.ok || got != tt.want {
				t.Errorf("ParseSemver(%q) = (%v, %v), want (%v, %v)",
					tt.input, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestSemver_String(t *testing.T) {
	tests := []struct {
		v    Semver
		want string
	}{
		{Semver{1, 2, 3, "", ""}, "v1.2.3"},
		{Semver{0, 0, 0, "", ""}, "v0.0.0"},
		{Semver{10, 0, 0, "", ""}, "v10.0.0"},
		{Semver{1, 0, 0, "alpha", ""}, "v1.0.0-alpha"},
		{Semver{1, 0, 0, "alpha.1", ""}, "v1.0.0-alpha.1"},
		{Semver{1, 0, 0, "", "build.123"}, "v1.0.0+build.123"},
		{Semver{1, 0, 0, "beta", "exp.sha.5114f85"}, "v1.0.0-beta+exp.sha.5114f85"},
	}
	for _, tt := range tests {
		if got := tt.v.String(); got != tt.want {
			t.Errorf("Semver%v.String() = %q, want %q", tt.v, got, tt.want)
		}
	}
}

func TestSemver_String_roundTrip(t *testing.T) {
	inputs := []string{
		"v1.0.0",
		"v1.2.3-alpha",
		"v1.0.0-alpha.1",
		"v1.0.0+build",
		"v1.0.0-rc.1+build.123",
		"v0.0.0",
	}
	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			v, ok := ParseSemver(input)
			if !ok {
				t.Fatalf("ParseSemver(%q) failed", input)
			}
			if got := v.String(); got != input {
				t.Errorf("round-trip: %q → %v → %q", input, v, got)
			}
		})
	}
}

func TestSemver_Bump(t *testing.T) {
	tests := []struct {
		name string
		v    Semver
		bump int
		want Semver
	}{
		{"patch", Semver{Major: 1, Minor: 2, Patch: 3}, BumpPatch, Semver{Major: 1, Minor: 2, Patch: 4}},
		{"minor", Semver{Major: 1, Minor: 2, Patch: 3}, BumpMinor, Semver{Major: 1, Minor: 3}},
		{"major", Semver{Major: 1, Minor: 2, Patch: 3}, BumpMajor, Semver{Major: 2}},
		{"patch from zero", Semver{}, BumpPatch, Semver{Patch: 1}},
		{"minor from zero", Semver{}, BumpMinor, Semver{Minor: 1}},
		{"major from zero", Semver{}, BumpMajor, Semver{Major: 1}},
		{"minor resets patch", Semver{Major: 1, Minor: 2, Patch: 5}, BumpMinor, Semver{Major: 1, Minor: 3}},
		{"major resets minor and patch", Semver{Major: 1, Minor: 2, Patch: 5}, BumpMajor, Semver{Major: 2}},
		{"unknown level returns input", Semver{Major: 1, Minor: 2, Patch: 3}, 999, Semver{Major: 1, Minor: 2, Patch: 3}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.Bump(tt.bump); got != tt.want {
				t.Errorf("(%v).Bump(%d) = %v, want %v", tt.v, tt.bump, got, tt.want)
			}
		})
	}
}

func TestSemver_Compare(t *testing.T) {
	tests := []struct {
		name string
		a, b Semver
		want int
	}{
		// Core version comparison.
		{"equal", Semver{1, 0, 0, "", ""}, Semver{1, 0, 0, "", ""}, 0},
		{"major greater", Semver{2, 0, 0, "", ""}, Semver{1, 9, 9, "", ""}, 1},
		{"major less", Semver{0, 9, 0, "", ""}, Semver{1, 0, 0, "", ""}, -1},
		{"minor greater", Semver{1, 2, 0, "", ""}, Semver{1, 1, 0, "", ""}, 1},
		{"minor less", Semver{1, 0, 0, "", ""}, Semver{1, 1, 0, "", ""}, -1},
		{"patch greater", Semver{1, 0, 2, "", ""}, Semver{1, 0, 1, "", ""}, 1},
		{"patch less", Semver{1, 0, 0, "", ""}, Semver{1, 0, 1, "", ""}, -1},

		// Pre-release vs release.
		{"prerelease < release", Semver{1, 0, 0, "alpha", ""}, Semver{1, 0, 0, "", ""}, -1},
		{"release > prerelease", Semver{1, 0, 0, "", ""}, Semver{1, 0, 0, "alpha", ""}, 1},

		// Pre-release ordering (spec rule 11 example).
		{"alpha < alpha.1", Semver{1, 0, 0, "alpha", ""}, Semver{1, 0, 0, "alpha.1", ""}, -1},
		{"alpha.1 < alpha.beta", Semver{1, 0, 0, "alpha.1", ""}, Semver{1, 0, 0, "alpha.beta", ""}, -1},
		{"alpha.beta < beta", Semver{1, 0, 0, "alpha.beta", ""}, Semver{1, 0, 0, "beta", ""}, -1},
		{"beta < beta.2", Semver{1, 0, 0, "beta", ""}, Semver{1, 0, 0, "beta.2", ""}, -1},
		{"beta.2 < beta.11", Semver{1, 0, 0, "beta.2", ""}, Semver{1, 0, 0, "beta.11", ""}, -1},
		{"beta.11 < rc.1", Semver{1, 0, 0, "beta.11", ""}, Semver{1, 0, 0, "rc.1", ""}, -1},

		// Numeric vs alphanumeric: numeric sorts first.
		{"numeric < alpha", Semver{1, 0, 0, "1", ""}, Semver{1, 0, 0, "alpha", ""}, -1},

		// Numeric comparison (not lexical).
		{"2 < 11 numerically", Semver{1, 0, 0, "2", ""}, Semver{1, 0, 0, "11", ""}, -1},

		// Build metadata ignored.
		{"build ignored equal", Semver{1, 0, 0, "", "build.1"}, Semver{1, 0, 0, "", "build.2"}, 0},
		{"build ignored with prerelease", Semver{1, 0, 0, "alpha", "1"}, Semver{1, 0, 0, "alpha", "2"}, 0},

		// Pre-release equal.
		{"same prerelease", Semver{1, 0, 0, "alpha.1", ""}, Semver{1, 0, 0, "alpha.1", ""}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.Compare(tt.b); got != tt.want {
				t.Errorf("%v.Compare(%v) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

// TestSemver_Compare_specExample verifies the full precedence example from the spec:
// 1.0.0-alpha < 1.0.0-alpha.1 < 1.0.0-alpha.beta < 1.0.0-beta < 1.0.0-beta.2 < 1.0.0-beta.11 < 1.0.0-rc.1 < 1.0.0
func TestSemver_Compare_specExample(t *testing.T) {
	versions := []string{
		"1.0.0-alpha",
		"1.0.0-alpha.1",
		"1.0.0-alpha.beta",
		"1.0.0-beta",
		"1.0.0-beta.2",
		"1.0.0-beta.11",
		"1.0.0-rc.1",
		"1.0.0",
	}
	for i := 0; i < len(versions)-1; i++ {
		a, aOK := ParseSemver(versions[i])
		b, bOK := ParseSemver(versions[i+1])
		if !aOK || !bOK {
			t.Fatalf("parse failed: %q=%v, %q=%v", versions[i], aOK, versions[i+1], bOK)
		}
		if c := a.Compare(b); c != -1 {
			t.Errorf("%s.Compare(%s) = %d, want -1", versions[i], versions[i+1], c)
		}
		if c := b.Compare(a); c != 1 {
			t.Errorf("%s.Compare(%s) = %d, want 1", versions[i+1], versions[i], c)
		}
	}
}

// TestParseNumericID tests the internal numeric identifier validation.
func TestParseNumericID(t *testing.T) {
	tests := []struct {
		input string
		want  int
		ok    bool
	}{
		{"0", 0, true},
		{"1", 1, true},
		{"123", 123, true},
		{"01", 0, false},  // leading zero
		{"00", 0, false},  // leading zero
		{"", 0, false},    // empty
		{"-1", 0, false},  // negative
		{"abc", 0, false}, // not a number
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, ok := parseNumericID(tt.input)
			if ok != tt.ok || got != tt.want {
				t.Errorf("parseNumericID(%q) = (%d, %v), want (%d, %v)",
					tt.input, got, ok, tt.want, tt.ok)
			}
		})
	}
}
