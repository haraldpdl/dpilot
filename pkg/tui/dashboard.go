package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/haraldpdl/dpilot/pkg/config"
	"github.com/haraldpdl/dpilot/pkg/ddev"
	"github.com/haraldpdl/dpilot/pkg/orchestrator"
)

type dashMode int

const (
	modeList dashMode = iota
	modeDescribe
	modeConfirmDelete
	modeEditor
)

const refreshInterval = 3 * time.Second

// GroupRow is one dashboard row.
type GroupRow struct {
	Name    string
	Members int
	Running int
}

// Loader supplies the dashboard's data and side effects, injected for testability.
type Loader struct {
	Rows     func() ([]GroupRow, error)
	Statuses func(string) ([]orchestrator.MemberState, error)
	Delete   func(string) error
	Projects func() ([]ddev.Project, error)
	Load     func(string) (*config.Group, error)
	Save     func(*config.Group) error
	Exists   func(string) bool
	Exec     func(verb, group string) tea.Cmd
}

type rowsMsg struct {
	rows []GroupRow
	err  error
}

type statusesMsg struct {
	states []orchestrator.MemberState
	err    error
}

type refreshMsg struct{}

type tickMsg struct{}

// Dashboard is the bubbletea model for the group dashboard.
type Dashboard struct {
	loader   Loader
	mode     dashMode
	rows     []GroupRow
	cursor   int
	err      string
	describe []orchestrator.MemberState
	editor   Editor
}

// NewDashboard builds a Dashboard from a Loader.
func NewDashboard(loader Loader) Dashboard { return Dashboard{loader: loader} }

func (d Dashboard) Init() tea.Cmd { return tea.Batch(d.loadRows(), tickCmd()) }

func tickCmd() tea.Cmd {
	return tea.Tick(refreshInterval, func(time.Time) tea.Msg { return tickMsg{} })
}

func (d Dashboard) loadRows() tea.Cmd {
	loader := d.loader
	return func() tea.Msg {
		rows, err := loader.Rows()
		return rowsMsg{rows: rows, err: err}
	}
}

func (d Dashboard) loadStatuses(name string) tea.Cmd {
	loader := d.loader
	return func() tea.Msg {
		st, err := loader.Statuses(name)
		return statusesMsg{states: st, err: err}
	}
}

func (d Dashboard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case rowsMsg:
		if m.err != nil {
			d.err = m.err.Error()
		} else {
			d.err = ""
			d.rows = m.rows
			if d.cursor >= len(d.rows) {
				d.cursor = max(0, len(d.rows)-1)
			}
		}
		return d, nil
	case statusesMsg:
		if m.err != nil {
			d.err = m.err.Error()
		} else {
			d.describe = m.states
			d.mode = modeDescribe
		}
		return d, nil
	case refreshMsg:
		return d, d.loadRows()
	case tickMsg:
		if d.mode == modeList {
			return d, tea.Batch(d.loadRows(), tickCmd())
		}
		return d, tickCmd()
	case tea.KeyMsg:
		return d.handleKey(m)
	}
	if d.mode == modeEditor {
		nm, cmd := d.editor.Update(msg)
		d.editor = nm.(Editor)
		return d, cmd
	}
	return d, nil
}

func (d Dashboard) handleKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch d.mode {
	case modeEditor:
		nm, cmd := d.editor.Update(k)
		d.editor = nm.(Editor)
		if d.editor.Done() {
			if d.editor.Saved() {
				if err := d.loader.Save(d.editor.Result()); err != nil {
					d.err = err.Error()
				}
			}
			d.mode = modeList
			return d, d.loadRows()
		}
		return d, cmd
	case modeDescribe:
		d.mode = modeList
		return d, nil
	case modeConfirmDelete:
		if keyRune(k, 'y') && len(d.rows) > 0 {
			name := d.rows[d.cursor].Name
			d.mode = modeList
			if err := d.loader.Delete(name); err != nil {
				d.err = err.Error()
			}
			return d, d.loadRows()
		}
		d.mode = modeList
		return d, nil
	default:
		return d.handleListKey(k)
	}
}

