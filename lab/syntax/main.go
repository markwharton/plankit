// Command syntax reads a design written as text and prints its JSON, prints a design's JSON as text,
// and writes a design's text as the printer writes it.
//
//	syntax read A|B FILE [SHOWN]     the JSON of a design written as text
//	syntax print A|B FILE.json       a design as text
//	syntax format A|B FILE [SHOWN]   a design's text as the printer writes it
//
// FILE - is the standard input, and SHOWN is the name a refusal gives it. A refusal exits 2 with
// its reason; a command it does not know exits 1.
package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	known := map[string]bool{"read": true, "print": true, "format": true}
	if len(args) < 3 || len(args) > 4 || !known[args[0]] || (args[1] != "A" && args[1] != "B") || (args[0] == "print" && len(args) != 3) {
		fmt.Fprintln(stderr, "usage: syntax read A|B FILE [SHOWN] | syntax print A|B FILE.json | syntax format A|B FILE [SHOWN]")
		return 1
	}
	form, file := Form(args[1]), args[2]
	shown := file
	if len(args) == 4 {
		shown = args[3]
	}
	var data []byte
	var err error
	if file == "-" {
		data, err = io.ReadAll(stdin)
	} else {
		data, err = os.ReadFile(file)
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	switch args[0] {
	case "read", "format":
		doc, err := Read(form, shown, string(data))
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		if args[0] == "format" {
			text, err := Print(form, doc)
			if err != nil {
				fmt.Fprintf(stderr, "%s: %v\n", shown, err)
				return 2
			}
			fmt.Fprint(stdout, text)
			return 0
		}
		out, err := encode(doc)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		fmt.Fprintf(stdout, "%s\n", out)
		return 0
	}
	doc, err := Decode(data)
	if err != nil {
		fmt.Fprintf(stderr, "%s: %v\n", shown, err)
		return 2
	}
	text, err := Print(form, doc)
	if err != nil {
		fmt.Fprintf(stderr, "%s: %v\n", shown, err)
		return 2
	}
	fmt.Fprint(stdout, text)
	return 0
}
