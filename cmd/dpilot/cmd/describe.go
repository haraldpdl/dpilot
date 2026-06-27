package cmd

import (
	"context"

	"github.com/haraldpdl/dpilot/pkg/config"
	"github.com/haraldpdl/dpilot/pkg/orchestrator"
	"github.com/haraldpdl/dpilot/pkg/output"
	"github.com/spf13/cobra"
)

var describeCmd = &cobra.Command{
	Use:     "describe <group>",
	Aliases: []string{"status"},
	Short:   "Show a group's members and their live ddev state",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOut, _ := cmd.Flags().GetBool("json-output")
		g, err := config.Load(args[0])
		if err != nil {
			return err
		}
		states, err := orchestrator.New(newClient()).Statuses(context.Background(), g)
		if err != nil {
			return err
		}
		rows := make([]output.MemberRow, 0, len(states))
		for _, s := range states {
			rows = append(rows, output.MemberRow{Name: s.Name, Status: string(s.Status)})
		}
		return output.Describe(cmd.OutOrStdout(), g.Name, rows, jsonOut)
	},
}

func init() {
	describeCmd.Flags().BoolP("json-output", "j", false, "output as JSON")
	rootCmd.AddCommand(describeCmd)
}
