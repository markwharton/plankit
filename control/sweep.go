package main

// Fault is one place broken on one path of a run, and its verdict. The
// trace it was decided from is Traces[Trace] of its run.
type Fault struct {
	Place     string `json:"place"`
	Path      int    `json:"path"`
	Verdict   string `json:"verdict"`
	RefusedAt string `json:"refusedAt,omitempty"`
	Held      bool   `json:"held"`
	Trace     int    `json:"trace"`
}

// RunView is the sweep of one step of the system: its paths, a
// baseline trace for each with nothing broken, and a fault for each
// place it touches on each path.
type RunView struct {
	Run       string   `json:"run"`
	Paths     [][]Pick `json:"paths"`
	Traces    []Trace  `json:"traces"`
	Baselines []int    `json:"baselines"`
	Faults    []Fault  `json:"faults"`
}

// sweep breaks each place each step of the system touches, one at a
// time, on each of its paths, and plays it: view.md, "The sweep".
func sweep(m *model, root string) []RunView {
	var out []RunView
	for _, run := range m.elem[m.f.Top].Children {
		rv := RunView{Run: run, Paths: m.paths(run), Traces: []Trace{}, Baselines: []int{}, Faults: []Fault{}}
		for i, path := range rv.Paths {
			rv.Baselines = append(rv.Baselines, len(rv.Traces))
			rv.Traces = append(rv.Traces, play(m, run, Break{}, path))
			for _, place := range m.touched(run) {
				t := play(m, run, Break{Place: place}, path)
				f := Fault{Place: place, Path: i, Verdict: m.verdict(t), Trace: len(rv.Traces)}
				if len(t.Refusals) > 0 {
					f.RefusedAt = t.Refusals[0].From
					f.Held = f.Verdict == "refused" && missing(root, m.elem[f.RefusedAt].Rejects.Ev, "hold", "test", false) == ""
				}
				rv.Traces = append(rv.Traces, t)
				rv.Faults = append(rv.Faults, f)
			}
		}
		out = append(out, rv)
	}
	return out
}
