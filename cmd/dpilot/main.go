package main

import (
	"fmt"
	"os"

	"github.com/haraldpdl/dpilot/cmd/dpilot/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, cmd.ErrorLine(err))
		os.Exit(1)
	}
}
