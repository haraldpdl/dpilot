package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/haraldpdl/dpilot/pkg/config"
	"github.com/haraldpdl/dpilot/pkg/ddev"
)

type editorPhase int

const (
	phaseName editorPhase = iota
	phaseSelect
	phaseTimeout
	phaseNoProjects
	editorDone
)

// EditorOptions configures a group editor.
type EditorOptions struct {
	Name           string
	NameFixed      bool
	Projects       []ddev.Project
	InitialMembers []string
	InitialTimeout time.Duration
	NameExists     func(string) bool
}

// Editor is the bubbletea model for creating or editing a group.
type Editor struct {
	opts  EditorOptions
	phase editorPhase
	// rows is what the picker shows: every project ddev lists, plus any
	// current member ddev no longer knows (marked missing) so it can still be
	// seen, reordered and removed.
	rows      []ddev.Project
	name      string
	cursor    int
	order     []string
	timeout   time.Duration
	saved     bool
	errMsg    string
	nameInput textinput.Model
	toInput   textinput.Model
	width     int // terminal columns, 0 = unknown
	height    int // terminal rows, 0 = unknown
}

// NewEditor builds an Editor. It starts at the name phase only when the name is
// not fixed and is empty; otherwise at the select phase.
func NewEditor(opts EditorOptions) Editor {
	to := opts.InitialTimeout
	if to == 0 {
		to = config.DefaultWaitTimeout
	}
	ni := textinput.New()
	ni.Placeholder = "group name"
	ni.SetValue(opts.Name)
	ti := textinput.New()
	ti.SetValue(to.String())
	e := Editor{
		opts:      opts,
		rows:      pickerRows(opts.Projects, opts.InitialMembers),
		name:      opts.Name,
		order:     append([]string(nil), opts.InitialMembers...),
		timeout:   to,
		nameInput: ni,
		toInput:   ti,
	}
	if len(e.rows) == 0 {
		e.phase = phaseNoProjects
	} else if !opts.NameFixed && opts.Name == "" {
		e.phase = phaseName
		e.nameInput.Focus()
	} else {
		e.phase = phaseSelect
	}
	return e
}

func (e Editor) Init() tea.Cmd { return textinput.Blink }

// pickerRows lists members ddev no longer knows first, marked missing, so the
// thing the user must act on is in view, followed by every ddev project.
func pickerRows(projects []ddev.Project, members []string) []ddev.Project {
	known := map[string]bool{}
	for _, p := range projects {
		known[p.Name] = true
	}
	var rows []ddev.Project
	for _, m := range members {
		if !known[m] {
			rows = append(rows, ddev.Project{Name: m, Status: ddev.StatusMissing})
			known[m] = true
		}
	}
	return append(rows, projects...)
}

func (e Editor) orderOf(name string) int {
	for i, n := range e.order {
		if n == name {
			return i + 1
		}
	}
	return 0
}

func (e *Editor) toggle(name string) {
	if e.orderOf(name) > 0 {
		out := e.order[:0:0]
		for _, n := range e.order {
			if n != name {
				out = append(out, n)
			}
		}
		e.order = out
		return
	}
	e.order = append(e.order, name)
}

func (e *Editor) move(name string, delta int) {
	idx := -1
	for i, n := range e.order {
		if n == name {
			idx = i
			break
		}
	}
	if idx < 0 {
		return
	}
	j := idx + delta
	if j < 0 || j >= len(e.order) {
		return
	}
	e.order[idx], e.order[j] = e.order[j], e.order[idx]
}

