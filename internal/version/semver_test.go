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

// TestCompare walks the spec's own precedence example, then the strings
// this repository meets: a release against a source build's
// pseudo-version, and build metadata that must not count.
func TestCompare(t *testing.T) {
	ascending := []string{
		"1.0.0-alpha", "1.0.0-alpha.1", "1.0.0-alpha.beta", "1.0.0-beta", "1.0.0-beta.2",
		"1.0.0-beta.11", "1.0.0-rc.1", "1.0.0", "1.0.1", "1.1.0", "2.0.0",
		"1.4.0", // out of order on purpose: checked against its neighbours below
	}
	parse := func(s string) Semver {
		v, ok := ParseSemver(s)
		if !ok {
			t.Fatalf("ParseSemver(%q) failed", s)
		}
		return v
	}
	for i := 1; i < len(ascending)-1; i++ {
		lo, hi := parse(ascending[i-1]), parse(ascending[i])
		if lo.Compare(hi) != -1 || hi.Compare(lo) != 1 || lo.Compare(lo) != 0 {
			t.Errorf("%s < %s not ordered", ascending[i-1], ascending[i])
		}
	}
	cases := []struct {
		a, b string
		want int
	}{
		{"1.4.0", "v1.5.0-0.20260930220111-f79562029de1+dirty", -1},
		{"1.5.0", "v1.5.0-0.20260930220111-f79562029de1+dirty", 1},
		{"1.4.0+build.1", "1.4.0+build.2", 0},
		{"v1.4.0", "1.4.0", 0},
	}
	for _, tc := range cases {
		if got := parse(tc.a).Compare(parse(tc.b)); got != tc.want {
			t.Errorf("Compare(%s, %s) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}
