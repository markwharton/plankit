package main

import (
	"encoding/json"
	"strings"
	"testing"
)

const dataOpen = `<script type="application/json" id="view-data">`

// pageData is the data a page holds, read as the page's script reads it.
func pageData(t *testing.T, page string) map[string]any {
	t.Helper()
	_, rest, ok := strings.Cut(page, dataOpen)
	if !ok {
		t.Fatal("the page has no data")
	}
	text, _, ok := strings.Cut(rest, "</script>")
	if !ok {
		t.Fatal("the page's data does not end")
	}
	var v map[string]any
	if err := json.Unmarshal([]byte(text), &v); err != nil {
		t.Fatalf("the page's data is not JSON: %v", err)
	}
	return v
}

// The view shows the findings check prints for the same files, in the
// same order, accepted ones included.
func TestTheViewShowsTheFindingsCheckPrints(t *testing.T) {
	accept := map[string]string{"library.accepted.json": `{"accepted": [{"finding": "replace-checks-owner  update writes shelf", "reason": "one reader owns every book"}]}`}
	_, stdout, _ := baselineRun(t, noBelongs, accept, "--format", "json")
	var check struct {
		Findings, Accepted []struct{ Check, Subject string }
	}
	if err := json.Unmarshal([]byte(stdout), &check); err != nil {
		t.Fatal(err)
	}
	var want []string
	for _, f := range check.Findings {
		want = append(want, f.Check+"  "+f.Subject)
	}
	for _, f := range check.Accepted {
		want = append(want, "accepted "+f.Check+"  "+f.Subject)
	}
	_, page, _ := baselineCommand(t, "view", noBelongs, accept)
	var got []string
	for _, x := range pageData(t, page)["findings"].([]any) {
		f := x.(map[string]any)
		line := f["check"].(string) + "  " + f["subject"].(string)
		if f["accepted"].(bool) {
			line = "accepted " + line
		}
		got = append(got, line)
	}
	if len(want) < 2 {
		t.Fatalf("the break gives %d findings; the test needs an open one and an accepted one", len(want))
	}
	sameOrder(t, "view and check", got, want)
}

func TestTheViewIsOneFileWithNoNetwork(t *testing.T) {
	_, page, _ := baselineCommand(t, "view", nil, nil)
	if !strings.HasPrefix(page, "<!doctype html>") || strings.Count(page, dataOpen) != 1 {
		t.Errorf("the page does not begin as HTML with one data block")
	}
	for _, s := range []string{"http:", "https:", "src=", "<link", "url(", "@import"} {
		if strings.Contains(page, s) {
			t.Errorf("the page contains %q, and must load nothing", s)
		}
	}
}

// A name that would end the data block early stays inside it.
func TestTheViewEscapesItsData(t *testing.T) {
	name := "</script><b>"
	_, page, _ := baselineCommand(t, "view", func(def, intake, audit obj) {
		for _, f := range []obj{def, audit} {
			place(f, "summary")["name"] = name
			writeTo(element(f, "audit"), "summary")["place"] = name
			if r, ok := element(f, "audit")["rejects"].(obj); ok && r["to"] == "summary" {
				r["to"] = name
			}
		}
	}, nil)
	found := false
	for _, x := range pageData(t, page)["places"].([]any) {
		if x.(map[string]any)["name"] == name {
			found = true
		}
	}
	if !found {
		t.Errorf("the place %q is not in the page's data", name)
	}
}

// The page's script walks lists; a list it is given is never null.
func TestTheViewDataHasNoNullLists(t *testing.T) {
	_, page, _ := baselineCommand(t, "view", nil, nil)
	var walk func(v any, at string)
	walk = func(v any, at string) {
		switch x := v.(type) {
		case nil:
			t.Errorf("%s is null", at)
		case map[string]any:
			for k, val := range x {
				walk(val, at+"."+k)
			}
		case []any:
			for _, item := range x {
				walk(item, at+"[]")
			}
		}
	}
	walk(pageData(t, page), "the data")
}

// The view draws findings; it does not fail on them.
func TestTheViewExitsZeroWithFindings(t *testing.T) {
	code, page, stderr := baselineCommand(t, "view", noBelongs, nil)
	if code != 0 || page == "" || !strings.Contains(stderr, "library.json: 2 findings, 0 accepted; 13 faults played; 0 tests to write") {
		t.Errorf("exit %d, %d bytes, stderr %q", code, len(page), stderr)
	}
}
