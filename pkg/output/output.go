package output

import (
	"encoding/json"
	"io"
	"os"

	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/mattn/go-isatty"
)

// GroupRow is one row of `dpilot list`.
type GroupRow struct {
	Name    string `json:"name"`
	Members int    `json:"members"`
	Running int    `json:"running"`
}

// MemberRow is one row of `dpilot describe`.
type MemberRow struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func colorize(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return os.Getenv("NO_COLOR") == "" && isatty.IsTerminal(f.Fd())
}

func colorStatus(s string, enabled bool) string {
	if !enabled {
		return s
	}
	var c *color.Color
	switch s {
	case "running":
		c = color.New(color.FgGreen)
	case "missing":
		c = color.New(color.FgRed)
	default:
		c = color.New(color.FgYellow)
	}
	c.EnableColor()
	return c.Sprint(s)
}

// Groups renders the group list as a table or JSON.
func Groups(w io.Writer, rows []GroupRow, jsonOut bool) error {
	if jsonOut {
		return writeJSON(w, rows)
	}
	t := table.NewWriter()
	t.SetOutputMirror(w)
	t.AppendHeader(table.Row{"GROUP", "MEMBERS", "RUNNING"})
	for _, r := range rows {
		t.AppendRow(table.Row{r.Name, r.Members, r.Running})
	}
	t.Render()
	return nil
}

// Describe renders a group's members as a table or JSON.
func Describe(w io.Writer, group string, rows []MemberRow, jsonOut bool) error {
	if jsonOut {
		return writeJSON(w, map[string]any{"name": group, "members": rows})
	}
	enabled := colorize(w)
	t := table.NewWriter()
	t.SetOutputMirror(w)
	t.AppendHeader(table.Row{"#", "PROJECT", "STATUS"})
	for i, r := range rows {
		t.AppendRow(table.Row{i + 1, r.Name, colorStatus(r.Status, enabled)})
	}
	t.Render()
	return nil
}
