package changelog

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// updateVersionFile rewrites the root-level "version" field in a JSON
// file by splicing the new value into the original bytes, preserving all
// formatting, key order, and indentation.
func updateVersionFile(path, ver string) error {
	content, err := readFile(path)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	updated, err := spliceJSONVersion(content, ver)
	if err != nil {
		return err
	}
	return writeFile(path, updated)
}

// spliceJSONVersion locates the root-level "version" key with a
// streaming decoder and replaces only its value bytes.
func spliceJSONVersion(content []byte, newVersion string) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(content))
	tok, err := dec.Token()
	if err != nil {
		return nil, fmt.Errorf("expected JSON object: %w", err)
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '{' {
		return nil, fmt.Errorf("expected JSON object, got %v", tok)
	}
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := keyTok.(string)
		if !ok {
			return nil, fmt.Errorf("expected string key, got %T", keyTok)
		}
		beforeValue := dec.InputOffset()
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return nil, err
		}
		if key != "version" {
			continue
		}
		afterValue := dec.InputOffset()
		segment := content[beforeValue:afterValue]
		rawIdx := bytes.Index(segment, raw)
		if rawIdx < 0 {
			return nil, fmt.Errorf("could not locate version value in source")
		}
		absStart := int(beforeValue) + rawIdx
		absEnd := absStart + len(raw)
		newValue, _ := json.Marshal(newVersion)
		result := make([]byte, 0, len(content)-len(raw)+len(newValue))
		result = append(result, content[:absStart]...)
		result = append(result, newValue...)
		result = append(result, content[absEnd:]...)
		return result, nil
	}
	return nil, fmt.Errorf("no version field found at root level")
}
