package run

import "text/template"

var templ = template.Must(template.New("").Parse(`
package main

import (
{{range $import, $ignored := .Imports -}}
	"{{$import}}"
{{end}}
)

const version = "{{.Version}}"


func main() {
	log.SetFlags(0)
	{{if not .HasRetVal}}
	if os.Getenv("GORRAM_TEMPLATE") != "" {
		log.Fatalf("No return value to use with templates.")
	}
	{{end}}
	
	{{.SrcInit}}

	{{if gt .NumCLIArgs 0}}
	// strip off the executable name and the -- that we put in so that go run
	// won't treat arguments to the script as files to run.
	var args []string
	if len(os.Args) > 2 {
		args = os.Args[2:]
	}
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


	{{.Results}}{{.PkgName}}.{{if .GlobalVar}}{{.GlobalVar}}.{{end}}{{.Func}}({{.Args}}{{.Variadic}})
	{{.ErrCheck}}
	{{if ne .DstIdx -1}}
	{{.DstToStdout}}
	{{else}}
	{{if .HasRetVal}}
	t := os.Getenv("GORRAM_TEMPLATE")
	if t != "" {
		tmpl, err := template.New("").Parse(t)
		if err != nil {
			log.Fatal(err)
		}
		err = tmpl.Execute(os.Stdout, val)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("")
		os.Exit(0)
	}
	{{end}}
	{{.PrintVal}}
	{{end}}
}
{{.ArgsToSrc}}
{{.StdinToSrc}}
{{range .ArgConvFuncs}}
{{.}}
{{end}}
`))
