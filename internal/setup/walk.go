package setup

import (
	"os"
	"path/filepath"
	"strings"
)

// WalkRuleFiles calls visit for every .md file under dir, descending into
// subdirectories. rel is the file's slash-separated path relative to dir
// (e.g. "plankit/git-discipline.md"), the form status, teardown, and rules
// all use as a display label. A missing directory (at any level) yields
// nothing; any other readDir error, or an error from visit, aborts the walk.
func WalkRuleFiles(readDir func(string) ([]os.DirEntry, error), dir string, visit func(path, rel string) error) error {
	return walkRuleFiles(readDir, dir, "", visit)
}

func walkRuleFiles(readDir func(string) ([]os.DirEntry, error), dir, rel string, visit func(path, rel string) error) error {
	entries, err := readDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		sub := name
		if rel != "" {
			sub = rel + "/" + name
		}
		if entry.IsDir() {
			if err := walkRuleFiles(readDir, filepath.Join(dir, name), sub, visit); err != nil {
				return err
			}
			continue
		}
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		if err := visit(filepath.Join(dir, name), sub); err != nil {
			return err
		}
	}
	return nil
}

// WalkSkillFiles calls visit for every <name>/SKILL.md under dir (the skills
// layout), with rel "<name>/SKILL.md". A missing directory yields nothing;
// any other readDir error, or an error from visit, aborts the walk.
func WalkSkillFiles(readDir func(string) ([]os.DirEntry, error), dir string, visit func(path, rel string) error) error {
	entries, err := readDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if err := visit(filepath.Join(dir, name, "SKILL.md"), name+"/SKILL.md"); err != nil {
			return err
		}
	}
	return nil
}
