package hooks

import (
	"fmt"
	"os"
	"testing"
)

func TestRunScript(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		if err := RunScript("true", nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("failure", func(t *testing.T) {
		if err := RunScript("false", nil); err == nil {
			t.Fatal("expected error for failing command")
		}
	})

	t.Run("env vars are set", func(t *testing.T) {
		key := "PK_TEST_SCRIPT_VAR"
		want := "hello"
		script := fmt.Sprintf(`test "$%s" = "%s"`, key, want)
		if err := RunScript(script, map[string]string{key: want}); err != nil {
			t.Fatalf("env var not available in script: %v", err)
		}
	})

	t.Run("env vars do not leak to parent", func(t *testing.T) {
		key := "PK_TEST_LEAK_CHECK"
		RunScript("true", map[string]string{key: "leaked"})
		if os.Getenv(key) != "" {
			t.Errorf("%s leaked into parent environment", key)
		}
	})

	t.Run("env var expanded in command", func(t *testing.T) {
		env := map[string]string{"VERSION": "1.2.3"}
		if err := RunScript(`test "$VERSION" = "1.2.3"`, env); err != nil {
			t.Fatalf("env var not expanded in command: %v", err)
		}
	})

	t.Run("braced env var expanded in command", func(t *testing.T) {
		env := map[string]string{"VERSION": "1.2.3"}
		if err := RunScript(`test "${VERSION}" = "1.2.3"`, env); err != nil {
			t.Fatalf("braced env var not expanded in command: %v", err)
		}
	})

	t.Run("unknown vars preserved for shell", func(t *testing.T) {
		env := map[string]string{"VERSION": "1.2.3"}
		// $HOME is not in env, so it should pass through to the shell which expands it.
		if err := RunScript(`test -n "$HOME"`, env); err != nil {
			t.Fatalf("unknown var should pass through to shell: %v", err)
		}
	})

	t.Run("no expansion without env", func(t *testing.T) {
		// With nil env, no expansion occurs. $HOME is expanded by the shell.
		if err := RunScript(`test -n "$HOME"`, nil); err != nil {
			t.Fatalf("nil env should not affect shell expansion: %v", err)
		}
	})
}
