package release

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/markwharton/plankit/internal/cli"
	"github.com/markwharton/plankit/internal/msg"
)

// PinCmd updates a version pin in a file: release hooks call it with
// $VERSION so pinned copies (workflows, install scripts, docs) track the
// release. Ported from v1's setup package; the pin formats survive even
// though setup did not.
var PinCmd = &cli.Command{
	Name:    "pin",
	Summary: "Update a version pin in a file (for release hooks)",
	Flags: []cli.FlagSpec{
		{Name: "file", Type: cli.StringFlag, Usage: "File containing the pin (relative to the project directory)"},
		{Name: "name", Type: cli.StringFlag, Usage: "Identifier of a named pin; default is the SOMETHING_VERSION=\"v...\" shell form"},
	},
	Run: runPin,
}

func runPin(ctx *cli.Context) error {
	file := ctx.String("file")
	if file == "" {
		return cli.Usagef("--file is required")
	}
	args := ctx.Args()
	if len(args) != 1 {
		return cli.Usagef("exactly one version argument is required (release hooks pass $VERSION)")
	}
	ver := args[0]
	path := file
	if !filepath.IsAbs(path) {
		path = filepath.Join(ctx.ProjectDir, file)
	}

	var updated bool
	var err error
	if name := ctx.String("name"); name != "" {
		updated, err = pinVersionNamed(path, name, ver)
	} else {
		updated, err = pinVersion(path, ver)
	}
	switch {
	case err == nil && updated:
		if !ctx.Quiet {
			msg.Notef(ctx.Stderr, "pinned %s to %s", file, ver)
		}
		return nil
	case err == nil:
		// Missing file is a no-op by contract: a pin target that a repo
		// does not carry never aborts a release from a hook.
		if !ctx.Quiet {
			msg.Notef(ctx.Stderr, "%s does not exist; nothing to pin", file)
		}
		return nil
	default:
		var noPin *noPinError
		if asNoPin(err, &noPin) {
			// Present but pinless is a warning, not a failure, so a
			// renamed or reformatted target never aborts a release.
			msg.Warnf(ctx.Stderr, "%v", err)
			return nil
		}
		return err
	}
}

// pinVersion updates the first SOMETHING_VERSION="vX.Y.Z" line (any
// uppercase variable ending in VERSION). updated=false with nil error
// means the file does not exist; a file without a pin returns
// *noPinError.
func pinVersion(path, ver string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, nil
	}
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		name, ok := versionPinName(line)
		if !ok {
			continue
		}
		lines[i] = fmt.Sprintf(`%s="v%s"`, name, strings.TrimPrefix(ver, "v"))
		if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o755); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, &noPinError{File: filepath.Base(path)}
}

// versionPinName matches SOMETHING_VERSION="v..." and returns the
// variable name.
func versionPinName(line string) (string, bool) {
	idx := strings.Index(line, `VERSION="v`)
	if idx < 0 {
		return "", false
	}
	name := line[:idx+len("VERSION")]
	for _, c := range name {
		if !((c >= 'A' && c <= 'Z') || c == '_') {
			return "", false
		}
	}
	if !strings.HasSuffix(line, `"`) {
		return "", false
	}
	return name, true
}

// pinVersionNamed updates the first line assigning a quoted string to
// the named identifier (=, :=, or a bare colon for YAML). The v prefix
// is inferred from the existing value.
func pinVersionNamed(path, name, ver string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, nil
	}
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		m, ok := matchNamedPin(line, name)
		if !ok {
			continue
		}
		newVer := strings.TrimPrefix(ver, "v")
		if strings.HasPrefix(m.value, "v") {
			newVer = "v" + newVer
		}
		lines[i] = m.linePrefix + newVer + string(m.quote) + m.lineSuffix
		if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, &noPinError{File: filepath.Base(path), Name: name}
}

type namedPinMatch struct {
	linePrefix string // through the opening quote
	lineSuffix string // after the closing quote
	value      string
	quote      byte
}

// matchNamedPin finds `name = "value"` at a word boundary, accepting =,
// :=, or a bare colon, with either quote character.
func matchNamedPin(line, name string) (namedPinMatch, bool) {
	pos := 0
	for {
		idx := strings.Index(line[pos:], name)
		if idx < 0 {
			return namedPinMatch{}, false
		}
		idx += pos
		if idx > 0 && isIdentChar(line[idx-1]) {
			pos = idx + len(name)
			continue
		}
		after := idx + len(name)
		if after < len(line) && isIdentChar(line[after]) {
			pos = after
			continue
		}
		i := after
		for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
			i++
		}
		if i >= len(line) {
			pos = after
			continue
		}
		switch line[i] {
		case '=':
			i++
		case ':':
			i++
			if i < len(line) && line[i] == '=' {
				i++
			}
		default:
			pos = after
			continue
		}
		for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
			i++
		}
		if i >= len(line) {
			pos = after
			continue
		}
		q := line[i]
		if q != '"' && q != '\'' {
			pos = after
			continue
		}
		i++
		closeIdx := strings.IndexByte(line[i:], q)
		if closeIdx < 0 {
			pos = after
			continue
		}
		return namedPinMatch{
			linePrefix: line[:i],
			lineSuffix: line[i+closeIdx+1:],
			value:      line[i : i+closeIdx],
			quote:      q,
		}, true
	}
}

func isIdentChar(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_'
}

// noPinError reports a file that exists but carries no matching pin.
type noPinError struct {
	File string
	Name string
}

func (e *noPinError) Error() string {
	if e.Name == "" {
		return e.File + " has no VERSION pin"
	}
	return fmt.Sprintf("%s has no pin for %q", e.File, e.Name)
}

func asNoPin(err error, target **noPinError) bool {
	np, ok := err.(*noPinError)
	if ok {
		*target = np
	}
	return ok
}
