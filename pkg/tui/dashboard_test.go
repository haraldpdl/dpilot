package tui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

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
	nm, cmd := d.Update(runes("n"))
	d = runCmd(nm.(Dashboard), cmd)
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

// runCmd executes a command (unwrapping batches) and delivers every resulting
// message to the dashboard, so tests can observe loader calls.
func runCmd(d Dashboard, cmd tea.Cmd) Dashboard {
	if cmd == nil {
		return d
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			d = runCmd(d, c)
		}
		return d
	}
	nm, _ := d.Update(msg)
	return nm.(Dashboard)
}

func TestDashboardDeleteTargetsRowConfirmedNotCurrentCursor(t *testing.T) {
	rec := &recorder{}
	rows := []GroupRow{{Name: "a"}, {Name: "b"}, {Name: "c"}}
	d := seeded(NewDashboard(testLoader(rec, rows, nil)), rows)
	d = dsend(d, kt(tea.KeyDown), kt(tea.KeyDown), runes("D")) // confirm delete of c
	d = seeded(d, rows[:2])                                    // a refresh drops c; cursor clamps onto b
	dsend(d, runes("y"))
	if rec.deleted != "c" {
		t.Fatalf("y must delete the group named in the prompt (c), deleted %q", rec.deleted)
	}
}

func TestDashboardActionFailureIsNamedAndSticky(t *testing.T) {
	rec := &recorder{}
	rows := []GroupRow{{Name: "g"}}
	d := seeded(NewDashboard(testLoader(rec, rows, nil)), rows)
	d = dsend(d, actionDoneMsg{verb: "start", group: "g", err: errors.New("exit status 1")})
	if !strings.Contains(d.View(), `start "g" failed`) {
		t.Fatalf("failure should name the action and group, view:\n%s", d.View())
	}
	d = seeded(d, rows) // a successful refresh must not wipe it
	if !strings.Contains(d.View(), `start "g" failed`) {
		t.Fatal("action failure must survive a rows refresh")
	}
	d = dsend(d, kt(tea.KeyDown)) // any key dismisses it
	if strings.Contains(d.View(), "failed") {
		t.Fatal("a key press should clear the notice")
	}
}

func TestDashboardEditOpensEditorWithoutBlockingUpdate(t *testing.T) {
	rec := &recorder{}
	calls := 0
	rows := []GroupRow{{Name: "g"}}
	loader := testLoader(rec, rows, projs("db"))
	loader.Projects = func() ([]ddev.Project, error) { calls++; return projs("db"), nil }
	d := seeded(NewDashboard(loader), rows)
	nm, cmd := d.Update(runes("e"))
	d = nm.(Dashboard)
	if calls != 0 || d.mode == modeEditor || cmd == nil {
		t.Fatalf("e must not call ddev inside Update: calls=%d mode=%v cmd=%v", calls, d.mode, cmd != nil)
	}
	d = runCmd(d, cmd)
	if calls != 1 || d.mode != modeEditor {
		t.Fatalf("editor should open once the async load lands: calls=%d mode=%v", calls, d.mode)
	}
	if got := d.editor.Result(); got.Name != "g" || len(got.Members) != 1 || got.Members[0] != "db" {
		t.Fatalf("editor not preloaded from Load: %+v", got)
	}
}

func TestDashboardNewOpensEditorWithoutBlockingUpdate(t *testing.T) {
	rec := &recorder{}
	calls := 0
	loader := testLoader(rec, nil, projs("db"))
	loader.Projects = func() ([]ddev.Project, error) { calls++; return projs("db"), nil }
	d := seeded(NewDashboard(loader), nil)
	nm, cmd := d.Update(runes("n"))
	d = nm.(Dashboard)
	if calls != 0 || d.mode == modeEditor {
		t.Fatalf("n must not call ddev inside Update: calls=%d mode=%v", calls, d.mode)
	}
	d = runCmd(d, cmd)
	if calls != 1 || d.mode != modeEditor || d.editor.phase != phaseName {
		t.Fatalf("editor should open at the name phase: calls=%d mode=%v phase=%d", calls, d.mode, d.editor.phase)
	}
}

