package tui

import (
	"fmt"
	"strings"

	"github.com/haraldpdl/dpilot/pkg/orchestrator"
)

// describeView renders a group's members and their live states, windowed to
// the terminal height (0 = unknown) like the other lists.
func describeView(states []orchestrator.MemberState, height int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", titleStyle.Render("describe"))
	budget := 0
	if height > 0 {
		budget = max(1, height-6) // border (2), title + blank, blank + footer
	}
	start, end, above, below := window(len(states), 0, budget)
	if above > 0 {
		fmt.Fprintf(&b, "%s\n", dimStyle.Render(fmt.Sprintf("  … %d more above", above)))
	}
	for i, s := range states[start:end] {
		fmt.Fprintf(&b, " %d  %-20s %s\n", start+i+1, s.Name, statusColor(string(s.Status)))
	}
	if below > 0 {
		fmt.Fprintf(&b, "%s\n", dimStyle.Render(fmt.Sprintf("  … %d more below", below)))
	}
	b.WriteString(dimStyle.Render("\nany key to return"))
	return borderStyle.Render(b.String())
}
