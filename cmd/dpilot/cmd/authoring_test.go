package cmd

import (
	"bytes"
	"context"
	"testing"

	"github.com/haraldpdl/dpilot/pkg/config"
	"github.com/haraldpdl/dpilot/pkg/ddev"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type stubClient struct{ list []ddev.Project }

func (s stubClient) List(context.Context) ([]ddev.Project, error)             { return s.list, nil }
func (s stubClient) Describe(context.Context, string) (*ddev.Describe, error) { return nil, nil }
func (s stubClient) Start(context.Context, string) error                      { return nil }
func (s stubClient) Stop(context.Context, string) error                       { return nil }

func resetFlags(c *cobra.Command) {
	c.Flags().VisitAll(func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	})
	for _, sub := range c.Commands() {
		resetFlags(sub)
	}
}

func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	resetFlags(rootCmd)
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	return buf.String(), err
}

func withProjects(names ...string) {
	var ps []ddev.Project
	for _, n := range names {
		ps = append(ps, ddev.Project{Name: n, Status: ddev.StatusStopped})
	}
	newClient = func() ddev.Client { return stubClient{list: ps} }
}

func TestCreateThenAddInOrder(t *testing.T) {
	t.Setenv("DPILOT_HOME", t.TempDir())
	withProjects("db", "api")
	if _, err := run(t, "create", "mystack"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := run(t, "add", "mystack", "db"); err != nil {
		t.Fatalf("add db: %v", err)
	}
	if _, err := run(t, "add", "mystack", "api"); err != nil {
		t.Fatalf("add api: %v", err)
	}
	g, err := config.Load("mystack")
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Members) != 2 || g.Members[0] != "db" || g.Members[1] != "api" {
		t.Fatalf("unexpected members: %v", g.Members)
	}
}

func TestAddRejectsUnknownProject(t *testing.T) {
	t.Setenv("DPILOT_HOME", t.TempDir())
	withProjects("db")
	_, _ = run(t, "create", "mystack")
	if _, err := run(t, "add", "mystack", "ghost"); err == nil {
		t.Fatal("expected error adding unknown ddev project")
	}
}

func TestAddAfterInserts(t *testing.T) {
	t.Setenv("DPILOT_HOME", t.TempDir())
	withProjects("db", "api", "cache")
	_, _ = run(t, "create", "mystack")
	_, _ = run(t, "add", "mystack", "db")
	_, _ = run(t, "add", "mystack", "api")
	if _, err := run(t, "add", "mystack", "cache", "--after", "db"); err != nil {
		t.Fatalf("add --after: %v", err)
	}
	g, _ := config.Load("mystack")
	if g.Members[1] != "cache" {
		t.Fatalf("expected cache after db, got %v", g.Members)
	}
}

func TestAddAfterUnknownAnchorErrors(t *testing.T) {
	t.Setenv("DPILOT_HOME", t.TempDir())
	withProjects("db", "api")
	_, _ = run(t, "create", "mystack")
	_, _ = run(t, "add", "mystack", "db")
	if _, err := run(t, "add", "mystack", "api", "--after", "ghost"); err == nil {
		t.Fatal("expected error when --after anchor is not a member")
	}
}

func TestRemoveMember(t *testing.T) {
	t.Setenv("DPILOT_HOME", t.TempDir())
	withProjects("db", "api")
	_, _ = run(t, "create", "mystack")
	_, _ = run(t, "add", "mystack", "db")
	_, _ = run(t, "add", "mystack", "api")
	if _, err := run(t, "remove", "mystack", "db"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	g, _ := config.Load("mystack")
	if len(g.Members) != 1 || g.Members[0] != "api" {
		t.Fatalf("unexpected members: %v", g.Members)
	}
}

func TestDeleteGroup(t *testing.T) {
	t.Setenv("DPILOT_HOME", t.TempDir())
	withProjects("db")
	_, _ = run(t, "create", "mystack")
	if _, err := run(t, "delete", "mystack", "-y"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if ok, _ := config.Exists("mystack"); ok {
		t.Fatal("group should be gone")
	}
}

func TestDeleteRefusesWithoutYes(t *testing.T) {
	t.Setenv("DPILOT_HOME", t.TempDir())
	withProjects("db")
	if _, err := run(t, "create", "mystack"); err != nil {
		t.Fatal(err)
	}
	if _, err := run(t, "delete", "mystack"); err == nil {
		t.Fatal("expected delete to refuse without -y")
	}
	if ok, _ := config.Exists("mystack"); !ok {
		t.Fatal("group should still exist after a refused delete")
	}
}

func TestAddRejectsDuplicate(t *testing.T) {
	t.Setenv("DPILOT_HOME", t.TempDir())
	withProjects("db")
	_, _ = run(t, "create", "mystack")
	if _, err := run(t, "add", "mystack", "db"); err != nil {
		t.Fatal(err)
	}
	if _, err := run(t, "add", "mystack", "db"); err == nil {
		t.Fatal("expected duplicate add to error")
	}
}

func TestCreateRejectsExisting(t *testing.T) {
	t.Setenv("DPILOT_HOME", t.TempDir())
	withProjects("db")
	if _, err := run(t, "create", "mystack"); err != nil {
		t.Fatal(err)
	}
	if _, err := run(t, "create", "mystack"); err == nil {
		t.Fatal("expected create on existing group to error")
	}
}

func TestRemoveRejectsAbsent(t *testing.T) {
	t.Setenv("DPILOT_HOME", t.TempDir())
	withProjects("db")
	_, _ = run(t, "create", "mystack")
	if _, err := run(t, "remove", "mystack", "ghost"); err == nil {
		t.Fatal("expected remove of absent member to error")
	}
}
