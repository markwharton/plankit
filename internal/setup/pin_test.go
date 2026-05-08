package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScriptVersion_found(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "install-pk.sh")
	os.WriteFile(path, []byte("#!/usr/bin/env bash\nPK_VERSION=\"v0.8.0\"\n"), 0755)

	ver, found := ScriptVersion(os.ReadFile, path)
	if !found {
		t.Fatal("ScriptVersion did not find PK_VERSION")
	}
	if ver != "v0.8.0" {
		t.Errorf("ScriptVersion = %q, want %q", ver, "v0.8.0")
	}
}

func TestScriptVersion_customName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "install.sh")
	os.WriteFile(path, []byte("#!/usr/bin/env bash\nMY_APP_VERSION=\"v1.2.3\"\n"), 0755)

	ver, found := ScriptVersion(os.ReadFile, path)
	if !found {
		t.Fatal("ScriptVersion did not find MY_APP_VERSION")
	}
	if ver != "v1.2.3" {
		t.Errorf("ScriptVersion = %q, want %q", ver, "v1.2.3")
	}
}

func TestScriptVersion_notFound(t *testing.T) {
	_, found := ScriptVersion(os.ReadFile, filepath.Join(t.TempDir(), "missing.sh"))
	if found {
		t.Error("ScriptVersion should return false when file does not exist")
	}
}

func TestScriptVersion_noVersionLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "script.sh")
	os.WriteFile(path, []byte("#!/usr/bin/env bash\necho hello\n"), 0755)

	_, found := ScriptVersion(os.ReadFile, path)
	if found {
		t.Error("ScriptVersion should return false when no VERSION line")
	}
}

func TestPinVersion_updatesVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "install-pk.sh")
	os.WriteFile(path, []byte("#!/usr/bin/env bash\nPK_VERSION=\"v0.8.0\"\ninstall_dir=\"$HOME/.local/bin\"\n"), 0755)

	updated, err := PinVersion(os.ReadFile, os.WriteFile, path, "0.8.1")
	if err != nil {
		t.Fatalf("PinVersion() error = %v", err)
	}
	if !updated {
		t.Fatal("PinVersion should return updated=true")
	}

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), `PK_VERSION="v0.8.1"`) {
		t.Errorf("script should contain v0.8.1, got: %s", string(data))
	}
}

func TestPinVersion_customName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "install.sh")
	os.WriteFile(path, []byte("#!/usr/bin/env bash\nMY_APP_VERSION=\"v1.0.0\"\n"), 0755)

	updated, err := PinVersion(os.ReadFile, os.WriteFile, path, "1.1.0")
	if err != nil {
		t.Fatalf("PinVersion() error = %v", err)
	}
	if !updated {
		t.Fatal("PinVersion should return updated=true")
	}

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), `MY_APP_VERSION="v1.1.0"`) {
		t.Errorf("script should contain v1.1.0, got: %s", string(data))
	}
}

func TestPinVersion_noFile(t *testing.T) {
	updated, err := PinVersion(os.ReadFile, os.WriteFile, filepath.Join(t.TempDir(), "missing.sh"), "0.8.1")
	if err != nil {
		t.Fatalf("PinVersion should not error when file doesn't exist, got: %v", err)
	}
	if updated {
		t.Error("PinVersion should return updated=false when file doesn't exist")
	}
}

func TestPinVersion_vPrefix(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "install-pk.sh")
	os.WriteFile(path, []byte("#!/usr/bin/env bash\nPK_VERSION=\"v0.8.0\"\n"), 0755)

	PinVersion(os.ReadFile, os.WriteFile, path, "v0.8.1")
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), `"vv0.8.1"`) {
		t.Error("PinVersion should not double-prefix v")
	}
}

// --- Named pin tests ---

func TestMatchNamedPin_goConst(t *testing.T) {
	m, ok := matchNamedPin(`const version = "0.1.0"`, "version")
	if !ok {
		t.Fatal("expected match")
	}
	if m.value != "0.1.0" {
		t.Errorf("value = %q, want %q", m.value, "0.1.0")
	}
	if m.quote != '"' {
		t.Errorf("quote = %c, want %c", m.quote, '"')
	}
}

