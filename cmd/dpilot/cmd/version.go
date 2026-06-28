package cmd

import (
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the dpilot version",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.Printf("dpilot %s\n", resolveVersion())
		return nil
	},
}

// resolveVersion returns the release version injected via -ldflags, falling back
// to the module version from the build info so `go install` builds report their
// tag instead of "dev".
func resolveVersion() string {
	if Version != "dev" && Version != "" {
		return Version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			return strings.TrimPrefix(v, "v")
		}
	}
	return Version
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
