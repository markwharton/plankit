package cli

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/markwharton/plankit/internal/git"
)

// Style is the presentation decision for this invocation's stdout.
type Style int

const (
	StyleNone Style = iota // undecorated: no escape codes
	StyleANSI              // conservative ANSI: bold, dim, underline, 16-color
)

// Context carries everything a command needs, resolved once by the frame
// before Run is called. Commands never inspect flags, the environment, or
// the terminal themselves.
type Context struct {
	Command    string
	Commands   []*Command // full registry, for help and usage generation
	ProjectDir string     // absolute; "." resolves to the enclosing git root
	// ProjectDirExplicit is true when the person stated the project
	// directory (--project-dir or PK_PROJECT_DIR) rather than letting it
	// default to the current directory. Hooks use it to keep an explicit
	// instruction above anything the environment or payload says.
	ProjectDirExplicit bool
	Format             string // "text" or "json"
	Style              Style
	Width              int  // wrap width for styled text output; 0 means no wrapping
	IsTTY              bool // stdout is a terminal
	Quiet              bool
	Stdin              io.Reader
	Stdout             io.Writer
	Stderr             io.Writer

	bools map[string]*bool
	strs  map[string]*string
	args  []string
}

// Bool returns a declared bool flag's value. Unknown names are programmer
// errors and panic.
func (c *Context) Bool(name string) bool {
	v, ok := c.bools[name]
	if !ok {
		panic("cli: undeclared bool flag " + name)
	}
	return *v
}

// String returns a declared string flag's value. Unknown names are
// programmer errors and panic.
func (c *Context) String(name string) string {
	v, ok := c.strs[name]
	if !ok {
		panic("cli: undeclared string flag " + name)
	}
	return *v
}

// Args returns the positional arguments after flag parsing.
func (c *Context) Args() []string { return c.args }

// resolve applies the resolution stack: flags, then environment, then
// detection. It fills ProjectDir, Format, Style, Width, and IsTTY.
func (c *Context) resolve(projectDir, format string, formats []string, plain bool) error {
	if len(formats) == 0 {
		formats = defaultFormats
	}
	if !accepts(formats, format) {
		return Usagef("invalid --format %q (must be %s)", format, JoinList(formats, "or"))
	}
	c.Format = format

	dir := projectDir
	if dir == "" {
		dir = os.Getenv("PK_PROJECT_DIR")
	}
	c.ProjectDirExplicit = dir != ""
	if dir == "" {
		dir = "."
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return Usagef("resolving --project-dir: %v", err)
	}
	// A default or explicit "." means "this project": resolve to the
	// enclosing git root, so pk works the same from any subdirectory.
	// Any other explicit path is taken as given.
	if dir == "." {
		if root, ok := git.FindRoot(abs); ok {
			abs = root
		}
	}
	c.ProjectDir = abs

	c.Style, c.Width, c.IsTTY = presentation(c.Stdout, plain)
	return nil
}

// defaultFormats are the values --format accepts when a command names
// no set of its own: the data formats.
var defaultFormats = []string{"text", "json"}

func accepts(formats []string, format string) bool {
	for _, f := range formats {
		if f == format {
			return true
		}
	}
	return false
}

// JoinList joins items into the list a sentence carries: "a", "a and
// b", "a, b, and c", with conjunction as the word before the last item.
// It is the one joiner: a usage error and the generated pages both read
// their lists through it, so they punctuate alike.
func JoinList(items []string, conjunction string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	case 2:
		return items[0] + " " + conjunction + " " + items[1]
	}
	return strings.Join(items[:len(items)-1], ", ") + ", " + conjunction + " " + items[len(items)-1]
}

// presentation decides style and width for a writer.
//
// Order: --plain, then NO_COLOR, then CLICOLOR_FORCE, then the TTY probe.
// Width comes from the terminal, clamped to 40..100; non-TTY output gets
// width 0, meaning raw, unwrapped bytes for whoever is parsing them.
func presentation(w io.Writer, plain bool) (Style, int, bool) {
	width, tty := 0, false
	var fd uintptr
	if f, ok := w.(*os.File); ok {
		fd = f.Fd()
		width, tty = termSize(fd)
	}
	if tty {
		width = clamp(width, 40, 100)
	}

	style := StyleNone
	switch {
	case plain:
		return StyleNone, 0, tty
	case os.Getenv("NO_COLOR") != "":
		style = StyleNone
	case forceColor():
		style = StyleANSI
	case tty:
		style = StyleANSI
	}
	if style == StyleANSI && tty && !enableVT(fd) {
		style = StyleNone // console refused VT processing: no escape litter
	}
	return style, width, tty
}

func forceColor() bool {
	v := os.Getenv("CLICOLOR_FORCE")
	return v != "" && v != "0"
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
