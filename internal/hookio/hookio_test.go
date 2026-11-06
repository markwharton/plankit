package hookio

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// No payload is not an error: a nil reader and a stdin that carries
// nothing both report that none arrived. A payload that does not parse
// is the error.
func TestReadInputTellsNoPayloadFromABrokenOne(t *testing.T) {
	if _, ok, err := ReadInput(nil); ok || err != nil {
		t.Fatalf("nil reader: ok=%v err=%v, want no payload", ok, err)
	}
	for _, empty := range []string{"", "  \n"} {
		if _, ok, err := ReadInput(strings.NewReader(empty)); ok || err != nil {
			t.Fatalf("empty stdin %q: ok=%v err=%v, want no payload", empty, ok, err)
		}
	}
	if _, ok, err := ReadInput(strings.NewReader("not json")); ok || err == nil || !strings.Contains(err.Error(), "does not parse") {
		t.Fatalf("malformed payload: ok=%v err=%v", ok, err)
	}
}

func TestReadInputParsesPayload(t *testing.T) {
	in, ok, err := ReadInput(strings.NewReader(`{"cwd":"/x","tool_input":{"command":"git push","file_path":"a.md"},"tool_response":"done"}`))
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if in.CWD != "/x" || in.ToolInput.Command != "git push" || in.ToolInput.FilePath != "a.md" {
		t.Fatalf("payload: %+v", in)
	}
	if in.ToolResponseString() != "done" {
		t.Fatalf("response: %q", in.ToolResponseString())
	}
}

func TestToolResponseObjectPassesThrough(t *testing.T) {
	in := Input{ToolResponse: json.RawMessage(`{"filePath":"/p.md"}`)}
	if got := in.ToolResponseString(); got != `{"filePath":"/p.md"}` {
		t.Fatalf("got %q", got)
	}
}

func TestResolveDirPrecedence(t *testing.T) {
	env := func(k string) string {
		if k == "CLAUDE_PROJECT_DIR" {
			return "/started-here"
		}
		return ""
	}
	noEnv := func(string) string { return "" }
	if got := ResolveDir(env, "/session-is-here", "/stated", true); got != "/stated" {
		t.Fatalf("an explicit project dir beats everything: %q", got)
	}
	if got := ResolveDir(env, "/session-is-here", "/default", false); got != "/session-is-here" {
		t.Fatalf("the payload's cwd beats where the session started: %q", got)
	}
	if got := ResolveDir(env, "", "/default", false); got != "/started-here" {
		t.Fatalf("CLAUDE_PROJECT_DIR fills in when the payload has no cwd: %q", got)
	}
	if got := ResolveDir(noEnv, "", "/default", false); got != "/default" {
		t.Fatalf("the process directory is the last resort: %q", got)
	}
}

func TestWritePermissionDecision(t *testing.T) {
	var b bytes.Buffer
	if err := WritePermissionDecision(&b, PermissionDeny, "because"); err != nil {
		t.Fatal(err)
	}
	var out struct {
		H struct {
			Event    string `json:"hookEventName"`
			Decision string `json:"permissionDecision"`
			Reason   string `json:"permissionDecisionReason"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal(b.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.H.Event != "PreToolUse" || out.H.Decision != "deny" || out.H.Reason != "because" {
		t.Fatalf("decision: %+v", out.H)
	}
}

func TestWritePostToolUseOmitsEmptyContext(t *testing.T) {
	var b bytes.Buffer
	WritePostToolUse(&b, "saved", "")
	if strings.Contains(b.String(), "hookSpecificOutput") {
		t.Fatalf("empty context should omit hookSpecificOutput: %s", b.String())
	}
	b.Reset()
	WritePostToolUse(&b, "ready", "tell them")
	s := b.String()
	if !strings.Contains(s, `"hookEventName":"PostToolUse"`) || !strings.Contains(s, "tell them") {
		t.Fatalf("context form: %s", s)
	}
}

// A lifecycle hook command reaches the shell with pk's variables
// filled in and every other reference as it was written.
func TestRunScriptLeavesOtherVariablesToTheShell(t *testing.T) {
	var out bytes.Buffer
	command := `FOO=bar; echo "[${FOO}_x] [${NOPE:-fallback}] [v${VERSION}] [$VERSION] [${VERSION:-none}]"`
	if err := RunScript(&out, t.TempDir(), command, map[string]string{"VERSION": "1.2.3"}); err != nil {
		t.Fatalf("%v: %s", err, out.String())
	}
	if got, want := strings.TrimSpace(out.String()), "[bar_x] [fallback] [v1.2.3] [1.2.3] [1.2.3]"; got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}

func TestExpandSubstitutesOnlyItsOwnNames(t *testing.T) {
	env := map[string]string{"VERSION": "1.2.3", "TAG": "v1.2.3"}
	cases := map[string]string{
		"$VERSION":            "1.2.3",
		"${VERSION}":          "1.2.3",
		"v${VERSION}-$TAG":    "v1.2.3-v1.2.3",
		"$VERSIONS":           "$VERSIONS",
		"${VERSION:-none}":    "${VERSION:-none}",
		"${FOO}_x":            "${FOO}_x",
		"${NOPE:-fallback}":   "${NOPE:-fallback}",
		"$(date) $$ $? $1":    "$(date) $$ $? $1",
		"${VERSION":           "${VERSION",
		"cost: 5$":            "cost: 5$",
		"${}":                 "${}",
		"no references here":  "no references here",
		"$TAG$VERSION${TAG}$": "v1.2.31.2.3v1.2.3$",
	}
	for in, want := range cases {
		if got := expand(in, env); got != want {
			t.Errorf("expand(%q) = %q, want %q", in, got, want)
		}
	}
	if got := expand("$VERSION", nil); got != "$VERSION" {
		t.Errorf("with no env: %q", got)
	}
}
