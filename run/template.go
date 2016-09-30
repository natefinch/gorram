package run

import "html/template"

var templ = template.Must(template.New("").Parse(`
package main

import (
{{.ImportStatements}}
)

func main() {
	log.SetFlags(0)
    expectedCLIArgs := {{.NumCLIArgs}}

    {{.SrcInit}}

	// strip off the executable name and the -- that we put in so that go run
	// won't treat arguments to the script as files to run.
	args := os.Args[2:]

    {{if .SrcInit}}
	switch len(args) {
	case expectedCLIArgs - 1:
		src = stdinToSrc()
	case expectedCLIArgs:
		src, args = argToSrc(args)
	default:
		log.Fatalf("Expected %d or %d arguments, but got %d args.\n\n", expectedCLIArgs-1, expectedCLIArgs, len(args))
	}
    {{end}}
    {{range .ArgInits}}
    {{.}}
    {{end}}
    {{.DstInit}}

   	val := md5.Sum(data)
	if _, err := fmt.Fprintf(os.Stdout, "%x\n", val); err != nil {
		log.Fatal(err)
	}
}
{{.ArgsToSrc}}
{{.StdinToSrc}}
{{.DstToStdout}}
{{range .ArgConvFuncs}}
{{.}}
{{end}}
`))
