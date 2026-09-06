package version

import (
	"bytes"
	"encoding/json"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/markwharton/plankit/internal/cli"
)

func TestTextOutput(t *testing.T) {
	var out, errw bytes.Buffer
	code := cli.RunIO([]string{"pk", "version"}, []*cli.Command{Cmd}, nil, &out, &errw)
	if code != cli.ExitOK || !strings.HasPrefix(out.String(), "pk ") {
		t.Fatalf("code=%d out=%q err=%q", code, out.String(), errw.String())
	}
}

func TestJSONOutput(t *testing.T) {
	var out, errw bytes.Buffer
	code := cli.RunIO([]string{"pk", "version", "--format", "json"}, []*cli.Command{Cmd}, nil, &out, &errw)
	if code != cli.ExitOK {
		t.Fatalf("code=%d err=%q", code, errw.String())
	}
	var v map[string]string
	if err := json.Unmarshal(out.Bytes(), &v); err != nil || v["version"] == "" {
		t.Fatalf("json: %v, %q", err, out.String())
	}
}

func TestFromBuildInfo(t *testing.T) {
	bi := func(main string, settings map[string]string) *debug.BuildInfo {
		info := &debug.BuildInfo{}
		info.Main.Version = main
		for k, v := range settings {
			info.Settings = append(info.Settings, debug.BuildSetting{Key: k, Value: v})
		}
		return info
	}
	cases := []struct {
		name string
		info *debug.BuildInfo
		want string
	}{
		{"go install tagged build", bi("v1.0.0", nil), "1.0.0"},
		{"source checkout clean", bi("(devel)", map[string]string{"vcs.revision": "abcdef1234567890"}), "dev+abcdef123"},
		{"source checkout dirty", bi("(devel)", map[string]string{"vcs.revision": "abcdef1234567890", "vcs.modified": "true"}), "dev+abcdef123.dirty"},
		{"no metadata at all", bi("", nil), "dev"},
	}
	for _, tc := range cases {
		if got := fromBuildInfo(tc.info); got != tc.want {
			t.Errorf("%s: fromBuildInfo = %q, want %q", tc.name, got, tc.want)
		}
	}
}
