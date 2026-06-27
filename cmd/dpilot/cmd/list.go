package cmd

import (
	"context"

	"github.com/haraldpdl/dpilot/pkg/orchestrator"
	"github.com/haraldpdl/dpilot/pkg/output"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"l"},
	Short:   "List groups",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		jsonOut, _ := cmd.Flags().GetBool("json-output")
		summaries, err := orchestrator.GroupSummaries(context.Background(), newClient())
		if err != nil {
			return err
		}
		rows := make([]output.GroupRow, 0, len(summaries))
		for _, s := range summaries {
			rows = append(rows, output.GroupRow{Name: s.Name, Members: s.Members, Running: s.Running})
		}
		return output.Groups(cmd.OutOrStdout(), rows, jsonOut)
	},
}

func init() {
	listCmd.Flags().BoolP("json-output", "j", false, "output as JSON")
	rootCmd.AddCommand(listCmd)
}
