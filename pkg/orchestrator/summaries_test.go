package orchestrator

import (
	"context"
	"errors"
	"os"
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

func TestGroupSummariesNoGroupsSkipsDdev(t *testing.T) {
	t.Setenv("DPILOT_HOME", t.TempDir())
	got, err := GroupSummaries(context.Background(), errListClient{})
	if err != nil {
		t.Fatalf("no groups should not error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no summaries, got %v", got)
	}
}

type errListClient struct{}

func (errListClient) List(context.Context) ([]ddev.Project, error) {
	return nil, errors.New("ddev List must not be called when there are no groups")
}
func (errListClient) Describe(context.Context, string) (*ddev.Describe, error) { return nil, nil }
func (errListClient) Start(context.Context, string) error                      { return nil }
func (errListClient) Stop(context.Context, string) error                       { return nil }

func TestGroupSummariesReportsInvalidGroupsAndContinues(t *testing.T) {
	t.Setenv("DPILOT_HOME", t.TempDir())
	if err := config.Save(&config.Group{Name: "good", Members: []string{"a"}}); err != nil {
		t.Fatal(err)
	}
	dir, _ := config.Dir()
	if err := os.WriteFile(dir+"/broken.yaml", []byte("bogus: 1\nmembers: [a]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f := newFakeClient()
	f.list = []ddev.Project{{Name: "a", Status: ddev.StatusRunning}}
	got, err := GroupSummaries(context.Background(), f)
	if err != nil {
		t.Fatalf("one broken file must not fail the listing: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 summaries (broken + good), got %+v", got)
	}
	if got[0].Name != "broken" || got[0].Err == "" {
		t.Fatalf("broken group should carry its error: %+v", got[0])
	}
	if got[1].Name != "good" || got[1].Err != "" || got[1].Running != 1 {
		t.Fatalf("good group should be unaffected: %+v", got[1])
	}
}
