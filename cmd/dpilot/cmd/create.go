package cmd

import (
	"context"
	"fmt"

	"github.com/haraldpdl/dpilot/pkg/config"
	"github.com/haraldpdl/dpilot/pkg/tui"
	"github.com/spf13/cobra"
)

// runEditor runs the interactive group editor; overridden in tests.
var runEditor = tui.RunEditor

var createCmd = &cobra.Command{
	Use:   "create <group>",
	Short: "Create a new group (interactive on a terminal)",
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
		if !isInteractive() {
			if err := config.Save(&config.Group{Name: name}); err != nil {
				return err
			}
			cmd.Printf("created group %q\n", name)
			return nil
		}
		projects, err := newClient().List(context.Background())
		if err != nil {
			return err
		}
		g, err := runEditor(tui.EditorOptions{
			Name:           name,
			NameFixed:      true,
			Projects:       projects,
			InitialTimeout: config.DefaultWaitTimeout,
		})
		if err != nil {
			return err
		}
		if g == nil {
			cmd.Println("canceled")
			return nil
		}
		if err := config.Save(g); err != nil {
			return err
		}
		cmd.Printf("created group %q with %d member(s)\n", g.Name, len(g.Members))
		return nil
	},
}

func init() { rootCmd.AddCommand(createCmd) }
