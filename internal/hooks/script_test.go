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
}
