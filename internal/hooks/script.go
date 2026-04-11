package hooks

import (
	"os"
	"os/exec"
)

// RunScript runs a shell command via "sh -c", inheriting stdout and stderr
// from the parent. Optional environment variables are added on top of the
// parent's environment. Used by command lifecycle hooks (e.g., preCommit,
// preRelease, postVersion).
func RunScript(command string, env map[string]string) error {
	cmd := exec.Command("sh", "-c", command)
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
