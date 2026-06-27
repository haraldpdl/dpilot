package cmd

import (
	"github.com/haraldpdl/dpilot/pkg/ddev"
	"github.com/spf13/cobra"
)

// Version is set via -ldflags at release time; "dev" otherwise.
var Version = "dev"

// newClient builds the ddev client; tests override it.
var newClient = func() ddev.Client { return ddev.New() }

var rootCmd = &cobra.Command{
	Use:           "dpilot",
	Short:         "Orchestrate ordered groups of ddev projects",
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
