package run

import (
	"errors"
	"fmt"
	"go/parser"
	"go/types"
	"log"
	"os/exec"
	"path"
	"text/template"

	"golang.org/x/tools/go/loader"

	"npf.io/deputy"
)

func init() {
	conf := loader.Config{
		ImportPkgs: map[string]bool{
			"bytes": false,
			"io":    false,
		},
	}
	prog, err := conf.Load()
	if err != nil {
		panic(err)
	}
	bytesBufferType = prog.Package("bytes").Pkg.Scope().Lookup("Buffer").Type()
	ptrBytesBufferType = types.NewPointer(bytesBufferType)
	errorType = types.Universe.Lookup("error").Type()
	ioReaderType = prog.Package("io").Pkg.Scope().Lookup("Reader").Type()
	ioWriterType = prog.Package("io").Pkg.Scope().Lookup("Writer").Type()
	ioReader = ioReaderType.Underlying().(*types.Interface)
	ioWriter = ioWriterType.Underlying().(*types.Interface)
}

// Used for type comparison.
var bytesBufferType types.Type
var ptrBytesBufferType types.Type
var errorType types.Type
var ioReaderType types.Type
var ioWriterType types.Type
var byteSliceType = types.NewSlice(types.Typ[types.Byte])
var stringType = types.Typ[types.String]

// Used for types.Implements.
var ioReader *types.Interface
var ioWriter *types.Interface

// Command contains the definition of a command that gorram can execute. It
// represents a package and either global function or a method call on a global
// variable.  If GlobalVar is non-empty, it's the latter.
type Command struct {
	Package   string
	Function  string
	GlobalVar string
}

// Run generates the gorram .go file if it doesn't already exist and then runs
// it with the given args.
func Run(cmd Command, args []string) error {
	path, err := Generate(cmd)
	if err != nil {
		return err
	}
	return run(path, args)
}

func run(path string, args []string) error {
	d := deputy.Deputy{
		Errors:    deputy.FromStderr,
		StdoutLog: func(b []byte) { log.Print(string(b)) },
	}
	// put a -- between the filename and the args so we don't confuse go run
	// into thinking the first arg is another file to run.
	realArgs := append([]string{"run", path, "--"}, args...)
	return d.Run(exec.Command("go", realArgs...))
}

// Generate creates the gorram .go file for the given command.
func Generate(cmd Command) (path string, err error) {
	// let's see if this is even a valid package
	conf := loader.Config{ParserMode: parser.ParseComments}
	conf.Import(cmd.Package)
	lprog, err := conf.Load()
	if err != nil {
		log.Fatal(err)
	}

	// Find the package and package-level object.
	pkg := lprog.Package(cmd.Package).Pkg
	scope := pkg.Scope()
	// TODO(natefinch): support globalvar.Method
	// if cmd.GlobalVar != nil {
	// 	obj := scope.Lookup(cmd.GlobalVar)
	// 	obj.Type()
	// 	obj := obj.Scope().Lookup(cmd.Function)
	// }
	obj := scope.Lookup(cmd.Function)
	if obj == nil {
		return "", fmt.Errorf("%s.%s not found", cmd.Package, cmd.Function)
	}
	f, ok := obj.(*types.Func)
	if !ok {
		return "", fmt.Errorf("%s.%s is not a function", cmd.Package, cmd.Function)
	}
	if !f.Exported() {
		return "", fmt.Errorf("%s.%s is not exported", cmd.Package, cmd.Function)
	}
	// guaranteed to work per types.Cloud docs.
	sig := f.Type().(*types.Signature)
	data, err := compileData(cmd, sig)
	if err != nil {
		return "", err
	}
	return gen(cmd, data)
}

func gen(cmd Command, data templateData) (path string, err error) {
	f, path, err := createFile(cmd)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if err := templ.Execute(f, data); err != nil {
		return "", err
	}
	return path, nil
}

type templateData struct {
	NumResults  int
	NumParams   int
	ImportPath  string
	PkgName     string
	Func        string
	SrcIdx      int
	DstIdx      int
	HasError    bool
	HasLen      bool
	ArgsToSrc   string
	StdinToSrc  string
	DstInit     string
	DstToStdout string
	ParamTypes  map[types.Type]struct{}
}

func compileData(cmd Command, sig *types.Signature) (templateData, error) {
	if err := validateResults(sig.Results()); err != nil {
		return templateData{}, err
	}
	data := templateData{
		NumResults: sig.Results().Len(),
		NumParams:  sig.Params().Len(),
		ImportPath: path.Base(cmd.Package),
		PkgName:    cmd.Package,
		Func:       cmd.Function,
		HasError:   hasError(sig),
		HasLen:     hasLen(sig),
		SrcIdx:     -1,
		DstIdx:     -1,
		ParamTypes: map[types.Type]struct{}{},
	}
	if src, dst, ok := checkSrcDst(sig.Params()); ok {
		if err := data.setSrcDst(src, dst, sig.Params()); err != nil {
			return templateData{}, err
		}
	}
	if err := data.parseParams(sig.Params()); err != nil {
		return templateData{}, err
	}

	return data, nil
}

