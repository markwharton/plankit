package hooks

import (
	"os"
	"os/exec"
	"runtime"
)

// RunScript runs a shell command, inheriting stdout and stderr from the
// parent. Uses "cmd /c" on Windows, "sh -c" elsewhere. Optional environment
// variables are added on top of the parent's environment.
//
// Variables from env are pre-expanded in the command string using $VAR and
// ${VAR} syntax so that hook authors can write $VERSION once and it works
// on all platforms (sh -c expands $VAR natively, cmd /c does not).
func RunScript(command string, env map[string]string) error {
	if len(env) > 0 {
		command = os.Expand(command, func(key string) string {
			if v, ok := env[key]; ok {
				return v
			}
			return "$" + key
		})
	}

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
