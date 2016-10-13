package cli

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseAndRun(t *testing.T) {
	t.Parallel()
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(dir)
	tests := []struct {
		args     []string
		expected string
	}{
		{args: []string{"math", "Sqrt", "25.4"}, expected: "5.039841267341661\n"},
	}
	for _, test := range tests {
		t.Run(strings.Join(test.args, " "), func(t *testing.T) {
			t.Parallel()
			stderr := &bytes.Buffer{}
			stdout := &bytes.Buffer{}
			env := OSEnv{
				Stderr: stderr,
				Stdout: stdout,
				Args:   append([]string{"gorram"}, test.args...),
			}
			code := ParseAndRun(env)
			checkCode(code, filename, t)
			out := stdout.String()
			if out != test.expected {
				t.Errorf("Expected %q but got %q", test.expected, out)
			}
			if msg := stderr.String(); msg != "" {
				t.Errorf("Expected no stderr output but got %q", msg)
			}
		})
	}
}

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
	env := OSEnv{
		Stderr: stderr,
		Stdout: stdout,
		Args:   []string{"gorram", "time", "Now"},
	}
	code := ParseAndRun(env)
	checkCode(code, filename, t)
	out := stdout.String()
	expected := fmt.Sprint(time.Now()) + "\n"

	// have to fudge the output checking since obviously the two times will not
	// be exactly the same.  We do this by checking the prefix, up to the hours,
	// and the suffix for the time zone.
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
	env := OSEnv{
		Stderr: stderr,
		Stdout: stdout,
		Stdin:  strings.NewReader(`{"foo":"bar"}`),
		Args:   []string{"gorram", "encoding/json", "Indent", "", "  "},
	}
	code := ParseAndRun(env)
	checkCode(code, filename, t)
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
	env := OSEnv{
		Stderr: stderr,
		Stdout: stdout,
		Args:   []string{"gorram", "net/http", "Get", ts.URL},
	}
	code := ParseAndRun(env)
	checkCode(code, filename, t)
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
	env := OSEnv{
		Stderr: stderr,
		Stdout: stdout,
		Args:   []string{"gorram", "-t", "{{.Status}}", "net/http", "Get", ts.URL},
	}
	code := ParseAndRun(env)
	checkCode(code, filename, t)

	out := stdout.String()
	expected := "200 OK\n"
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
func TestNetHTTPGetWithFileTemplate(t *testing.T) {
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
	filename := filepath.Join(dir, "template.txt")
	if err := ioutil.WriteFile(filename, []byte("{{.Status}}"), 0600); err != nil {
		t.Fatal(err)
	}
	stderr := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	env := OSEnv{
		Stderr: stderr,
		Stdout: stdout,
		Args:   []string{"gorram", "-t", filename, "net/http", "Get", ts.URL},
	}
	code := ParseAndRun(env)
	checkCode(code, filename, t)

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
	env := OSEnv{
		Stderr: stderr,
		Stdout: stdout,
		Args:   []string{"gorram", "encoding/base64", "StdEncoding.EncodeToString", filename},
	}
	code := ParseAndRun(env)
	checkCode(code, filename, t)

	out := stdout.String()
	expected := "MTIzNDU=\n"
	if out != expected {
		t.Errorf("Expected %q but got %q", expected, out)
	}
	if msg := stderr.String(); msg != "" {
		t.Errorf("Expected no stderr output but got %q", msg)
	}
}

func checkCode(code int, filename string, t *testing.T) {
	if code == 0 {
		return
	}
	t.Errorf("unexpected exit code: %v", code)
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Logf("error reading generated file %q: %v", filename, err)
	}
	t.Log("Generated file contents:")
	t.Log(string(b))
}
