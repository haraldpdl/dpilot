package tui

import (
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
	e = send(e, runes("t"))
	e.toInput.SetValue("nope")
	e = send(e, kt(tea.KeyEnter))
	if e.errMsg == "" {
		t.Fatal("expected error for invalid duration")
	}
	if e.Result().WaitTimeout.Duration() != 90*time.Second {
		t.Fatal("timeout should be unchanged after invalid input")
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
