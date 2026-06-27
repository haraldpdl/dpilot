package cmd

import (
	"fmt"

	"github.com/haraldpdl/dpilot/pkg/config"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <group>",
	Short: "Delete a group",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		yes, _ := cmd.Flags().GetBool("yes")
		if !yes {
			return fmt.Errorf("refusing to delete group %q without -y", name)
		}
		if err := config.Delete(name); err != nil {
			return err
		}
		cmd.Printf("deleted group %q\n", name)
		return nil
	},
}

func init() {
	deleteCmd.Flags().BoolP("yes", "y", false, "confirm deletion")
	rootCmd.AddCommand(deleteCmd)
}
