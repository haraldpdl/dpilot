package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/haraldpdl/dpilot/pkg/ddev"
)

var (
	borderStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	titleStyle  = lipgloss.NewStyle().Bold(true)
	dimStyle    = lipgloss.NewStyle().Faint(true)
)

// statusColor renders a ddev status in the tone ddev uses, from the same
// mapping as the CLI tables (ddev.ProjectStatus.Tone). Like ddev's own TUI it
// keeps the raw word; only the CLI tables say "OK" for running.
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
	return lipgloss.NewStyle().Foreground(c).Render(string(st))
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

// window picks the slice [start,end) of n rows to draw so the cursor stays
// visible within a budget of terminal rows. When rows are hidden and the
// budget allows, two lines go to "more above/below" markers; with a budget
// under three the cursor row wins and no markers are drawn.
func window(n, cursor, budget int) (start, end, above, below int) {
	if budget <= 0 || n <= budget {
		return 0, n, 0, 0
	}
	size := budget
	if budget >= 3 {
		size = budget - 2
	}
	start = max(0, cursor-size/2)
	end = start + size
	if end > n {
		end = n
		start = max(0, end-size)
	}
	if budget < 3 {
		return start, end, 0, 0
	}
	return start, end, start, n - end
}

// box draws content inside the rounded border, first cutting every line to
// the terminal width (0 = unknown) so a long name or notice cannot push the
// right border off screen.
func box(content string, width int) string {
	if inner := width - 4; width > 0 && inner > 0 { // border (2) + padding (2)
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			if lipgloss.Width(line) > inner {
				cut := ansi.Truncate(line, inner, "…")
				if strings.Contains(cut, "\x1b[") {
					cut += "\x1b[0m" // the cut may have dropped the style reset
				}
				lines[i] = cut
			}
		}
		content = strings.Join(lines, "\n")
	}
	return borderStyle.Render(content)
}
