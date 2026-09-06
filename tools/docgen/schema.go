package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/markwharton/plankit/internal/config"
)

// The policy file's JSON Schema is a derived view: the structs give
// every key and its type, the settings table gives enumerations,
// defaults, and descriptions, and strict decoding is
// additionalProperties false. It is written into the site as
// config.SchemaFile and served at config.SchemaURL.

const schemaDraft = "https://json-schema.org/draft/2020-12/schema"

// buildSchema returns the schema document for PkConfig.
func buildSchema() map[string]any {
	table := map[string]config.Setting{}
	for _, s := range config.Settings() {
		table[s.Key] = s
	}
	root := schemaFor(reflect.TypeOf(config.PkConfig{}), "", table)
	root["$schema"] = schemaDraft
	root["$id"] = config.SchemaURL
	root["title"] = config.FileName
	root["description"] = "plankit's policy file. Every key is optional; an absent key means its default. Unknown keys are refused."
	return root
}

// schemaFor describes a struct type as a JSON object schema. path is
// the dotted key prefix used to look up the settings table.
func schemaFor(t reflect.Type, path string, table map[string]config.Setting) map[string]any {
	props := map[string]any{}
	var required []string
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := f.Tag.Get("json")
		name, opts, _ := strings.Cut(tag, ",")
		if name == "" || name == "-" {
			continue
		}
		key := name
		if path != "" {
			key = path + "." + name
		}
		prop := schemaForType(f.Type, key, table)
		if s, ok := table[key]; ok {
			if s.Doc != "" {
				prop["description"] = s.Doc
			}
			if len(s.Values) > 0 {
				if s.Values[0] == "true" || s.Values[0] == "false" {
					prop["default"] = s.Default == "true"
				} else {
					prop["enum"] = s.Values
					prop["default"] = s.Default
				}
			}
		}
		if name == "$schema" {
			prop["description"] = config.SchemaDescription
			prop["format"] = "uri"
		}
		props[name] = prop
		if !strings.Contains(opts, "omitempty") {
			required = append(required, name)
		}
	}
	obj := map[string]any{
		"type":                 "object",
		"properties":           props,
		"additionalProperties": false,
	}
	if len(required) > 0 {
		obj["required"] = required
	}
	return obj
}

func schemaForType(t reflect.Type, key string, table map[string]config.Setting) map[string]any {
	switch t.Kind() {
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Struct:
		return schemaFor(t, key, table)
	case reflect.Slice:
		return map[string]any{"type": "array", "items": schemaForType(t.Elem(), key, table)}
	default:
		panic(fmt.Sprintf("schema: unsupported field type %s at %s", t, key))
	}
}

// schemaJSON renders the schema, keys sorted, two-space indent.
func schemaJSON() ([]byte, error) {
	b, err := json.MarshalIndent(buildSchema(), "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}
