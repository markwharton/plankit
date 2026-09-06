package main

import (
	"strings"
	"testing"

	"github.com/markwharton/plankit/internal/config"
)

func TestSetRegion(t *testing.T) {
	page := "# pk x\n\nProse.\n"
	// Absent region with a body: the section is appended.
	out, err := setRegion(page, "flags", "## Flags", "```\nA\n```\n")
	if err != nil || !strings.HasSuffix(out, "## Flags\n\n<!-- generated: flags -->\n```\nA\n```\n<!-- /generated: flags -->\n") {
		t.Fatalf("append: %v\n%s", err, out)
	}
	// Present region: only its content changes; the second run is a no-op.
	out2, _ := setRegion(out, "flags", "## Flags", "```\nB\n```\n")
	if !strings.Contains(out2, "```\nB\n```") || strings.Contains(out2, "```\nA\n```") {
		t.Fatalf("replace:\n%s", out2)
	}
	if again, _ := setRegion(out2, "flags", "## Flags", "```\nB\n```\n"); again != out2 {
		t.Fatal("rewriting with the same body must not change the page")
	}
	// Empty body removes the section and its heading.
	gone, _ := setRegion(out2, "flags", "## Flags", "")
	if strings.Contains(gone, "## Flags") || strings.Contains(gone, "generated") || !strings.HasSuffix(gone, "Prose.\n") {
		t.Fatalf("remove:\n%q", gone)
	}
	// Absent region with an empty body is a no-op.
	if same, _ := setRegion(page, "flags", "## Flags", ""); same != page {
		t.Fatal("no flags, no section")
	}
	// A lone marker is an error.
	if _, err := setRegion(page+"<!-- generated: flags -->\n", "flags", "## Flags", "x"); err == nil {
		t.Fatal("one marker must be an error")
	}
}

func TestRegionMarkersCompileToNothing(t *testing.T) {
	d, err := compile("x", []byte("---\nname: x\ndescription: d\n---\n\n# x\n\n<!-- generated: flags -->\n```\n--a\n```\n<!-- /generated: flags -->\n"))
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range d.Blocks {
		if b.Type != "heading" && b.Type != "codeblock" {
			t.Fatalf("unexpected block %q; markers must compile to nothing", b.Type)
		}
	}
}

func TestShapeBlockNestsOnDots(t *testing.T) {
	got := shapeBlock("release", config.SettingsFor("release"))
	want := `"release": {
  "branch": "<branch>",
  "hooks": {
    "prePush": "<command>",
    "preRelease": "<command>"
  }
}
`
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
	if g := shapeBlock("guard", config.SettingsFor("guard")); !strings.Contains(g, `"mode": "block" | "ask" | "off",`) || !strings.Contains(g, `"branches": ["<branch>", ...],`) {
		t.Fatalf("guard shape:\n%s", g)
	}
}
