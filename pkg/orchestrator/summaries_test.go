package orchestrator

import (
	"context"
	"testing"

	"github.com/haraldpdl/dpilot/pkg/config"
	"github.com/haraldpdl/dpilot/pkg/ddev"
)

func TestGroupSummaries(t *testing.T) {
	t.Setenv("DPILOT_HOME", t.TempDir())
	if err := config.Save(&config.Group{Name: "g1", Members: []string{"a", "b"}}); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(&config.Group{Name: "g2", Members: []string{"c"}}); err != nil {
		t.Fatal(err)
	}
	f := newFakeClient()
	f.list = []ddev.Project{{Name: "a", Status: ddev.StatusRunning}, {Name: "c", Status: ddev.StatusRunning}, {Name: "b", Status: ddev.StatusStopped}}
	got, err := GroupSummaries(context.Background(), f)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 summaries, got %d", len(got))
	}
	if got[0].Name != "g1" || got[0].Members != 2 || got[0].Running != 1 {
		t.Fatalf("g1 summary wrong: %+v", got[0])
	}
	if got[1].Name != "g2" || got[1].Members != 1 || got[1].Running != 1 {
		t.Fatalf("g2 summary wrong: %+v", got[1])
	}
}
