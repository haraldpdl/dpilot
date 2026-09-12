package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/haraldpdl/dpilot/pkg/ddev"
)

var (
	borderStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	titleStyle  = lipgloss.NewStyle().Bold(true)
	dimStyle    = lipgloss.NewStyle().Faint(true)
)

// statusColor renders a ddev status with the label and tone ddev uses, from
// the same mapping as the CLI tables (ddev.ProjectStatus.Tone/Label).
func statusColor(status string) string {
	if status == "" {
		return status
	}
	st := ddev.ProjectStatus(status)
	var c lipgloss.Color
	switch st.Tone() {
	case ddev.ToneWarn:
		c = lipgloss.Color("3")
	case ddev.ToneBad:
		c = lipgloss.Color("1")
	default:
		c = lipgloss.Color("2")
	}
	return lipgloss.NewStyle().Foreground(c).Render(st.Label())
}

// keyRune reports whether a key message is exactly the single rune r.
func keyRune(k tea.KeyMsg, r rune) bool {
	return k.Type == tea.KeyRunes && len(k.Runes) == 1 && k.Runes[0] == r
}
