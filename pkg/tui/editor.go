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
	opts      EditorOptions
	phase     editorPhase
	name      string
	cursor    int
	order     []string
	timeout   time.Duration
	saved     bool
	errMsg    string
	nameInput textinput.Model
	toInput   textinput.Model
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
		name:      opts.Name,
		order:     append([]string(nil), opts.InitialMembers...),
		timeout:   to,
		nameInput: ni,
		toInput:   ti,
	}
	if !opts.NameFixed && opts.Name == "" {
		e.phase = phaseName
		e.nameInput.Focus()
	} else {
		e.phase = phaseSelect
	}
	return e
}

func (e Editor) Init() tea.Cmd { return textinput.Blink }

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
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return e, nil
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
			if err != nil {
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
	case phaseSelect:
		switch {
		case key.Type == tea.KeyUp || keyRune(key, 'k'):
			if e.cursor > 0 {
				e.cursor--
			}
		case key.Type == tea.KeyDown || keyRune(key, 'j'):
			if e.cursor < len(e.opts.Projects)-1 {
				e.cursor++
			}
		case key.Type == tea.KeySpace:
			if len(e.opts.Projects) > 0 {
				e.toggle(e.opts.Projects[e.cursor].Name)
			}
		case keyRune(key, 'K'):
			if len(e.opts.Projects) > 0 {
				e.move(e.opts.Projects[e.cursor].Name, -1)
			}
		case keyRune(key, 'J'):
			if len(e.opts.Projects) > 0 {
				e.move(e.opts.Projects[e.cursor].Name, 1)
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
	default:
		fmt.Fprintf(&b, "%s\n\n", titleStyle.Render("Select projects for "+e.name))
		for i, p := range e.opts.Projects {
			cursor := "  "
			if i == e.cursor {
				cursor = "> "
			}
			mark := "[ ]"
			if n := e.orderOf(p.Name); n > 0 {
				mark = fmt.Sprintf("[%d]", n)
			}
			fmt.Fprintf(&b, "%s%s %-20s %s\n", cursor, mark, p.Name, statusColor(string(p.Status)))
		}
		fmt.Fprintf(&b, "\nwait_timeout: %s\n", e.timeout)
		b.WriteString(dimStyle.Render("\nspace add/remove · K/J reorder · t timeout · enter save · esc cancel"))
	}
	if e.errMsg != "" {
		fmt.Fprintf(&b, "\n%s", e.errMsg)
	}
	return borderStyle.Render(b.String())
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
