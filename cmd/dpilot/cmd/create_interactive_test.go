package cmd

import (
	"testing"
	"time"

	"github.com/haraldpdl/dpilot/pkg/config"
	"github.com/haraldpdl/dpilot/pkg/tui"
)

func TestCreateInteractiveSavesEditorResult(t *testing.T) {
	t.Setenv("DPILOT_HOME", t.TempDir())
	withProjects("db", "api")
	isInteractive = func() bool { return true }
	runEditor = func(opts tui.EditorOptions) (*config.Group, error) {
		if opts.Name != "mystack" || !opts.NameFixed || len(opts.Projects) != 2 {
			t.Fatalf("editor opts not wired: %+v", opts)
		}
		return &config.Group{Name: "mystack", WaitTimeout: config.Duration(90 * time.Second), Members: []string{"api", "db"}}, nil
	}
	defer func() { isInteractive = tui.IsInteractive; runEditor = tui.RunEditor }()
	if _, err := run(t, "create", "mystack"); err != nil {
		t.Fatalf("create: %v", err)
	}
	g, err := config.Load("mystack")
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Members) != 2 || g.Members[0] != "api" || g.WaitTimeout.Duration() != 90*time.Second {
		t.Fatalf("interactive result not saved: %+v", g)
	}
}

func TestCreateInteractiveCanceledSavesNothing(t *testing.T) {
	t.Setenv("DPILOT_HOME", t.TempDir())
	withProjects("db")
	isInteractive = func() bool { return true }
	runEditor = func(tui.EditorOptions) (*config.Group, error) { return nil, nil }
	defer func() { isInteractive = tui.IsInteractive; runEditor = tui.RunEditor }()
	if _, err := run(t, "create", "mystack"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if ok, _ := config.Exists("mystack"); ok {
		t.Fatal("canceled interactive create should not save a group")
	}
}

func TestCreateNonInteractiveEmpty(t *testing.T) {
	t.Setenv("DPILOT_HOME", t.TempDir())
	withProjects("db")
	isInteractive = func() bool { return false }
	defer func() { isInteractive = tui.IsInteractive }()
	if _, err := run(t, "create", "mystack"); err != nil {
		t.Fatalf("create: %v", err)
	}
	g, err := config.Load("mystack")
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Members) != 0 {
		t.Fatalf("non-interactive create should be empty, got %v", g.Members)
	}
}