func (e Editor) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		e.width, e.height = ws.Width, ws.Height
		return e, nil
	}
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		switch e.phase {
		case phaseName:
			var cmd tea.Cmd
			e.nameInput, cmd = e.nameInput.Update(msg)
			return e, cmd
		case phaseTimeout:
			var cmd tea.Cmd
			e.toInput, cmd = e.toInput.Update(msg)
			return e, cmd
		}
		return e, nil
	}
	if key.Type == tea.KeyCtrlC {
		e.saved, e.phase = false, editorDone
		return e, tea.Quit
	}
	switch e.phase {
	case phaseName:
		switch key.Type {
		case tea.KeyEnter:
			name := strings.TrimSpace(e.nameInput.Value())
			if name == "" {
				e.errMsg = "name cannot be empty"
				return e, nil
			}
			if err := config.ValidateName(name); err != nil {
				e.errMsg = err.Error()
				return e, nil
			}
			if e.opts.NameExists != nil && e.opts.NameExists(name) {
				e.errMsg = fmt.Sprintf("group %q already exists", name)
				return e, nil
			}
			e.name, e.errMsg, e.phase = name, "", phaseSelect
			return e, nil
		case tea.KeyEsc:
			e.phase = editorDone
			return e, tea.Quit
		default:
			var cmd tea.Cmd
			e.nameInput, cmd = e.nameInput.Update(msg)
			return e, cmd
		}
	case phaseTimeout:
		switch key.Type {
		case tea.KeyEnter:
			d, err := time.ParseDuration(strings.TrimSpace(e.toInput.Value()))
			if err != nil || d <= 0 {
				e.errMsg = "invalid duration (try 120s, 2m)"
				return e, nil
			}
			e.timeout, e.errMsg, e.phase = d, "", phaseSelect
			return e, nil
		case tea.KeyEsc:
			e.toInput.SetValue(e.timeout.String())
			e.errMsg, e.phase = "", phaseSelect
			return e, nil
		default:
			var cmd tea.Cmd
			e.toInput, cmd = e.toInput.Update(msg)
			return e, cmd
		}
	case phaseNoProjects:
		e.phase = editorDone
		return e, tea.Quit
	case phaseSelect:
		switch {
		case key.Type == tea.KeyUp || keyRune(key, 'k'):
			if e.cursor > 0 {
				e.cursor--
			}
		case key.Type == tea.KeyDown || keyRune(key, 'j'):
			if e.cursor < len(e.rows)-1 {
				e.cursor++
			}
		case key.Type == tea.KeySpace:
			if len(e.rows) > 0 {
				e.toggle(e.rows[e.cursor].Name)
			}
		case keyRune(key, 'K'):
			if len(e.rows) > 0 {
				e.move(e.rows[e.cursor].Name, -1)
			}
		case keyRune(key, 'J'):
			if len(e.rows) > 0 {
				e.move(e.rows[e.cursor].Name, 1)
			}
		case keyRune(key, 't'):
			e.toInput.SetValue(e.timeout.String())
			e.toInput.Focus()
			e.phase = phaseTimeout
		case key.Type == tea.KeyEnter:
			e.saved, e.phase = true, editorDone
			return e, tea.Quit
		case key.Type == tea.KeyEsc || keyRune(key, 'q'):
			e.phase = editorDone
			return e, tea.Quit
		}
		return e, nil
	}
	return e, nil
}

func (e Editor) View() string {
	if e.phase == editorDone {
		return ""
	}
	var b strings.Builder
	switch e.phase {
	case phaseName:
		fmt.Fprintf(&b, "New group name:\n\n%s\n", e.nameInput.View())
	case phaseTimeout:
		fmt.Fprintf(&b, "wait_timeout:\n\n%s\n", e.toInput.View())
	case phaseNoProjects:
		b.WriteString("No ddev projects found. Press any key to exit.\n")
	default:
		fmt.Fprintf(&b, "%s\n\n", titleStyle.Render("Select projects for "+e.name))
		start, end, above, below := window(len(e.rows), e.cursor, e.rowBudget())
		if above > 0 {
			fmt.Fprintf(&b, "%s\n", dimStyle.Render(fmt.Sprintf("  … %d more above", above)))
		}
		for i, p := range e.rows[start:end] {
			cursor := "  "
			if start+i == e.cursor {
				cursor = "> "
			}
			mark := "[ ]"
			if n := e.orderOf(p.Name); n > 0 {
				mark = fmt.Sprintf("[%d]", n)
			}
			fmt.Fprintf(&b, "%s%s %-20s %s\n", cursor, mark, p.Name, statusColor(string(p.Status)))
		}
		if below > 0 {
			fmt.Fprintf(&b, "%s\n", dimStyle.Render(fmt.Sprintf("  … %d more below", below)))
		}
		fmt.Fprintf(&b, "\nwait_timeout: %s\n", e.timeout)
		b.WriteString(dimStyle.Render("\nspace add/remove · K/J reorder · t timeout · enter save · esc cancel"))
	}
	if e.errMsg != "" {
		fmt.Fprintf(&b, "\n%s", e.errMsg)
	}
	return box(b.String(), e.width)
}

// rowBudget is how many terminal rows the project list may use: the height
// minus the border, title, timeout line, footer and any error; 0 when unknown.
func (e Editor) rowBudget() int {
	if e.height <= 0 {
		return 0
	}
	fixed := 8 // border (2), title + blank, blank + wait_timeout, blank + footer
	if e.errMsg != "" {
		fixed++
	}
	return max(1, e.height-fixed)
}

// Done reports whether the editor has finished (saved or canceled).
func (e Editor) Done() bool { return e.phase == editorDone }

// Saved reports whether the editor finished with a save.
func (e Editor) Saved() bool { return e.saved }

// Result returns the group described by the editor's current state.
func (e Editor) Result() *config.Group {
	return &config.Group{
		Name:        e.name,
		WaitTimeout: config.Duration(e.timeout),
		Members:     append([]string(nil), e.order...),
	}
}

// RunEditor runs the editor as a standalone program and returns the saved group,
// or nil if the user canceled.
func RunEditor(opts EditorOptions) (*config.Group, error) {
	final, err := tea.NewProgram(NewEditor(opts)).Run()
	if err != nil {
		return nil, err
	}
	e := final.(Editor)
	if !e.saved {
		return nil, nil
	}
	return e.Result(), nil
}
