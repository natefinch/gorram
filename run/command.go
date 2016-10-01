package run

import (
	"errors"
	"fmt"
	"go/parser"
	"go/types"
	"log"
	"os/exec"
	"path"
	"sort"
	"strings"

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
	Results      string
	Args         string
	NumCLIArgs   int
	PkgName      string
	Func         string
	SrcIdx       int
	DstIdx       int
	ErrCheck     string
	HasLen       bool
	SrcInit      string
	ArgsToSrc    string
	StdinToSrc   string
	DstInit      string
	DstToStdout  string
	PrintVal     string
	ParamTypes   map[types.Type]struct{}
	Imports      map[string]struct{}
	ArgConvFuncs []string
	ArgInits     []string
}

func compileData(cmd Command, sig *types.Signature) (templateData, error) {
	data := templateData{
		PkgName:    path.Base(cmd.Package),
		Func:       cmd.Function,
		HasLen:     hasLen(sig.Results()),
		SrcIdx:     -1,
		DstIdx:     -1,
		ParamTypes: map[types.Type]struct{}{},
		Imports: map[string]struct{}{
			cmd.Package: struct{}{},
			"log":       struct{}{},
			"os":        struct{}{},
		},
	}
	if err := data.parseResults(sig.Results()); err != nil {
		return templateData{}, err
	}
	if src, dst, ok := checkSrcDst(sig.Params()); ok {
		if err := data.setSrcDst(src, dst, sig.Params()); err != nil {
			return templateData{}, err
		}
	}
	if err := data.parseParams(sig.Params()); err != nil {
		return templateData{}, err
	}
	data.NumCLIArgs = sig.Params().Len()
	if data.DstIdx != -1 {
		data.NumCLIArgs--
	}
	return data, nil
}

func (data *templateData) setSrcDst(src, dst int, params *types.Tuple) error {
	data.SrcIdx = src
	data.DstIdx = dst
	srcType := params.At(src).Type()
	srcH, ok := srcHandlers[srcType]
	if !ok {
		return fmt.Errorf("should be impossible: src type %q has no handler", srcType)
	}
	data.ArgsToSrc = fmt.Sprintf(srcH.ArgToSrc, src)
	data.StdinToSrc = srcH.StdinToSrc
	for _, imp := range srcH.Imports {
		data.Imports[imp] = struct{}{}
	}
	data.SrcInit = srcH.Init

	dstType := params.At(dst).Type()
	dstH, ok := dstHandlers[dstType]
	if !ok {
		return fmt.Errorf("should be impossible: dst type %q has no handler", dstType)
	}
	data.DstInit = dstH.Init
	data.DstToStdout = dstH.ToStdout
	for _, imp := range dstH.Imports {
		data.Imports[imp] = struct{}{}
	}
	return nil
}

func (data *templateData) parseParams(params *types.Tuple) error {
	pos := 1
	var args []string
	for x := 0; x < params.Len(); x++ {
		if x == data.SrcIdx {
			args = append(args, "src")
			continue
		}
		if x == data.DstIdx {
			args = append(args, "dst")
			continue
		}
		p := params.At(x)
		t := p.Type()
		conv, ok := argConverters[t]
		if !ok {
			return fmt.Errorf("don't understand how to convert arg %q from CLI", p.Name())
		}
		args = append(args, fmt.Sprintf("arg%d", pos))
		data.ParamTypes[t] = struct{}{}
		data.ArgInits = append(data.ArgInits, fmt.Sprintf(conv.Assign, pos, x))
		pos++
	}
	data.Args = strings.Join(args, ", ")
	for t := range data.ParamTypes {
		data.ArgConvFuncs = append(data.ArgConvFuncs, argConverters[t].Func)
		for _, imp := range argConverters[t].Imports {
			data.Imports[imp] = struct{}{}
		}
	}

	// sort so we have consistent output.
	sort.Strings(data.ArgConvFuncs)
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
		default:
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
	_, ok := dstHandlers[t.Underlying()]
	return ok
}

func isSrcType(t types.Type) bool {
	_, ok := srcHandlers[t.Underlying()]
	return ok
}

const justPrintIt = `
	if _, err := fmt.Fprintf(os.Stdout, "%v\n", val); err != nil {
		log.Fatal(err)
	}
`

