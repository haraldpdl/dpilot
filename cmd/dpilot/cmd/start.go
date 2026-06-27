package cmd

import (
	"context"

	"github.com/haraldpdl/dpilot/pkg/config"
	"github.com/haraldpdl/dpilot/pkg/orchestrator"
	"github.com/spf13/cobra"
)

func orch(cmd *cobra.Command) *orchestrator.Orchestrator {
	o := orchestrator.New(newClient())
	o.Out = cmd.OutOrStdout()
	return o
}

var startCmd = &cobra.Command{
	Use:   "start <group>",
	Short: "Start all projects in a group, in order",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		g, err := config.Load(args[0])
		if err != nil {
			return err
		}
		return orch(cmd).Start(context.Background(), g)
	},
}

func init() { rootCmd.AddCommand(startCmd) }
