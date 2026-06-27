package cmd

import (
	"strings"
	"testing"

	"github.com/haraldpdl/dpilot/pkg/ddev"
	"github.com/haraldpdl/dpilot/pkg/tui"
)

func TestBareDpilotInteractiveLaunchesDashboard(t *testing.T) {
	called := false
	isInteractive = func() bool { return true }
	runDashboard = func(tui.Loader) error { called = true; return nil }
	defer func() { isInteractive = tui.IsInteractive; runDashboard = tui.RunDashboard }()
	newClient = func() ddev.Client { return stubClient{} }
	if _, err := run(t); err != nil {
		t.Fatalf("bare dpilot: %v", err)
	}
	if !called {
		t.Fatal("interactive bare dpilot should launch the dashboard")
	}
}

func TestBareDpilotNonInteractivePrintsHelp(t *testing.T) {
	called := false
	isInteractive = func() bool { return false }
	runDashboard = func(tui.Loader) error { called = true; return nil }
	defer func() { isInteractive = tui.IsInteractive; runDashboard = tui.RunDashboard }()
	out, err := run(t)
	if err != nil {
		t.Fatalf("bare dpilot: %v", err)
	}
	if called {
		t.Fatal("non-interactive bare dpilot must not launch the dashboard")
	}
	if !strings.Contains(out, "Usage") {
		t.Fatalf("expected help output, got %q", out)
	}
}