func (data *templateData) setSrcDst(src, dst int, params *types.Tuple) error {
	data.SrcIdx = src
	data.DstIdx = dst
	srcType := params.At(src).Type()
	s, ok := argsToSrcFuncs[srcType]
	if !ok {
		return fmt.Errorf("should be impossible: src type %q has no args conversion func", srcType)
	}
	data.ArgsToSrc = fmt.Sprintf(s, src)
	s, ok = stdinToSrcFuncs[srcType]
	if !ok {
		return fmt.Errorf("should be impossible: src type %q has no stdin conversion func", srcType)
	}
	data.StdinToSrc = s

	dstType := params.At(dst).Type()
	s, ok = dstInit[dstType]
	if !ok {
		return fmt.Errorf("should be impossible: dst type %q has no init value", dstType)
	}
	data.DstInit = s
	s, ok = dstToStdoutFuncs[dstType]
	if !ok {
		return fmt.Errorf("should be impossible: dst type %q has no stdout conversion func", dstType)
	}
	data.StdinToSrc = s
	return nil
}

func (data *templateData) parseParams(params *types.Tuple) error {
	for x := 0; x < params.Len(); x++ {
		if x == data.SrcIdx || x == data.DstIdx {
			continue
		}
		p := params.At(x)
		t := p.Type()
		if _, ok := argToBuiltin[t]; !ok {
			return fmt.Errorf("don't understand how to convert arg %q from CLI", p.Name())
		}
		data.ParamTypes[t] = struct{}{}
	}
	return nil
}

func checkSrcDst(params *types.Tuple) (dst, src int, ok bool) {
	dst, src = -1, -1
	for x := 0; x < params.Len(); x++ {
		p := params.At(x)
		switch p.Name() {
		case "dst":
			if isDstType(p.Type()) {
				dst = x
			}
		case "src":
			if isSrcType(p.Type()) {
				src = x
			}
		}
	}
	if src != -1 && dst != -1 {
		return dst, src, true
	}
	return -1, -1, false
}

func isDstType(t types.Type) bool {
	return types.Identical(t, ptrBytesBufferType) ||
		types.Identical(t, byteSliceType) ||
		types.Implements(t, ioWriter)
	// anything else?
}

func isSrcType(t types.Type) bool {
	return types.Identical(t, byteSliceType) ||
		types.Identical(t, byteSliceType) ||
		types.Implements(t, ioWriter)
	// anything else?
}

// validateResults ensures that the return value on the signature is one that we
// can support.
func validateResults(results *types.Tuple) error {
	switch results.Len() {
	case 0, 1:
		// always fine.
		return nil
	case 2:
		// val, err is ok.
		if types.Identical(results.At(1).Type(), errorType) {
			return nil
		}
		return errors.New("can't understand function with multiple non-error return values")
	default:
		return errors.New("can't understand functions with more than two return values")
	}
}

// hasError determines if the function can fail. For this, we assume the last
// value returned is the one that determines whether or not the function may
// fail.  We also assume that only the builtin error interface indicates an
// error.
func hasError(sig *types.Signature) bool {
	if len := sig.Results().Len(); len > 0 {
		// We only care about the last value.
		return types.Identical(sig.Results().At(len-1).Type(), errorType)
	}
	return false
}

// hasLen determines if the function returns a value indicating a number of
// bytes written.  This is a common go idiom, and is usually the first value
// returned, with a variable name called n of type int.
func hasLen(sig *types.Signature) bool {
	if sig.Results().Len() > 0 {
		// We only care about the last value.
		val := sig.Results().At(0)
		return val.Name() == "n" && types.Identical(val.Type(), types.Typ[types.Int])
	}
	return false
}

var templ = template.Must(template.New("").Parse(`
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	{{ if not (eq .ImportPath "log" "fmt" "os" "io/ioutil") }}
	"{{.ImportPath}}"
	{{- end -}}
)

func main() {
	log.SetFlags(0)
	// strip off the executable name and the -- that we put in so that go run
	// won't treat arguments to the script as files to run.
	args := os.Args[2:]
	var data []byte
	switch len(args) {
	{{ if and (.StreamIdx gt -1) (.Params.Len gt 0) -}}
	case {{dec(.Params.Len)}}:
		// read from stdin
		b, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			log.Fatal(err)
		}
		data = b
	{{- end }}
	case  {{.Params.Len}}:
		// treat it as a filename
		b, err := ioutil.ReadFile(args[0])
		if err != nil {
			log.Fatal(err)
		}
		data = b
	default:
		log.Fatalf("Expected 0 or 1 arguments, but got %d args.\n\n", len(args))
	}

	val := md5.Sum(data)
	if _, err := fmt.Fprintf(os.Stdout, "%x\n", val); err != nil {
		log.Fatal(err)
	}
}
`))

