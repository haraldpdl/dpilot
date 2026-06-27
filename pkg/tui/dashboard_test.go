package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/haraldpdl/dpilot/pkg/config"
	"github.com/haraldpdl/dpilot/pkg/ddev"
	"github.com/haraldpdl/dpilot/pkg/orchestrator"
)

type recorder struct {
	execVerb, execGroup string
	deleted             string
	saved               *config.Group
}

func testLoader(rec *recorder, rows []GroupRow, projects []ddev.Project) Loader {
	return Loader{
		Rows: func() ([]GroupRow, error) { return rows, nil },
		Statuses: func(string) ([]orchestrator.MemberState, error) {
			return []orchestrator.MemberState{{Name: "db", Status: ddev.StatusRunning}}, nil
		},
		Delete:   func(n string) error { rec.deleted = n; return nil },
		Projects: func() ([]ddev.Project, error) { return projects, nil },
		Load:     func(n string) (*config.Group, error) { return &config.Group{Name: n, Members: []string{"db"}}, nil },
		Save:     func(g *config.Group) error { rec.saved = g; return nil },
		Exists:   func(string) bool { return false },
		Exec:     func(verb, group string) tea.Cmd { rec.execVerb, rec.execGroup = verb, group; return nil },
	}
}

func dsend(d Dashboard, msgs ...tea.Msg) Dashboard {
	for _, m := range msgs {
		nm, _ := d.Update(m)
		d = nm.(Dashboard)
	}
	return d
}

func seeded(d Dashboard, rows []GroupRow) Dashboard {
	nm, _ := d.Update(rowsMsg{rows: rows})
	return nm.(Dashboard)
}

func TestDashboardNavAndStart(t *testing.T) {
	rec := &recorder{}
	rows := []GroupRow{{Name: "mystack", Members: 3, Running: 2}, {Name: "blog", Members: 1}}
	d := seeded(NewDashboard(testLoader(rec, rows, nil)), rows)
	dsend(d, kt(tea.KeyDown), runes("s")) // move to blog, start
	if rec.execVerb != "start" || rec.execGroup != "blog" {
		t.Fatalf("expected start blog, got %s %s", rec.execVerb, rec.execGroup)
	}
}

func TestDashboardDeleteConfirm(t *testing.T) {
	rec := &recorder{}
	rows := []GroupRow{{Name: "mystack", Members: 1}}
	d := seeded(NewDashboard(testLoader(rec, rows, nil)), rows)
	d = dsend(d, runes("D"))
	if d.mode != modeConfirmDelete {
		t.Fatal("D should enter confirm mode")
	}
	d = dsend(d, runes("y"))
	if rec.deleted != "mystack" || d.mode != modeList {
		t.Fatalf("y should delete and return to list, deleted=%q mode=%v", rec.deleted, d.mode)
	}
}

func TestDashboardEnterDescribe(t *testing.T) {
	rec := &recorder{}
	rows := []GroupRow{{Name: "mystack", Members: 1}}
	d := seeded(NewDashboard(testLoader(rec, rows, nil)), rows)
	nm, cmd := d.Update(kt(tea.KeyEnter))
	d = nm.(Dashboard)
	if cmd == nil {
		t.Fatal("enter should return a statuses-loading command")
	}
	d = dsend(d, cmd()) // deliver statusesMsg
	if d.mode != modeDescribe || len(d.describe) != 1 {
		t.Fatalf("enter should load statuses and switch to describe mode: mode=%v n=%d", d.mode, len(d.describe))
	}
}

func TestDashboardNewOpensEditorAndSaves(t *testing.T) {
	rec := &recorder{}
	d := seeded(NewDashboard(testLoader(rec, nil, projs("db"))), nil)
	d = dsend(d, runes("n"))
	if d.mode != modeEditor {
		t.Fatal("n should open the editor")
	}
	// name phase: set a fresh name, advance, select db, save
	d.editor.nameInput.SetValue("fresh")
	d = dsend(d, kt(tea.KeyEnter), kt(tea.KeySpace), kt(tea.KeyEnter))
	if rec.saved == nil || rec.saved.Name != "fresh" || len(rec.saved.Members) != 1 {
		t.Fatalf("editor save not propagated to loader: %+v", rec.saved)
	}
	if d.mode != modeList {
		t.Fatal("after editor save, should return to list mode")
	}
}

func TestDashboardTickRefreshesListOnly(t *testing.T) {
	rec := &recorder{}
	rows := []GroupRow{{Name: "mystack", Members: 1}}
	d := seeded(NewDashboard(testLoader(rec, rows, nil)), rows)
	_, cmd := d.Update(tickMsg{})
	if cmd == nil {
		t.Fatal("tick in list mode should return a command (refresh + next tick)")
	}
	d.mode = modeDescribe
	d2 := dsend(d, tickMsg{})
	if d2.mode != modeDescribe {
		t.Fatal("tick must not change mode while describing")
	}
}
