package version

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/markwharton/plankit/internal/cli"
)

func TestTextOutput(t *testing.T) {
	var out, errw bytes.Buffer
	code := cli.RunIO([]string{"pk", "version"}, []*cli.Command{Cmd}, &out, &errw)
	if code != cli.ExitOK || !strings.HasPrefix(out.String(), "pk ") {
		t.Fatalf("code=%d out=%q err=%q", code, out.String(), errw.String())
	}
}

func TestJSONOutput(t *testing.T) {
	var out, errw bytes.Buffer
	code := cli.RunIO([]string{"pk", "version", "--format", "json"}, []*cli.Command{Cmd}, &out, &errw)
	if code != cli.ExitOK {
		t.Fatalf("code=%d err=%q", code, errw.String())
	}
	var v map[string]string
	if err := json.Unmarshal(out.Bytes(), &v); err != nil || v["version"] == "" {
		t.Fatalf("json: %v, %q", err, out.String())
	}
}
