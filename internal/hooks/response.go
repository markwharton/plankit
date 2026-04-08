package hooks

import (
	"fmt"
	"io"
)

// WriteBlockDecision writes a PreToolUse block response to w.
// The reason is Go-quoted which produces valid JSON string escaping.
func WriteBlockDecision(w io.Writer, reason string) {
	fmt.Fprintf(w, `{"decision":"block","reason":%q}`, reason)
}
