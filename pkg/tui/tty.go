package tui

import (
	"os"

	"github.com/mattn/go-isatty"
)

// IsInteractive reports whether both stdin and stdout are terminals.
func IsInteractive() bool {
	return isatty.IsTerminal(os.Stdin.Fd()) && isatty.IsTerminal(os.Stdout.Fd())
}
