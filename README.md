# gorram (WIP)

![river](https://cloud.githubusercontent.com/assets/3185864/18798443/97829e60-81a0-11e6-99a2-d8a788dd9279.jpg)

<sup><sub>image: &copy; [SubSuid](http://subsuid.deviantart.com/art/River-Tam-Speed-Drawing-282223915)</sub></sup>

A command line helper for interfacing with go packages

Automagically understands how to produce an interface from the command line into a Go function.

For example:

```
cat ugly.json | gorram json.Indent "" "\t"
```

The above will run encoding/json.Indent, with src = stdin and dst being a bytes.Buffer that gets written to stdout 
if json.Indent returns without an error.



