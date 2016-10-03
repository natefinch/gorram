package cli // import "npf.io/gorram/cli"

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"npf.io/gorram/run"
)

// CacheEnv is the environment variable that users may set to change the
// location where gorram stores its script files.
const CacheEnv = "GORRAM_CACHE"

// Parse converts the gorram command line (not including executable name) and
// returns a gorram run command and the unused args. The return value is the
// code to exit with.
func Parse(args []string) int {
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	regen := fs.Bool("r", false, "regenerate script file even if it already exists")
	err := fs.Parse(os.Args[1:])
	switch {
	case err == flag.ErrHelp:
		usage()
		return 0
	case err != nil:
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 2
	}
	args = fs.Args()
	if len(args) == 0 {
		usage()
		return 0
	}
	cmd := run.Command{}
	cmd.Regen = regen != nil && *regen
	parts := strings.Split(args[0], ".")
	switch len(parts) {
	case 2:
		cmd.Package = parts[0]
		cmd.Function = parts[1]
	case 3:
		cmd.Package = parts[0]
		cmd.GlobalVar = parts[1]
		cmd.Function = parts[2]
	default:
		fmt.Fprintf(os.Stderr, `Command %q invalid. Expected "package.Function" or "package.Varable.Method".`, args[0])
		return 2
	}
	if err := run.Run(cmd, args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	return 0
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintf(os.Stderr, `
%s [OPTION] <pkg.func | pkg.var.func> [args...]

Options:
  -r	regenerate the script generated for the given function

Executes a go function or an method on a global variable defined in a package in
the stdlib or a package in your GOPATH.  Package must be the full package import
path, e.g. encoding/json.  Only exported functions, methods, and variables may
be called.

Most builtin types are supported, and streams of input (via io.Reader or []byte
for example) may be read from stdin.  If specified as an argument, the argument
to a stream input is expected to be a filename.

Return values are printed to stdout.  If the function has an output argument,
like io.Reader or *bytes.Buffer, it is automatically passed in and then written
to stdout. 
`, os.Args[0])
}
