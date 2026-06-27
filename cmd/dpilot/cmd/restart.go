package cmd

import (
	"github.com/haraldpdl/dpilot/pkg/config"
	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart <group>",
	Short: "Stop then start all projects in a group",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		g, err := config.Load(args[0])
		if err != nil {
			return err
		}
		ctx, stop := signalCtx()
		defer stop()
		return orch(cmd).Restart(ctx, g)
	},
}

func init() { rootCmd.AddCommand(restartCmd) }
