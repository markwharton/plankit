package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// shimFixture copies bin/pk into a scratch plugin with nothing beside it
// and returns the shim's path, the plugin root, and a directory on PATH
// holding a fake pk that reports what the shim told it.
func shimFixture(t *testing.T) (shim, root, pathDir string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("bin/pk is the POSIX shim; bin/pk.cmd is not run by this suite")
	}
	src, err := os.ReadFile(filepath.Join(repoRoot(t), "bin", "pk"))
	if err != nil {
		t.Fatal(err)
	}
	root = t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	shim = filepath.Join(root, "bin", "pk")
	if err := os.WriteFile(shim, src, 0o755); err != nil {
		t.Fatal(err)
	}
	pathDir = t.TempDir()
	fake := "#!/bin/sh\necho \"root=$PK_PLUGIN_ROOT args=$*\"\nexit 7\n"
	if err := os.WriteFile(filepath.Join(pathDir, "pk"), []byte(fake), 0o755); err != nil {
		t.Fatal(err)
	}
	return shim, root, pathDir
}

func runShim(t *testing.T, shim, path string, args ...string) (int, string, string) {
	t.Helper()
	cmd := exec.Command(shim, args...)
	// The shim needs dirname and uname from the system; neither
	// directory holds a pk on a development machine.
	cmd.Env = []string{"PATH=" + path + ":/usr/bin:/bin", "HOME=" + t.TempDir()}
	var out, errw strings.Builder
	cmd.Stdout, cmd.Stderr = &out, &errw
	err := cmd.Run()
	code := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		code = exitErr.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return code, out.String(), errw.String()
}

// TestShimFallsBackToPathAndNamesThePlugin: with no binary beside it,
// the shim runs the pk on PATH, skipping its own directory even when
// that comes first, passes the arguments and the exit code through,
// and names the plugin root.
func TestShimFallsBackToPathAndNamesThePlugin(t *testing.T) {
	shim, root, pathDir := shimFixture(t)
	resolved, _ := filepath.EvalSymlinks(root)
	code, out, errw := runShim(t, shim, filepath.Dir(shim)+":"+pathDir, "version", "--x")
	if code != 7 || out != "root="+resolved+" args=version --x\n" {
		t.Fatalf("code=%d out=%q errw=%q", code, out, errw)
	}
}

func TestShimWithoutAnyPkNamesTheInstall(t *testing.T) {
	shim, _, _ := shimFixture(t)
	code, out, errw := runShim(t, shim, filepath.Dir(shim), "version")
	if code != 3 || out != "" || !strings.Contains(errw, "go install github.com/markwharton/plankit/cmd/pk@latest") {
		t.Fatalf("code=%d out=%q errw=%q", code, out, errw)
	}
}

func TestShimPrefersTheBinaryBesideIt(t *testing.T) {
	shim, _, pathDir := shimFixture(t)
	sibling := filepath.Join(filepath.Dir(shim), "pk-"+runtime.GOOS+"-"+runtime.GOARCH)
	if err := os.WriteFile(sibling, []byte("#!/bin/sh\necho \"sibling root=$PK_PLUGIN_ROOT\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	code, out, errw := runShim(t, shim, pathDir, "version")
	if code != 0 || out != "sibling root=\n" {
		t.Fatalf("code=%d out=%q errw=%q", code, out, errw)
	}
}
