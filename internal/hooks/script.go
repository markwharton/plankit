package hooks

import (
	"os"
	"os/exec"
	"runtime"
)

// RunScript runs a shell command, inheriting stdout and stderr from the
// parent. Uses "cmd /c" on Windows, "sh -c" elsewhere. Optional environment
// variables are added on top of the parent's environment.
func RunScript(command string, env map[string]string) error {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if len(env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}
	return cmd.Run()
}
