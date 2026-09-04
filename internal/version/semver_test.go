package version

import "testing"

func TestParseSemver(t *testing.T) {
	valid := map[string]string{
		"v1.2.3": "v1.2.3", "1.2.3": "v1.2.3",
		"v0.0.0": "v0.0.0", "v1.0.0-alpha.1": "v1.0.0-alpha.1",
		"v2.0.0+build.5": "v2.0.0+build.5", "v2.0.0-rc.1+sha.abc": "v2.0.0-rc.1+sha.abc",
	}
	for in, want := range valid {
		sv, ok := ParseSemver(in)
		if !ok || sv.String() != want {
			t.Errorf("ParseSemver(%q) = %v %v, want %s", in, sv, ok, want)
		}
	}
	invalid := []string{"", "v1.2", "v1.2.3.4", "v01.2.3", "v1.2.3-", "v1.2.3-01", "v1.2.3+", "va.b.c", "v1.2.3 "}
	for _, in := range invalid {
		if _, ok := ParseSemver(in); ok {
			t.Errorf("ParseSemver(%q) should fail", in)
		}
	}
}

func TestBump(t *testing.T) {
	base, _ := ParseSemver("v1.2.3-rc.1+b7")
	cases := map[int]string{BumpPatch: "v1.2.4", BumpMinor: "v1.3.0", BumpMajor: "v2.0.0"}
	for level, want := range cases {
		if got := base.Bump(level).String(); got != want {
			t.Errorf("Bump(%d) = %s, want %s (pre-release and build must drop)", level, got, want)
		}
	}
}
