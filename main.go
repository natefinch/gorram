package main // import "npf.io/gorram"

import (
	"os"

	"npf.io/gorram/cli"
)

func main() {
	os.Exit(cli.Run())
}
