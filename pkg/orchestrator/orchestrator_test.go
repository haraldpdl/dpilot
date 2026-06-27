package orchestrator

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/haraldpdl/dpilot/pkg/config"
	"github.com/haraldpdl/dpilot/pkg/ddev"
)

func testOrch(c ddev.Client) *Orchestrator {
	return &Orchestrator{Client: c, Clock: &fakeClock{}, Poll: time.Second, Out: io.Discard}
}

func grp(timeout time.Duration, members ...string) *config.Group {
	return &config.Group{Name: "g", WaitTimeout: config.Duration(timeout), Members: members}
}

func TestStartInOrderWaitsForReadiness(t *testing.T) {
	f := newFakeClient()
	// api becomes ready on the second poll.
	f.describeSeq["db"] = []*ddev.Describe{running("db")}
	f.describeSeq["api"] = []*ddev.Describe{stopped("api"), running("api")}
	o := testOrch(f)
	if err := o.Start(context.Background(), grp(120*time.Second, "db", "api")); err != nil {
		t.Fatalf("start: %v", err)
	}
	if strings.Join(f.started, ",") != "db,api" {
		t.Fatalf("expected order db,api, got %v", f.started)
	}
}

func TestStartFailsFastOnStartError(t *testing.T) {
	f := newFakeClient()
	f.startErr["api"] = errFor("api")
	f.describeSeq["db"] = []*ddev.Describe{running("db")}
	o := testOrch(f)
	err := o.Start(context.Background(), grp(120*time.Second, "db", "api", "web"))
	if err == nil {
		t.Fatal("expected error")
	}
	for _, s := range f.started {
		if s == "web" {
			t.Fatal("web should not start after api failed")
		}
	}
}

func TestStartTimesOut(t *testing.T) {
	f := newFakeClient()
	f.describeSeq["db"] = []*ddev.Describe{stopped("db")} // never ready
	o := testOrch(f)
	err := o.Start(context.Background(), grp(3*time.Second, "db"))
	if err == nil || !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("expected timeout error, got %v", err)
	}
}

func TestStopReverseBestEffort(t *testing.T) {
	f := newFakeClient()
	f.stopErr["api"] = errFor("api")
	o := testOrch(f)
	err := o.Stop(context.Background(), grp(0, "db", "api", "web"))
	if err == nil {
		t.Fatal("expected joined error for api")
	}
	// reverse order, and db still attempted despite api failing
	if strings.Join(f.stopped, ",") != "web,db" {
		t.Fatalf("expected stop web,db, got %v", f.stopped)
	}
}

func TestStatusesMapsMissing(t *testing.T) {
	f := newFakeClient()
	f.list = []ddev.Project{{Name: "db", Status: ddev.StatusRunning}}
	o := testOrch(f)
	states, err := o.Statuses(context.Background(), grp(0, "db", "ghost"))
	if err != nil {
		t.Fatal(err)
	}
	if states[0].Status != ddev.StatusRunning || states[1].Status != ddev.StatusMissing {
		t.Fatalf("unexpected states: %+v", states)
	}
}

func TestRestartStartsEvenIfStopErrors(t *testing.T) {
	f := newFakeClient()
	f.stopErr["api"] = errFor("api")
	f.describeSeq["db"] = []*ddev.Describe{running("db")}
	f.describeSeq["api"] = []*ddev.Describe{running("api")}
	o := testOrch(f)
	if err := o.Restart(context.Background(), grp(120*time.Second, "db", "api")); err != nil {
		t.Fatalf("restart: %v", err)
	}
	if strings.Join(f.started, ",") != "db,api" {
		t.Fatalf("expected restart to start db,api despite a stop error, got %v", f.started)
	}
}
