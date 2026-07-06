// Package paths centralizes the on-disk names pk reads and writes, so the
// agreed locations live in one place rather than as repeated literals.
package paths

import "path/filepath"

const (
	ClaudeDir     = ".claude"       // project-local Claude Code dir
	PkConfig      = ".pk.json"      // project-level pk config
	SettingsFile  = "settings.json" // under ClaudeDir
	InstallScript = "install-pk.sh" // under ClaudeDir
	RulesDir      = "rules"         // under ClaudeDir

	// InstallScriptRel is InstallScript's repo-relative path, in the slash
	// form hook commands and version checks use.
	InstallScriptRel = ClaudeDir + "/" + InstallScript

	docsDir  = "docs"
	plansDir = "plans"
)

// Plans returns the immutable preserved-plans directory for a project root.
// protect (which blocks edits) and preserve (which writes) must agree on it.
func Plans(root string) string { return filepath.Join(root, docsDir, plansDir) }

// PlansRel returns the repo-relative path to a preserved plan file, e.g.
// "docs/plans/<filename>". preserve uses this when reporting committed paths.
func PlansRel(filename string) string { return filepath.Join(docsDir, plansDir, filename) }
