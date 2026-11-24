package main

import (
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strings"
)

// viewPage is the view's HTML. It holds the marker viewData once, where the
// data goes; its script draws the data and decides nothing.
//
//go:embed view.html
var viewPage string

const viewData = "/*view-data*/"

// viewModel is everything the page draws, computed here: view.md says
// what each part is.
type viewModel struct {
	System       string      `json:"system"`
	Definition   string      `json:"definition"`
	Descriptions []string    `json:"descriptions"`
	Nodes        []vNode     `json:"nodes"`
	Places       []vPlace    `json:"places"`
	Findings     []vFinding  `json:"findings"`
	Runs         []RunView   `json:"runs"`
	Tests        []TestEntry `json:"tests"`
}

type vNode struct {
	Name      string     `json:"name"`
	Shape     string     `json:"shape"`
	Parent    string     `json:"parent,omitempty"`
	Children  []string   `json:"children"`
	Branches  []vBranch  `json:"branches"`
	Otherwise string     `json:"otherwise,omitempty"`
	Label     string     `json:"label,omitempty"`
	Reads     []string   `json:"reads"`
	Writes    []vWrite   `json:"writes"`
	Owns      []string   `json:"owns"`
	Refusals  []vRefusal `json:"refusals"`
	Findings  []int      `json:"findings"`
}

type vBranch struct {
	When  string `json:"when"`
	Child string `json:"child"`
}

type vWrite struct {
	Place   string   `json:"place"`
	Mode    string   `json:"mode"`
	Key     string   `json:"key,omitempty"`
	Records string   `json:"records,omitempty"`
	Before  []vCheck `json:"before"`
}

type vCheck struct {
	Check string `json:"check"`
	Then  string `json:"then"`
	Held  bool   `json:"held"`
	Test  string `json:"test,omitempty"`
}

type vRefusal struct {
	Kind   string `json:"kind"`
	Reason string `json:"reason"`
	To     string `json:"to,omitempty"`
	Held   bool   `json:"held"`
	Test   string `json:"test,omitempty"`
}

type vPlace struct {
	Name              string   `json:"name"`
	Owner             string   `json:"owner,omitempty"`
	Kind              string   `json:"kind"`
	Keyed             bool     `json:"keyed"`
	OnExisting        string   `json:"onExisting,omitempty"`
	Invalid           string   `json:"invalid"`
	Writers           []string `json:"writers"`
	Readers           []string `json:"readers"`
	WritesFromOutside []string `json:"writesFromOutside"`
	Findings          []int    `json:"findings"`
}

type vFinding struct {
	Check    string `json:"check"`
	Subject  string `json:"subject"`
	Element  string `json:"element,omitempty"`
	Place    string `json:"place,omitempty"`
	Message  string `json:"message,omitempty"`
	Accepted bool   `json:"accepted"`
	Reason   string `json:"reason,omitempty"`
}

func runView(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("view", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", ".", "the repository's root, which the paths in holds and sources start from")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(stderr, "control view: name a definition file, then any description files")
		return 1
	}
	run, refused := checkFiles(fs.Arg(0), fs.Args()[1:], *root)
	if len(refused) > 0 {
		for _, line := range refused {
			fmt.Fprintln(stderr, line)
		}
		return 2
	}
	v := buildView(run, fs.Arg(0), fs.Args()[1:], *root)
	data, err := json.Marshal(v)
	if err != nil {
		fmt.Fprintf(stderr, "control view: %v\n", err)
		return 2
	}
	io.WriteString(stdout, strings.Replace(viewPage, viewData, string(data), 1))

	open, accepted, faults := 0, 0, 0
	for _, f := range run.findings {
		if f.Accepted {
			accepted++
		} else {
			open++
		}
	}
	for _, rv := range v.Runs {
		faults += len(rv.Faults)
	}
	fmt.Fprintf(stderr, "%s: %s, %d accepted; %s played; %s to write\n", fs.Arg(0), plural(open, "finding"), accepted, plural(faults, "fault"), plural(len(v.Tests), "test"))
	return 0
}

