package tui

import (
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/haraldpdl/dpilot/pkg/config"
	"github.com/haraldpdl/dpilot/pkg/ddev"
)

func projs(names ...string) []ddev.Project {
	var ps []ddev.Project
	for _, n := range names {
		ps = append(ps, ddev.Project{Name: n, Status: ddev.StatusStopped})
	}
	return ps
}

func send(e Editor, msgs ...tea.Msg) Editor {
	for _, m := range msgs {
		nm, _ := e.Update(m)
		e = nm.(Editor)
	}
	return e
}

func runes(s string) tea.KeyMsg   { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }
func kt(t tea.KeyType) tea.KeyMsg { return tea.KeyMsg{Type: t} }

func TestEditorSelectCapturesOrder(t *testing.T) {
	e := NewEditor(EditorOptions{Name: "g", NameFixed: true, Projects: projs("a", "b", "c")})
	e = send(e, kt(tea.KeyDown), kt(tea.KeyDown), kt(tea.KeySpace)) // select c
	e = send(e, kt(tea.KeyUp), kt(tea.KeyUp), kt(tea.KeySpace))     // select a
	got := e.Result().Members
	if len(got) != 2 || got[0] != "c" || got[1] != "a" {
		t.Fatalf("expected [c a], got %v", got)
	}
}

func TestEditorUnselectRenumbers(t *testing.T) {
	e := NewEditor(EditorOptions{Name: "g", NameFixed: true, Projects: projs("a", "b", "c")})
	e = send(e, kt(tea.KeySpace))                  // a
	e = send(e, kt(tea.KeyDown), kt(tea.KeySpace)) // b
	e = send(e, kt(tea.KeyDown), kt(tea.KeySpace)) // c -> [a b c]
	e = send(e, kt(tea.KeyUp), kt(tea.KeySpace))   // cursor on b, unselect
	got := e.Result().Members
	if len(got) != 2 || got[0] != "a" || got[1] != "c" {
		t.Fatalf("expected [a c], got %v", got)
	}
}

func TestEditorReorder(t *testing.T) {
	e := NewEditor(EditorOptions{Name: "g", NameFixed: true, Projects: projs("a", "b")})
	e = send(e, kt(tea.KeySpace), kt(tea.KeyDown), kt(tea.KeySpace)) // [a b], cursor on b
	e = send(e, runes("K"))                                          // move b earlier
	got := e.Result().Members
	if got[0] != "b" || got[1] != "a" {
		t.Fatalf("expected [b a], got %v", got)
	}
}

func TestEditorTimeoutParse(t *testing.T) {
	e := NewEditor(EditorOptions{Name: "g", NameFixed: true, Projects: projs("a")})
	e = send(e, runes("t")) // enter timeout phase
	e.toInput.SetValue("90s")
	e = send(e, kt(tea.KeyEnter))
	if e.Result().WaitTimeout.Duration() != 90*time.Second {
		t.Fatalf("expected 90s, got %v", e.Result().WaitTimeout.Duration())
	}
	for _, bad := range []string{"nope", "0s", "-5s"} {
		e = send(e, runes("t"))
		e.toInput.SetValue(bad)
		e = send(e, kt(tea.KeyEnter))
		if e.errMsg == "" {
			t.Fatalf("expected error for duration %q", bad)
		}
		if e.Result().WaitTimeout.Duration() != 90*time.Second {
			t.Fatalf("timeout should be unchanged after invalid input %q", bad)
		}
		e = send(e, kt(tea.KeyEsc)) // back to select phase
	}
}

func TestEditorNameUniqueness(t *testing.T) {
	e := NewEditor(EditorOptions{Projects: projs("a"), NameExists: func(n string) bool { return n == "dup" }})
	if e.phase != phaseName {
		t.Fatal("new (unfixed, empty name) editor should start at name phase")
	}
	e.nameInput.SetValue("dup")
	e = send(e, kt(tea.KeyEnter))
	if e.errMsg == "" || e.phase != phaseName {
		t.Fatal("duplicate name should error and stay on name phase")
	}
	e.nameInput.SetValue("fresh")
	e = send(e, kt(tea.KeyEnter))
	if e.phase != phaseSelect {
		t.Fatal("fresh name should advance to select phase")
	}
}

