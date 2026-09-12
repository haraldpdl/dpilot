package cmd

import (
	"runtime/debug"
	"strings"

	"github.com/haraldpdl/dpilot/pkg/output"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the dpilot version",
	Args:  cobra.NoArgs,
	RunE:  func(cmd *cobra.Command, _ []string) error { return printVersion(cmd) },
}

// printVersion writes the version like ddev does: plain text, or under -j an
// envelope whose raw payload carries the version.
func printVersion(cmd *cobra.Command) error {
	v := resolveVersion()
	text := "dpilot version " + v + "\n"
	if jsonOutput() {
		return output.Info(cmd.OutOrStdout(), text, map[string]string{"version": v})
	}
	cmd.Print(text)
	return nil
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
	// --version / -v as in ddev. Handled in the root command rather than via
	// cobra's Version field so that -j is honoured.
	rootCmd.PersistentFlags().BoolP("version", "v", false, "Print the dpilot version")
}
