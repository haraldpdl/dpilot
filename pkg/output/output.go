package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/mattn/go-isatty"
)

// GroupRow is one row of `dpilot list`. Error is set for a group whose file
// could not be loaded; its counts are then zero.
type GroupRow struct {
	Name    string `json:"name"`
	Members int    `json:"members"`
	Running int    `json:"running"`
	Error   string `json:"error,omitempty"`
}

// MemberRow is one row of `dpilot describe`.
type MemberRow struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// envelope is the shape ddev's -j output uses: the rendered text in msg and
// the data in raw, so scripts written for ddev keep working.
type envelope struct {
	Level string `json:"level"`
	Msg   string `json:"msg"`
	Raw   any    `json:"raw"`
	Time  string `json:"time"`
}

// now is the envelope clock; tests may replace it.
var now = time.Now

func writeJSON(w io.Writer, msg string, raw any) error {
	return json.NewEncoder(w).Encode(envelope{
		Level: "info",
		Msg:   strings.TrimRight(msg, "\n"),
		Raw:   raw,
		Time:  now().Format(time.RFC3339),
	})
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

// Groups renders the group list as a table or JSON. Invalid groups appear in
// the table marked "invalid", with their errors listed below it.
func Groups(w io.Writer, rows []GroupRow, jsonOut bool) error {
	text := renderGroups(rows)
	if jsonOut {
		if rows == nil {
			rows = []GroupRow{}
		}
		return writeJSON(w, text, rows)
	}
	_, err := io.WriteString(w, text)
	return err
}

func renderGroups(rows []GroupRow) string {
	if len(rows) == 0 {
		return "No dpilot groups found. Run 'dpilot create <group>' to make one.\n"
	}
	var b strings.Builder
	t := table.NewWriter()
	t.SetOutputMirror(&b)
	t.AppendHeader(table.Row{"GROUP", "MEMBERS", "RUNNING"})
	var invalid []GroupRow
	for _, r := range rows {
		if r.Error != "" {
			t.AppendRow(table.Row{r.Name, "invalid", ""})
			invalid = append(invalid, r)
			continue
		}
		t.AppendRow(table.Row{r.Name, r.Members, r.Running})
	}
	t.Render()
	for _, r := range invalid {
		fmt.Fprintf(&b, "%s: %s\n", r.Name, r.Error)
	}
	return b.String()
}

// Describe renders a group's members as a table or JSON.
func Describe(w io.Writer, group string, rows []MemberRow, jsonOut bool) error {
	if jsonOut {
		if rows == nil {
			rows = []MemberRow{}
		}
		return writeJSON(w, renderDescribe(rows, false), map[string]any{"name": group, "members": rows})
	}
	_, err := io.WriteString(w, renderDescribe(rows, colorize(w)))
	return err
}

func renderDescribe(rows []MemberRow, color bool) string {
	var b strings.Builder
	t := table.NewWriter()
	t.SetOutputMirror(&b)
	t.AppendHeader(table.Row{"#", "PROJECT", "STATUS"})
	for i, r := range rows {
		t.AppendRow(table.Row{i + 1, r.Name, colorStatus(r.Status, color)})
	}
	t.Render()
	return b.String()
}
