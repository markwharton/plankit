package setup

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:embed skills/*/SKILL.md
var skillsFS embed.FS

//go:embed rules/*.md
var rulesFS embed.FS

// Skill represents a skill file to install.
type Skill struct {
	Name    string
	Content string
}

// skills returns the skills to install from the embedded filesystem.
func skills() ([]Skill, error) {
	entries, err := fs.ReadDir(skillsFS, "skills")
	if err != nil {
		return nil, err
	}

	var result []Skill
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		content, err := fs.ReadFile(skillsFS, "skills/"+entry.Name()+"/SKILL.md")
		if err != nil {
			return nil, err
		}
		result = append(result, Skill{
			Name:    entry.Name(),
			Content: string(content),
		})
	}
	return result, nil
}

// Rule represents a rules file to install.
type Rule struct {
	Name    string
	Content string
}

// rules returns the rules to install from the embedded filesystem.
func rules() ([]Rule, error) {
	entries, err := fs.ReadDir(rulesFS, "rules")
	if err != nil {
		return nil, err
	}

	var result []Rule
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		content, err := fs.ReadFile(rulesFS, "rules/"+entry.Name())
		if err != nil {
			return nil, err
		}
		name := strings.TrimSuffix(entry.Name(), ".md")
		result = append(result, Rule{
			Name:    name,
			Content: string(content),
		})
	}
	return result, nil
}

const commentPrefix = "<!-- pk:sha256:"
const commentSuffix = " -->"
const frontmatterKey = "pk_sha256: "

// ContentSHA computes the SHA256 hash of content.
func ContentSHA(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}

// ExtractSHA extracts a pk SHA and the hashed content from a file.
// Supports two formats:
//   - HTML comment on first line: <!-- pk:sha256:... --> (for CLAUDE.md)
//   - YAML frontmatter field: pk_sha256: ... (for skills with frontmatter)
//
// Returns (sha, hashedContent, found).
func ExtractSHA(fileContent string) (string, string, bool) {
	// Try HTML comment on first line.
	firstNewline := strings.IndexByte(fileContent, '\n')
	if firstNewline > 0 {
		firstLine := fileContent[:firstNewline]
		if strings.HasPrefix(firstLine, commentPrefix) && strings.HasSuffix(firstLine, commentSuffix) {
			sha := firstLine[len(commentPrefix) : len(firstLine)-len(commentSuffix)]
			content := fileContent[firstNewline+1:]
			return sha, content, true
		}
	}

	// Try frontmatter pk_sha256 field.
	if strings.HasPrefix(fileContent, "---\n") {
		closeIdx := strings.Index(fileContent[4:], "\n---\n")
		if closeIdx >= 0 {
			frontmatter := fileContent[4 : 4+closeIdx]
			body := fileContent[4+closeIdx+5:] // skip past \n---\n
			for _, line := range strings.Split(frontmatter, "\n") {
				if strings.HasPrefix(line, frontmatterKey) {
					sha := strings.TrimSpace(line[len(frontmatterKey):])
					return sha, body, true
				}
			}
		}
	}

	return "", "", false
}

// embedSHA embeds a SHA into content using the appropriate format.
// Skills (content starting with ---) use a frontmatter field.
// Other files use an HTML comment on the first line.
func embedSHA(content string, sha string) string {
	if strings.HasPrefix(content, "---\n") {
		// Insert pk_sha256 field into existing frontmatter.
		closeIdx := strings.Index(content[4:], "\n---\n")
		if closeIdx >= 0 {
			frontmatter := content[4 : 4+closeIdx]
			body := content[4+closeIdx+5:]
			return "---\n" + frontmatter + "\n" + frontmatterKey + sha + "\n---\n" + body
		}
	}
	// HTML comment on first line.
	return commentPrefix + sha + commentSuffix + "\n" + content
}

// shouldUpdate checks whether a managed file should be updated.
// Returns (true, reason) if the file should be written, (false, reason) if it should be skipped.
func shouldUpdate(readFile func(string) ([]byte, error), path string, newContent string, force bool) (bool, string) {
	data, err := readFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return true, "created"
		}
		return false, "skipped (unreadable)"
	}

	if force {
		return true, "updated (forced)"
	}

	storedSHA, hashedContent, found := ExtractSHA(string(data))
	if !found {
		return false, "skipped (not managed by pk)"
	}

	if ContentSHA(hashedContent) != storedSHA {
		return false, "skipped (modified by user)"
	}

	return true, "updated"
}

// writeManaged writes content to path with a SHA marker embedded in the file.
// Skills with YAML frontmatter get a pk_sha256 field; other files get an HTML comment on line 1.
// If the file exists and has been modified by the user, it is skipped unless force is true.
// Returns (changed, error). changed is true only when the bytes actually written differ from what was on disk.
func writeManaged(cfg Config, path string, content string, force bool) (bool, error) {
	update, reason := shouldUpdate(cfg.ReadFile, path, content, force)
	if !update {
		fmt.Fprintf(cfg.Stderr, "  %s: %s\n", displayName(path), reason)
		return false, nil
	}

	// Compute SHA over the body that will be hashed (content after frontmatter for skills,
	// content after the comment line for CLAUDE.md). Since embedSHA splits at the same
	// boundaries as ExtractSHA, we hash the original content which becomes the body.
	var sha string
	if strings.HasPrefix(content, "---\n") {
		// For skills: SHA covers the body after frontmatter.
		closeIdx := strings.Index(content[4:], "\n---\n")
		if closeIdx >= 0 {
			body := content[4+closeIdx+5:]
			sha = ContentSHA(body)
		} else {
			sha = ContentSHA(content)
		}
	} else {
		// For non-frontmatter files: SHA covers the full content.
		sha = ContentSHA(content)
	}

	managed := embedSHA(content, sha)

	// Read existing bytes before writing so we can report whether content actually changed.
	existing, _ := cfg.ReadFile(path)

	if err := cfg.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return false, fmt.Errorf("failed to create directory for %s: %w", path, err)
	}
	if err := cfg.WriteFile(path, []byte(managed), 0644); err != nil {
		return false, fmt.Errorf("failed to write %s: %w", path, err)
	}
	fmt.Fprintf(cfg.Stderr, "  %s: %s\n", displayName(path), reason)
	return string(existing) != managed, nil
}

// displayName returns a short display name for a managed file path.
// Uses parent/file for skills (e.g., "init/SKILL.md") and just the filename otherwise.
func displayName(path string) string {
	dir := filepath.Base(filepath.Dir(path))
	base := filepath.Base(path)
	if base == "SKILL.md" {
		return dir + "/" + base
	}
	return base
}

// pruneSkills removes skill directories under skillsDir whose name isn't in
// kept (the set of currently-embedded skill names). Per-file safety:
//   - SKILL.md missing or unreadable -> skip silently.
//   - SKILL.md has no pk_sha256 -> user-created, skip silently.
//   - pk_sha256 matches body -> pk wrote it, untouched, remove.
//   - pk_sha256 mismatches body -> user modified, preserve, warn.
//
// After removing the SKILL.md, the parent dir is removed only if empty.
// Returns true if any file was removed.
func pruneSkills(cfg Config, skillsDir string, kept map[string]bool) bool {
	entries, err := cfg.ReadDir(skillsDir)
	if err != nil {
		return false
	}
	var removed bool
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if kept[entry.Name()] {
			continue
		}
		skillFile := filepath.Join(skillsDir, entry.Name(), "SKILL.md")
		switch evaluateRemoval(cfg.ReadFile, skillFile) {
		case "remove":
			if err := cfg.Remove(skillFile); err == nil {
				fmt.Fprintf(cfg.Stderr, "  %s/SKILL.md: removed\n", entry.Name())
				cfg.Remove(filepath.Join(skillsDir, entry.Name()))
				removed = true
			}
		case "preserve":
			fmt.Fprintf(cfg.Stderr, "  %s/SKILL.md: preserved (modified locally; pk no longer manages it — remove manually if no longer needed)\n", entry.Name())
		}
	}
	return removed
}

// pruneRules removes rule files under rulesDir whose name (without .md) isn't
// in kept. Same per-file safety rules as pruneSkills. Returns true if any file
// was removed.
func pruneRules(cfg Config, rulesDir string, kept map[string]bool) bool {
	entries, err := cfg.ReadDir(rulesDir)
	if err != nil {
		return false
	}
	var removed bool
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".md")
		if kept[name] {
			continue
		}
		ruleFile := filepath.Join(rulesDir, entry.Name())
		switch evaluateRemoval(cfg.ReadFile, ruleFile) {
		case "remove":
			if err := cfg.Remove(ruleFile); err == nil {
				fmt.Fprintf(cfg.Stderr, "  %s: removed\n", entry.Name())
				removed = true
			}
		case "preserve":
			fmt.Fprintf(cfg.Stderr, "  %s: preserved (modified locally; pk no longer manages it — remove manually if no longer needed)\n", entry.Name())
		}
	}
	return removed
}

// evaluateRemoval inspects a managed-file candidate and reports whether it can
// be safely removed, must be preserved (user modified), or should be skipped
// (user-created, no pk marker).
func evaluateRemoval(readFile func(string) ([]byte, error), path string) string {
	data, err := readFile(path)
	if err != nil {
		return "skip"
	}
	storedSHA, body, found := ExtractSHA(string(data))
	if !found {
		return "skip"
	}
	if ContentSHA(body) == storedSHA {
		return "remove"
	}
	return "preserve"
}
