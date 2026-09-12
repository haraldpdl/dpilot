package cmd

import (
	"encoding/json"
	"time"

	"github.com/haraldpdl/dpilot/pkg/ddev"
	"github.com/haraldpdl/dpilot/pkg/tui"
	"github.com/spf13/cobra"
)

// Version is set via -ldflags at release time; "dev" otherwise.
var Version = "dev"

// newClient builds the ddev client; tests override it.
var newClient = func() ddev.Client { return ddev.New() }

// isInteractive is the TTY check; overridden in tests.
var isInteractive = tui.IsInteractive

// runDashboard runs the dashboard program; overridden in tests.
var runDashboard = tui.RunDashboard

var rootCmd = &cobra.Command{
	Use:           "dpilot",
	Short:         "Orchestrate ordered groups of ddev projects",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if v, _ := cmd.Flags().GetBool("version"); v {
			return printVersion(cmd)
		}
		if len(args) == 0 && isInteractive() {
			return runDashboard(tui.ProductionLoader(newClient()))
		}
		return cmd.Help()
	},
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

// jsonOutput reports whether -j/--json-output was given.
func jsonOutput() bool {
	v, _ := rootCmd.PersistentFlags().GetBool("json-output")
	return v
}

// ErrorLine formats a top-level error for stderr: a plain "dpilot: ..." line,
// or, with -j, a ddev-style fatal JSON line.
func ErrorLine(err error) string {
	if !jsonOutput() {
		return "dpilot: " + err.Error()
	}
	b, _ := json.Marshal(map[string]string{
		"level": "fatal",
		"msg":   err.Error(),
		"time":  time.Now().Format(time.RFC3339),
	})
	return string(b)
}

func init() {
	// Persistent like ddev's, so both `dpilot -j list` and `dpilot list -j` work.
	rootCmd.PersistentFlags().BoolP("json-output", "j", false, "If true, user-oriented output will be in JSON format.")
}
