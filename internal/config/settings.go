package config

import "strings"

// Setting describes one key of the policy file: its allowed values or
// kind, its default, and one line of meaning. This table is the single
// source for what the file may contain. Validate reads it for the
// enumerated keys, and docgen writes each page's Settings section from
// it, so the page and the loader cannot disagree.
type Setting struct {
	Key     string   // dotted path in .pk.json
	Values  []string // allowed values for an enumerated setting; empty for others
	Kind    string   // what the value is, for a non-enumerated setting
	Shape   string   // the value's JSON shape, for the reference block; enums derive theirs
	Default string   // the default, as shown to a reader
	Doc     string   // one sentence of meaning
	value   func(*PkConfig) string
}

// Settings returns every documented key, sorted by key.
func Settings() []Setting {
	return []Setting{
		{Key: "changelog.hooks.postVersion", Kind: "a shell command", Shape: `"<command>"`, Default: "none",
			Doc: "Runs after the version files are stamped and before CHANGELOG.md is written, with `$VERSION`."},
		{Key: "changelog.hooks.preCommit", Kind: "a shell command", Shape: `"<command>"`, Default: "none",
			Doc: "Runs before the release commit, with `$VERSION`."},
		{Key: "changelog.showScope", Values: []string{"true", "false"}, Default: "false",
			Doc:   "Includes each commit's scope in its changelog entry.",
			value: func(c *PkConfig) string { return boolText(c.Changelog.ShowScope) }},
		{Key: "changelog.types", Kind: "a list of `{\"type\", \"section\", \"hidden\"}` rows", Shape: `[{ "hidden": true | false, "section": "<section>", "type": "<type>" }, ...]`, Default: "the table under `pk help changelog`",
			Doc: "Maps commit types to changelog sections; a hidden type is tracked and never listed; an empty list means the default table."},
		{Key: "changelog.versionFiles", Kind: "a list of `{\"path\", \"type\"}` rows", Shape: `[{ "path": "<path>", "type": "json" }, ...]`, Default: "none",
			Doc: "JSON files whose root `version` field is stamped in place at release."},
		{Key: "guard.branches", Kind: "a list of branch names", Shape: `["<branch>", ...]`, Default: "the release branch",
			Doc: "The protected branches: a git mutation while one is checked out is judged by `guard.mode`."},
		{Key: "guard.breaking", Values: []string{"ask", "off"}, Default: DefaultGuardBreaking,
			Doc:   "A commit whose message carries a breaking marker (`!` or `BREAKING CHANGE`) is questioned, or not.",
			value: func(c *PkConfig) string { return c.Guard.Breaking }},
		{Key: "guard.mode", Values: []string{"block", "ask", "off"}, Default: DefaultGuardMode,
			Doc:   "A git mutation while a protected branch is checked out is denied, questioned, or ignored.",
			value: func(c *PkConfig) string { return c.Guard.Mode }},
		{Key: "guard.push", Values: []string{"block", "ask", "off"}, Default: DefaultGuardPush,
			Doc:   "Any `git push`, on any branch, is denied, questioned, or ignored.",
			value: func(c *PkConfig) string { return c.Guard.Push }},
		{Key: "preserve.mode", Values: []string{"auto", "manual", "off"}, Default: DefaultPreserveMode,
			Doc:   "An approved plan is committed at once, recorded for `/plankit:preserve` to commit, or ignored.",
			value: func(c *PkConfig) string { return c.Preserve.Mode }},
		{Key: "release.branch", Kind: "a branch name", Shape: `"<branch>"`, Default: "the branch `pk init` ran on",
			Doc: "The branch releases merge into; empty selects the trunk flow, which tags the default branch."},
		{Key: "release.hooks.prePush", Kind: "a shell command", Shape: `"<command>"`, Default: "none",
			Doc: "Runs after the tag exists and before the push, with `$VERSION` and `$TAG`."},
		{Key: "release.hooks.preRelease", Kind: "a shell command", Shape: `"<command>"`, Default: "none",
			Doc: "Runs before the tag is created, with `$VERSION` and `$TAG`; `--dry-run` rehearses it."},
	}
}

// Sections returns the top-level keys the table documents, in order:
// each is a section of the policy file and the name of the page that
// carries its Settings.
func Sections() []string {
	var out []string
	seen := map[string]bool{}
	for _, s := range Settings() {
		sec, _, _ := strings.Cut(s.Key, ".")
		if !seen[sec] {
			seen[sec] = true
			out = append(out, sec)
		}
	}
	return out
}

// SettingsFor returns the settings whose key begins with section and
// a dot: the ones a page owns.
func SettingsFor(section string) []Setting {
	var out []Setting
	for _, s := range Settings() {
		if strings.HasPrefix(s.Key, section+".") {
			out = append(out, s)
		}
	}
	return out
}

// LoadRule is the sentence every Settings section ends with: what the
// loader does with a wrong key or value.
const LoadRule = "An unknown key or a value outside these fails the whole file when it loads, with a message naming the key: `pk` commands exit 2, and each hook reports the message and takes no action until it is fixed. An absent key means its default. `pk status` reads the file back and reports the first problem."

func boolText(b bool) string {
	if b {
		return "true"
	}
	return ""
}

// ShapeText returns the value's shape for the reference block: an
// enumeration as quoted alternatives (booleans bare), otherwise the
// declared Shape.
func (s Setting) ShapeText() string {
	if len(s.Values) == 0 {
		return s.Shape
	}
	if len(s.Values) == 2 && s.Values[0] == "true" && s.Values[1] == "false" {
		return "true | false"
	}
	parts := make([]string, len(s.Values))
	for i, v := range s.Values {
		parts[i] = `"` + v + `"`
	}
	return strings.Join(parts, " | ")
}
