package cli

import "fmt"

// Exit codes are a stable contract; scripts and hooks may rely on them.
const (
	ExitOK       = 0 // success
	ExitUsage    = 1 // bad invocation: unknown command, flag, or argument
	ExitState    = 2 // precondition not met: not configured, not a repo, dirty tree
	ExitInternal = 3 // pk itself failed
)

// exitError carries an exit code, a message, and an optional hint that
// names the fix. Commands construct these via Usagef and Statef.
type exitError struct {
	code int
	msg  string
	hint string
}

func (e *exitError) Error() string { return e.msg }

// Usagef reports a bad invocation (exit 1).
func Usagef(format string, a ...any) error {
	return &exitError{code: ExitUsage, msg: fmt.Sprintf(format, a...)}
}

// Statef reports an unmet precondition (exit 2).
func Statef(format string, a ...any) error {
	return &exitError{code: ExitState, msg: fmt.Sprintf(format, a...)}
}

// Silent returns an error that carries only an exit code. Commands use
// it when the report is already written and the code is the remaining
// signal (pk status in an unconfigured repository).
func Silent(code int) error { return &exitError{code: code} }

// WithHint attaches a fix-naming hint to an error created by Usagef or
// Statef. Hints print after the error unless --quiet is set.
func WithHint(err error, format string, a ...any) error {
	if e, ok := err.(*exitError); ok {
		e.hint = fmt.Sprintf(format, a...)
	}
	return err
}
