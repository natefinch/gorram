package cli

import (
	"os"
	"path/filepath"
	"runtime"
)

// CacheEnv is the environment variable that users may set to change the
// location where gorram stores its script files.
const CacheEnv = "GORRAM_CACHE"

// confDir returns the default gorram configuration directory.
func confDir(env map[string]string) string {
	d := env[CacheEnv]
	if d != "" {
		return d
	}
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("HOMEDRIVE"), os.Getenv("HOMEPATH"), "gorram")
	default:
		return filepath.Join(os.Getenv("HOME"), ".gorram")
	}
}
