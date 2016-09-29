package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/natefinch/gorram/run"
)

// Parse converts the gorram command line (not including executable name) and
// returns a gorram run command and the unused args.
func Parse(args []string) (cmd run.Command, remaining []string, err error) {
	if len(args) == 0 {
		return cmd, nil, errors.New("No command specified.")
	}
	parts := strings.Split(args[0], ".")
	switch len(parts) {
	case 2:
		cmd.Package = parts[0]
		cmd.Function = parts[1]
	case 3:
		cmd.Package = parts[0]
		cmd.GlobalVar = parts[1]
		cmd.Function = parts[2]
	default:
		return cmd, nil, fmt.Errorf(`Command %q invalid. Expected "package.Function" or "package.Varable.Method".`, args[0])
	}
	return cmd, args[1:], nil
}
