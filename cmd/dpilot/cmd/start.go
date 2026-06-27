package cmd

import (
	"context"
	"os"
	"os/signal"

	"github.com/haraldpdl/dpilot/pkg/config"
	"github.com/haraldpdl/dpilot/pkg/orchestrator"
	"github.com/spf13/cobra"
)

func orch(cmd *cobra.Command) *orchestrator.Orchestrator {
	o := orchestrator.New(newClient())
	o.Out = cmd.OutOrStdout()
	return o
}

func signalCtx() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt)
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
		ctx, stop := signalCtx()
		defer stop()
		return orch(cmd).Start(ctx, g)
	},
}

func init() { rootCmd.AddCommand(startCmd) }
