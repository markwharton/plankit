// Package hookio is the Claude Code hook wire format: the stdin payload
// and the stdout decisions. Ported from v1, which already spoke the
// current schema (hookSpecificOutput with permissionDecision for
// PreToolUse, systemMessage plus additionalContext for PostToolUse).
//
// A hook command never blocks work by failing. It exits 0 when it acts
// or declines, reports a problem it cannot act on (a policy file that
// fails to load, a payload that does not parse) with ExitReport, and
// never exits 2: stdin comes from the trusted harness, and failing
// closed would block every tool call while, say, .pk.json is mid-edit.
// The hooks are guardrails against an agent following its defaults,
// not a security boundary against an adversary.
package hookio

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/markwharton/plankit/internal/cli"
	"github.com/markwharton/plankit/internal/msg"
)

// Input is the JSON payload Claude Code writes to a hook's stdin.
type Input struct {
	ToolInput    *ToolInput      `json:"tool_input,omitempty"`
	ToolResponse json.RawMessage `json:"tool_response,omitempty"`
	CWD          string          `json:"cwd,omitempty"`
}

// ToolInput carries the tool-call parameters the hooks inspect.
type ToolInput struct {
	FilePath string `json:"file_path,omitempty"`
	Command  string `json:"command,omitempty"`
}

// ReadInput reads and parses the hook payload. The bool reports that a
// payload arrived; the error is a payload that could not be read or
// does not parse. No payload is neither: a nil reader, a terminal
// (someone ran the hook command by hand, and it returns at once instead
// of blocking), or a stdin that carries nothing. An open pipe with
// nothing on it (a shell tool that keeps stdin attached) gets the same
// answer after a short wait: a real hook writes its payload and closes
// stdin at once, so a second of silence means no payload. The wait is a
// goroutine racing a timer because stdin is a blocking descriptor
// outside Go's poller and read deadlines cannot attach to it; on
// timeout the reader goroutine is abandoned, which is fine for a
// process about to exit.
func ReadInput(r io.Reader) (Input, bool, error) {
	if r == nil {
		return Input{}, false, nil
	}
	var data []byte
	var err error
	if f, ok := r.(*os.File); ok {
		if stat, serr := f.Stat(); serr == nil && stat.Mode()&os.ModeCharDevice != 0 {
			return Input{}, false, nil
		}
		type result struct {
			data []byte
			err  error
		}
		done := make(chan result, 1)
		go func() {
			d, e := io.ReadAll(f)
			done <- result{d, e}
		}()
		select {
		case res := <-done:
			data, err = res.data, res.err
		case <-time.After(time.Second):
			return Input{}, false, nil
		}
	} else {
		data, err = io.ReadAll(r)
	}
	if err != nil {
		return Input{}, false, err
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return Input{}, false, nil
	}
	var input Input
	if err := json.Unmarshal(data, &input); err != nil {
		return Input{}, false, fmt.Errorf("the payload does not parse: %v", err)
	}
	return input, true, nil
}

// ToolResponseString returns tool_response as text: unquoted when it is
// a JSON string, raw JSON otherwise, "" when absent.
func (i Input) ToolResponseString() string {
	if len(i.ToolResponse) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(i.ToolResponse, &s); err == nil {
		return s
	}
	return string(i.ToolResponse)
}

// ResolveDir picks the directory a hook acts on. Precedence, most
// deliberate first: a project directory the person stated
// (--project-dir or PK_PROJECT_DIR) always wins; then the payload's cwd,
// the hook's own statement of where the session is now; then
// CLAUDE_PROJECT_DIR, where the session was started, for payloads that
// carry no cwd; then the context's default, the process directory.
// Explicit beats ambient, and where the session is beats where it began.
func ResolveDir(env func(string) string, payloadCWD, contextDir string, contextExplicit bool) string {
	if contextExplicit && contextDir != "" {
		return contextDir
	}
	if payloadCWD != "" {
		return payloadCWD
	}
	if dir := env("CLAUDE_PROJECT_DIR"); dir != "" {
		return dir
	}
	return contextDir
}

// Permission decisions accepted inside hookSpecificOutput for PreToolUse.
const (
	PermissionAllow = "allow"
	PermissionDeny  = "deny"
	PermissionAsk   = "ask"
)

