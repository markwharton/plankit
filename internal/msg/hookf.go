package msg

import (
	"fmt"
	"io"
)

// Hookf writes a hook diagnostic to w. Hook commands fail open, so these
// stderr lines are the only trace when something goes wrong; the prefix
// names which hook spoke.
func Hookf(w io.Writer, hook, format string, a ...any) {
	fmt.Fprintf(w, "pk %s: %s\n", hook, fmt.Sprintf(format, a...))
}
