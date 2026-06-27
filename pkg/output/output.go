package output

import (
	"encoding/json"
	"io"

	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
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

func colorStatus(s string) string {
	switch s {
	case "running":
		return color.GreenString(s)
	case "missing":
		return color.RedString(s)
	default:
		return color.YellowString(s)
	}
}

// Describe renders a group's members as a table or JSON.
func Describe(w io.Writer, group string, rows []MemberRow, jsonOut bool) error {
	if jsonOut {
		return writeJSON(w, map[string]any{"name": group, "members": rows})
	}
	t := table.NewWriter()
	t.SetOutputMirror(w)
	t.AppendHeader(table.Row{"#", "PROJECT", "STATUS"})
	for i, r := range rows {
		t.AppendRow(table.Row{i + 1, r.Name, colorStatus(r.Status)})
	}
	t.Render()
	return nil
}
