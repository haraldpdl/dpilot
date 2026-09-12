package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/haraldpdl/dpilot/pkg/config"
	"github.com/haraldpdl/dpilot/pkg/ddev"
	"github.com/haraldpdl/dpilot/pkg/orchestrator"
	"github.com/haraldpdl/dpilot/pkg/output"
)

type dashMode int

const (
	modeList dashMode = iota
	modeDescribe
	modeConfirmDelete
	modeEditor
)

var refreshInterval = 3 * time.Second

// GroupRow is an alias for output.GroupRow (identical fields) to avoid duplication.
type GroupRow = output.GroupRow

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

// editorReadyMsg carries the data an editor needs, loaded off the event loop.
type editorReadyMsg struct {
	opts EditorOptions
	err  error
}

type actionDoneMsg struct {
	verb, group string
	err         error
}

type tickMsg struct{}

// Dashboard is the bubbletea model for the group dashboard.
type Dashboard struct {
	loader   Loader
	mode     dashMode
	rows     []GroupRow
	cursor   int
	loading  bool   // a rows load is in flight
	busy     string // transient status shown while an editor loads
	err      string // last rows-load error; cleared by the next successful load
	notice   string // action/save/delete failure; sticky until the next key press
	describe []orchestrator.MemberState
	editor   Editor
	// pendingDelete is the group named in the confirm prompt, captured when
	// the prompt opens so a concurrent refresh cannot retarget it.
	pendingDelete string
}

// NewDashboard builds a Dashboard from a Loader. Init issues the first load,
// so the dashboard starts with a load in flight.
func NewDashboard(loader Loader) Dashboard { return Dashboard{loader: loader, loading: true} }

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

// refresh starts a rows load unless one is already running.
func (d *Dashboard) refresh() tea.Cmd {
	if d.loading {
		return nil
	}
	d.loading = true
	return d.loadRows()
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
		d.loading = false
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
			d.notice = m.err.Error()
		} else {
			d.describe = m.states
			d.mode = modeDescribe
		}
		return d, nil
	case editorReadyMsg:
		if d.busy == "" || d.mode != modeList {
			// Stale or duplicate: the user navigated away, or an editor is
			// already open. Never replace an editor the user is typing in.
			return d, nil
		}
		d.busy = ""
		if m.err != nil {
			d.notice = m.err.Error()
			return d, nil
		}
		d.editor = NewEditor(m.opts)
		d.mode = modeEditor
		return d, d.editor.Init()
	case actionDoneMsg:
		// A child killed by the user's own Ctrl-C is not a failure to report.
		if m.err != nil && !strings.Contains(m.err.Error(), "signal: interrupt") {
			d.notice = fmt.Sprintf("%s %q failed: %v", m.verb, m.group, m.err)
		}
		cmd := d.refresh()
		return d, cmd
	case tickMsg:
		if d.mode == modeList {
			cmd := d.refresh()
			return d, tea.Batch(cmd, tickCmd())
		}
		return d, tickCmd()
	case tea.KeyMsg:
		if m.Type == tea.KeyCtrlC {
			return d, tea.Quit
		}
		d.notice = ""
		return d.handleKey(m)
	}
	if d.mode == modeEditor {
		return d.updateEditor(msg)
	}
	return d, nil
}

// updateEditor forwards a message to the editor and, once it finishes, saves
// the result and returns to the list.
func (d Dashboard) updateEditor(msg tea.Msg) (tea.Model, tea.Cmd) {
	nm, cmd := d.editor.Update(msg)
	d.editor = nm.(Editor)
	if !d.editor.Done() {
		return d, cmd
	}
	if d.editor.Saved() {
		if err := d.loader.Save(d.editor.Result()); err != nil {
			d.notice = err.Error()
		}
	}
	d.mode = modeList
	cmd = d.refresh() // the editor's own quit command is deliberately dropped
	return d, cmd
}

func (d Dashboard) handleKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch d.mode {
	case modeEditor:
		return d.updateEditor(k)
	case modeDescribe:
		d.mode = modeList
		return d, nil
	case modeConfirmDelete:
		name := d.pendingDelete
		d.pendingDelete = ""
		d.mode = modeList
		if !keyRune(k, 'y') || name == "" {
			return d, nil
		}
		if err := d.loader.Delete(name); err != nil {
			d.notice = err.Error()
		}
		cmd := d.refresh()
		return d, cmd
	default:
		return d.handleListKey(k)
	}
}