// Our simplest case - no args, one return, like time.Now().
var zeroOne = template.Must(template.New("").Parse(`
package main

import (
	"fmt"
	"log"
	"os"
	{{ if not (eq .Import "log" "fmt" "os") }}
	"{{.Import}}"
	{{- end -}}
)

func main() {
	log.SetFlags(0)
	if len(os.Args) > 1 {
		log.Fatalf("Expected no arguments, but got %d args.\n\n", len(os.Args)-1)
	}
	{{if .Params.Len eq }}
	val := {{.Package}}.{{.Func}}()
	if _, err := fmt.Fprintln(os.Stdout, val); err != nil {
		log.Fatal(err)
	}
}
`))

// argsToSrcFuncs holds the definitions of functions we put at the bottom of the
// file to convert the src CLI arg into the proper format for the function.  It
// is expected to contain a format directive taking the index of the src arg
// from the cli args.
var argsToSrcFuncs = map[types.Type]string{
	byteSliceType: `
func argsToSrc(args []string) ([]byte, []string) {
	srcIdx := %d
	src, err = ioutil.ReadFile(args[srcIdx])
	if err != nil {
		log.Fatal(err)
	}
	// Take out the src arg.
	args = append(args[:srcIdx], args[srcIdx+1]...)
	return src, args
}
`,
	ioReaderType: `
func argsToSrc(args []string) (io.Reader, []string) {
	srcIdx := %d
	// yes, I know I never close this. It gets closed when the process exits.
	// it's ugly, but it works.
	src, err = os.Open(args[srcIdx])
	if err != nil {
		log.Fatal(err)
	}
	// Take out the src arg.
	args = append(args[:srcIdx], args[srcIdx+1]...)
	return src, args
}
`,
}

var stdinToSrcFuncs = map[types.Type]string{
	byteSliceType: `
func stdinToSrc() []byte {
	src, err = ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}
	return src
}
`,
	ioReaderType: `
func stdinToSrc() io.Reader {
	return os.Stdin
}
`,
}

var dstToStdoutFuncs = map[types.Type]string{
	ptrBytesBufferType: `
	if _, err := io.Copy(os.Stdout, dst); err != nil {
		log.Fatal(err)
	}
`,
	ioWriterType: "", // we pass in os.Stdout right into the function.
}

var dstInit = map[types.Type]string{
	ptrBytesBufferType: "dst := &bytes.Buffer{}",
	ioWriterType:       "dst := os.Stdout",
}

var argToBuiltin = map[types.Type]string{
	types.Typ[types.String]:  "arg%d = args[%d]",
	types.Typ[types.Int]:     "arg%d = argToInt(args[%d])",
	types.Typ[types.Uint]:    "arg%d = argToUint(args[%d])",
	types.Typ[types.Int64]:   "arg%d = argToInt64(args[%d])",
	types.Typ[types.Uint64]:  "arg%d = argToUint64(args[%d])",
	types.Typ[types.Bool]:    "arg%d = argToBool(args[%d])",
	types.Typ[types.Float64]: "arg%d = argToFloat64(args[%d])",
}

const (
	arrayOutput = `
	if _, err := fmt.Fprintf(os.Stdout, "%x\n", val); err != nil {
		log.Fatal(err)
	}
`

	errCheck = `
	if err != nil {
		log.Fatal(err)
	}
`
)

var argToBuiltinFuncs = map[types.Type]string{
	types.Typ[types.Int]: `
func argToInt(s string) int {
	i, err := strconv.ParseInt(s, 0, 0)
	if err != nil {
		log.Fatal(err)
	}
	return int(i)
}
`,
	types.Typ[types.Uint]: `
func argToUint(s string) int {
	u, err := strconv.ParseUint(s, 0, 0)
	if err != nil {
		log.Fatal(err)
	}
	return uint(u)
}
`,
	types.Typ[types.Float64]: `
func argToFloat64(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		log.Fatal(err)
	}
	return f
}
`,
	types.Typ[types.Bool]: `
func argToBool(s string) bool {
	b, err := strconv.ParseBool(s)
	if err != nil {
		log.Fatal(err)
	}
	return b
}
`,
	types.Typ[types.Int64]: `
func argToInt64(s string) int64 {
	i, err := strconv.ParseInt(s, 0, 64)
	if err != nil {
		log.Fatal(err)
	}
	return i
}
`,
	types.Typ[types.Uint64]: `
func argToUint(s string) uint64 {
	u, err := strconv.ParseUint(s, 0, 64)
	if err != nil {
		log.Fatal(err)
	}
	return u
}
`,
}
