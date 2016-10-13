package run

import (
	"bytes"
	"fmt"
	"go/parser"
	"go/token"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// func Now() Time
// tests zero arg Function.
// Tests printing of value with ToString method.
func TestTimeNow(t *testing.T) {
	t.Parallel()
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(dir)
	stderr := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	env := Env{
		Stderr: stderr,
		Stdout: stdout,
	}
	c := &Command{
		Package:  "time",
		Function: "Now",
		Cache:    dir,
		Env:      env,
	}
	err = Run(c)
	checkRunErr(err, c.script(), t)
	out := stdout.String()
	expected := fmt.Sprint(time.Now()) + "\n"

	// we have to fudge the test since obviously the milliseconds won't be the
	// same, and if we're unlucky, bigger units also won't be the same.
	if !strings.HasPrefix(out, expected[:15]) {
		t.Errorf("Expected ~%q but got %q", expected, out)
	}
	if !strings.HasSuffix(out, expected[len(expected)-9:]) {
		t.Errorf("Expected ~%q but got %q", expected, out)
	}
	if msg := stderr.String(); msg != "" {
		t.Errorf("Expected no stderr output but got %q", msg)
	}
}

// func Sqrt(x float64) float64
// Tests float parsing arguments and outputs.
func TestMathSqrt(t *testing.T) {
	t.Parallel()
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(dir)
	stderr := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	env := Env{
		Stderr: stderr,
		Stdout: stdout,
	}
	c := &Command{
		Package:  "math",
		Function: "Sqrt",
		Args:     []string{"25.4"},
		Cache:    dir,
		Env:      env,
	}
	err = Run(c)
	checkRunErr(err, c.script(), t)
	out := stdout.String()
	expected := "5.039841267341661\n"
	if out != expected {
		t.Errorf("Expected %q but got %q", expected, out)
	}
	if msg := stderr.String(); msg != "" {
		t.Errorf("Expected no stderr output but got %q", msg)
	}
}

// func Indent(dst *bytes.Buffer, src []byte, prefix, indent string) error
// Tests stdin to []byte argument.
// Tests a dst *bytes.Buffer with a []byte src.
// Tests string arguments.
func TestJsonIndentStdin(t *testing.T) {
	t.Parallel()
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(dir)
	stderr := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	stdin := strings.NewReader(`{ "foo" : "bar" }`)
	env := Env{
		Stderr: stderr,
		Stdout: stdout,
		Stdin:  stdin,
	}
	c := &Command{
		Package:  "encoding/json",
		Function: "Indent",
		Args:     []string{"", "  "},
		Cache:    dir,
		Env:      env,
	}
	err = Run(c)
	checkRunErr(err, c.script(), t)
	out := stdout.String()
	expected := `
{
  "foo": "bar"
}
`[1:]
	if out != expected {
		t.Errorf("Expected %q but got %q", expected, out)
	}
	if msg := stderr.String(); msg != "" {
		t.Errorf("Expected no stderr output but got %q", msg)
	}
}

// func Get(url string) (resp *Response, err error)
// Tests a single string argument.
// Tests val, err return value.
// Tests struct return value that contains an io.Reader.
func TestNetHTTPGet(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, client")
	}))
	defer ts.Close()

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(dir)
	stderr := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	env := Env{
		Stderr: stderr,
		Stdout: stdout,
	}
	c := &Command{
		Package:  "net/http",
		Function: "Get",
		Args:     []string{ts.URL},
		Cache:    dir,
		Env:      env,
	}
	err = Run(c)
	checkRunErr(err, c.script(), t)
	out := stdout.String()
	expected := "Hello, client\n\n"
	if out != expected {
		t.Errorf("Expected %q but got %q", expected, out)
	}
	if msg := stderr.String(); msg != "" {
		t.Errorf("Expected no stderr output but got %q", msg)
	}
}

// func Get(url string) (resp *Response, err error)
// Tests a single string argument.
// Tests val, err return value.
// Tests template output.
func TestNetHTTPGetWithTemplate(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, client")
	}))
	defer ts.Close()

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(dir)
	stderr := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	env := Env{
		Stderr: stderr,
		Stdout: stdout,
	}
	c := &Command{
		Package:  "net/http",
		Function: "Get",
		Args:     []string{ts.URL},
		Cache:    dir,
		Env:      env,
		Template: "{{.Status}}",
	}
	err = Run(c)
	checkRunErr(err, c.script(), t)
	out := stdout.String()
	expected := "200 OK\n"
	if out != expected {
		t.Errorf("Expected %q but got %q", expected, out)
	}
	if msg := stderr.String(); msg != "" {
		t.Errorf("Expected no stderr output but got %q", msg)
	}
}

