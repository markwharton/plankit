// Package hookio is the Claude Code hook wire format: the stdin payload
// and the stdout decisions. Ported from v1, which already spoke the
// current schema (hookSpecificOutput with permissionDecision for
// PreToolUse, systemMessage plus additionalContext for PostToolUse).
//
// Hook commands always exit 0 and fail open: stdin comes from the
// trusted harness, and failing closed would block every tool call
// while, say, .pk.json is mid-edit. The hooks are guardrails against
// an agent following its defaults, not a security boundary against an
// adversary.
package hookio

import (
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"runtime"
	"time"
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

// ReadInput reads and parses the hook payload. When stdin is a terminal
// (someone ran the hook command by hand), it returns io.EOF immediately
// instead of blocking; callers treat that as "no payload". An open pipe
// with nothing on it (a shell tool that keeps stdin attached) gets the
// same answer after a short wait: a real hook writes its payload and
// closes stdin at once, so a second of silence means no payload. The
// wait is a goroutine racing a timer because stdin is a blocking
// descriptor outside Go's poller and read deadlines cannot attach to
// it; on timeout the reader goroutine is abandoned, which is fine for
// a process about to exit.
func ReadInput(r io.Reader) (Input, error) {
	if r == nil {
		return Input{}, io.EOF
	}
	var data []byte
	var err error
	if f, ok := r.(*os.File); ok {
		if stat, serr := f.Stat(); serr == nil && stat.Mode()&os.ModeCharDevice != 0 {
			return Input{}, io.EOF
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
			return Input{}, io.EOF
		}
	} else {
		data, err = io.ReadAll(r)
	}
	if err != nil {
		return Input{}, err
	}
	var input Input
	if err := json.Unmarshal(data, &input); err != nil {
		return Input{}, err
	}
	return input, nil
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

// WritePermissionDecision emits a PreToolUse decision. The schema
// requires hookEventName whenever hookSpecificOutput is present, so it
// is set here; callers don't have to remember.
func WritePermissionDecision(w io.Writer, decision, reason string) error {
	resp := struct {
		HookSpecificOutput struct {
			HookEventName            string `json:"hookEventName"`
			PermissionDecision       string `json:"permissionDecision"`
			PermissionDecisionReason string `json:"permissionDecisionReason"`
		} `json:"hookSpecificOutput"`
	}{}
	resp.HookSpecificOutput.HookEventName = "PreToolUse"
	resp.HookSpecificOutput.PermissionDecision = decision
	resp.HookSpecificOutput.PermissionDecisionReason = reason
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
