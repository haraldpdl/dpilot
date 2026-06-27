package cmd

import (
	"fmt"

	"github.com/haraldpdl/dpilot/pkg/config"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove <group> <project>",
	Short: "Remove a project from a group",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		groupName, project := args[0], args[1]
		g, err := config.Load(groupName)
		if err != nil {
			return err
		}
		out := g.Members[:0:0]
		found := false
		for _, m := range g.Members {
			if m == project {
				found = true
				continue
			}
			out = append(out, m)
		}
		if !found {
			return fmt.Errorf("%q is not a member of %q", project, groupName)
		}
		g.Members = out
		if err := config.Save(g); err != nil {
			return err
		}
		cmd.Printf("removed %q from %q\n", project, groupName)
		return nil
	},
}

func init() { rootCmd.AddCommand(removeCmd) }
