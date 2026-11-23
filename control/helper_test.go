package main

import "bytes"

// runCLI runs the command and returns its exit code and what it wrote
// to standard output and standard error.
func runCLI(args ...string) (code int, stdout, stderr string) {
	var out, errb bytes.Buffer
	code = run(args, &out, &errb)
	return code, out.String(), errb.String()
}
