package cmd

import (
	"context"

	"github.com/haraldpdl/dpilot/pkg/config"
	"github.com/haraldpdl/dpilot/pkg/ddev"
	"github.com/haraldpdl/dpilot/pkg/output"
	"github.com/spf13/cobra"
)

var listJSON bool

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"l"},
	Short:   "List groups",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		names, err := config.List()
		if err != nil {
			return err
		}
		projects, err := newClient().List(context.Background())
		if err != nil {
			return err
		}
		running := map[string]bool{}
		for _, p := range projects {
			if p.Status == ddev.StatusRunning {
				running[p.Name] = true
			}
		}
		rows := make([]output.GroupRow, 0, len(names))
		for _, n := range names {
			g, err := config.Load(n)
			if err != nil {
				return err
			}
			r := output.GroupRow{Name: n, Members: len(g.Members)}
			for _, m := range g.Members {
				if running[m] {
					r.Running++
				}
			}
			rows = append(rows, r)
		}
		return output.Groups(cmd.OutOrStdout(), rows, listJSON)
	},
}

func init() {
	listCmd.Flags().BoolVarP(&listJSON, "json-output", "j", false, "output as JSON")
	rootCmd.AddCommand(listCmd)
}
