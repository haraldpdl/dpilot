package cmd

import (
	"context"
	"fmt"

	"github.com/haraldpdl/dpilot/pkg/config"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <group> <project>",
	Short: "Add a ddev project to a group",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		groupName, project := args[0], args[1]
		after, _ := cmd.Flags().GetString("after")
		g, err := config.Load(groupName)
		if err != nil {
			return err
		}
		// Validate the project exists in ddev.
		projects, err := newClient().List(context.Background())
		if err != nil {
			return err
		}
		known := false
		for _, p := range projects {
			if p.Name == project {
				known = true
				break
			}
		}
		if !known {
			return fmt.Errorf("ddev project %q not found (run 'ddev list')", project)
		}
		for _, m := range g.Members {
			if m == project {
				return fmt.Errorf("%q is already a member of %q", project, groupName)
			}
		}
		if after != "" {
			found := false
			for _, m := range g.Members {
				if m == after {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("--after %q is not a member of %q", after, groupName)
			}
		}
		g.Members = insert(g.Members, project, after)
		if err := config.Save(g); err != nil {
			return err
		}
		cmd.Printf("added %q to %q\n", project, groupName)
		return nil
	},
}

// insert appends project, or places it after `after` when set.
func insert(members []string, project, after string) []string {
	if after == "" {
		return append(members, project)
	}
	out := make([]string, 0, len(members)+1)
	for _, m := range members {
		out = append(out, m)
		if m == after {
			out = append(out, project)
		}
	}
	// If `after` was not found, fall back to append.
	if len(out) == len(members) {
		out = append(out, project)
	}
	return out
}

func init() {
	addCmd.Flags().String("after", "", "insert after this member instead of appending")
	rootCmd.AddCommand(addCmd)
}
