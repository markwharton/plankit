package brief

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/markwharton/plankit/internal/jsonsplice"
	"github.com/markwharton/plankit/internal/version"
)

// pluginRootEnv is set by the plugin's shim when it found no binary
// beside it and ran the pk on PATH instead. It names the plugin's root,
// so this pk can tell when it and the plugin's skills are not the same
// release. Both shims set it; nothing else does.
const pluginRootEnv = "PK_PLUGIN_ROOT"

// installHint is the one way plankit's hints name an update of pk.
const installHint = "go install github.com/markwharton/plankit/cmd/pk@latest"

// binaryVersion is this pk's version; tests stub it, since a test
// binary reports dev and dev is never compared.
var binaryVersion = version.Version

// PluginVersion returns the version of the plugin whose shim dispatched
// this pk, read from <PK_PLUGIN_ROOT>/.claude-plugin/plugin.json, and ""
// when no shim fell back to this binary. The shim said where the plugin
// is, so a manifest that is missing or has no version is an error.
func PluginVersion(getenv func(string) string) (string, error) {
	root := getenv(pluginRootEnv)
	if root == "" {
		return "", nil
	}
	path := filepath.Join(root, ".claude-plugin", "plugin.json")
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("%s names the plugin, but: %v", pluginRootEnv, err)
	}
	raw, ok, err := jsonsplice.Get(content, "version")
	if err != nil {
		return "", fmt.Errorf("%s: %v", path, err)
	}
	if !ok {
		return "", fmt.Errorf("%s: no version field", path)
	}
	var v string
	if err := json.Unmarshal(raw, &v); err != nil {
		return "", fmt.Errorf("%s: version is not a string", path)
	}
	return v, nil
}

// opening is the brief's first sentence. With no plugin version, or
// when the two are the same release, or when either is not a semantic
// version (a dev build), it names this binary. Otherwise it names the
// plugin, which is what the session's skills describe, and says which
// side is behind and what levels them.
func opening(binary, plugin string) string {
	same := fmt.Sprintf("plankit %s is configured in this repository.", binary)
	if plugin == "" {
		return same
	}
	b, bok := version.ParseSemver(binary)
	p, pok := version.ParseSemver(plugin)
	if !bok || !pok {
		return same
	}
	switch b.Compare(p) {
	case -1:
		return fmt.Sprintf("plankit %s is configured in this repository. This session's pk is %s, from the PATH and behind the plugin: update it with %s.", plugin, binary, installHint)
	case 1:
		return fmt.Sprintf("plankit %s is configured in this repository. This session's pk is %s, from the PATH and ahead of the plugin: /plugin update plankit@plankit brings the skills level.", plugin, binary)
	}
	return same
}
