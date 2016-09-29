# gorram <sup><sub>(WIP)</sub></sup>

![river](https://cloud.githubusercontent.com/assets/3185864/18798443/97829e60-81a0-11e6-99a2-d8a788dd9279.jpg)

<sup><sub>image: &copy; [SubSuid](http://subsuid.deviantart.com/art/River-Tam-Speed-Drawing-282223915)</sub></sup>

It's like go run for any go function.

Automagically understands how to produce an interface from the command line into a Go function.

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

The first time you run Gorram with a specific function name, Gorram analyzes the package function and generates a file for use with `go run`.  Gorram intelligently converts stdin or cli arguments into string, []byte, io.Reader, or bytes.Buffer arguments for the function. Output is converted similarly to stdout.  The code is cached in a local directory so that later runs don't incur the generation overhead.



