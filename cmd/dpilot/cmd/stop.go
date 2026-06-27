package cmd

import (
	"context"

	"github.com/haraldpdl/dpilot/pkg/config"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop <group>",
	Short: "Stop all projects in a group, in reverse order",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		g, err := config.Load(args[0])
		if err != nil {
			return err
		}
		return orch(cmd).Stop(context.Background(), g)
	},
}

func init() { rootCmd.AddCommand(stopCmd) }
