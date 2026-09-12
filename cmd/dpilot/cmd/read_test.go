package cmd

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/haraldpdl/dpilot/pkg/config"
	"github.com/haraldpdl/dpilot/pkg/ddev"
)

type listClient struct{ list []ddev.Project }

func (c listClient) List(context.Context) ([]ddev.Project, error)             { return c.list, nil }
func (c listClient) Describe(context.Context, string) (*ddev.Describe, error) { return nil, nil }
func (c listClient) Start(context.Context, string) error                      { return nil }
func (c listClient) Stop(context.Context, string) error                       { return nil }

func TestListShowsGroups(t *testing.T) {
	t.Setenv("DPILOT_HOME", t.TempDir())
	_ = config.Save(&config.Group{Name: "mystack", Members: []string{"db", "api"}})
	newClient = func() ddev.Client {
		return listClient{list: []ddev.Project{{Name: "db", Status: ddev.StatusRunning}}}
	}
	out, err := run(t, "list")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "mystack") {
		t.Fatalf("expected mystack in output: %q", out)
	}
}

func TestListAliasL(t *testing.T) {
	t.Setenv("DPILOT_HOME", t.TempDir())
	_ = config.Save(&config.Group{Name: "mystack", Members: []string{"db"}})
	newClient = func() ddev.Client { return listClient{} }
	if _, err := run(t, "l"); err != nil {
		t.Fatalf("alias l failed: %v", err)
	}
}

func TestDescribeJSON(t *testing.T) {
	t.Setenv("DPILOT_HOME", t.TempDir())
	_ = config.Save(&config.Group{Name: "mystack", Members: []string{"db", "ghost"}})
	newClient = func() ddev.Client {
		return listClient{list: []ddev.Project{{Name: "db", Status: ddev.StatusRunning}}}
	}
	out, err := run(t, "describe", "mystack", "-j")
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Members []struct{ Name, Status string } `json:"members"`
	}
	if err := json.Unmarshal(decodeEnvelope(t, out).Raw, &payload); err != nil {
		t.Fatalf("describe -j not valid json: %v (%q)", err, out)
	}
	if payload.Members[1].Name != "ghost" || payload.Members[1].Status != "missing" {
		t.Fatalf("expected ghost missing: %+v", payload.Members)
	}
}

func TestStatusAlias(t *testing.T) {
	t.Setenv("DPILOT_HOME", t.TempDir())
	_ = config.Save(&config.Group{Name: "mystack", Members: []string{"db"}})
	newClient = func() ddev.Client { return listClient{} }
	if _, err := run(t, "status", "mystack"); err != nil {
		t.Fatalf("alias status failed: %v", err)
	}
}

func TestListJSON(t *testing.T) {
	t.Setenv("DPILOT_HOME", t.TempDir())
	_ = config.Save(&config.Group{Name: "mystack", Members: []string{"db", "api"}})
	newClient = func() ddev.Client {
		return listClient{list: []ddev.Project{{Name: "db", Status: ddev.StatusRunning}}}
	}
	out, err := run(t, "list", "-j")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Name    string `json:"name"`
		Members int    `json:"members"`
		Running int    `json:"running"`
	}
	if err := json.Unmarshal(decodeEnvelope(t, out).Raw, &rows); err != nil {
		t.Fatalf("list -j not valid json: %v (%q)", err, out)
	}
	if len(rows) != 1 || rows[0].Name != "mystack" || rows[0].Members != 2 || rows[0].Running != 1 {
		t.Fatalf("unexpected list -j (want mystack members=2 running=1): %+v", rows)
	}
}

func TestListSurvivesBrokenGroupFile(t *testing.T) {
	t.Setenv("DPILOT_HOME", t.TempDir())
	if err := config.Save(&config.Group{Name: "good", Members: []string{"db"}}); err != nil {
		t.Fatal(err)
	}
	dir, _ := config.Dir()
	if err := os.WriteFile(dir+"/broken.yaml", []byte("bogus: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	newClient = func() ddev.Client { return listClient{list: []ddev.Project{{Name: "db", Status: ddev.StatusRunning}}} }
	out, err := run(t, "list")
	if err != nil {
		t.Fatalf("list must not fail because of one broken file: %v", err)
	}
	if !strings.Contains(out, "good") || !strings.Contains(out, "broken") || !strings.Contains(out, "invalid") {
		t.Fatalf("both groups should be listed, the broken one marked invalid: %q", out)
	}
	if !strings.Contains(out, "|       1 |       1 |") {
		t.Fatalf("the good group's counts should still render: %q", out)
	}
}
