package tui

import (
	"fmt"
	"strings"

	"github.com/haraldpdl/dpilot/pkg/orchestrator"
)

// describeView renders a group's members and their live states.
func describeView(states []orchestrator.MemberState) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", titleStyle.Render("describe"))
	for i, s := range states {
		fmt.Fprintf(&b, " %d  %-20s %s\n", i+1, s.Name, statusColor(string(s.Status)))
	}
	b.WriteString(dimStyle.Render("\nany key to return"))
	return borderStyle.Render(b.String())
}
