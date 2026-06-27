package cmd

import (
	"fmt"

	"github.com/haraldpdl/dpilot/pkg/config"
	"github.com/spf13/cobra"
)

var deleteYes bool

var deleteCmd = &cobra.Command{
	Use:   "delete <group>",
	Short: "Delete a group",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if !deleteYes {
			cmd.Printf("re-run with -y to delete group %q\n", name)
			return fmt.Errorf("confirmation required")
		}
		if err := config.Delete(name); err != nil {
			return err
		}
		cmd.Printf("deleted group %q\n", name)
		return nil
	},
}

func init() {
	deleteCmd.Flags().BoolVarP(&deleteYes, "yes", "y", false, "confirm deletion")
	rootCmd.AddCommand(deleteCmd)
}
