package changelog

import (
	"fmt"

	"github.com/markwharton/plankit/internal/jsonsplice"
)

// updateVersionFile rewrites the root-level "version" field in a JSON
// file by splicing the new value into the original bytes, preserving all
// formatting, key order, and indentation.
func updateVersionFile(path, ver string) error {
	content, err := readFile(path)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	updated, err := jsonsplice.Replace(content, ver, "version")
	if err != nil {
		return err
	}
	return writeFile(path, updated)
}