func (d Dashboard) handleListKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case k.Type == tea.KeyUp || keyRune(k, 'k'):
		if d.cursor > 0 {
			d.cursor--
		}
	case k.Type == tea.KeyDown || keyRune(k, 'j'):
		if d.cursor < len(d.rows)-1 {
			d.cursor++
		}
	case keyRune(k, 'q') || k.Type == tea.KeyCtrlC:
		return d, tea.Quit
	case keyRune(k, 'n'):
		return d.openEditorNew()
	case keyRune(k, 'e'):
		return d.openEditorEdit()
	case keyRune(k, 'D'):
		if len(d.rows) > 0 {
			d.mode = modeConfirmDelete
		}
	case k.Type == tea.KeyEnter:
		if len(d.rows) > 0 {
			return d, d.loadStatuses(d.rows[d.cursor].Name)
		}
	case keyRune(k, 's'):
		if len(d.rows) > 0 {
			return d, d.loader.Exec("start", d.rows[d.cursor].Name)
		}
	case keyRune(k, 'x'):
		if len(d.rows) > 0 {
			return d, d.loader.Exec("stop", d.rows[d.cursor].Name)
		}
	case keyRune(k, 'r'):
		if len(d.rows) > 0 {
			return d, d.loader.Exec("restart", d.rows[d.cursor].Name)
		}
	}
	return d, nil
}

func (d Dashboard) openEditorNew() (tea.Model, tea.Cmd) {
	projects, err := d.loader.Projects()
	if err != nil {
		d.err = err.Error()
		return d, nil
	}
	d.editor = NewEditor(EditorOptions{
		Projects:       projects,
		InitialTimeout: config.DefaultWaitTimeout,
		NameExists:     d.loader.Exists,
	})
	d.mode = modeEditor
	return d, d.editor.Init()
}

func (d Dashboard) openEditorEdit() (tea.Model, tea.Cmd) {
	if len(d.rows) == 0 {
		return d, nil
	}
	name := d.rows[d.cursor].Name
	g, err := d.loader.Load(name)
	if err != nil {
		d.err = err.Error()
		return d, nil
	}
	projects, err := d.loader.Projects()
	if err != nil {
		d.err = err.Error()
		return d, nil
	}
	d.editor = NewEditor(EditorOptions{
		Name:           g.Name,
		NameFixed:      true,
		Projects:       projects,
		InitialMembers: g.Members,
		InitialTimeout: g.WaitTimeout.Duration(),
	})
	d.mode = modeEditor
	return d, d.editor.Init()
}

func (d Dashboard) View() string {
	if d.mode == modeEditor {
		return d.editor.View()
	}
	var b strings.Builder
	if d.mode == modeDescribe {
		fmt.Fprintf(&b, "%s\n\n", titleStyle.Render("describe"))
		for i, s := range d.describe {
			fmt.Fprintf(&b, " %d  %-20s %s\n", i+1, s.Name, statusColor(string(s.Status)))
		}
		b.WriteString(dimStyle.Render("\nany key to return"))
		return borderStyle.Render(b.String())
	}
	fmt.Fprintf(&b, "%s\n\n", titleStyle.Render("dpilot groups"))
	if len(d.rows) == 0 {
		b.WriteString("no groups yet\n")
	}
	for i, r := range d.rows {
		cursor := "  "
		if i == d.cursor {
			cursor = "> "
		}
		fmt.Fprintf(&b, "%s%-20s  members %d  running %d\n", cursor, r.Name, r.Members, r.Running)
	}
	if d.mode == modeConfirmDelete && len(d.rows) > 0 {
		fmt.Fprintf(&b, "\ndelete %q? [y/N]", d.rows[d.cursor].Name)
	} else {
		b.WriteString(dimStyle.Render("\n[s]tart [x]stop [r]estart [enter]describe [n]ew [e]dit [D]elete [q]uit"))
	}
	if d.err != "" {
		fmt.Fprintf(&b, "\n%s", d.err)
	}
	return borderStyle.Render(b.String())
}
