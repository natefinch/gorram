package run

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"
)

func TestTimeNow(t *testing.T) {
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
	err = Run(Command{Package: "time", Function: "Now", Cache: dir}, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := stdout.String()
	expected := fmt.Sprint(time.Now()) + "\n"
	if !strings.HasPrefix(out, expected[:15]) {
		t.Fatalf("Expected ~%q but got %q", expected, out)
	}
	if !strings.HasSuffix(out, expected[len(expected)-9:]) {
		t.Fatalf("Expected ~%q but got %q", expected, out)
	}
}

func TestMathSqrt(t *testing.T) {
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
	err = Run(Command{Package: "math", Function: "Sqrt", Args: []string{"25"}, Cache: dir}, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := stdout.String()
	expected := "5\n"
	if out != expected {
		t.Fatalf("Expected %q but got %q", expected, out)
	}
}

func TestJsonIndentStdin(t *testing.T) {
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
	err = Run(Command{Package: "encoding/json", Function: "Indent", Args: []string{"", "  "}, Cache: dir}, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := stdout.String()
	expected := `
{
  "foo": "bar"
}
`[1:]
	if out != expected {
		t.Fatalf("Expected %q but got %q", expected, out)
	}
}
