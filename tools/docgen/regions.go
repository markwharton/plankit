package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/markwharton/plankit/internal/cli"
	"github.com/markwharton/plankit/internal/config"
	"github.com/markwharton/plankit/internal/registry"
)

// Generated regions. A page can carry a block that docgen writes from
// the code, between markers docgen owns:
//
//	<!-- generated: flags -->
//	...
//	<!-- /generated: flags -->
//
// The Flags section of a command page lists the command's own flags
// as --help prints them; the overview lists the flags every command
// accepts. Both are cli.FlagBlock renderings of the same FlagSpec
// values the parser registers, so a page and --help cannot disagree.
// make docs rewrites the regions before compiling, and CI's drift
// check fails a page whose region is stale.

const (
	regionOpen  = "<!-- generated: %s -->"
	regionClose = "<!-- /generated: %s -->"
)

// updateRegions rewrites every generated region under skillsDir and
// returns the pages it changed.
func updateRegions(skillsDir string) ([]string, error) {
	var changed []string
	for _, cmd := range registry.Commands() {
		path := filepath.Join(skillsDir, cmd.Name, "SKILL.md")
		raw, err := os.ReadFile(path)
		if err != nil {
			continue // the invariant test reports a missing page
		}
		var body string
		if len(cmd.Flags) > 0 {
			body = "```\n" + cli.FlagBlock(cmd.Flags) + "```\n"
		}
		out, err := setRegion(string(raw), "flags", "## Flags", body)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		if out != string(raw) {
			if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
				return nil, err
			}
			changed = append(changed, cmd.Name)
		}
	}
	// Settings, on the page named after each section of the policy
	// file. A section without a page is an error: every documented key
	// must have a home.
	for _, section := range config.Sections() {
		settings := config.SettingsFor(section)
		path := filepath.Join(skillsDir, section, "SKILL.md")
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("settings section %q has no page skills/%s/SKILL.md to carry it", section, section)
		}
		out, err := setRegion(string(raw), "settings", "## Settings", settingsSection(section, settings))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		if out != string(raw) {
			if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
				return nil, err
			}
			changed = append(changed, section)
		}
	}
	// Universal flags, once, on the overview.
	path := filepath.Join(skillsDir, "plankit", "SKILL.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		return changed, nil
	}
	body := "Every command accepts these:\n\n```\n" + cli.FlagBlock(cli.UniversalFlags()) + "```\n"
	out, err := setRegion(string(raw), "universal-flags", "## Universal flags", body)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if out != string(raw) {
		if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
			return nil, err
		}
		changed = append(changed, "plankit")
	}
	return changed, nil
}

// setRegion replaces the named region's content, appends the section
// with the region when the page lacks it and body is non-empty, and
// removes the section when body is empty.
func setRegion(page, name, heading, body string) (string, error) {
	open := fmt.Sprintf(regionOpen, name)
	close := fmt.Sprintf(regionClose, name)
	start := strings.Index(page, open)
	end := strings.Index(page, close)
	switch {
	case start >= 0 && end < 0, start < 0 && end >= 0:
		return "", fmt.Errorf("region %q has one marker without the other", name)
	case start >= 0 && end < start:
		return "", fmt.Errorf("region %q closes before it opens", name)
	case start < 0:
		if body == "" {
			return page, nil
		}
		return strings.TrimRight(page, "\n") + "\n\n" + heading + "\n\n" + open + "\n" + body + close + "\n", nil
	}
	// Replace the existing region, and its heading and section when the
	// body is now empty.
	if body == "" {
		sectionStart := strings.LastIndex(page[:start], heading)
		if sectionStart < 0 {
			sectionStart = start
		}
		return strings.TrimRight(page[:sectionStart], "\n") + "\n" + strings.TrimLeft(page[end+len(close):], "\n"), nil
	}
	return page[:start] + open + "\n" + body + close + page[end+len(close):], nil
}

// settingsSection renders a page's Settings: the section's shape in
// .pk.json as a block, then the list of keys with allowed values or
// kind, default, and meaning, then the load rule.
func settingsSection(section string, settings []config.Setting) string {
	var b strings.Builder
	b.WriteString("The `" + section + "` section of `.pk.json`:\n\n```\n")
	b.WriteString(shapeBlock(section, settings))
	b.WriteString("```\n\n")
	b.WriteString(settingsList(settings))
	return b.String()
}

// shapeBlock renders the section's keys as pseudo-JSON, nesting on
// dots, one key per line, sorted as the table is.
func shapeBlock(section string, settings []config.Setting) string {
	type node struct {
		keys     []string
		children map[string]*node
		shape    string
	}
	root := &node{children: map[string]*node{}}
	for _, s := range settings {
		parts := strings.Split(strings.TrimPrefix(s.Key, section+"."), ".")
		cur := root
		for i, p := range parts {
			if cur.children[p] == nil {
				cur.children[p] = &node{children: map[string]*node{}}
				cur.keys = append(cur.keys, p)
			}
			cur = cur.children[p]
			if i == len(parts)-1 {
				cur.shape = s.ShapeText()
			}
		}
	}
	var b strings.Builder
	var emit func(n *node, indent string)
	emit = func(n *node, indent string) {
		for i, k := range n.keys {
			child := n.children[k]
			comma := ","
			if i == len(n.keys)-1 {
				comma = ""
			}
			if len(child.children) > 0 {
				b.WriteString(indent + `"` + k + `": {` + "\n")
				emit(child, indent+"  ")
				b.WriteString(indent + "}" + comma + "\n")
			} else {
				b.WriteString(indent + `"` + k + `": ` + child.shape + comma + "\n")
			}
		}
	}
	b.WriteString(`"` + section + `": {` + "\n")
	emit(root, "  ")
	b.WriteString("}\n")
	return b.String()
}

// settingsList renders the keys as a list: key, allowed values or
// kind, default, meaning; then the load rule.
func settingsList(settings []config.Setting) string {
	var b strings.Builder
	for _, s := range settings {
		b.WriteString("- `" + s.Key + "`: ")
		if n := len(s.Values); n > 0 {
			for i, v := range s.Values {
				switch {
				case n == 2 && i == 1:
					b.WriteString(" or `" + v + "`")
				case n > 2 && i == n-1:
					b.WriteString(", or `" + v + "`")
				case i > 0:
					b.WriteString(", `" + v + "`")
				default:
					b.WriteString("`" + v + "`")
				}
			}
		} else {
			b.WriteString(s.Kind)
		}
		def := s.Default
		if len(s.Values) > 0 {
			def = "`" + def + "`"
		}
		b.WriteString("; default " + def + ". " + s.Doc + "\n")
	}
	b.WriteString("\n" + config.LoadRule + "\n")
	return b.String()
}