func TestDashboardTickSkipsRefreshWhileOneIsInFlight(t *testing.T) {
	old := refreshInterval
	refreshInterval = time.Millisecond
	defer func() { refreshInterval = old }()
	rec := &recorder{}
	calls := 0
	loader := testLoader(rec, nil, nil)
	loader.Rows = func() ([]GroupRow, error) { calls++; return nil, nil }
	d := NewDashboard(loader) // Init issues the first load; nothing has answered yet
	_, cmd := d.Update(tickMsg{})
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			if m := c(); m != (tickMsg{}) {
				calls++ // any non-tick message came from an extra Rows load
			}
		}
	} else if msg != (tickMsg{}) {
		calls++
	}
	if calls != 0 {
		t.Fatalf("tick must not start another load while one is in flight, got %d", calls)
	}
	d = seeded(d, nil) // first load answered
	_, cmd = d.Update(tickMsg{})
	runCmd(d, cmd)
	if calls != 1 {
		t.Fatalf("tick after the load returned should refresh once, got %d", calls)
	}
}

func TestDashboardCtrlCQuitsInEveryMode(t *testing.T) {
	rec := &recorder{}
	rows := []GroupRow{{Name: "g"}}
	for _, mode := range []dashMode{modeList, modeDescribe, modeConfirmDelete} {
		d := seeded(NewDashboard(testLoader(rec, rows, nil)), rows)
		d.mode = mode
		_, cmd := d.Update(kt(tea.KeyCtrlC))
		if cmd == nil {
			t.Fatalf("mode %v: ctrl+c returned no command", mode)
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Fatalf("mode %v: ctrl+c should quit", mode)
		}
	}
}

func TestProductionLoaderBoundsDdevCalls(t *testing.T) {
	var seen context.Context
	client := &ctxClient{onCall: func(ctx context.Context) { seen = ctx }}
	_, _ = ProductionLoader(client).Projects()
	if seen == nil {
		t.Fatal("client not called")
	}
	if _, ok := seen.Deadline(); !ok {
		t.Fatal("dashboard ddev calls must carry a deadline")
	}
}

type ctxClient struct{ onCall func(context.Context) }

func (c *ctxClient) List(ctx context.Context) ([]ddev.Project, error) { c.onCall(ctx); return nil, nil }
func (c *ctxClient) Describe(ctx context.Context, _ string) (*ddev.Describe, error) {
	c.onCall(ctx)
	return &ddev.Describe{}, nil
}
func (c *ctxClient) Start(ctx context.Context, _ string) error { c.onCall(ctx); return nil }
func (c *ctxClient) Stop(ctx context.Context, _ string) error  { c.onCall(ctx); return nil }

func TestSelfArgsTerminateFlags(t *testing.T) {
	if got := selfArgs("start", "-legacy"); len(got) != 3 || got[1] != "--" || got[2] != "-legacy" {
		t.Fatalf("group must follow --, got %v", got)
	}
}

func TestDashboardShowsInvalidGroupRow(t *testing.T) {
	rec := &recorder{}
	rows := []GroupRow{{Name: "broken", Error: "parse group \"broken\": yaml: unmarshal errors:\n  line 1: field bogus not found in type config.Group"}}
	d := seeded(NewDashboard(testLoader(rec, rows, nil)), rows)
	v := d.View()
	if !strings.Contains(v, "broken") || !strings.Contains(v, "invalid: yaml: unmarshal errors: line 1: field bogus") {
		t.Fatalf("invalid group should be visible on one row without its name repeated:\n%s", v)
	}
	for _, key := range []tea.KeyMsg{runes("s"), runes("x"), runes("r"), runes("e"), kt(tea.KeyEnter)} {
		nm, cmd := d.Update(key)
		if cmd != nil || rec.execVerb != "" || !strings.Contains(nm.(Dashboard).View(), "press D to delete") {
			t.Fatalf("key %v on an invalid row should explain instead of acting", key)
		}
	}
}
