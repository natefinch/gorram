# gorram

![river](https://cloud.githubusercontent.com/assets/3185864/18798443/97829e60-81a0-11e6-99a2-d8a788dd9279.jpg)

<sup><sub>image: &copy; [SubSuid](http://subsuid.deviantart.com/art/River-Tam-Speed-Drawing-282223915)</sub></sup>

It's like go run for any go function.

Automagically understands how to produce an interface from the command line into a Go function.

## Installation

```
go get npf.io/gorram
```

## Examples

Pretty print JSON:

```
$ echo '{ "foo" : "bar" }' | gorram encoding/json.Indent "" $'\t'
{
    "foo" : "bar"
}
```

Calculate a sha256 sum:

```
$ gorram crypto/sha256.Sum256 foo.gz
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
$ cat foo.zip | gorram crypto/sha1.Sum
or
$ gorram crypto/sha1.Sum foo.zip

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
$ gorram encoding/json.Indent foo.json "" $'\t'
or
$ cat foo.json | gorram encoding/json.Indent "" $'\t'

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
$ gorram math.Cos 25

function:
// math
func Cos(x float64) float64
```

Gorram understands how to convert CLI arguments using the stringconv.Parse*
functions, and will print outputs with `fmt.Printf("%v\n", val)`.


```
usage:
$ echo 12345 | gorram encoding/base64.StdEncoding.EncodeToString
MTIzNDU2Cg==

function:
// base64
func (e *Encoding) EncodeToString(b []byte]) string
```
Gorram understands that packages have global variables that have methods you can
call.