func TestEditorSaveAndCancel(t *testing.T) {
	e := NewEditor(EditorOptions{Name: "g", NameFixed: true, Projects: projs("a")})
	e = send(e, kt(tea.KeyEsc))
	if !e.Done() || e.Saved() {
		t.Fatal("esc should finish without saving")
	}
	e2 := NewEditor(EditorOptions{Name: "g", NameFixed: true, Projects: projs("a")})
	e2 = send(e2, kt(tea.KeySpace), kt(tea.KeyEnter))
	if !e2.Done() || !e2.Saved() {
		t.Fatal("enter should finish saved")
	}
	if g := e2.Result(); g.Name != "g" || len(g.Members) != 1 || g.Members[0] != "a" || g.WaitTimeout.Duration() != config.DefaultWaitTimeout {
		t.Fatalf("unexpected result: %+v", g)
	}
}

func TestEditorEditModePreloaded(t *testing.T) {
	e := NewEditor(EditorOptions{Name: "g", NameFixed: true, Projects: projs("a", "b"), InitialMembers: []string{"b", "a"}, InitialTimeout: 60 * time.Second})
	if e.phase != phaseSelect {
		t.Fatal("edit should start at select phase")
	}
	g := e.Result()
	if g.Members[0] != "b" || g.Members[1] != "a" {
		t.Fatalf("preloaded order wrong: %v", g.Members)
	}
	if g.WaitTimeout.Duration() != 60*time.Second {
		t.Fatalf("preloaded timeout wrong: %v", g.WaitTimeout.Duration())
	}
}

func TestEditorLowercaseNav(t *testing.T) {
	e := NewEditor(EditorOptions{Name: "g", NameFixed: true, Projects: projs("a", "b", "c")})
	e = send(e, runes("j"), runes("j"), kt(tea.KeySpace)) // j j -> cursor on c, select
	if got := e.Result().Members; len(got) != 1 || got[0] != "c" {
		t.Fatalf("expected [c] via lowercase j nav, got %v", got)
	}
}

func TestEditorReorderNoOpAtEnds(t *testing.T) {
	e := NewEditor(EditorOptions{Name: "g", NameFixed: true, Projects: projs("a", "b")})
	e = send(e, kt(tea.KeySpace), kt(tea.KeyDown), kt(tea.KeySpace)) // [a b], cursor on b
	e = send(e, runes("J"))                                          // b is last; move later is a no-op
	if got := e.Result().Members; got[0] != "a" || got[1] != "b" {
		t.Fatalf("J at end should be a no-op, got %v", got)
	}
	e = send(e, kt(tea.KeyUp), runes("K")) // cursor on a (first); move earlier is a no-op
	if got := e.Result().Members; got[0] != "a" || got[1] != "b" {
		t.Fatalf("K at start should be a no-op, got %v", got)
	}
}

func TestEditorReorderNoOpWhenUnselected(t *testing.T) {
	e := NewEditor(EditorOptions{Name: "g", NameFixed: true, Projects: projs("a", "b")})
	e = send(e, kt(tea.KeySpace))                        // select a; cursor on a
	e = send(e, kt(tea.KeyDown), runes("K"), runes("J")) // cursor on b (unselected); reorder no-ops
	if got := e.Result().Members; len(got) != 1 || got[0] != "a" {
		t.Fatalf("reorder on an unselected project should be a no-op, got %v", got)
	}
}

