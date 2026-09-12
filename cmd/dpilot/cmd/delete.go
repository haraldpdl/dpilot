package cmd

import (
	"bufio"
	"fmt"
	"strings"

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
			if !isInteractive() {
				return fmt.Errorf("refusing to delete group %q without -y", name)
			}
			if !confirm(cmd, fmt.Sprintf("OK to delete group %q?", name)) {
				cmd.Println("delete cancelled")
				return nil
			}
		}
		if err := config.Delete(name); err != nil {
			return err
		}
		cmd.Printf("deleted group %q\n", name)
		return nil
	},
}

// confirm asks a yes/no question the way ddev does: blank answers take the
// default (yes), and after three unreadable answers it gives up with no.
func confirm(cmd *cobra.Command, prompt string) bool {
	in := bufio.NewReader(cmd.InOrStdin())
	for range 3 {
		cmd.Printf("%s [Y/n] (yes): ", prompt)
		line, err := in.ReadString('\n')
		if err != nil && line == "" {
			return false
		}
		switch strings.ToLower(strings.TrimSpace(line)) {
		case "", "y", "yes":
			return true
		case "n", "no":
			return false
		}
	}
	return false
}

func init() {
	deleteCmd.Flags().BoolP("yes", "y", false, "Yes - skip confirmation prompt")
	rootCmd.AddCommand(deleteCmd)
}
