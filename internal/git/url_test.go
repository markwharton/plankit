package git

import "testing"

func TestParseRepoURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"SSH", "git@github.com:markwharton/plankit.git", "https://github.com/markwharton/plankit"},
		{"HTTPS with .git", "https://github.com/markwharton/plankit.git", "https://github.com/markwharton/plankit"},
		{"HTTPS without .git", "https://github.com/markwharton/plankit", "https://github.com/markwharton/plankit"},
		{"SSH with trailing newline", "git@github.com:owner/repo.git\n", "https://github.com/owner/repo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseRepoURL(tt.input); got != tt.want {
				t.Errorf("ParseRepoURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