func TestMatchNamedPin_goVar(t *testing.T) {
	m, ok := matchNamedPin(`var version = "0.1.0"`, "version")
	if !ok {
		t.Fatal("expected match")
	}
	if m.value != "0.1.0" {
		t.Errorf("value = %q, want %q", m.value, "0.1.0")
	}
}

func TestMatchNamedPin_python(t *testing.T) {
	m, ok := matchNamedPin(`__version__ = "0.1.0"`, "__version__")
	if !ok {
		t.Fatal("expected match")
	}
	if m.value != "0.1.0" {
		t.Errorf("value = %q, want %q", m.value, "0.1.0")
	}
}

func TestMatchNamedPin_singleQuote(t *testing.T) {
	m, ok := matchNamedPin(`__version__ = '0.1.0'`, "__version__")
	if !ok {
		t.Fatal("expected match")
	}
	if m.value != "0.1.0" {
		t.Errorf("value = %q, want %q", m.value, "0.1.0")
	}
	if m.quote != '\'' {
		t.Errorf("quote = %c, want %c", m.quote, '\'')
	}
}

func TestMatchNamedPin_toml(t *testing.T) {
	m, ok := matchNamedPin(`version = "0.1.0"`, "version")
	if !ok {
		t.Fatal("expected match")
	}
	if m.value != "0.1.0" {
		t.Errorf("value = %q, want %q", m.value, "0.1.0")
	}
}

func TestMatchNamedPin_vPrefix(t *testing.T) {
	m, ok := matchNamedPin(`version = "v0.1.0"`, "version")
	if !ok {
		t.Fatal("expected match")
	}
	if m.value != "v0.1.0" {
		t.Errorf("value = %q, want %q", m.value, "v0.1.0")
	}
}

func TestMatchNamedPin_noMatch(t *testing.T) {
	_, ok := matchNamedPin(`comment = "hello"`, "version")
	if ok {
		t.Fatal("expected no match")
	}
}

func TestMatchNamedPin_partialName(t *testing.T) {
	_, ok := matchNamedPin(`my_version = "1.0.0"`, "version")
	if ok {
		t.Fatal("expected no match for partial name")
	}
}

func TestMatchNamedPin_partialNameSuffix(t *testing.T) {
	_, ok := matchNamedPin(`versioning = "1.0.0"`, "version")
	if ok {
		t.Fatal("expected no match for suffix overlap")
	}
}

func TestMatchNamedPin_colonEquals(t *testing.T) {
	m, ok := matchNamedPin(`version := "1.0.0"`, "version")
	if !ok {
		t.Fatal("expected match")
	}
	if m.value != "1.0.0" {
		t.Errorf("value = %q, want %q", m.value, "1.0.0")
	}
}

func TestMatchNamedPin_tabs(t *testing.T) {
	m, ok := matchNamedPin("\tversion\t=\t\"1.0.0\"", "version")
	if !ok {
		t.Fatal("expected match with tabs")
	}
	if m.value != "1.0.0" {
		t.Errorf("value = %q, want %q", m.value, "1.0.0")
	}
}

func TestMatchNamedPin_trailingContent(t *testing.T) {
	m, ok := matchNamedPin(`const version = "1.0.0" // app version`, "version")
	if !ok {
		t.Fatal("expected match")
	}
	if m.value != "1.0.0" {
		t.Errorf("value = %q, want %q", m.value, "1.0.0")
	}
	if m.lineSuffix != " // app version" {
		t.Errorf("lineSuffix = %q, want %q", m.lineSuffix, " // app version")
	}
}

func TestMatchNamedPin_unclosedQuote(t *testing.T) {
	_, ok := matchNamedPin(`version = "0.1.0`, "version")
	if ok {
		t.Fatal("expected no match for unclosed quote")
	}
}

func TestMatchNamedPin_nameAtEndOfLine(t *testing.T) {
	_, ok := matchNamedPin(`version`, "version")
	if ok {
		t.Fatal("expected no match when name is at end of line with no operator")
	}
}

func TestMatchNamedPin_nameFollowedByWhitespaceOnly(t *testing.T) {
	_, ok := matchNamedPin(`version   `, "version")
	if ok {
		t.Fatal("expected no match when no operator follows")
	}
}

