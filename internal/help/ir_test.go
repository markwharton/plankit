package help

import (
	"strings"
	"testing"
)

func TestLoadRejectsUnknownField(t *testing.T) {
	_, err := Load([]byte(`{"schema":1,"meta":{"name":"x","description":"y"},"blocks":[],"extra":1}`))
	if err == nil || !strings.Contains(err.Error(), "extra") {
		t.Fatalf("err = %v, want unknown field rejection", err)
	}
}

func TestLoadRejectsWrongSchema(t *testing.T) {
	_, err := Load([]byte(`{"schema":2,"meta":{"name":"x","description":"y"},"blocks":[]}`))
	if err == nil || !strings.Contains(err.Error(), "schema") {
		t.Fatalf("err = %v", err)
	}
}

func TestLoadValidatesBlocks(t *testing.T) {
	bad := `{"schema":1,"meta":{"name":"x","description":"y"},"blocks":[{"type":"heading","level":4,"id":"a"}]}`
	if _, err := Load([]byte(bad)); err == nil {
		t.Fatal("heading level 4 should fail validation")
	}
	bad = `{"schema":1,"meta":{"name":"x","description":"y"},"blocks":[{"type":"wat"}]}`
	if _, err := Load([]byte(bad)); err == nil {
		t.Fatal("unknown block type should fail validation")
	}
}
