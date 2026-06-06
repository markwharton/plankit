package setup

import (
	"io/fs"
	"testing"

	"github.com/markwharton/plankit/internal/safety"
)

// embeddedTexts returns every managed text asset that pk setup ships into
// downstream repositories, keyed by display path. These files are read by AI
// agents in every Claude Code session, so they must carry no hidden or control
// characters that could smuggle instructions past a human reviewer (the
// "Trojan Source" class, CVE-2021-42574). The scan covers exactly what ships:
// the embedded bytes, not the working-tree source.
func embeddedTexts(t *testing.T) map[string]string {
	t.Helper()
	texts := map[string]string{
		"internal/setup/template/install-pk.sh": installScriptTemplate,
	}
	for _, fsys := range []fs.FS{templateFS, rulesFS, skillsFS} {
		err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			b, err := fs.ReadFile(fsys, path)
			if err != nil {
				return err
			}
			texts["internal/setup/"+path] = string(b)
			return nil
		})
		if err != nil {
			t.Fatalf("walk embedded FS: %v", err)
		}
	}
	return texts
}

// TestEmbeddedManagedFilesHaveNoHiddenCharacters is the build-time guard
// against a Trojan-source contribution to a file pk distributes downstream. The
// scan policy lives in internal/safety so `pk rules --lint` shares it.
func TestEmbeddedManagedFilesHaveNoHiddenCharacters(t *testing.T) {
	texts := embeddedTexts(t)
	if len(texts) == 0 {
		t.Fatal("no embedded managed files found to scan")
	}
	for path, content := range texts {
		if v := safety.ScanHidden(content); len(v) > 0 {
			t.Errorf("%s contains hidden/control characters: %v", path, v)
		}
	}
}