func TestEditorNoProjectsExits(t *testing.T) {
	e := NewEditor(EditorOptions{Name: "g", NameFixed: true, Projects: nil})
	if e.phase != phaseNoProjects {
		t.Fatal("no projects should start in phaseNoProjects")
	}
	e = send(e, kt(tea.KeyEnter))
	if !e.Done() || e.Saved() {
		t.Fatal("any key in phaseNoProjects should exit without saving")
	}
}

func TestEditorCtrlCCancelsInEveryPhase(t *testing.T) {
	cases := map[string]Editor{
		"select":  NewEditor(EditorOptions{Name: "g", NameFixed: true, Projects: projs("a")}),
		"name":    NewEditor(EditorOptions{Projects: projs("a")}),
		"timeout": send(NewEditor(EditorOptions{Name: "g", NameFixed: true, Projects: projs("a")}), runes("t")),
	}
	for phase, e := range cases {
		nm, cmd := e.Update(kt(tea.KeyCtrlC))
		e = nm.(Editor)
		if !e.Done() || e.Saved() {
			t.Fatalf("%s phase: ctrl+c should cancel (done=%v saved=%v)", phase, e.Done(), e.Saved())
		}
		if cmd == nil {
			t.Fatalf("%s phase: ctrl+c should return the quit command", phase)
		}
	}
}

func TestEditorNameRejectsInvalid(t *testing.T) {
	e := NewEditor(EditorOptions{Projects: projs("a")})
	e.nameInput.SetValue("-x")
	e = send(e, kt(tea.KeyEnter))
	if e.errMsg == "" || e.phase != phaseName {
		t.Fatalf("invalid name should error and stay on name phase (err=%q phase=%d)", e.errMsg, e.phase)
	}
}

func TestEditorWindowsProjectsAroundCursorAtEveryHeight(t *testing.T) {
	var names []string
	for i := range 30 {
		names = append(names, fmt.Sprintf("proj%02d", i))
	}
	for h := 9; h <= 30; h++ {
		e := NewEditor(EditorOptions{Name: "g", NameFixed: true, Projects: projs(names...)})
		e = send(e, tea.WindowSizeMsg{Width: 80, Height: h})
		for range 25 {
			e = send(e, kt(tea.KeyDown))
		}
		v := e.View()
		if got := strings.Count(v, "\n") + 1; got > h {
			t.Errorf("height %d: view has %d lines:\n%s", h, got, v)
		}
		if !strings.Contains(v, "proj25") || strings.Contains(v, "proj00") {
			t.Errorf("height %d: list should scroll with the cursor:\n%s", h, v)
		}
		e.errMsg = "invalid duration"
		if got := strings.Count(e.View(), "\n") + 1; got > h && h >= 10 {
			t.Errorf("height %d with error: view has %d lines", h, got)
		}
	}
}

func TestEditorShowsOrphanMembersFirstAndLetsUserRemoveThem(t *testing.T) {
	e := NewEditor(EditorOptions{Name: "g", NameFixed: true, Projects: projs("a", "b"), InitialMembers: []string{"a", "ghost"}})
	v := e.View()
	if !strings.Contains(v, "ghost") || !strings.Contains(v, "missing") {
		t.Fatalf("a member ddev no longer lists must be shown as missing:\n%s", v)
	}
	if strings.Index(v, "ghost") > strings.Index(v, "[1] a") {
		t.Fatalf("orphaned members should be listed first so they are seen:\n%s", v)
	}
	e = send(e, kt(tea.KeySpace)) // cursor starts on the orphan; unselect it
	if got := e.Result().Members; len(got) != 1 || got[0] != "a" {
		t.Fatalf("expected ghost removed, got %v", got)
	}
}

func TestEditorEditsGroupWithOrphansWhenDdevListsNothing(t *testing.T) {
	e := NewEditor(EditorOptions{Name: "g", NameFixed: true, Projects: nil, InitialMembers: []string{"ghost"}})
	if e.phase != phaseSelect {
		t.Fatalf("a group with members must be editable even when ddev lists no projects, phase=%d", e.phase)
	}
}
