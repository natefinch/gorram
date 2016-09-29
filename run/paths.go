package run

import (
	"os"
	"path/filepath"
	"runtime"
)

func confDir() string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("HOMEDRIVE"), os.Getenv("HOMEPATH"), "gorram")
	default:
		return filepath.Join(os.Getenv("HOME"), ".gorram")
	}
}

func createFile(cmd Command) (f *os.File, path string, err error) {
	dir := filepath.Join(confDir(), filepath.FromSlash(cmd.Package))
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, "", err
	}
	name := cmd.Function
	if cmd.GlobalVar != "" {
		name = cmd.GlobalVar + "." + cmd.Function
	}
	path = filepath.Join(dir, name+".go")
	f, err = os.Create(path)
	return f, path, err
}
