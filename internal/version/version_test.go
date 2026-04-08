package version

import "testing"

func TestParseSemver(t *testing.T) {
	tests := []struct {
		input string
		want  Semver
		ok    bool
	}{
		{"v1.2.3", Semver{1, 2, 3}, true},
		{"1.2.3", Semver{1, 2, 3}, true},
		{"v0.0.0", Semver{0, 0, 0}, true},
		{"v10.20.30", Semver{10, 20, 30}, true},
		{"invalid", Semver{}, false},
		{"v1.2", Semver{}, false},
		{"v1.2.abc", Semver{}, false},
		{"", Semver{}, false},
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
		{Semver{1, 2, 3}, "v1.2.3"},
		{Semver{0, 0, 0}, "v0.0.0"},
		{Semver{10, 0, 0}, "v10.0.0"},
	}
	for _, tt := range tests {
		if got := tt.v.String(); got != tt.want {
			t.Errorf("Semver%v.String() = %q, want %q", tt.v, got, tt.want)
		}
	}
}

func TestSemver_Compare(t *testing.T) {
	tests := []struct {
		name string
		a, b Semver
		want int
	}{
		{"equal", Semver{1, 0, 0}, Semver{1, 0, 0}, 0},
		{"major greater", Semver{2, 0, 0}, Semver{1, 9, 9}, 1},
		{"major less", Semver{0, 9, 0}, Semver{1, 0, 0}, -1},
		{"minor greater", Semver{1, 2, 0}, Semver{1, 1, 0}, 1},
		{"minor less", Semver{1, 0, 0}, Semver{1, 1, 0}, -1},
		{"patch greater", Semver{1, 0, 2}, Semver{1, 0, 1}, 1},
		{"patch less", Semver{1, 0, 0}, Semver{1, 0, 1}, -1},
		{"zeros", Semver{0, 0, 0}, Semver{0, 0, 0}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.Compare(tt.b); got != tt.want {
				t.Errorf("%v.Compare(%v) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