// buildView computes what the page draws from one run of the checker.
func buildView(run *checked, definition string, descriptions []string, root string) viewModel {
	m := run.def
	held := func(ev *Evidence) (bool, string) {
		if ev == nil {
			return false, ""
		}
		return missing(root, ev, "hold", "test", false) == "", ev.Name
	}
	runs := sweep(m, root)
	v := viewModel{System: m.f.Top, Definition: definition, Descriptions: append([]string{}, descriptions...), Findings: []vFinding{}, Runs: runs, Tests: testEntries(m, runs, root)}
	// In the order check prints them: the open findings, then the
	// accepted ones.
	var ordered []*Finding
	for _, accepted := range []bool{false, true} {
		for _, f := range run.findings {
			if f.Accepted == accepted {
				ordered = append(ordered, f)
			}
		}
	}
	for _, f := range ordered {
		v.Findings = append(v.Findings, vFinding{Check: f.Check, Subject: f.Subject, Element: f.Element, Place: f.Place, Message: f.Message, Accepted: f.Accepted, Reason: f.Reason})
	}
	findingsOf := func(match func(f *Finding) bool) []int {
		out := []int{}
		for i, f := range ordered {
			if match(f) {
				out = append(out, i)
			}
		}
		return out
	}

	for _, e := range m.f.Elements {
		n := vNode{Name: e.Name, Shape: e.Shape, Parent: m.parent[e.Name], Children: append([]string{}, e.Children...), Branches: []vBranch{}, Otherwise: e.Otherwise,
			Reads: []string{}, Writes: []vWrite{}, Owns: []string{}, Refusals: []vRefusal{}}
		for _, b := range e.Branches {
			n.Branches = append(n.Branches, vBranch{When: b.When, Child: b.Do})
		}
		for _, r := range e.Reads {
			n.Reads = append(n.Reads, r.Place)
		}
		var outs []string
		for _, w := range e.Writes {
			vw := vWrite{Place: w.Place, Mode: w.Mode, Key: w.Key, Records: w.Records, Before: []vCheck{}}
			for _, c := range w.Before {
				h, test := held(c.Ev)
				vw.Before = append(vw.Before, vCheck{Check: c.Kind, Then: c.Then, Held: h, Test: test})
			}
			n.Writes = append(n.Writes, vw)
			outs = append(outs, w.Place)
		}
		if e.Shape == "leaf" {
			n.Label = strings.Join(outs, ", ") + " = " + e.Name + "(" + strings.Join(n.Reads, ", ") + ")"
		}
		for _, p := range m.f.Places {
			if p.BelongsTo == e.Name {
				n.Owns = append(n.Owns, p.Name)
			}
		}
		for _, r := range []struct {
			kind string
			r    *Refusal
		}{{"rejects", e.Rejects}, {"declines", e.Declines}} {
			if r.r != nil {
				h, test := held(r.r.Ev)
				n.Refusals = append(n.Refusals, vRefusal{Kind: r.kind, Reason: r.r.Reason, To: r.r.To, Held: h, Test: test})
			}
		}
		name := e.Name
		n.Findings = findingsOf(func(f *Finding) bool { return f.Element == name })
		v.Nodes = append(v.Nodes, n)
	}

	for _, p := range m.f.Places {
		vp := vPlace{Name: p.Name, Owner: p.BelongsTo, Keyed: p.Keyed, OnExisting: p.OnExisting, Invalid: p.Invalid,
			Writers: []string{}, Readers: []string{}, WritesFromOutside: []string{}}
		switch {
		case p.Outside:
			vp.Kind = "outside"
		case p.Lasting:
			vp.Kind = "lasting"
		default:
			vp.Kind = "in a run"
		}
		for _, e := range m.leaves() {
			if e.reads(p.Name) {
				vp.Readers = append(vp.Readers, e.Name)
			}
			if e.write(p.Name) != nil {
				vp.Writers = append(vp.Writers, e.Name)
				if !m.ownedBy(e.Name, p) {
					vp.WritesFromOutside = append(vp.WritesFromOutside, e.Name)
				}
			}
		}
		name := p.Name
		vp.Findings = findingsOf(func(f *Finding) bool { return f.Place == name })
		v.Places = append(v.Places, vp)
	}
	return v
}
