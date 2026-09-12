package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop [group]",
	Short: "Stop all projects in a group, in reverse order",
	RunE: func(cmd *cobra.Command, args []string) error {
		groups, err := groupsFor(cmd, args)
		if err != nil {
			return err
		}
		ctx, stop := signalCtx()
		defer stop()
		o := orch(cmd)
		var errs []error
		for i := len(groups) - 1; i >= 0; i-- { // --all stops groups in reverse name order
			if err := o.Stop(ctx, groups[i]); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.Join(errs...)
	},
}

func init() {
	lifecycleArgs(stopCmd)
	rootCmd.AddCommand(stopCmd)
}