// arrayOutput is the text we dump instead of using %v to print out return
// values.  Most functions that return an array of bytes intend for them to
// be printed out with %x, so that you get a hex string, instead of a bunch
// of byte values.
const arrayOutput = `
if _, err := fmt.Fprintf(os.Stdout, "%x\n", val); err != nil {
		log.Fatal(err)
	}
`

// yay go!  (no, really, I actually do like go's error handling)
const errCheck = `
	if err != nil {
		log.Fatal(err)
	}
`

// parseResults ensures that the return value on the signature is one that we
// can support, and creates the data to output in the template data.
func (data *templateData) parseResults(results *types.Tuple) error {
	switch results.Len() {
	case 0:
		return nil
	case 1:
		if types.Identical(results.At(0).Type().Underlying(), errorType) {
			data.Results = "err := "
			data.ErrCheck = errCheck
			data.Imports["log"] = struct{}{}
			return nil
		}
		if hasLen(results) {
			data.Results = "_ = "
			return nil
		}
		data.Results = "val := "
		data.PrintVal = justPrintIt
		data.Imports["os"] = struct{}{}
		data.Imports["fmt"] = struct{}{}
		return nil
	case 2:
		// val, err is ok.
		if types.Identical(results.At(1).Type().Underlying(), errorType) {
			if hasLen(results) {
				data.Results = "_, err := "
				data.ErrCheck = errCheck
				data.Imports["log"] = struct{}{}
				return nil
			}
			data.Results = "val, err := "
			data.ErrCheck = errCheck
			data.PrintVal = justPrintIt
			data.Imports["os"] = struct{}{}
			data.Imports["fmt"] = struct{}{}
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
		return types.Identical(sig.Results().At(len-1).Type().Underlying(), errorType)
	}
	return false
}

// hasLen determines if the function returns a value indicating a number of
// bytes written.  This is a common go idiom, and is usually the first value
// returned, with a variable name called n of type int.
func hasLen(results *types.Tuple) bool {
	if results.Len() > 0 {
		// We only care about the last value.
		val := results.At(0)
		return val.Name() == "n" && types.Identical(val.Type().Underlying(), types.Typ[types.Int])
	}
	return false
}

// // Our simplest case - no args, one return, like time.Now().
// var zeroOne = template.Must(template.New("").Parse(`
// package main

// import (
// 	"fmt"
// 	"log"
// 	"os"
// 	{{ if not (eq .Import "log" "fmt" "os") }}
// 	"{{.Import}}"
// 	{{- end -}}
// )

// func main() {
// 	log.SetFlags(0)
// 	if len(os.Args) > 1 {
// 		log.Fatalf("Expected no arguments, but got %d args.\n\n", len(os.Args)-1)
// 	}
// 	{{if .Params.Len eq }}
// 	val := {{.Package}}.{{.Func}}()
// 	if _, err := fmt.Fprintln(os.Stdout, val); err != nil {
// 		log.Fatal(err)
// 	}
// }
// `))

type srcHandler struct {
	// Imports holds the packages needed for the functions this handler uses.
	Imports []string
	// Init holds the line that initializes the src variable.
	Init string
	// ArgToSrc holds the definition of a function that is put at the bottom of the
	// file to convert the src CLI arg into the proper format for the function.  It
	// is expected to contain a %d format directive taking the index of the src arg
	// from the cli args.
	ArgToSrc string
	// StdInToSrc holds the definition of a function that is put at the bottom
	// of the file to convert data sent to stdin into a format suitable to pass
	// to the function.
	StdinToSrc string
}

var srcHandlers = map[types.Type]srcHandler{
	byteSliceType: srcHandler{
		Imports: []string{"io/ioutil", "log"},
		Init:    "var src []byte",
		ArgToSrc: `
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
		StdinToSrc: `
func stdinToSrc() []byte {
	src, err = ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}
	return src
}
`},

	ioReaderType: srcHandler{
		Imports: []string{"io", "os"},
		Init:    "var src io.Reader",
		ArgToSrc: `
func argsToSrc(args []string) (io.Reader, []string) {
	srcIdx := %d
	// yes, I know I never close this. It gets closed when the process exits.
	// It's ugly, but it works and it simplifies the code.  Sorry.
	src, err = os.Open(args[srcIdx])
	if err != nil {
		log.Fatal(err)
	}
	// Take out the src arg.
	args = append(args[:srcIdx], args[srcIdx+1]...)
	return src, args
}
`,
		StdinToSrc: `
func stdinToSrc() io.Reader {
	return os.Stdin
}
`},
}

// dstHandler contains the code to handle destination arguments in a function.
type dstHandler struct {
	// Imports holds the packages needed for the functions this handler uses.
	Imports []string
	// Init contains the initialization line necessary for creating a variable
	// called dst that is used in functions that have a source and destination
	// arguments.
	Init string
	//  ToStdout contains the code that handles writing the data written to dst
	//  to stdout.
	ToStdout string
}

var dstHandlers = map[types.Type]dstHandler{
	ptrBytesBufferType: dstHandler{
		Imports: []string{"bytes"},
		Init:    "dst := &bytes.Buffer{}",
		ToStdout: `
	if _, err := io.Copy(os.Stdout, dst); err != nil {
		log.Fatal(err)
	}
`},
	ioWriterType: dstHandler{
		Imports: []string{"os"},
		Init:    "dst := os.Stdout",
		// no ToStdout needed, since dst *is* stdout.
	},
}

// converter is a type that holds information about argument conversions from
// CLI strings to function arguments of various types.  If a function takes an
// argument that is not declared here, and is not a destination or source
// argument, we can't handle it.
type converter struct {
	// Assign is a format string that is used to make the conversion between CLI
	// arg x and function arg y in the body of the main function.  It should
	// take the cli arg index and the function arg index as %d format values.
	// Ideally, it is a single line of code, which may call a helper function.
	// If it calls a helper function, that function must be listed in Func.
	Assign string
	// Imports is the list of imports that Func uses, so we can make sure
	// they're added to the list of imports.
	Imports []string
	// Func is the declaration of the conversion function between a string (the
	// CLI arg) and a given type.  It must only return a single value of the
	// appropriate type.  Errors should be handled with log.Fatal(err).  It
	// should be named argTo<type> to avoid collision with other conversion
	// function.
	Func string
}

// argConverters is a map of types to helper functions that we dump at the
// end of the file to make the rest of the file easier to construct and read.  The values
var argConverters = map[types.Type]converter{
	// string is a special flower because it doesn't need a converter, but we
	// keep an empty converter here so that we don't need to special case it
	// elsewhere.
	types.Typ[types.String]: converter{Assign: "arg%d = args[%d]"},
	types.Typ[types.Int]: converter{
		Assign:  "arg%d := argToInt(args[%d])",
		Imports: []string{"strconv", "log"},
		Func: `
func argToInt(s string) int {
	i, err := strconv.ParseInt(s, 0, 0)
	if err != nil {
		log.Fatal(err)
	}
	return int(i)
}
`},
	types.Typ[types.Uint]: converter{
		Assign:  "arg%d := argToUint(args[%d])",
		Imports: []string{"strconv", "log"},
		Func: `
func argToUint(s string) int {
	u, err := strconv.ParseUint(s, 0, 0)
	if err != nil {
		log.Fatal(err)
	}
	return uint(u)
}
`},
	types.Typ[types.Float64]: converter{
		Assign:  "arg%d := argToFloat64(args[%d])",
		Imports: []string{"strconv", "log"},
		Func: `
func argToFloat64(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		log.Fatal(err)
	}
	return f
}
`},
	types.Typ[types.Bool]: converter{
		Assign:  "arg%d := argToBool(args[%d])",
		Imports: []string{"strconv", "log"},
		Func: `
func argToBool(s string) bool {
	b, err := strconv.ParseBool(s)
	if err != nil {
		log.Fatal(err)
	}
	return b
}
`},
	types.Typ[types.Int64]: converter{
		Assign:  "arg%d := argToInt64(args[%d])",
		Imports: []string{"strconv", "log"},
		Func: `
func argToInt64(s string) int64 {
	i, err := strconv.ParseInt(s, 0, 64)
	if err != nil {
		log.Fatal(err)
	}
	return i
}
`},
	types.Typ[types.Uint64]: converter{
		Assign:  "arg%d := argToUint64(args[%d])",
		Imports: []string{"strconv", "log"},
		Func: `
func argToUint(s string) uint64 {
	u, err := strconv.ParseUint(s, 0, 64)
	if err != nil {
		log.Fatal(err)
	}
	return u
}
`},
}
