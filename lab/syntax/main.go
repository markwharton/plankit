// Command syntax reads a design written as text and prints its JSON, or prints a design's JSON as text.
//
//	syntax read A|B FILE         the JSON of a design written as text
//	syntax print A|B FILE.json   a design as text
//
// A refusal exits 2 with its reason; a command it does not know exits 1.
package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) != 3 || (args[0] != "read" && args[0] != "print") || (args[1] != "A" && args[1] != "B") {
		fmt.Fprintln(stderr, "usage: syntax read A|B FILE | syntax print A|B FILE.json")
		return 1
	}
	form, file := Form(args[1]), args[2]
	data, err := os.ReadFile(file)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if args[0] == "read" {
		doc, err := Read(form, file, string(data))
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
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
		fmt.Fprintf(stderr, "%s: %v\n", file, err)
		return 2
	}
	text, err := Print(form, doc)
	if err != nil {
		fmt.Fprintf(stderr, "%s: %v\n", file, err)
		return 2
	}
	fmt.Fprint(stdout, text)
	return 0
}
