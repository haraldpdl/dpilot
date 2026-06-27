package cmd

import (
	"fmt"

	"github.com/haraldpdl/dpilot/pkg/config"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create <group>",
	Short: "Create a new empty group",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		ok, err := config.Exists(name)
		if err != nil {
			return err
		}
		if ok {
			return fmt.Errorf("group %q already exists", name)
		}
		if err := config.Save(&config.Group{Name: name}); err != nil {
			return err
		}
		cmd.Printf("created group %q\n", name)
		return nil
	},
}

func init() { rootCmd.AddCommand(createCmd) }
