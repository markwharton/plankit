// Package hooks provides shared types and utilities for Claude Code hook commands.
package hooks

import (
	"encoding/json"
	"io"
)

// Input represents the JSON payload received from Claude Code hooks via stdin.
type Input struct {
	ToolInput    *ToolInput      `json:"tool_input,omitempty"`
	ToolResponse json.RawMessage `json:"tool_response,omitempty"`
	CWD          string          `json:"cwd,omitempty"`
}

// ToolInput contains the parameters passed to Edit/Write tool calls.
type ToolInput struct {
	FilePath string `json:"file_path,omitempty"`
}

// ReadInput reads and parses the Claude Code hook JSON payload from the given reader.
func ReadInput(r io.Reader) (Input, error) {
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