func (d Dashboard) handleListKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	// A group whose file will not load cannot be started, described or
	// edited; say so instead of shelling out to a command that fails.
	if len(d.rows) > 0 && d.rows[d.cursor].Error != "" {
		if k.Type == tea.KeyEnter || keyRune(k, 's') || keyRune(k, 'x') || keyRune(k, 'r') || keyRune(k, 'e') {
			d.err = "group file is invalid: press D to delete it, or fix ~/.dpilot/groups/" + d.rows[d.cursor].Name + ".yaml"
			return d, nil
		}
	}
	switch {
	case k.Type == tea.KeyUp || keyRune(k, 'k'):
		if d.cursor > 0 {
			d.cursor--
		}
	case k.Type == tea.KeyDown || keyRune(k, 'j'):
		if d.cursor < len(d.rows)-1 {
			d.cursor++
		}
	case keyRune(k, 'q'):
		return d, tea.Quit
	case keyRune(k, 'n'):
		if d.busy != "" {
			break // an editor open is already pending
		}
		d.busy = "loading projects..."
		return d, d.openEditorNew()
	case keyRune(k, 'e'):
		if len(d.rows) > 0 && d.busy == "" {
			d.busy = "loading projects..."
			return d, d.openEditorEdit()
		}
	case keyRune(k, 'D'):
		if len(d.rows) > 0 {
			d.busy = "" // abandon a pending editor open
			d.pendingDelete = d.rows[d.cursor].Name
			d.mode = modeConfirmDelete
		}
	case k.Type == tea.KeyEnter:
		if len(d.rows) > 0 {
			d.busy = ""
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

// openEditorNew loads the ddev project list off the event loop and opens an
// editor for a new group once it arrives.
func (d Dashboard) openEditorNew() tea.Cmd {
	loader := d.loader
	return func() tea.Msg {
		projects, err := loader.Projects()
		if err != nil {
			return editorReadyMsg{err: err}
		}
		return editorReadyMsg{opts: EditorOptions{
			Projects:       projects,
			InitialTimeout: config.DefaultWaitTimeout,
			NameExists:     loader.Exists,
		}}
	}
}

// openEditorEdit loads the selected group and the ddev project list off the
// event loop and opens an editor preloaded with them.
func (d Dashboard) openEditorEdit() tea.Cmd {
	name := d.rows[d.cursor].Name
	loader := d.loader
	return func() tea.Msg {
		g, err := loader.Load(name)
		if err != nil {
			return editorReadyMsg{err: err}
		}
		projects, err := loader.Projects()
		if err != nil {
			return editorReadyMsg{err: err}
		}
		return editorReadyMsg{opts: EditorOptions{
			Name:           g.Name,
			NameFixed:      true,
			Projects:       projects,
			InitialMembers: g.Members,
			InitialTimeout: g.WaitTimeout.Duration(),
		}}
	}
}

func (d Dashboard) View() string {
	if d.mode == modeEditor {
		return d.editor.View()
	}
	if d.mode == modeDescribe {
		return describeView(d.describe)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", titleStyle.Render("dpilot groups"))
	if len(d.rows) == 0 {
		b.WriteString("no groups yet. press n to create one\n")
	}
	for i, r := range d.rows {
		cursor := "  "
		if i == d.cursor {
			cursor = "> "
		}
		if r.Error != "" {
			fmt.Fprintf(&b, "%s%-20s  invalid: %s\n", cursor, r.Name, shortError(r.Name, r.Error))
			continue
		}
		fmt.Fprintf(&b, "%s%-20s  members %d  running %d\n", cursor, r.Name, r.Members, r.Running)
	}
	if d.mode == modeConfirmDelete {
		fmt.Fprintf(&b, "\ndelete %q? [y/N]", d.pendingDelete)
	} else {
		b.WriteString(dimStyle.Render("\n[s]tart [x]stop [r]estart [enter]describe [n]ew [e]dit [D]elete [q]uit"))
	}
	if d.busy != "" {
		fmt.Fprintf(&b, "\n%s", dimStyle.Render(d.busy))
	}
	if d.err != "" {
		fmt.Fprintf(&b, "\n%s", oneLine(d.err))
	}
	if d.notice != "" {
		fmt.Fprintf(&b, "\n%s", d.notice)
	}
	return borderStyle.Render(b.String())
}
