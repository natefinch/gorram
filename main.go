// gorram is a command line helper for interfacing with go packages.
// Automagically understands how to produce an interface from the command line
// into a Go function.
//
// For example:
//
//     cat ugly.json | gorram json.Indent "" "\t"
//
// The above will run encoding/json.Indent, with src = stdin and dst being a
// bytes.Buffer that gets written to stdout if json.Indent returns without an
// error.
package main

import (
	"os"

	"npf.io/gorram/cli"
)

func main() {
	os.Exit(cli.Run())
}
