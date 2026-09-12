package tui

import (
	"context"
	"os"
	"os/exec"
	"time"

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
	return tea.ExecProcess(exec.Command(self, selfArgs(verb, group)...), func(err error) tea.Msg {
		return actionDoneMsg{verb: verb, group: group, err: err}
	})
}

// selfArgs builds the argv for re-invoking dpilot; the group name goes after
// "--" so a legacy dash-prefixed name is never parsed as a flag.
func selfArgs(verb, group string) []string { return []string{verb, "--", group} }

// RunDashboard runs the dashboard program full-screen.
func RunDashboard(loader Loader) error {
	_, err := tea.NewProgram(NewDashboard(loader), tea.WithAltScreen()).Run()
	return err
}

// ddevTimeout bounds every ddev call the dashboard makes, so a wedged Docker
// daemon cannot hang the TUI for more than this plus the client's short kill
// grace, after which no ddev child is left running.
const ddevTimeout = 30 * time.Second

func ddevCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), ddevTimeout)
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
			ctx, cancel := ddevCtx()
			defer cancel()
			return orchestrator.New(client).Statuses(ctx, g)
		},
		Delete: config.Delete,
		Projects: func() ([]ddev.Project, error) {
			ctx, cancel := ddevCtx()
			defer cancel()
			return client.List(ctx)
		},
		Load:   config.Load,
		Save:   config.Save,
		Exists: func(n string) bool { ok, _ := config.Exists(n); return ok },
		Exec:   execAction,
	}
}

func groupRows(client ddev.Client) ([]GroupRow, error) {
	ctx, cancel := ddevCtx()
	defer cancel()
	summaries, err := orchestrator.GroupSummaries(ctx, client)
	if err != nil {
		return nil, err
	}
	rows := make([]GroupRow, 0, len(summaries))
	for _, s := range summaries {
		rows = append(rows, GroupRow{Name: s.Name, Members: s.Members, Running: s.Running, Error: s.Err})
	}
	return rows, nil
}
