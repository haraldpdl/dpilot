package orchestrator

import (
	"context"
	"errors"
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
	if got := strings.Join(f.calls, ","); got != "stop:api,stop:db,start:db,start:api" {
		t.Fatalf("expected every stop attempted (reverse order) before any start, got %q", got)
	}
}

func TestStartAbortsAfterCancellation(t *testing.T) {
	f := newFakeClient()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f.hook = func(name string) {
		if name == "db" {
			cancel() // Ctrl-C lands while db is starting
		}
	}
	f.describeSeq["db"] = []*ddev.Describe{running("db")}
	err := testOrch(f).Start(ctx, grp(120*time.Second, "db", "api"))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected a cancellation error, got %v", err)
	}
	if strings.Join(f.started, ",") != "db" {
		t.Fatalf("api must not be started after cancellation, started %v", f.started)
	}
}

func TestStopAbortsAfterCancellation(t *testing.T) {
	f := newFakeClient()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f.hook = func(name string) {
		if name == "web" {
			cancel()
		}
	}
	var out strings.Builder
	o := testOrch(f)
	o.Out = &out
	err := o.Stop(ctx, grp(0, "db", "api", "web"))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected a cancellation error, got %v", err)
	}
	if strings.Join(f.stopped, ",") != "web" {
		t.Fatalf("remaining members must not be attempted after cancellation, stopped %v", f.stopped)
	}
	if !strings.Contains(out.String(), "not stopped") || !strings.Contains(out.String(), "api") || !strings.Contains(out.String(), "db") {
		t.Fatalf("the user should be told which members were left running, got %q", out.String())
	}
	if n := strings.Count(err.Error(), "context canceled"); n != 1 {
		t.Fatalf("one cancellation should yield one error, got %d in %q", n, err)
	}
}

func TestRestartDoesNotStartAfterCancelledStop(t *testing.T) {
	f := newFakeClient()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f.hook = func(name string) { cancel() }
	err := testOrch(f).Restart(ctx, grp(120*time.Second, "db", "api"))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected a cancellation error, got %v", err)
	}
	if len(f.started) != 0 {
		t.Fatalf("restart must not start members after a cancelled stop, started %v", f.started)
	}
}

func TestReadinessWaitReturnsPromptlyOnCancellation(t *testing.T) {
	f := newFakeClient()
	f.describeSeq["db"] = []*ddev.Describe{stopped("db")} // never ready
	o := &Orchestrator{Client: f, Clock: realClock{}, Poll: 5 * time.Second, Out: io.Discard}
	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(20 * time.Millisecond); cancel() }()
	begin := time.Now()
	err := o.Start(ctx, grp(120*time.Second, "db"))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected a cancellation error, got %v", err)
	}
	if elapsed := time.Since(begin); elapsed > time.Second {
		t.Fatalf("cancellation should interrupt the poll sleep, took %v", elapsed)
	}
}
