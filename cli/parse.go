package cli // import "npf.io/gorram/cli"

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"npf.io/gorram/run"
)

// CacheEnv is the environment variable that users may set to change the
// location where gorram stores its script files.
const CacheEnv = "GORRAM_CACHE"

// OSEnv encapsulates the gorram environment.
type OSEnv struct {
	Args   []string
	Stderr io.Writer
	Stdout io.Writer
	Stdin  io.Reader
	Env    map[string]string
}

// ParseAndRun parses the environment to create a run.Command and runs it.  It
// returns the code that should be used for os.Exit.
func ParseAndRun(env OSEnv) int {
	cmd, err := Parse(env)
	if err != nil {
		fmt.Fprintln(env.Stderr, err.Error())
		return code(err)
	}
	renv := run.Env{
		Stderr: env.Stderr,
		Stdout: env.Stdout,
		Stdin:  env.Stdin,
	}
	if err := run.Run(cmd, renv); err != nil {
		fmt.Fprintln(env.Stderr, err.Error())
		return 1
	}
	return 0
}

func code(err error) int {
	type coded interface {
		Code() int
	}
	if c, ok := err.(coded); ok {
		return c.Code()
	}
	return 1
}

// Parse converts the gorram command line.  If an error is returned, the program
// should exit with the code specified by the error's Code() int function.
func Parse(env OSEnv) (run.Command, error) {
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	regen := fs.Bool("r", false, "")
	err := fs.Parse(env.Args[1:])
	switch {
	case err == flag.ErrHelp:
		return run.Command{}, codeError{code: 0, msg: usage}
	case err != nil:
		return run.Command{}, codeError{code: 2, msg: err.Error()}
	}
	args := fs.Args()
	if len(args) == 0 {
		return run.Command{}, codeError{code: 0, msg: usage}
	}
	if len(args) == 1 {
		return run.Command{}, codeError{code: 2, msg: usage}
	}
	cmd := run.Command{
		Args:    args[2:],
		Regen:   regen != nil && *regen,
		Package: args[0],
	}
	parts := strings.Split(args[1], ".")
	switch len(parts) {
	case 1:
		cmd.Function = parts[0]
	case 2:
		cmd.GlobalVar = parts[0]
		cmd.Function = parts[1]
	default:
		return run.Command{}, codeError{code: 2, msg: fmt.Sprintf(`Command %q invalid. Expected "importpath Function" or "importpath Varable.Method".`, args[0])}
	}
	if d := env.Env[CacheEnv]; d != "" {
		cmd.Cache = d
	}
	return cmd, nil
}

type codeError struct {
	code int
	msg  string
}

func (c codeError) Error() string {
	return c.msg
}

func (c codeError) Code() int {
	return c.code
}

const usage = `Usage:
gorram [OPTION] <pkg> <func | var.method> [args...]

Options:
  -r	      regenerate the script generated for the given function
  -h, --help  display this help

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

Example:

$ echo '{"a":"b"}' | gorram encoding/json Indent "" "  "
{
  "a" : "b"
} 
`
