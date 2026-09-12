package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	borderStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	titleStyle  = lipgloss.NewStyle().Bold(true)
	dimStyle    = lipgloss.NewStyle().Faint(true)
)

// statusColor styles a ddev status string to match the output package's colors.
func statusColor(status string) string {
	switch status {
	case "running":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render(status)
	case "missing":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render(status)
	case "":
		return status
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render(status)
	}
}

// keyRune reports whether a key message is exactly the single rune r.
func keyRune(k tea.KeyMsg, r rune) bool {
	return k.Type == tea.KeyRunes && len(k.Runes) == 1 && k.Runes[0] == r
}

// shortError makes a load error fit one dashboard row: the group name is
// already in the row, so its "group "x": " prefix goes, and yaml's multi-line
// messages are flattened and capped.
func shortError(name, err string) string {
	for _, prefix := range []string{"parse group \"" + name + "\": ", "group \"" + name + "\": "} {
		err = strings.TrimPrefix(err, prefix)
	}
	return oneLine(err)
}

// oneLine flattens whitespace and caps a message so it cannot break a row.
func oneLine(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 60 {
		s = s[:59] + "…"
	}
	return s
}
