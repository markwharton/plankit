// Package paths names plankit's repository layout in one place.
package paths

import "path/filepath"

// PlansRel is the plans directory relative to the repository root, in
// slash form for git and display.
const PlansRel = "docs/plans"

// Plans returns the absolute plans directory for a repository root.
func Plans(root string) string {
	return filepath.Join(root, "docs", "plans")
}

// PlanRel returns a plan's repo-relative path in slash form, as git and
// hook messages want it.
func PlanRel(filename string) string {
	return PlansRel + "/" + filename
}
