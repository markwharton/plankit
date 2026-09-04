package release

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/markwharton/plankit/internal/cli"
)

func runPinCmd(t *testing.T, dir string, args ...string) (int, string) {
	t.Helper()
	var out, errw bytes.Buffer
	argv := append([]string{"pk", "pin", "--project-dir", dir}, args...)
	code := cli.RunIO(argv, []*cli.Command{PinCmd}, nil, &out, &errw)
	return code, errw.String()
}

func TestShellPin(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "install.sh")
	os.WriteFile(path, []byte("#!/bin/sh\nPK_VERSION=\"v0.1.0\"\necho $PK_VERSION\n"), 0o755)

	code, errw := runPinCmd(t, dir, "--file", "install.sh", "0.2.0")
	if code != cli.ExitOK || !strings.Contains(errw, "pinned install.sh to 0.2.0") {
		t.Fatalf("code=%d errw=%q", code, errw)
	}
	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), `PK_VERSION="v0.2.0"`) {
		t.Fatalf("pin not updated: %s", got)
	}
}

func TestNamedPinForms(t *testing.T) {
	cases := []struct{ line, name, want string }{
		{`version: "1.0.0"`, "version", `version: "2.5.0"`},
		{`pk := "v1.0.0"`, "pk", `pk := "v2.5.0"`},
		{`const V = 'v1.0.0' // pinned`, "V", `const V = 'v2.5.0' // pinned`},
	}
	for _, tc := range cases {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "f"), []byte(tc.line+"\n"), 0o644)
		code, _ := runPinCmd(t, dir, "--file", "f", "--name", tc.name, "v2.5.0")
		if code != cli.ExitOK {
			t.Fatalf("%s: exit %d", tc.line, code)
		}
		got, _ := os.ReadFile(filepath.Join(dir, "f"))
		if strings.TrimRight(string(got), "\n") != tc.want {
			t.Errorf("%s: got %q want %q", tc.line, got, tc.want)
		}
	}
}

func TestNamedPinWordBoundary(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "f"), []byte("myversion: \"1.0.0\"\nversion: \"1.0.0\"\n"), 0o644)
	runPinCmd(t, dir, "--file", "f", "--name", "version", "2.0.0")
	got, _ := os.ReadFile(filepath.Join(dir, "f"))
	if !strings.Contains(string(got), `myversion: "1.0.0"`) || !strings.Contains(string(got), `version: "2.0.0"`) {
		t.Fatalf("boundary wrong: %s", got)
	}
}

func TestMissingFileIsANoOp(t *testing.T) {
	code, errw := runPinCmd(t, t.TempDir(), "--file", "absent.sh", "1.0.0")
	if code != cli.ExitOK || !strings.Contains(errw, "does not exist; nothing to pin") {
		t.Fatalf("code=%d errw=%q", code, errw)
	}
}

func TestPinlessFileWarnsButSucceeds(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "f"), []byte("nothing here\n"), 0o644)
	code, errw := runPinCmd(t, dir, "--file", "f", "1.0.0")
	if code != cli.ExitOK || !strings.Contains(errw, "has no VERSION pin") {
		t.Fatalf("code=%d errw=%q", code, errw)
	}
}

func TestUsageErrors(t *testing.T) {
	if code, _ := runPinCmd(t, t.TempDir(), "1.0.0"); code != cli.ExitUsage {
		t.Fatal("--file is required")
	}
	if code, _ := runPinCmd(t, t.TempDir(), "--file", "f"); code != cli.ExitUsage {
		t.Fatal("version argument is required")
	}
}
