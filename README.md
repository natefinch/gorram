# gorram [![Codeship](https://img.shields.io/codeship/ba651390-71e8-0134-7f3a-1a37cb97ae34.svg?maxAge=0)](https://app.codeship.com/projects/178461)
![river](https://cloud.githubusercontent.com/assets/3185864/18798443/97829e60-81a0-11e6-99a2-d8a788dd9279.jpg)

<sup><sub>image: &copy; [SubSuid](http://subsuid.deviantart.com/art/River-Tam-Speed-Drawing-282223915)</sub></sup>

It's like go run for any go function.

Automagically understands how to produce an interface from the command line into
a Go function.

*Sometimes, magic is just someone spending more time on something than anyone else might reasonably expect.* -Teller

## Installation

```
go get -u npf.io/gorram
```

Note: gorram depends on having a working go environment to function, since it
dynamically analyzes go code in the stdlib and in your GOPATH.

## Usage

```
Usage: gorram [OPTION] <pkg> <func | var.method> [args...]

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

```



## Examples

Pretty print JSON:

```
$ echo '{ "foo" : "bar" }' | gorram encoding/json Indent "" $'\t'
{
    "foo" : "bar"
}
```

Calculate a sha256 sum:

```
$ gorram crypto/sha256 Sum256 foo.gz
abcdef012345678
```


## How it works

The first time you run Gorram with a specific function name, Gorram analyzes the
package function and generates a file for use with `go run`.  Gorram
intelligently converts stdin and/or cli arguments into arguments for the
function. Output is converted similarly to stdout.  The code is cached in a
local directory so that later runs don't incur the generation overhead.

## Heuristics

By default, Gorram just turns CLI args into function args and prints out the
return value of a function using fmt's %v.  However, there are some special
heuristics that it uses to be smarter about inputs and outputs, based on common
go idioms.

For example:

```
usage:
$ cat foo.zip | gorram crypto/sha1 Sum
or
$ gorram crypto/sha1 Sum foo.zip

function:
// crypto/sha1
func Sum(data []byte) [Size]byte
```

Gorram understands functions that take a single slice of bytes (or an io.Reader)
should read from stdin, or if an argument is specified, the argument is treated
as a filename to be read.

Return values that are an array of bytes are understood to be intended to be
printed with fmt's %x, so that you get `2c37424d58` instead of `[44 55 66 77
88]`.

```
usage:
$ gorram encoding/json Indent foo.json "" $'\t'
or
$ cat foo.json | gorram encoding/json Indent "" $'\t'

function:
// encoding/json
func Indent(dst *bytes.Buffer, src []byte, prefix, indent string) error
```

Gorram understands that functions with a src argument that is an io.Reader or
[]bytes and a dst argument that is a []byte, *bytes.Buffer, or io.Writer will
read from stdin (or use an argument as a file to open), and write what is
written to dst to stdout.

Gorram understands that if the function returns a non-nil error, the error
should be written to stderr, the program exits with a non-zero exit status, and
nothing is written to stdout.

Gorram understands that prefix and indent are arguments that need to be
specified in the command line.


```
usage:
$ gorram math Cos 25

function:
// math
func Cos(x float64) float64
```

Gorram understands how to convert CLI arguments using the stringconv.Parse*
functions, and will print outputs with `fmt.Printf("%v\n", val)`.


```
usage:
$ echo 12345 | gorram encoding/base64 StdEncoding.EncodeToString
MTIzNDU2Cg==

function:
// base64
func (e *Encoding) EncodeToString(b []byte]) string
```
Gorram understands that packages have global variables that have methods you can
call.

```
usage: 
$ gorram net/http Get https://google.com
<some long html output>

function:
// net/http
func Get(url string) (resp *Response, err error)
```

Gorram understands that if a function returns a struct, and one of the fields of
the struct is an io.Reader, then it will output the contents of that reader.  
(if there are no contents or it's nil, the result value will be printed with
%v).

## Development

See the [project page](https://github.com/natefinch/gorram/projects/1) for what's
being worked on now. 

## Hacking

Gorram requires go 1.7 to run the tests (thank you subtests!)  But only requires
it builds with earlier versions (at least 1.6, I haven't tried earlier ones, but
they should be fine, too).
