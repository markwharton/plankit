// Package hooks provides shared types and utilities for Claude Code hook commands.
package hooks

import (
	"encoding/json"
	"io"
	"os"
)

// Input represents the JSON payload received from Claude Code hooks via stdin.
type Input struct {
	ToolInput    *ToolInput      `json:"tool_input,omitempty"`
	ToolResponse json.RawMessage `json:"tool_response,omitempty"`
	CWD          string          `json:"cwd,omitempty"`
}

// ToolInput contains the parameters passed to tool calls.
type ToolInput struct {
	FilePath string `json:"file_path,omitempty"`
	Command  string `json:"command,omitempty"`
}

// ReadInput reads and parses the Claude Code hook JSON payload from the given reader.
// If stdin is a terminal (not a pipe), it returns an error immediately to avoid blocking.
func ReadInput(r io.Reader) (Input, error) {
	if f, ok := r.(*os.File); ok {
		if stat, err := f.Stat(); err == nil {
			if stat.Mode()&os.ModeCharDevice != 0 {
				return Input{}, io.EOF
			}
		}
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return Input{}, err
	}
	var input Input
	if err := json.Unmarshal(data, &input); err != nil {
		return Input{}, err
	}
	return input, nil
}

// ResolveProjectDir determines the project directory from the CLAUDE_PROJECT_DIR
// environment variable or the hook payload's CWD field.
func ResolveProjectDir(env func(string) string, inputCWD string) string {
	projectDir := env("CLAUDE_PROJECT_DIR")
	if projectDir == "" {
		projectDir = inputCWD
	}
	return projectDir
}

// ToolResponseString returns the tool_response field as a string.
// If tool_response is a JSON string, it is unquoted.
// If it is an object or array, it is returned as raw JSON text.
// If empty or null, an empty string is returned.
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
