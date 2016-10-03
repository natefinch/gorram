package run

import "text/template"

var templ = template.Must(template.New("").Parse(`
package main

import (
{{range $import, $ignored := .Imports -}}
	"{{$import}}"
{{end}}
)

func main() {
	log.SetFlags(0)

	{{.SrcInit}}

	{{if gt .NumCLIArgs 0}}
	// strip off the executable name and the -- that we put in so that go run
	// won't treat arguments to the script as files to run.
	args := os.Args[2:]
	{{end}}
	{{if ne .SrcIdx -1}}
	expectedCLIArgs := {{.NumCLIArgs}}
	switch len(args) {
	case expectedCLIArgs - 1:
		src = stdinToSrc()
	case expectedCLIArgs:
		src, args = argsToSrc(args)
	default:
		log.Fatalf("Expected %d or %d arguments, but got %d args.\n\n", expectedCLIArgs-1, expectedCLIArgs, len(args))
	}
	{{end}}
	{{range .ArgInits}}
	{{.}}
	{{end}}
	{{.DstInit}}


	{{.Results}}{{.PkgName}}.{{if .GlobalVar}}{{.GlobalVar}}.{{end}}{{.Func}}({{.Args}})
	{{.ErrCheck}}
	{{if ne .DstIdx -1}}
	{{.DstToStdout}}
	{{else}}
	{{.PrintVal}}
	{{end}}
}
{{.ArgsToSrc}}
{{.StdinToSrc}}
{{range .ArgConvFuncs}}
{{.}}
{{end}}
`))
