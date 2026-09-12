package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/haraldpdl/dpilot/pkg/ddev"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/mattn/go-isatty"
)

// ddevStyle is ddev's default table look (StyleLight box, separated rows,
// upper-case headers), so dpilot's tables sit next to ddev's unnoticed.
var ddevStyle = func() table.Style {
	s := table.StyleLight
	s.Options.SeparateRows = true
	s.Format.Header = text.FormatUpper
	return s
}()

func newTable(w io.Writer) table.Writer {
	t := table.NewWriter()
	t.SetOutputMirror(w)
	t.SetStyle(ddevStyle)
	return t
}

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

func writeJSON(w io.Writer, msg string, raw any) error {
	return json.NewEncoder(w).Encode(envelope{
		Level: "info",
		Msg:   msg, // as rendered, trailing newline included, like ddev
		Raw:   raw,
		Time:  time.Now().Format(time.RFC3339),
	})
}

// Info writes one ddev-style info envelope: text in msg, data in raw.
func Info(w io.Writer, msg string, raw any) error { return writeJSON(w, msg, raw) }

func colorize(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return os.Getenv("NO_COLOR") == "" && isatty.IsTerminal(f.Fd())
}

// colorStatus renders a status word the way ddev does: same label, same
// colour, and no escape codes when colour is off.
func colorStatus(s string, enabled bool) string {
	st := ddev.ProjectStatus(s)
	if !enabled {
		return st.Label()
	}
	var c text.Color
	switch st.Tone() {
	case ddev.ToneWarn:
		c = text.FgYellow
	case ddev.ToneBad:
		c = text.FgRed
	default:
		c = text.FgGreen
	}
	return c.Sprint(st.Label())
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
	t := newTable(&b)
	t.AppendHeader(table.Row{"GROUP", "MEMBERS", "RUNNING"})
	// Pin the numeric columns: an "invalid" cell would otherwise flip
	// go-pretty's content-based alignment for every row.
	t.SetColumnConfigs([]table.ColumnConfig{
		{Name: "MEMBERS", Align: text.AlignRight},
		{Name: "RUNNING", Align: text.AlignRight},
	})
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
	for _, r := range invalid { // every load error already names its group
		fmt.Fprintln(&b, r.Error)
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
	t := newTable(&b)
	t.AppendHeader(table.Row{"#", "PROJECT", "STATUS"})
	for i, r := range rows {
		t.AppendRow(table.Row{i + 1, r.Name, colorStatus(r.Status, color)})
	}
	t.Render()
	return b.String()
}
