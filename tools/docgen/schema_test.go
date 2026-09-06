package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/markwharton/plankit/internal/config"
)

// lookup walks the schema to the property at a dotted key, through
// array items where a key names a row field.
func lookup(schema map[string]any, key string) map[string]any {
	cur := schema
	parts := strings.Split(key, ".")
	for i, part := range parts {
		props, _ := cur["properties"].(map[string]any)
		next, ok := props[part].(map[string]any)
		if !ok {
			return nil
		}
		// Descend into a row object only when a deeper part follows;
		// the array itself carries the table's description.
		if items, ok := next["items"].(map[string]any); ok && items["type"] == "object" && i < len(parts)-1 {
			next = items
		}
		cur = next
	}
	return cur
}

func TestSchemaCarriesEveryTableKeyAndRefusesUnknowns(t *testing.T) {
	schema := buildSchema()
	if schema["$id"] != config.SchemaURL || schema["additionalProperties"] != false {
		t.Fatalf("root: %v", schema)
	}
	for _, s := range config.Settings() {
		prop := lookup(schema, s.Key)
		if prop == nil {
			t.Errorf("%s: not in the schema", s.Key)
			continue
		}
		if prop["description"] != s.Doc {
			t.Errorf("%s: description %q, want the table's", s.Key, prop["description"])
		}
		if len(s.Values) > 0 && s.Values[0] != "true" {
			enum, _ := prop["enum"].([]string)
			if strings.Join(enum, ",") != strings.Join(s.Values, ",") || prop["default"] != s.Default {
				t.Errorf("%s: enum %v default %v, want %v %q", s.Key, enum, prop["default"], s.Values, s.Default)
			}
		}
	}
	if p := lookup(schema, "$schema"); p == nil || p["format"] != "uri" {
		t.Errorf("$schema property missing or untyped: %v", p)
	}
	for _, sec := range config.Sections() {
		if p := lookup(schema, sec); p == nil || p["additionalProperties"] != false {
			t.Errorf("%s: object must refuse unknown keys", sec)
		}
	}
	row, _ := lookup(schema, "changelog.types")["items"].(map[string]any)
	if req, _ := row["required"].([]string); len(req) != 1 || req[0] != "type" {
		t.Errorf("a types row requires type: %v", row["required"])
	}
}

// TestWrittenConfigMatchesSchema walks the file pk init writes and
// checks every key has a property in the schema. Without a validator
// dependency this is the structural half of validation, and it is
// the half strict decoding already enforces at load.
func TestWrittenConfigMatchesSchema(t *testing.T) {
	dir := t.TempDir()
	if err := config.Write(dir, config.Default("main")); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(dir, config.FileName))
	var file map[string]any
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatal(err)
	}
	schema := buildSchema()
	var walk func(node map[string]any, sch map[string]any, path string)
	walk = func(node map[string]any, sch map[string]any, path string) {
		props, _ := sch["properties"].(map[string]any)
		for k, v := range node {
			p, ok := props[k].(map[string]any)
			if !ok {
				t.Errorf("%s.%s: written by pk init, absent from the schema", path, k)
				continue
			}
			switch vv := v.(type) {
			case map[string]any:
				walk(vv, p, path+"."+k)
			case []any:
				items, _ := p["items"].(map[string]any)
				for i, e := range vv {
					if m, ok := e.(map[string]any); ok {
						walk(m, items, path+"."+k+"["+string(rune('0'+i))+"]")
					}
				}
			}
		}
	}
	walk(file, schema, "$")
	if file["$schema"] != config.SchemaURL {
		t.Errorf("pk init must write $schema: %v", file["$schema"])
	}
}
