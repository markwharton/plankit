package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestAbsentFileIsNotConfigured(t *testing.T) {
	_, err := Load(t.TempDir())
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("err = %v, want ErrNotConfigured", err)
	}
}

func TestRoundTripDefault(t *testing.T) {
	dir := t.TempDir()
	if err := Write(dir, Default("main")); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Guard.ResolvedMode() != "block" || cfg.Preserve.ResolvedMode() != "manual" || cfg.Release.Branch != "main" {
		t.Fatalf("round trip: %+v", cfg)
	}
	if len(cfg.Changelog.Types) != 14 {
		t.Fatalf("type table: %d entries", len(cfg.Changelog.Types))
	}
}

func TestV1FileParsesUnchanged(t *testing.T) {
	// The shape plankit's own repository has carried since v1.
	dir := t.TempDir()
	write(t, dir, `{
  "changelog": {"types": [{"type": "plan", "section": "Plans", "hidden": true}]},
  "guard": {"branches": ["main"], "mode": "block", "push": "block"},
  "preserve": {"mode": "manual"},
  "release": {"branch": "main", "hooks": {"preRelease": "go test ./..."}}
}`)
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("v1 config rejected: %v", err)
	}
	if cfg.Release.Hooks.PreRelease == "" || !cfg.Changelog.Types[0].Hidden {
		t.Fatalf("fields lost: %+v", cfg)
	}
}

func TestUnknownKeyFailsLoudly(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, `{"guard": {"branchs": ["main"]}}`)
	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), "branchs") {
		t.Fatalf("err = %v, want the typo named", err)
	}
}

func TestInvalidModeFailsWithKey(t *testing.T) {
	cases := []struct{ body, key string }{
		{`{"guard": {"mode": "blok"}}`, "guard.mode"},
		{`{"guard": {"push": "maybe"}}`, "guard.push"},
		{`{"preserve": {"mode": "always"}}`, "preserve.mode"},
		{`{"changelog": {"types": [{"type": ""}]}}`, "changelog.types[0]"},
		{`{"changelog": {"types": [{"type": "feat"}]}}`, "section is required"},
	}
	for _, tc := range cases {
		dir := t.TempDir()
		write(t, dir, tc.body)
		_, err := Load(dir)
		if err == nil || !strings.Contains(err.Error(), tc.key) {
			t.Errorf("%s: err = %v, want %q named", tc.body, err, tc.key)
		}
	}
}

func TestAbsentModesResolveToDefaults(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, `{"guard": {"branches": ["main"]}}`)
	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Guard.ResolvedMode() != DefaultGuardMode || cfg.Guard.ResolvedPush() != DefaultGuardPush || cfg.Preserve.ResolvedMode() != DefaultPreserveMode {
		t.Fatalf("defaults: %+v", cfg)
	}
}

// TestWrittenConfigIsSorted holds the file pk init writes to sorted
// keys at every level. Struct field order is what json.Marshal
// follows, so a field appended out of order fails here.
func TestWrittenConfigIsSorted(t *testing.T) {
	data, err := json.Marshal(Default("main"))
	if err != nil {
		t.Fatal(err)
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	var walk func(path string)
	walk = func(path string) {
		tok, err := dec.Token()
		if err != nil {
			t.Fatal(err)
		}
		d, ok := tok.(json.Delim)
		if !ok {
			return // scalar
		}
		switch d {
		case '{':
			prev := ""
			for dec.More() {
				k, _ := dec.Token()
				key := k.(string)
				if prev != "" && key < prev {
					t.Errorf("%s: key %q follows %q; the written config must be sorted at every level", path, key, prev)
				}
				prev = key
				walk(path + "." + key)
			}
			dec.Token() // '}'
		case '[':
			for i := 0; dec.More(); i++ {
				walk(fmt.Sprintf("%s[%d]", path, i))
			}
			dec.Token() // ']'
		}
	}
	walk("$")
	if _, err := dec.Token(); err != io.EOF {
		t.Fatalf("trailing tokens: %v", err)
	}
}
