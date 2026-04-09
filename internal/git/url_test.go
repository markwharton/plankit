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
		{"GitLab SSH", "git@gitlab.com:team/project.git", "https://gitlab.com/team/project"},
		{"GitLab HTTPS", "https://gitlab.com/team/project.git", "https://gitlab.com/team/project"},
		{"Gitea SSH", "git@gitea.example.com:org/repo.git", "https://gitea.example.com/org/repo"},
		{"Bitbucket SSH", "git@bitbucket.org:workspace/repo.git", "https://bitbucket.org/workspace/repo"},
		{"empty string", "", ""},
		{"whitespace only", "  \n  ", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseRepoURL(tt.input); got != tt.want {
				t.Errorf("ParseRepoURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
