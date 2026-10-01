package repo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/markwharton/plankit/internal/jsonsplice"
)

// settingsFile is the repository's Claude Code settings, the developer's
// file. A repository whose contributors use a plugin carries two entries
// there: the marketplace, added when a teammate trusts the folder, and
// the plugin enabled, which project settings apply over user settings.
// pk splices them in and leaves every other byte alone.
const settingsFile = ".claude/settings.json"

const (
	marketplaceName = "plankit"
	marketplaceURL  = "https://plankit.com/marketplace.json"
	pluginID        = "plankit@plankit"
)

// pluginEntry is one entry under a root key of the settings file. The
// key names are the identity: an entry present under its key with
// another value is the developer's choice and is left as it is.
type pluginEntry struct {
	path  []string
	value any
}

func (e pluginEntry) String() string { return strings.Join(e.path, ".") }

// pluginEntries are the entries a configured repository carries, in the
// order they are written.
func pluginEntries() []pluginEntry {
	type source struct {
		Source string `json:"source"`
		URL    string `json:"url"`
	}
	type marketplace struct {
		Source source `json:"source"`
	}
	return []pluginEntry{
		{[]string{"extraKnownMarketplaces", marketplaceName}, marketplace{source{"url", marketplaceURL}}},
		{[]string{"enabledPlugins", pluginID}, true},
	}
}

// readSettings returns the settings file's bytes, or an empty object
// when there is no file; init then creates it. The file's own state,
// committed or not, is the caller's question.
func readSettings(root string) ([]byte, error) {
	content, err := os.ReadFile(filepath.Join(root, settingsFile))
	if os.IsNotExist(err) {
		return []byte("{}"), nil
	}
	return content, err
}

// pluginEntryState sorts the entries by what content holds: missing, or
// present with another value. The error names a document that is not a
// JSON object, or a root key that is not one.
func pluginEntryState(content []byte) (missing, differing []pluginEntry, err error) {
	for _, e := range pluginEntries() {
		raw, ok, err := jsonsplice.Get(content, e.path...)
		if err != nil {
			return nil, nil, err
		}
		switch {
		case !ok:
			missing = append(missing, e)
		case !sameValue(raw, e.value):
			differing = append(differing, e)
		}
	}
	return missing, differing, nil
}

// sameValue compares a raw document value with an intended one as JSON
// values, so spacing and key order do not count.
func sameValue(raw json.RawMessage, want any) bool {
	var got, expected any
	if json.Unmarshal(raw, &got) != nil {
		return false
	}
	wantJSON, err := json.Marshal(want)
	if err != nil || json.Unmarshal(wantJSON, &expected) != nil {
		return false
	}
	return reflect.DeepEqual(got, expected)
}

// addPluginEntries returns content with the missing entries spliced in.
func addPluginEntries(content []byte, missing []pluginEntry) ([]byte, error) {
	var err error
	for _, e := range missing {
		if content, err = jsonsplice.Insert(content, e.value, e.path...); err != nil {
			return nil, err
		}
	}
	return content, nil
}
