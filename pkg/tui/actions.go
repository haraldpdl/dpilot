package tui

import (
	"context"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/haraldpdl/dpilot/pkg/config"
	"github.com/haraldpdl/dpilot/pkg/ddev"
	"github.com/haraldpdl/dpilot/pkg/orchestrator"
)

// execAction suspends the TUI, runs `dpilot <verb> <group>` attached to the
// terminal so ddev output streams, then asks the dashboard to refresh.
func execAction(verb, group string) tea.Cmd {
	self, err := os.Executable()
	if err != nil {
		self = os.Args[0]
	}
	return tea.ExecProcess(exec.Command(self, verb, group), func(err error) tea.Msg {
		return actionDoneMsg{err: err}
	})
}

// RunDashboard runs the dashboard program full-screen.
func RunDashboard(loader Loader) error {
	_, err := tea.NewProgram(NewDashboard(loader), tea.WithAltScreen()).Run()
	return err
}

// ProductionLoader wires the dashboard to the real config/ddev/orchestrator.
func ProductionLoader(client ddev.Client) Loader {
	return Loader{
		Rows: func() ([]GroupRow, error) { return groupRows(client) },
		Statuses: func(name string) ([]orchestrator.MemberState, error) {
			g, err := config.Load(name)
			if err != nil {
				return nil, err
			}
			return orchestrator.New(client).Statuses(context.Background(), g)
		},
		Delete:   config.Delete,
		Projects: func() ([]ddev.Project, error) { return client.List(context.Background()) },
		Load:     config.Load,
		Save:     config.Save,
		Exists:   func(n string) bool { ok, _ := config.Exists(n); return ok },
		Exec:     execAction,
	}
}

func groupRows(client ddev.Client) ([]GroupRow, error) {
	summaries, err := orchestrator.GroupSummaries(context.Background(), client)
	if err != nil {
		return nil, err
	}
	rows := make([]GroupRow, 0, len(summaries))
	for _, s := range summaries {
		rows = append(rows, GroupRow{Name: s.Name, Members: s.Members, Running: s.Running})
	}
	return rows, nil
}