// WritePermissionDecision emits a PreToolUse decision for the named
// hook. The reason is written as "pk <hook>: <reason>", the prefix a
// hook's stderr lines carry, so the person reading it knows which hook
// spoke; callers write the reason alone. The schema requires
// hookEventName whenever hookSpecificOutput is present, so it is set
// here; callers don't have to remember.
func WritePermissionDecision(w io.Writer, hook, decision, reason string) error {
	resp := struct {
		HookSpecificOutput struct {
			HookEventName            string `json:"hookEventName"`
			PermissionDecision       string `json:"permissionDecision"`
			PermissionDecisionReason string `json:"permissionDecisionReason"`
		} `json:"hookSpecificOutput"`
	}{}
	resp.HookSpecificOutput.HookEventName = "PreToolUse"
	resp.HookSpecificOutput.PermissionDecision = decision
	resp.HookSpecificOutput.PermissionDecisionReason = "pk " + hook + ": " + reason
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// WritePostToolUse emits a PostToolUse response: systemMessage is shown
// to the person, additionalContext (optional) is injected into Claude's
// next turn. hookSpecificOutput is omitted entirely when there is no
// context, and hookEventName is set whenever it is present.
func WritePostToolUse(w io.Writer, systemMessage, additionalContext string) error {
	type specific struct {
		HookEventName     string `json:"hookEventName"`
		AdditionalContext string `json:"additionalContext,omitempty"`
	}
	resp := struct {
		SystemMessage      string    `json:"systemMessage,omitempty"`
		HookSpecificOutput *specific `json:"hookSpecificOutput,omitempty"`
	}{SystemMessage: systemMessage}
	if additionalContext != "" {
		resp.HookSpecificOutput = &specific{
			HookEventName:     "PostToolUse",
			AdditionalContext: additionalContext,
		}
	}
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// RunScript runs a lifecycle hook command through the platform shell
// ("sh -c", or "cmd /c" on Windows), inheriting output onto w. env
// variables are pre-expanded in the command string ($VAR and ${VAR}) and
// also set as real environment variables, so a hook can write $VERSION
// once and have it work on every platform (cmd does not expand $VAR).
func RunScript(w io.Writer, dir, command string, env map[string]string) error {
	command = expand(command, env)
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Stdout = w
	cmd.Stderr = w
	if len(env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}
	return cmd.Run()
}

// RunHook runs one lifecycle hook from the policy file. name is the
// hook's key there (postVersion, preCommit, preRelease, prePush), the
// word a person searches .pk.json for. It narrates the hook under that
// name, runs the command with RunScript, and returns the one refusal
// for a hook that fails. A hook that is not set runs nothing.
func RunHook(w io.Writer, dir, name, command string, env map[string]string) error {
	if command == "" {
		return nil
	}
	fmt.Fprintln(w, "")
	msg.Section(w, name+" hook")
	msg.Itemf(w, "%s", command)
	if err := RunScript(w, dir, command, env); err != nil {
		return cli.Statef("%s hook failed: %v", name, err)
	}
	msg.Itemf(w, "Hook passed")
	return nil
}

// expand substitutes $NAME and ${NAME} in command for the names env
// holds, and copies every other byte as it was written. A variable that
// is not pk's, and every other form the shell reads after a dollar sign
// (${NAME:-default}, $(command), $$), reaches the shell untouched.
func expand(command string, env map[string]string) string {
	var b strings.Builder
	for i := 0; i < len(command); {
		if command[i] == '$' {
			if name, end := reference(command, i); name != "" {
				if v, ok := env[name]; ok {
					b.WriteString(v)
					i = end
					continue
				}
			}
		}
		b.WriteByte(command[i])
		i++
	}
	return b.String()
}

// reference reads a plain variable reference at the dollar sign s[i],
// $NAME or ${NAME}, and returns the name and the index just past the
// reference. The name is "" when the dollar sign opens anything else.
func reference(s string, i int) (string, int) {
	j := i + 1
	braced := j < len(s) && s[j] == '{'
	if braced {
		j++
	}
	start := j
	for j < len(s) && isNameByte(s[j], j == start) {
		j++
	}
	name := s[start:j]
	if braced {
		if j >= len(s) || s[j] != '}' {
			return "", i
		}
		j++
	}
	return name, j
}

// isNameByte reports whether c may appear in a shell variable name:
// a letter or an underscore anywhere, a digit after the first byte.
func isNameByte(c byte, first bool) bool {
	switch {
	case c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z':
		return true
	case c >= '0' && c <= '9':
		return !first
	}
	return false
}

// WriteSessionStart emits a SessionStart response whose additionalContext
// is injected into the session as it begins. Empty context writes
// nothing at all, which Claude Code treats as no-op.
func WriteSessionStart(w io.Writer, additionalContext string) error {
	if additionalContext == "" {
		return nil
	}
	resp := struct {
		HookSpecificOutput struct {
			HookEventName     string `json:"hookEventName"`
			AdditionalContext string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}{}
	resp.HookSpecificOutput.HookEventName = "SessionStart"
	resp.HookSpecificOutput.AdditionalContext = additionalContext
	return json.NewEncoder(w).Encode(resp)
}

// Report writes a problem a hook cannot act on, under the hook's name,
// and returns the exit that makes Claude Code show it. It is the one
// way a hook says something went wrong: the line on stderr and
// ExitReport always travel together.
func Report(w io.Writer, hook string, err error) error {
	msg.Hookf(w, hook, "%v", err)
	return cli.Silent(ExitReport)
}

// ExitReport is the exit code a hook uses to report a problem it
// cannot act on: a policy file that fails to load, a payload that does
// not parse. Claude Code shows the hook's stderr to the person and
// continues; exit 2 would block the tool call, so a hook never uses
// it, and exit 0 would hide the message.
const ExitReport = 1
