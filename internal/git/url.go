package git

import "strings"

// ParseRepoURL converts a git remote URL to an HTTPS base URL.
// Handles SSH (git@github.com:owner/repo.git) and HTTPS formats.
func ParseRepoURL(remoteURL string) string {
	u := strings.TrimSpace(remoteURL)
	// SSH format: git@github.com:owner/repo.git
	if strings.HasPrefix(u, "git@") {
		u = strings.TrimPrefix(u, "git@")
		u = strings.Replace(u, ":", "/", 1)
		u = "https://" + u
	}
	u = strings.TrimSuffix(u, ".git")
	return u
}
