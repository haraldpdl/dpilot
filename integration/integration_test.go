//go:build integration

package integration

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/haraldpdl/dpilot/pkg/config"
	"github.com/haraldpdl/dpilot/pkg/ddev"
	"github.com/haraldpdl/dpilot/pkg/orchestrator"
)

// Requires a real ddev with a project named "dpilot-fixture" configured.
func TestRealStartDescribeStop(t *testing.T) {
	if _, err := exec.LookPath("ddev"); err != nil {
		t.Skip("ddev not installed")
	}
	t.Setenv("DPILOT_HOME", t.TempDir())
	g := &config.Group{Name: "it", WaitTimeout: config.Duration(180 * time.Second),
		Members: []string{"dpilot-fixture"}}
	if err := config.Save(g); err != nil {
		t.Fatal(err)
	}
	o := orchestrator.New(ddev.New())
	ctx := context.Background()

	if err := o.Start(ctx, g); err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(func() { _ = o.Stop(ctx, g) })
	states, err := o.Statuses(ctx, g)
	if err != nil {
		t.Fatal(err)
	}
	if len(states) == 0 {
		t.Fatal("statuses returned empty slice")
	}
	if states[0].Status != ddev.StatusRunning {
		t.Fatalf("expected running, got %q", states[0].Status)
	}
	if err := o.Stop(ctx, g); err != nil {
		t.Fatalf("stop: %v", err)
	}
}
