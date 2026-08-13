package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ScriptVersion reads the pinned version from a file.
// Returns the version string and true if found, or ("", false) if the file
// does not exist or has no VERSION pin.
func ScriptVersion(readFile func(string) ([]byte, error), filePath string) (string, bool) {
	data, err := readFile(filePath)
	if err != nil {
		return "", false
	}
	for _, line := range strings.Split(string(data), "\n") {
		if _, ok := versionPinName(line); ok {
			idx := strings.Index(line, `="`)
			if idx >= 0 && strings.HasSuffix(line, `"`) {
				return line[idx+2 : len(line)-1], true
			}
		}
	}
	return "", false
}

// PinVersion updates a shell-variable version pin in a file. It finds the first
// line matching SOMETHING_VERSION="vX.Y.Z" (any uppercase variable ending in
// VERSION) and replaces the version.
// Returns (updated, error). updated is true if the file was rewritten, false
// with nil error if the file does not exist (no-op); a missing VERSION pin
// returns a *NoPinError.
func PinVersion(readFile func(string) ([]byte, error), writeFile func(string, []byte, os.FileMode) error, filePath string, ver string) (bool, error) {
	data, err := readFile(filePath)
	if err != nil {
		return false, nil
	}
	if !strings.HasPrefix(ver, "v") {
		ver = "v" + ver
	}
	lines := strings.Split(string(data), "\n")
	found := false
	for i, line := range lines {
		if name, ok := versionPinName(line); ok {
			lines[i] = fmt.Sprintf(`%s="v%s"`, name, strings.TrimPrefix(ver, "v"))
			found = true
			break
		}
	}
	if !found {
		return false, &NoPinError{File: filepath.Base(filePath)}
	}
	if err := writeFile(filePath, []byte(strings.Join(lines, "\n")), 0755); err != nil {
		return false, err
	}
	return true, nil
}

// versionPinName checks if a line matches the pattern SOMETHING_VERSION="v..."
// and returns the variable name. Returns ("", false) if no match.
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

// NoPinError reports that the file exists but contains no matching pin.
// The CLI treats it as a warning rather than a fatal error, so a renamed or
// reformatted pin target never aborts a release from a preCommit hook.
type NoPinError struct {
	File string // base name of the target file
	Name string // identifier from --name; empty for the shell-variable format
}

func (e *NoPinError) Error() string {
	if e.Name == "" {
		return e.File + " has no VERSION pin"
	}
	return fmt.Sprintf("%s has no pin for %q", e.File, e.Name)
}

// namedPinMatch holds the result of scanning a line for a named version pin.
type namedPinMatch struct {
	linePrefix string // everything up to and including the opening quote
	lineSuffix string // everything after the closing quote
	value      string // version string between quotes
	quote      byte   // quote character (' or ")
}

// matchNamedPin checks if a line contains an assignment of a quoted string to
// the given identifier name. The name must appear at a word boundary and be
// followed by =, :=, or a bare colon, then a quoted value.
func matchNamedPin(line, name string) (namedPinMatch, bool) {
	pos := 0
	for {
		idx := strings.Index(line[pos:], name)
		if idx < 0 {
			return namedPinMatch{}, false
		}
		idx += pos

		// Word boundary before: must be start of line or non-identifier char.
		if idx > 0 && isIdentChar(line[idx-1]) {
			pos = idx + len(name)
			continue
		}
		// Word boundary after: must be end or non-identifier char.
		after := idx + len(name)
		if after < len(line) && isIdentChar(line[after]) {
			pos = after
			continue
		}

		// Skip whitespace after name.
		i := after
		for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
			i++
		}

		// Expect =, :=, or a bare colon (YAML and frontmatter).
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

		// Skip whitespace after operator.
		for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
			i++
		}

		// Expect opening quote.
		if i >= len(line) {
			pos = after
			continue
		}
		q := line[i]
		if q != '"' && q != '\'' {
			pos = after
			continue
		}
		i++ // skip opening quote

		// Find closing quote.
		closeIdx := strings.IndexByte(line[i:], q)
		if closeIdx < 0 {
			pos = after
			continue
		}

		value := line[i : i+closeIdx]
		return namedPinMatch{
			linePrefix: line[:i],
			lineSuffix: line[i+closeIdx+1:],
			value:      value,
			quote:      q,
		}, true
	}
}

func isIdentChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}

// PinVersionNamed updates a named version pin in a file. It finds the first
// line where the identifier name is assigned a quoted string value and replaces
// that value with ver. The v-prefix is inferred from the existing value.
// Returns (updated, error). updated is false with nil error if the file does
// not exist; a file with no matching pin returns a *NoPinError.
func PinVersionNamed(readFile func(string) ([]byte, error), writeFile func(string, []byte, os.FileMode) error, filePath, name, ver string) (bool, error) {
	data, err := readFile(filePath)
	if err != nil {
		return false, nil
	}
	lines := strings.Split(string(data), "\n")
	found := false
	for i, line := range lines {
		m, ok := matchNamedPin(line, name)
		if !ok {
			continue
		}
		// Infer v-prefix from existing value.
		newVer := strings.TrimPrefix(ver, "v")
		if strings.HasPrefix(m.value, "v") {
			newVer = "v" + newVer
		}
		lines[i] = m.linePrefix + newVer + string(m.quote) + m.lineSuffix
		found = true
		break
	}
	if !found {
		return false, &NoPinError{File: filepath.Base(filePath), Name: name}
	}
	if err := writeFile(filePath, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		return false, err
	}
	return true, nil
}

// ReadVersionNamed reads the pinned version for the given identifier name.
// Returns the version string and true if found, or ("", false) if not.
func ReadVersionNamed(readFile func(string) ([]byte, error), filePath, name string) (string, bool) {
	data, err := readFile(filePath)
	if err != nil {
		return "", false
	}
	for _, line := range strings.Split(string(data), "\n") {
		if m, ok := matchNamedPin(line, name); ok {
			return m.value, true
		}
	}
	return "", false
}
