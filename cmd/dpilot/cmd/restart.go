package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart [group ...]",
	Short: "Stop then start all projects in a group",
	RunE: func(cmd *cobra.Command, args []string) error {
		groups, err := groupsFor(cmd, args)
		if err != nil {
			return err
		}
		ctx, stop := signalCtx()
		defer stop()
		o := orch(cmd)
		if len(groups) == 1 {
			return o.Restart(ctx, groups[0])
		}
		// Several groups: like stop --all then start --all, so no group is
		// started while a later one is still to be stopped.
		var errs []error
		for i := len(groups) - 1; i >= 0; i-- {
			if err := o.Stop(ctx, groups[i]); err != nil {
				errs = append(errs, err)
			}
		}
		if err := ctx.Err(); err != nil {
			return errors.Join(errs...)
		}
		if len(errs) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "restart: stop reported errors, continuing to start: %v\n", errors.Join(errs...))
		}
		for _, g := range groups {
			if err := o.Start(ctx, g); err != nil {
				return err
			}
		}
		return nil
	},
}

func init() {
	lifecycleArgs(restartCmd, "Restart all groups")
	rootCmd.AddCommand(restartCmd)
}
