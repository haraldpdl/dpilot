package cmd

import (
	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart [group]",
	Short: "Stop then start all projects in a group",
	RunE: func(cmd *cobra.Command, args []string) error {
		groups, err := groupsFor(cmd, args)
		if err != nil {
			return err
		}
		ctx, stop := signalCtx()
		defer stop()
		o := orch(cmd)
		for _, g := range groups {
			if err := o.Restart(ctx, g); err != nil {
				return err
			}
		}
		return nil
	},
}

func init() {
	lifecycleArgs(restartCmd)
	rootCmd.AddCommand(restartCmd)
}