func TestMatchNamedPin_operatorNoQuote(t *testing.T) {
	_, ok := matchNamedPin(`version = 123`, "version")
	if ok {
		t.Fatal("expected no match when value is not quoted")
	}
}

func TestPinVersionNamed_updatesGoConst(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.go")
	os.WriteFile(path, []byte("package main\n\nconst version = \"0.1.0\"\n\nfunc main() {}\n"), 0644)

	updated, err := PinVersionNamed(os.ReadFile, os.WriteFile, path, "version", "0.2.0")
	if err != nil {
		t.Fatalf("PinVersionNamed() error = %v", err)
	}
	if !updated {
		t.Fatal("expected updated=true")
	}

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), `const version = "0.2.0"`) {
		t.Errorf("file should contain 0.2.0, got: %s", string(data))
	}
}

func TestPinVersionNamed_preservesVPrefix(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.go")
	os.WriteFile(path, []byte("const version = \"v0.1.0\"\n"), 0644)

	PinVersionNamed(os.ReadFile, os.WriteFile, path, "version", "0.2.0")
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), `"v0.2.0"`) {
		t.Errorf("should preserve v prefix, got: %s", string(data))
	}
}

func TestPinVersionNamed_stripsVWhenBare(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.go")
	os.WriteFile(path, []byte("const version = \"0.1.0\"\n"), 0644)

	PinVersionNamed(os.ReadFile, os.WriteFile, path, "version", "v0.2.0")
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), `"v0.2.0"`) {
		t.Errorf("should strip v prefix when existing value is bare, got: %s", string(data))
	}
	if !strings.Contains(string(data), `"0.2.0"`) {
		t.Errorf("should contain bare 0.2.0, got: %s", string(data))
	}
}

func TestPinVersionNamed_noFile(t *testing.T) {
	updated, err := PinVersionNamed(os.ReadFile, os.WriteFile, filepath.Join(t.TempDir(), "missing.go"), "version", "0.1.0")
	if err != nil {
		t.Fatalf("should not error when file doesn't exist, got: %v", err)
	}
	if updated {
		t.Error("should return updated=false")
	}
}

func TestPinVersionNamed_noMatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.go")
	os.WriteFile(path, []byte("package main\n\nfunc main() {}\n"), 0644)

	_, err := PinVersionNamed(os.ReadFile, os.WriteFile, path, "version", "0.1.0")
	if err == nil {
		t.Fatal("expected error for no match")
	}
	if !strings.Contains(err.Error(), "no pin for") {
		t.Errorf("error = %v, want 'no pin for'", err)
	}
}

func TestPinVersionNamed_firstMatchWins(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.go")
	content := "const version = \"0.1.0\"\nvar version = \"0.2.0\"\n"
	os.WriteFile(path, []byte(content), 0644)

	PinVersionNamed(os.ReadFile, os.WriteFile, path, "version", "0.9.0")
	data, _ := os.ReadFile(path)
	lines := strings.Split(string(data), "\n")
	if lines[0] != `const version = "0.9.0"` {
		t.Errorf("first line should be updated, got: %s", lines[0])
	}
	if lines[1] != `var version = "0.2.0"` {
		t.Errorf("second line should be unchanged, got: %s", lines[1])
	}
}

func TestReadVersionNamed_found(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.go")
	os.WriteFile(path, []byte("package main\n\nconst version = \"0.3.0\"\n"), 0644)

	ver, found := ReadVersionNamed(os.ReadFile, path, "version")
	if !found {
		t.Fatal("expected found=true")
	}
	if ver != "0.3.0" {
		t.Errorf("version = %q, want %q", ver, "0.3.0")
	}
}

func TestReadVersionNamed_notFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.go")
	os.WriteFile(path, []byte("package main\n"), 0644)

	_, found := ReadVersionNamed(os.ReadFile, path, "version")
	if found {
		t.Fatal("expected found=false")
	}
}

func TestReadVersionNamed_missingFile(t *testing.T) {
	_, found := ReadVersionNamed(os.ReadFile, filepath.Join(t.TempDir(), "missing.go"), "version")
	if found {
		t.Fatal("expected found=false for missing file")
	}
}
