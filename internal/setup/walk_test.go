package setup

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestWalkRuleFiles_recursesAndFiltersMD(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "top.md"))
	mustWrite(t, filepath.Join(dir, "notes.txt"))
	mustWrite(t, filepath.Join(dir, "plankit", "git-discipline.md"))
	mustWrite(t, filepath.Join(dir, "plankit", "deep", "nested.md"))

	var rels []string
	err := WalkRuleFiles(os.ReadDir, dir, func(path, rel string) error {
		if filepath.Base(path) != filepath.Base(rel) {
			t.Errorf("path %q and rel %q disagree on basename", path, rel)
		}
		rels = append(rels, rel)
		return nil
	})
	if err != nil {
		t.Fatalf("WalkRuleFiles() error = %v", err)
	}
	sort.Strings(rels)
	want := []string{"plankit/deep/nested.md", "plankit/git-discipline.md", "top.md"}
	if !reflect.DeepEqual(rels, want) {
		t.Errorf("rels = %v, want %v", rels, want)
	}
}

func TestWalkRuleFiles_missingDir(t *testing.T) {
	err := WalkRuleFiles(os.ReadDir, filepath.Join(t.TempDir(), "absent"), func(path, rel string) error {
		t.Errorf("visit called for %q in missing dir", rel)
		return nil
	})
	if err != nil {
		t.Fatalf("missing dir should yield nothing, got error %v", err)
	}
}

func TestWalkRuleFiles_visitErrorAborts(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "a.md"))
	wantErr := errors.New("boom")
	err := WalkRuleFiles(os.ReadDir, dir, func(path, rel string) error { return wantErr })
	if !errors.Is(err, wantErr) {
		t.Errorf("WalkRuleFiles() error = %v, want %v", err, wantErr)
	}
}

func TestWalkSkillFiles(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "ship", "SKILL.md"))
	mustWrite(t, filepath.Join(dir, "preserve", "SKILL.md"))
	mustWrite(t, filepath.Join(dir, "stray.md")) // top-level file: not a skill, skipped

	var rels []string
	err := WalkSkillFiles(os.ReadDir, dir, func(path, rel string) error {
		rels = append(rels, rel)
		return nil
	})
	if err != nil {
		t.Fatalf("WalkSkillFiles() error = %v", err)
	}
	sort.Strings(rels)
	want := []string{"preserve/SKILL.md", "ship/SKILL.md"}
	if !reflect.DeepEqual(rels, want) {
		t.Errorf("rels = %v, want %v", rels, want)
	}
}

func TestWalkSkillFiles_missingDir(t *testing.T) {
	err := WalkSkillFiles(os.ReadDir, filepath.Join(t.TempDir(), "absent"), func(path, rel string) error {
		t.Errorf("visit called for %q in missing dir", rel)
		return nil
	})
	if err != nil {
		t.Fatalf("missing dir should yield nothing, got error %v", err)
	}
}

func mustWrite(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("content\n"), 0644); err != nil {
		t.Fatal(err)
	}
}
