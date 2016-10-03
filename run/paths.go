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

func createFile(path string) (f *os.File, err error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	return os.Create(path)
}

func dir(cmd Command) string {
	base := cmd.Cache
	if cmd.Cache == "" {
		base = confDir()
	}
	return filepath.Join(base, filepath.FromSlash(cmd.Package))
}

func script(cmd Command) string {
	name := cmd.Function
	if cmd.GlobalVar != "" {
		name = cmd.GlobalVar + "." + cmd.Function
	}
	return filepath.Join(dir(cmd), name+".go")
}
