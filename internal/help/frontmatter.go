package help

import (
	"fmt"
	"strings"
)

// SplitFrontmatter reads the leading "---" block of a page as flat
// "key: value" lines, and returns the fields and the body after the
// closing fence, untrimmed. The frontmatter is deliberately not YAML:
// a few flat keys need no dependency, and anything fancier fails here.
//
// It is the one reader of frontmatter. A skill, a release note, and
// pk help document each go through it, and each checks the keys that
// are its own.
func SplitFrontmatter(raw []byte) (map[string]string, []byte, error) {
	const fence = "---\n"
	s := string(raw)
	if !strings.HasPrefix(s, fence) {
		return nil, nil, fmt.Errorf("missing frontmatter (file must start with ---)")
	}
	rest := s[len(fence):]
	end := strings.Index(rest, "\n"+fence)
	if end < 0 {
		return nil, nil, fmt.Errorf("unterminated frontmatter")
	}
	fields := map[string]string{}
	for _, line := range strings.Split(rest[:end], "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return nil, nil, fmt.Errorf("frontmatter line %q is not key: value", line)
		}
		fields[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return fields, []byte(rest[end+1+len(fence):]), nil
}
