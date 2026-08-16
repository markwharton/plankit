package setup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func rulesetConfig() Config {
	return Config{
		ReadFile:  os.ReadFile,
		WriteFile: os.WriteFile,
		MkdirAll:  os.MkdirAll,
	}
}

// rulesetRefs decodes a rendered ruleset and returns its name and the
// ref_name include list, walking the structure rather than grepping it.
func rulesetRefs(t *testing.T, data []byte) (string, []string) {
	t.Helper()
	var doc struct {
		Name       string `json:"name"`
		Conditions struct {
			RefName struct {
				Include []string `json:"include"`
				Exclude []string `json:"exclude"`
			} `json:"ref_name"`
		} `json:"conditions"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("ruleset is not valid JSON: %v\n%s", err, data)
	}
	return doc.Name, doc.Conditions.RefName.Include
}

func TestRulesetPath(t *testing.T) {
	tests := map[string]string{
		"main":        ".github/protect-main.json",
		"trunk":       ".github/protect-trunk.json",
		"release/1.x": ".github/protect-release-1.x.json",
	}
	for branch, want := range tests {
		if got := RulesetPath(branch); got != want {
			t.Errorf("RulesetPath(%q) = %q, want %q", branch, got, want)
		}
	}
}

func TestRenderRuleset_pinsTheReleaseBranch(t *testing.T) {
	// The ruleset guards the release branch by name. ~DEFAULT_BRANCH would
	// follow whatever the repository's default is, and a project that makes
	// its working branch the default (so pull requests base there) would end
	// up with the working branch guarded and the release branch open.
	for _, branch := range []string{"main", "trunk"} {
		data, err := RenderRuleset(branch)
		if err != nil {
			t.Fatalf("RenderRuleset(%q) error = %v", branch, err)
		}
		name, include := rulesetRefs(t, data)
		if want := "protect-" + branch; name != want {
			t.Errorf("RenderRuleset(%q) name = %q, want %q", branch, name, want)
		}
		if len(include) != 1 || include[0] != "refs/heads/"+branch {
			t.Errorf("RenderRuleset(%q) include = %v, want [refs/heads/%s]", branch, include, branch)
		}
		if bytes.Contains(data, []byte("~DEFAULT_BRANCH")) {
			t.Errorf("RenderRuleset(%q) still targets ~DEFAULT_BRANCH", branch)
		}
	}
}

func TestRenderRuleset_keepsThePolicyAndKeyOrder(t *testing.T) {
	data, err := RenderRuleset("main")
	if err != nil {
		t.Fatalf("RenderRuleset() error = %v", err)
	}
	// Everything but the derived fields is the embedded policy, in its order.
	var got, want map[string]json.RawMessage
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("rendered: %v", err)
	}
	if err := json.Unmarshal(rulesetTemplate, &want); err != nil {
		t.Fatalf("template: %v", err)
	}
	for _, key := range []string{"target", "enforcement", "rules", "bypass_actors"} {
		if !bytes.Equal(got[key], want[key]) {
			t.Errorf("%s changed in rendering:\n got %s\nwant %s", key, got[key], want[key])
		}
	}
	// The template's key order survives (a map would alphabetize it), and the
	// derived name still leads, where a reader importing the file looks first.
	if !bytes.HasPrefix(bytes.TrimSpace(data), []byte("{\n  \"name\": \"protect-main\"")) {
		t.Errorf("rendered ruleset does not start with the name:\n%s", data)
	}
	if !bytes.HasSuffix(data, []byte("}\n")) {
		t.Errorf("rendered ruleset should end with a single newline")
	}
}

func TestRenderRuleset_emptyBranch(t *testing.T) {
	if _, err := RenderRuleset(""); err == nil {
		t.Fatal("RenderRuleset(\"\") succeeded, want an error")
	}
}

func TestRenderRuleset_templateMatchesDocsCopy(t *testing.T) {
	// docs/protect-main.json is the UI export offered for download; apart from
	// the source fields the API rejects, it must be the same policy pk writes.
	docs, err := os.ReadFile(filepath.Join("..", "..", "docs", "protect-main.json"))
	if err != nil {
		t.Skipf("docs copy not readable: %v", err)
	}
	var d, r map[string]json.RawMessage
	if err := json.Unmarshal(docs, &d); err != nil {
		t.Fatalf("docs copy: %v", err)
	}
	rendered, err := RenderRuleset("main")
	if err != nil {
		t.Fatalf("RenderRuleset() error = %v", err)
	}
	if err := json.Unmarshal(rendered, &r); err != nil {
		t.Fatalf("rendered: %v", err)
	}
	delete(d, "source")
	delete(d, "source_type")
	if len(d) != len(r) {
		t.Fatalf("docs copy has keys %v, rendered has %v", keysOf(d), keysOf(r))
	}
	for k := range r {
		var dv, rv any
		if err := json.Unmarshal(d[k], &dv); err != nil {
			t.Fatalf("docs %s: %v", k, err)
		}
		if err := json.Unmarshal(r[k], &rv); err != nil {
			t.Fatalf("rendered %s: %v", k, err)
		}
		if fmt.Sprint(dv) != fmt.Sprint(rv) {
			t.Errorf("%s differs between docs/protect-main.json and the rendered ruleset:\n docs %s\n pk   %s", k, d[k], r[k])
		}
	}
}

func keysOf(m map[string]json.RawMessage) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestWriteRuleset_writesWhenAbsent(t *testing.T) {
	dir := t.TempDir()

	wrote, err := WriteRuleset(rulesetConfig(), dir, "main")
	if err != nil {
		t.Fatalf("WriteRuleset() error = %v", err)
	}
	if !wrote {
		t.Error("WriteRuleset() = false, want true for a fresh project")
	}

	got, err := os.ReadFile(filepath.Join(dir, ".github", "protect-main.json"))
	if err != nil {
		t.Fatalf("ruleset not written: %v", err)
	}
	want, _ := RenderRuleset("main")
	if !bytes.Equal(got, want) {
		t.Error("written ruleset differs from the rendered one")
	}
}

func TestWriteRuleset_followsTheReleaseBranch(t *testing.T) {
	dir := t.TempDir()

	if _, err := WriteRuleset(rulesetConfig(), dir, "trunk"); err != nil {
		t.Fatalf("WriteRuleset() error = %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dir, ".github", "protect-trunk.json"))
	if err != nil {
		t.Fatalf("ruleset not written under the release branch's name: %v", err)
	}
	name, include := rulesetRefs(t, got)
	if name != "protect-trunk" || len(include) != 1 || include[0] != "refs/heads/trunk" {
		t.Errorf("name = %q, include = %v; want protect-trunk guarding refs/heads/trunk", name, include)
	}
}

func TestWriteRuleset_leavesExistingCopyAlone(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".github"), 0755); err != nil {
		t.Fatalf("mkdir .github: %v", err)
	}
	custom := []byte(`{"name":"my-own-policy"}`)
	if err := os.WriteFile(filepath.Join(dir, ".github", "protect-main.json"), custom, 0644); err != nil {
		t.Fatalf("seed ruleset: %v", err)
	}

	wrote, err := WriteRuleset(rulesetConfig(), dir, "main")
	if err != nil {
		t.Fatalf("WriteRuleset() error = %v", err)
	}
	if wrote {
		t.Error("WriteRuleset() = true, want false when the project has its own copy")
	}

	got, err := os.ReadFile(filepath.Join(dir, ".github", "protect-main.json"))
	if err != nil {
		t.Fatalf("read ruleset: %v", err)
	}
	if !bytes.Equal(got, custom) {
		t.Errorf("overwrote the project's ruleset: got %s", got)
	}
}

func TestWriteRuleset_mkdirFailure(t *testing.T) {
	cfg := rulesetConfig()
	cfg.MkdirAll = func(string, os.FileMode) error { return fmt.Errorf("disk full") }

	wrote, err := WriteRuleset(cfg, t.TempDir(), "main")
	if err == nil {
		t.Fatal("WriteRuleset() succeeded, want a mkdir error")
	}
	if wrote {
		t.Error("WriteRuleset() = true despite the failure")
	}
	if !strings.Contains(err.Error(), "failed to create .github directory") {
		t.Errorf("error = %q, want it to name the failed step", err)
	}
}

func TestWriteRuleset_writeFailure(t *testing.T) {
	cfg := rulesetConfig()
	cfg.WriteFile = func(string, []byte, os.FileMode) error { return fmt.Errorf("read-only fs") }

	wrote, err := WriteRuleset(cfg, t.TempDir(), "main")
	if err == nil {
		t.Fatal("WriteRuleset() succeeded, want a write error")
	}
	if wrote {
		t.Error("WriteRuleset() = true despite the failure")
	}
	if !strings.Contains(err.Error(), "failed to write") {
		t.Errorf("error = %q, want it to name the failed step", err)
	}
}

func TestRenderRuleset_rejectsAMalformedTemplate(t *testing.T) {
	// The template is embedded, so these branches only fire if the file
	// shipped in the binary is broken. Swap it for a bad one to prove the
	// error is reported rather than a half-rendered ruleset written.
	saved := rulesetTemplate
	t.Cleanup(func() { rulesetTemplate = saved })

	cases := map[string]string{
		"not an object":       `[]`,
		"conditions not json": `{"name":"x","conditions":"nope"}`,
		"ref_name not json":   `{"name":"x","conditions":{"ref_name":42}}`,
	}
	for name, tmpl := range cases {
		t.Run(name, func(t *testing.T) {
			rulesetTemplate = []byte(tmpl)
			if _, err := RenderRuleset("main"); err == nil {
				t.Errorf("RenderRuleset() succeeded with template %s", tmpl)
			}
		})
	}
}

func TestWriteRuleset_renderFailureWritesNothing(t *testing.T) {
	saved := rulesetTemplate
	t.Cleanup(func() { rulesetTemplate = saved })
	rulesetTemplate = []byte(`[]`)

	dir := t.TempDir()
	wrote, err := WriteRuleset(rulesetConfig(), dir, "main")
	if err == nil || wrote {
		t.Fatalf("WriteRuleset() = (%v, %v), want an error and no write", wrote, err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, ".github", "protect-main.json")); statErr == nil {
		t.Error("a file was written despite the render failing")
	}
}
