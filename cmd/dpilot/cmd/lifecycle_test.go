package cmd

import (
	"context"
	"strings"
	"testing"

	"github.com/haraldpdl/dpilot/pkg/config"
	"github.com/haraldpdl/dpilot/pkg/ddev"
)

type recClient struct {
	started, stopped []string
	calls            []string // interleaved "start:x"/"stop:x" log, so tests can assert phase order
}

func (c *recClient) List(context.Context) ([]ddev.Project, error) { return nil, nil }
func (c *recClient) Describe(_ context.Context, name string) (*ddev.Describe, error) {
	return &ddev.Describe{Name: name, Status: ddev.StatusRunning}, nil
}
func (c *recClient) Start(_ context.Context, name string) error {
	c.started = append(c.started, name)
	c.calls = append(c.calls, "start:"+name)
	return nil
}
func (c *recClient) Stop(_ context.Context, name string) error {
	c.stopped = append(c.stopped, name)
	c.calls = append(c.calls, "stop:"+name)
	return nil
}

func TestStartCommandStartsInOrder(t *testing.T) {
	t.Setenv("DPILOT_HOME", t.TempDir())
	_ = config.Save(&config.Group{Name: "g", Members: []string{"db", "api"}})
	rec := &recClient{}
	newClient = func() ddev.Client { return rec }
	if _, err := run(t, "start", "g"); err != nil {
		t.Fatalf("start: %v", err)
	}
	if len(rec.started) != 2 || rec.started[0] != "db" || rec.started[1] != "api" {
		t.Fatalf("unexpected start order: %v", rec.started)
	}
}

func TestStopCommandStopsReverse(t *testing.T) {
	t.Setenv("DPILOT_HOME", t.TempDir())
	_ = config.Save(&config.Group{Name: "g", Members: []string{"db", "api"}})
	rec := &recClient{}
	newClient = func() ddev.Client { return rec }
	if _, err := run(t, "stop", "g"); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if len(rec.stopped) != 2 || rec.stopped[0] != "api" || rec.stopped[1] != "db" {
		t.Fatalf("unexpected stop order: %v", rec.stopped)
	}
}

func TestRestartCommandStopsReverseThenStartsInOrder(t *testing.T) {
	t.Setenv("DPILOT_HOME", t.TempDir())
	if err := config.Save(&config.Group{Name: "g", Members: []string{"db", "api"}}); err != nil {
		t.Fatal(err)
	}
	rec := &recClient{}
	orig := newClient
	newClient = func() ddev.Client { return rec }
	t.Cleanup(func() { newClient = orig })
	if _, err := run(t, "restart", "g"); err != nil {
		t.Fatalf("restart: %v", err)
	}
	if got := strings.Join(rec.calls, ","); got != "stop:api,stop:db,start:db,start:api" {
		t.Fatalf("restart must stop in reverse order, then start in order; got %q", got)
	}
}
