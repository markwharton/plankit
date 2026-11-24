// Command control checks a definition file and its description files:
// whether the design is sound, whether each of its rules names a hold,
// and whether the code, as the description files state it, follows the
// design. format.md, fields.md and checks.md are its reference.
//
// Exit codes: 0 no finding, or every finding accepted; 1 the command was
// given wrongly; 2 a finding is not accepted, or a file was refused.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

const usage = `usage:
  control check [--format text|json] [--root <directory>] <definition file> [<description file> ...]
  control checks
  control axioms

check reads a definition file and any description files, with the
acceptances file beside each, and prints one line for each finding on
standard output. checks lists the checks of the first question with the
sources of Hamilton's each comes from. axioms prints her axioms, theorem,
type map and control structures, with the check for each.`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, usage)
		return 1
	}
	switch args[0] {
	case "check":
		return runCheck(args[1:], stdout, stderr)
	case "checks":
		if len(args) > 1 {
			fmt.Fprintln(stderr, "control checks takes no arguments")
			return 1
		}
		for _, id := range soundChecks {
			fmt.Fprintf(stdout, "%-22s %s\n", id, strings.Join(sourcesOf(id), ", "))
		}
		return 0
	case "axioms":
		if len(args) > 1 {
			fmt.Fprintln(stderr, "control axioms takes no arguments")
			return 1
		}
		printSources(stdout)
		return 0
	case "help", "-h", "--help":
		fmt.Fprintln(stdout, usage)
		return 0
	}
	fmt.Fprintf(stderr, "control: unknown command %q\n%s\n", args[0], usage)
	return 1
}

func runCheck(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "text or json")
	root := fs.String("root", ".", "the repository's root, which the paths in holds and sources start from")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *format != "text" && *format != "json" {
		fmt.Fprintf(stderr, "control check: --format is text or json, not %q\n", *format)
		return 1
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(stderr, "control check: name a definition file, then any description files")
		return 1
	}
	findings, refused := checkFiles(fs.Arg(0), fs.Args()[1:], *root)
	if len(refused) > 0 {
		for _, line := range refused {
			fmt.Fprintln(stderr, line)
		}
		return 2
	}

	open, accepted := 0, 0
	for _, f := range findings {
		if f.Accepted {
			accepted++
		} else {
			open++
		}
	}
	if *format == "json" {
		type line struct {
			Check   string `json:"check"`
			Subject string `json:"subject"`
			Element string `json:"element,omitempty"`
			Place   string `json:"place,omitempty"`
			Message string `json:"message,omitempty"`
			Reason  string `json:"reason,omitempty"`
		}
		out := struct {
			Findings []line `json:"findings"`
			Accepted []line `json:"accepted"`
		}{Findings: []line{}, Accepted: []line{}}
		for _, f := range findings {
			if f.Accepted {
				out.Accepted = append(out.Accepted, line{Check: f.Check, Subject: f.Subject, Element: f.Element, Place: f.Place, Reason: f.Reason})
			} else {
				out.Findings = append(out.Findings, line{Check: f.Check, Subject: f.Subject, Element: f.Element, Place: f.Place, Message: f.Message})
			}
		}
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		enc.Encode(out)
	} else {
		for _, f := range findings {
			if !f.Accepted {
				fmt.Fprintf(stdout, "%s  %s: %s\n", f.Check, f.Subject, f.Message)
			}
		}
		for _, f := range findings {
			if f.Accepted {
				fmt.Fprintf(stdout, "accepted  %s  %s: %s\n", f.Check, f.Subject, f.Reason)
			}
		}
	}
	fmt.Fprintf(stderr, "%s: %s, %d accepted\n", fs.Arg(0), plural(open, "finding"), accepted)
	if open > 0 {
		return 2
	}
	return 0
}

// checkFiles makes a run: it reads every file, makes every check, and
// marks the accepted findings. The findings come back in the order they
// are printed. When any file is refused, it returns the reasons, one
// line each as "<file>: <reason>", and no findings.
func checkFiles(definition string, descriptions []string, root string) ([]*Finding, []string) {
	var refused []string
	refuse := func(path string, reasons []string) {
		for _, r := range reasons {
			refused = append(refused, path+": "+r)
		}
	}
	load := func(path string, description bool) *model {
		data, err := os.ReadFile(path)
		if err != nil {
			refuse(path, []string{"the file cannot be read"})
			return nil
		}
		f, problems := parseFile(path, data, description)
		if len(problems) > 0 {
			refuse(path, problems)
			return nil
		}
		m, problems := wellFormed(f)
		refuse(path, problems)
		return m
	}
	accepts := func(path string) []Acceptance {
		beside := strings.TrimSuffix(path, ".json") + ".accepted.json"
		data, err := os.ReadFile(beside)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			refuse(beside, []string{"the file cannot be read"})
			return nil
		}
		list, problems := parseAcceptances(data, checkOrder)
		refuse(beside, problems)
		return list
	}

	def := load(definition, false)
	defAccepts := accepts(definition)
	descs := make([]*model, len(descriptions))
	descAccepts := make([][]Acceptance, len(descriptions))
	for i, path := range descriptions {
		descs[i] = load(path, true)
		descAccepts[i] = accepts(path)
	}
	if len(refused) > 0 {
		return nil, refused
	}

	c := &collector{}
	sound(def, c)
	held(def, root, c)
	required := map[string]bool{}
	for _, u := range def.f.Unsaid {
		required[u.What] = true
	}
	described := map[string]bool{}
	for i, d := range descs {
		follows(def, d, i, c)
		sourced(d, required, root, i, c)
		described[d.f.Top] = true
	}
	undescribed(def, described, len(descs), c)

	// An acceptance matches a finding of its check and subject that
	// belongs in its file. One that matches no finding is stale.
	match := func(file int, list []Acceptance) {
		for _, a := range list {
			found := false
			for _, f := range c.list {
				if f.file == file && f.Check+"  "+f.Subject == a.Finding {
					f.Accepted, f.Reason, found = true, a.Reason, true
				}
			}
			if !found {
				c.add("stale", file, 0, Subject{Text: a.Finding}, "accepted, and no longer found")
			}
		}
	}
	match(-1, defAccepts)
	for i, list := range descAccepts {
		match(i, list)
	}

	rank := map[string]int{}
	for i, id := range checkOrder {
		rank[id] = i
	}
	sort.SliceStable(c.list, func(i, j int) bool {
		a, b := c.list[i], c.list[j]
		if rank[a.Check] != rank[b.Check] {
			return rank[a.Check] < rank[b.Check]
		}
		if a.order != b.order {
			return a.order < b.order
		}
		return a.Subject < b.Subject
	})
	return c.list, nil
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return fmt.Sprintf("%d %ss", n, word)
}
