package cmd

import (
	"github.com/haraldpdl/dpilot/pkg/ddev"
	"github.com/haraldpdl/dpilot/pkg/tui"
	"github.com/spf13/cobra"
)

// Version is set via -ldflags at release time; "dev" otherwise.
var Version = "dev"

// newClient builds the ddev client; tests override it.
var newClient = func() ddev.Client { return ddev.New() }

// isInteractive is the TTY check; overridden in tests.
var isInteractive = tui.IsInteractive

// runDashboard runs the dashboard program; overridden in tests.
var runDashboard = tui.RunDashboard

var rootCmd = &cobra.Command{
	Use:           "dpilot",
	Short:         "Orchestrate ordered groups of ddev projects",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 && isInteractive() {
			return runDashboard(tui.ProductionLoader(newClient()))
		}
		return cmd.Help()
	},
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