// func (enc *Encoding) EncodeToString(src []byte) string
// Tests calling a method on a global variable.
// Tests passing a filename as a []byte argument.
func TestBase64EncodeToStringFromFilename(t *testing.T) {
	t.Parallel()
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(dir)
	filename := filepath.Join(dir, "out.txt")
	if err := ioutil.WriteFile(filename, []byte("12345"), 0600); err != nil {
		t.Fatal(err)
	}

	stderr := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	env := Env{
		Stderr: stderr,
		Stdout: stdout,
	}
	c := &Command{
		Package:   "encoding/base64",
		GlobalVar: "StdEncoding",
		Function:  "EncodeToString",
		Args:      []string{filename},
		Cache:     dir,
		Env:       env,
	}
	err = Run(c)
	checkRunErr(err, c.script(), t)
	out := stdout.String()
	expected := "MTIzNDU=\n"
	if out != expected {
		t.Errorf("Expected %q but got %q", expected, out)
	}
	if msg := stderr.String(); msg != "" {
		t.Errorf("Expected no stderr output but got %q", msg)
	}
}

func TestVersionKeep(t *testing.T) {
	t.Parallel()
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(dir)
	stderr := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	env := Env{
		Stderr: stderr,
		Stdout: stdout,
	}

	c := &Command{
		Package:  "time",
		Function: "Now",
		Cache:    dir,
		Env:      env,
	}
	path := c.script()
	err = os.MkdirAll(filepath.Dir(path), 0700)
	if err != nil {
		t.Fatal(err)
	}

	err = ioutil.WriteFile(path, []byte(`
package main

import "fmt"

const version = "`+version+`"
func main() {
	fmt.Println("Hi!")
}
`), 0600)
	if err != nil {
		t.Fatal(err)
	}

	err = Run(c)
	checkRunErr(err, path, t)
	out := stdout.String()
	expected := "Hi!\n"
	if out != expected {
		t.Errorf("Expected %q but got %q", expected, out)
	}
	if msg := stderr.String(); msg != "" {
		t.Errorf("Expected no stderr output but got %q", msg)
	}
}

func TestVersionOverwrite(t *testing.T) {
	t.Parallel()
	versions := map[string]string{
		"EmptyVersion":     "",
		"TruncatedVersion": `const version = "` + version[:len(version)-1] + `"`,
		"CommentedVersion": `// const version = "` + version + `"`,
	}

	for name, ver := range versions {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir, err := ioutil.TempDir("", "")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(dir)
			stderr := &bytes.Buffer{}
			stdout := &bytes.Buffer{}
			env := Env{
				Stderr: stderr,
				Stdout: stdout,
			}

			c := &Command{
				Package:  "math",
				Function: "Sqrt",
				Args:     []string{"25"},
				Cache:    dir,
				Env:      env,
			}
			path := c.script()
			err = os.MkdirAll(filepath.Dir(path), 0700)
			if err != nil {
				t.Fatal(err)
			}
			err = ioutil.WriteFile(path, []byte(`
			package main
			import "fmt"
			`+ver+`
			func main() {
				fmt.Println("Hi!")
			}`), 0600)
			if err != nil {
				t.Fatal(err)
			}

			err = Run(c)
			checkRunErr(err, path, t)
			out := stdout.String()
			expected := "5\n"
			if out != expected {
				t.Errorf("Expected %q but got %q", expected, out)
			}
			if msg := stderr.String(); msg != "" {
				t.Errorf("Expected no stderr output but got %q", msg)
			}
		})
	}
}

func TestVersionMatches(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "foo.go", []byte(`
	package main

	import "fmt"

	const version = "`+version+`"
	func main() {
		fmt.Println("Hi!")
	}`), 0)
	if err != nil {
		t.Fatal(err)
	}
	if !versionMatches(f) {
		t.Fatal("Expected version to match but it did not.")
	}
}

func checkRunErr(err error, filename string, t *testing.T) {
	if err == nil {
		return
	}
	t.Errorf("unexpected error: %v", err)
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Logf("error reading generated file %q: %v", filename, err)
	}
	t.Log("Generated file contents:")
	t.Log(string(b))
}
