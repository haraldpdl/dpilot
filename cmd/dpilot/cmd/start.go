package cmd

import (
	"context"
	"errors"
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

// signalCtx cancels on the first Ctrl-C. It then hands the signal back to the
// default handler, so a second Ctrl-C ends dpilot at once instead of being
// swallowed while ddev is given its grace period.
func signalCtx() (context.Context, context.CancelFunc) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	go func() {
		<-ctx.Done()
		stop()
	}()
	return ctx, stop
}

// lifecycleArgs is the shared "<group ...> | --all" contract of
// start/stop/restart, mirroring ddev's "[projectname ...]" verbs.
func lifecycleArgs(c *cobra.Command, allHelp string) {
	c.Args = cobra.ArbitraryArgs
	c.Flags().BoolP("all", "a", false, allHelp)
}

// groupsFor resolves the groups a lifecycle verb acts on: the named groups in
// the order given, or every group in name order with --all.
func groupsFor(cmd *cobra.Command, args []string) ([]*config.Group, error) {
	all, _ := cmd.Flags().GetBool("all")
	names := args
	switch {
	case all && len(args) > 0:
		return nil, errors.New("give group names or --all, not both")
	case !all && len(args) == 0:
		return nil, errors.New("requires a group name or --all")
	case all:
		var err error
		if names, err = config.List(); err != nil {
			return nil, err
		}
	}
	groups := make([]*config.Group, 0, len(names))
	for _, n := range names {
		g, err := config.Load(n)
		if err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, nil
}

var startCmd = &cobra.Command{
	Use:   "start [group ...]",
	Short: "Start all projects in a group, in order",
	RunE: func(cmd *cobra.Command, args []string) error {
		groups, err := groupsFor(cmd, args)
		if err != nil {
			return err
		}
		ctx, stop := signalCtx()
		defer stop()
		o := orch(cmd)
		for _, g := range groups {
			if err := o.Start(ctx, g); err != nil {
				return err
			}
		}
		return nil
	},
}

func init() {
	lifecycleArgs(startCmd, "Start all groups")
	rootCmd.AddCommand(startCmd)
}
