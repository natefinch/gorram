// Package cli provides a CLI UI for the gorram command line tool.
package cli // import "npf.io/gorram/cli"

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"
	"strings"

	"npf.io/gorram/run"
)

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
	ui, err := Parse(env)
	switch {
	case err == flag.ErrHelp:
		fmt.Fprintln(env.Stderr, usage)
		return 0
	case err != nil:
		fmt.Fprintln(env.Stderr, err.Error())
		return 2
	}
	if len(ui.Args) == 0 {
		fmt.Fprintln(env.Stderr, usage)
		return 0
	}
	c, err := parseCommand(ui, env)
	if err != nil {
		fmt.Fprintln(env.Stderr, err.Error())
		return 2
	}
	if err := run.Run(c); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			// if there's a problem running os/Exec commands, we'll have alreday
			// printed out stdout and stderr, so we can just be silent on this
			// one
			return 1
		}
		fmt.Fprintln(env.Stderr, err.Error())
		return 1
	}
	return 0
}

// UI represents the UI of the CLI, including flags and actions.
type UI struct {
	Regen    bool
	Template string
	Cache    string
	Args     []string
}

// Parse converts the gorram command line.  If an error is returned, the program
// should exit with the code specified by the error's Code() int function.
func Parse(env OSEnv) (*UI, error) {
	ui := &UI{
		Cache: confDir(env.Env),
	}
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	fs.BoolVar(&ui.Regen, "r", false, "")
	fs.StringVar(&ui.Template, "t", "", "")
	if err := fs.Parse(env.Args[1:]); err != nil {
		return nil, err
	}
	if ui.Template != "" {
		// try to treat the template as a file on the assumption that no one
		// will ever have template that matched a local filename.
		b, err := ioutil.ReadFile(ui.Template)
		if err == nil {
			ui.Template = string(b)
		}
		// else, not a file, assume the template is just a raw template.
	}
	ui.Args = fs.Args()
	return ui, nil
}

func parseCommand(ui *UI, env OSEnv) (*run.Command, error) {
	if len(ui.Args) < 2 {
		return nil, errors.New(usage)
	}
	cmd := &run.Command{
		Args:     ui.Args[2:],
		Regen:    ui.Regen,
		Template: ui.Template,
		Package:  ui.Args[0],
		Cache:    ui.Cache,
		Env: run.Env{
			Stderr: env.Stderr,
			Stdout: env.Stdout,
			Stdin:  env.Stdin,
		},
	}
	parts := strings.Split(ui.Args[1], ".")
	switch len(parts) {
	case 1:
		cmd.Function = parts[0]
	case 2:
		cmd.GlobalVar = parts[0]
		cmd.Function = parts[1]
	default:
		return nil, fmt.Errorf(`Command %q invalid. Expected "importpath Function" or "importpath Varable.Method".`, ui.Args[0])
	}

	return cmd, nil
}

const usage = `Usage:
gorram [OPTION] <pkg> <func | var.method> [args...]

Options:
  -t <string>  format output with a go template
  -h, --help   display this help

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

If there's no output stream, the return value is simply written to stdout via
fmt.Println.  If the return value is a struct that has an exported field that is
an io.Reader (such as net/http.Request), then that will be treated as the output
value, unless it's empty, in which case we fall back to printing the output
value.

A template specified with -t may either be a template definition (e.g.
{{.Status}}) or a filename, in which case the contents of the file will be used
as the template.

Gorram creates a script file in $GORRAM_CACHE, or, if not set, in
$HOME/.gorram/importpath/Name.go.  Running with -r will re-generate that script
file, otherwise it is reused.

Example:

$ echo '{"a":"b"}' | gorram encoding/json Indent "" "  "
{
  "a" : "b"
} 
`
